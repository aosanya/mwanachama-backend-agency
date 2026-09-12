# mwanachama-backend-agency (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

| ID | Pri | Status | Task | Depends on |
|---|---|---|---|---|
| AG18 | P1 | Not started | 🐞 **SECURITY — a caller holding only `agency.write` forges a record's `id` and `created_at` on create, across every `Create<Type>` in this repo (Goal/Objective/Role/Workflow/WorkItem/Department/Level/Value/Rule/Draft/WorkPlan/ContextSource/EventFlowStep/Agency).** Every `routes/<type>.go`'s `Create<Type>` handler decodes the wire body straight into the bare `models.<Type>` struct (`var in models.Goal; readJSON(r, &in); m.Create<Type>(ctx, in)`) with no field-stripping, and every manager-layer `Create<Type>` (`<type>_impl.go`) only fills `CreatedAt`/`ID` when they arrive empty (`if g.CreatedAt == "" { g.CreatedAt = models.NowRFC3339() }`; `gormstore.<Type>Row.BeforeCreate`'s `if r.ID == "" { r.ID = uuid.NewString() }`) — a caller who sends `id`/`created_at` on the wire has both honoured verbatim. Confirmed live against a booted gateway (`AGENCY_ROUTES=true`), an ALICE-equivalent session holding `agency.write`: `POST /v1/agency/goals {"id":"forged-id-123","code":"G-FORGED","title":"My Goal","created_at":"1999-01-01T00:00:00Z"}` → `201 {"id":"forged-id-123","code":"G-1",...,"created_at":"1999-01-01T00:00:00Z",...}` — `code` (the AG16 human-readable handle) IS correctly re-minted server-side and the forged `G-FORGED` is discarded, but `id` and `created_at` both survive untouched. **Threat**: any agency.write holder can (a) forge a record's own creation timestamp, corrupting the audit/provenance trail Value/Rule and AgencyPublication's whole reason for existing is to give an AI agent acting as a Role a trustworthy account of "what was decided and when" — a client-set clock field is P1 by this routine's own severity rule; (b) choose a record's own primary key, breaking the UUID-space assumption every other row in this repo relies on and opening a same-capability actor to a duplicate-key 500 by pre-claiming an id, or to a subtler identity-confusion bug if any future code ever treats these ids as unguessable. Reproduces identically through `mwanachama-backend-api-gateway` (`AGENCY_ROUTES=true`) and `mwanachama-wakala-api` (both mount this repo's own unmodified `routes.Routes`, decision #1) — not a gateway-side bug, the fix belongs here. **Also affects the `mcp/` surface** (`mwanachama-wakala-api`'s `/mcp`, AG13): `mcp/goal.go`'s `mcpGoalParams.ID` doc comment claims "required for update, ignored for create", but `mcpCreateGoal`'s `toModel()` passes `p.ID` through unconditionally to `CreateGoal` with no create/update branch — the comment is aspirational, not enforced, and the same id-forgery hole is reachable via MCP tool calls, though `created_at` is not (no such field exists on any `mcp` Create params struct, so the clock-forgery half is REST-only). Not filed as a self-test anywhere (case B — the hole reproduces; a self-test would assert today's bad behaviour and break the day this is fixed). **Shape of a fix**: in every `Create<Type>` (routes handler or manager, whichever this repo's own convention prefers), unconditionally overwrite `in.ID`/`in.CreatedAt` from server state before persisting rather than only when empty — the existing `if x == ""` guards need to become unconditional assignments, mirroring how `Code`-minting in the same functions already ignores whatever the caller sent. Cheapest single fix point: since every manager `Create<Type>` already computes/derives `CreatedAt` unconditionally-when-empty and mints `ID` in `BeforeCreate`, the fix is to zero `in.ID`/`in.CreatedAt` at the top of each manager method (or, more centrally, in `<Type>ToRow` before the row is ever built) rather than trusting the caller's value at all. | — |

Cross-repo follow-ups from AGD-001/AG12 (not this repo's own board — filed
here per dev-research.md's "a row for another board is still filed here
first" convention):
- `mwanachama-wakala-api`: AG11's `UpdateAgency`/`PATCH /agency` route is
  gone — `internal/api/http/instance_handlers.go`,
  `mcp_tools_agency.go`/`mcp_tools_composite.go`, and
  `cmd/server/selftest/selftest_create_agency_mission_vision.go` all still
  reference the removed `Mission`/`Vision` fields and need updating.
- `mwanachama-wakala-studio`: `agency-canvas.tsx`'s `IdentitySection`/
  `ALL_SECTIONS`, `agency.new.tsx`'s `DRAFT_SECTIONS`,
  `WorkspaceMenu.tsx`'s section list, and `AgencyIdentityField.tsx`/
  `api/instances.ts` all assume Name/Mission/Vision.
- `mwanachama-backend-api-kazi`: K6's RACI lookup ("who to notify for a
  WorkItem") must resolve via the new `WorkItem.ObjectiveID` →
  `ListRoleAssignmentsForObjective`, not a direct WorkItem-scoped call.
- `mwanachama-backend-api-gateway`: AG12 added routes for
  Objective/Value/Rule and moved the Objective-nested role-assignments
  read — the gateway's `TestPostmanCollectionCoversEveryRoute` gate (see
  AG10's own note below) will need matching Postman coverage added.
