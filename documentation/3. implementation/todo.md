# mwanachama-backend-agency (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

| ID | Pri | Status | Task | Depends on |
|---|---|---|---|---|
| AG18 | P1 | Not started | 🐞 **SECURITY — a caller holding `agency.write` mints their own `id` and `created_at` on any Agency record.** Every `Create<Type>` in this package (`agency_impl.go`, `department_impl.go`, `goal_impl.go`, `level_impl.go`, `objective_impl.go`, `role_impl.go`, `roleassignment_impl.go`, `rule_impl.go`, `value_impl.go`, `workflow_impl.go`, `workitem_impl.go`, `workplan_impl.go`, `contextsource_impl.go`, `eventflowstep_impl.go`) only defaults `CreatedAt`/`ID` when the field arrives empty (`if g.CreatedAt == "" { g.CreatedAt = models.NowRFC3339() }`, mirrored by each gormstore Row's `BeforeCreate`'s `if r.ID == "" { r.ID = uuid.NewString() }`) — a non-empty value the caller supplies is written verbatim. `routes/*.go`'s `Create<Type>` handlers decode the wire body straight into the bare `models.X` struct (`var in models.Level; readJSON(r, &in)`), so both fields are one JSON key away from any `POST` on this surface. Live-verified 2026-09-11 against `mwanachama-backend-api-gateway`'s `/v1/agency/...` mount (`AGENCY_ROUTES=true`): `POST /v1/agency/levels {"id":"claimed-id-123","name":"Engineering","depth":0}` → `201 {"id":"claimed-id-123",...}` (server's own UUID minting skipped entirely); `POST /v1/agency/goals {"title":"Ship reliably","created_at":"2020-01-01T00:00:00Z"}` → `201 {...,"created_at":"2020-01-01T00:00:00Z",...}` (backdated by 6+ years). `Code` is NOT affected — confirmed the claimed `"code":"G-999"` in the same Goal request was correctly ignored and server-minted `"G-1"` (AG16's `NextCode` transaction has no such `if empty` escape hatch). **Threat**: any Instance/agent holding `agency.write` (org-wide, per AG10 — every write capability holder is affected, not a cross-tenant issue) can (a) pick an attacker-chosen id for any record, opening ID-collision/confusion angles against every other row that references one by id (`WorkItem.ObjectiveID`, `RoleAssignment.RoleID`/`.ObjectiveID`, `Level.HierarchyID`-shaped refs), and (b) backdate/forward-date `created_at` on any Goal/Level/Department/Objective/Role/Workflow/WorkItem/Value/Rule/WorkPlan/ContextSource/EventFlowStep, corrupting every consumer that orders or reasons over creation time (draft/publish content hashing, `ListXForY` ordering, any future audit narrative). **Exposure is not limited to this repo** — `mwanachama-wakala-api` mounts this same `routes/` package verbatim per its own `internal/agencies.Registry` Instance (its CLAUDE.md, WK5), so every registered Instance there inherits the identical hole; `mwanachama-backend-agency`'s own `mcp/` package (AG13) is **not** affected — its tool params are hand-written (`mcpCreateLevelParams{Name, Depth}` etc.), never an embedded `models.X`, so `id`/`created_at` never reach the manager from that surface. **Fix shape**: each `Create<Type>` should unconditionally overwrite `ID`/`CreatedAt` (and `LastUpdated`) from server state before the row is built, the same unconditional-overwrite `Update<Type>` already applies to `Code` (AG16) — never a `readJSON`-level allowlist, since every `Create<Type>` shares the same bare-struct-decode shape. Not fixed here — filing only, per this run's scope (security QA does not patch product code). | |

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
