# mwanachama-backend-agency (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

| ID | Pri | Status | Task | Depends on |
|---|---|---|---|---|
| AG21 | P1 | 📋 Not started | 🐞 **SECURITY — every `routes/Create<Type>` handler decodes the request body straight into the domain struct (`var in models.Goal`, etc.) and hands it to the manager verbatim, so a caller who legitimately holds write access can forge `created_at` (clock forgery) and, for every Draft-scoped type, `draft_id` (attaching a "live" write to a Draft that never went through `CreateDraft`'s fork).** Confirmed by reading every `Create<Type>` in `routes/` (`agency.go`, `department.go`, `draft.go`, `goal.go`, `level.go`, `objective.go`, `role.go`, `roleassignment.go`, `rule.go`, `value.go`, `workflow.go`, `workitem.go`, `workplan.go` — 13 handlers, all `readJSON(r, &in)` into the raw domain struct) and the corresponding `*_impl.go` `Create<Type>` methods: e.g. `goal_impl.go`'s `CreateGoal` only defaults `CreatedAt` `if g.CreatedAt == ""` — a caller-supplied non-empty value is written unchanged — and `g.DraftID`/`row := gormstore.GoalToRow(g)` carries the caller's `DraftID` straight through with no scope check. `Code` is the one field already correctly server-owned (`gormstore.NextCode` unconditionally overwrites it), proving the fix is a known, already-applied pattern here, just not extended to the other server-owned fields. Not this repo's own finding originally — reproduced and root-caused here by a scheduled cross-repo security-QA sweep after `mwanachama-wakala-api`'s own sweep (2026-09-12) live-verified the forgery against that repo's `POST /instances/{id}/goals` and filed it as that repo's WK17, correctly naming this package's `routes/` as the actual fix location rather than guarding it downstream. This library is mounted by at least two consumers reachable the same way: `mwanachama-wakala-api` (WK17, any signed-in caller, decision #13's "no per-tenant isolation" already accepted — this is not that, it holds regardless of who's calling) and `mwanachama-backend-api-gateway` (AG10, `/v1/agency/...` behind `AGENCY_ROUTES=true`, gated on `CapAgencyWrite` — that gateway's own security sweep run 61 already confirmed the *authorization* boundary holds there (403 for a caller without `agency.write`), but never tested field-forgery by a caller who *does* hold it, since that's a different property). **`Update<Type>` was re-read while filing this row, not left as an assumption**: `UpdateGoal` (`goal_impl.go`) correctly resets `CreatedAt`/`Code` from the existing row before writing, but its explicit `cols` map still includes `"draft_id": row.DraftID` from the caller-supplied struct — so a caller who can Update a live Goal can also move it into or out of a Draft's scope after the fact, the Update-time sibling of Create's forgery, not yet checked across the other 12 `Update<Type>` methods for the same pattern. Fix: clear/ignore `ID`/`CreatedAt` (and `DraftID` specifically on Create — Update legitimately needs to move scope only via the Draft fork/promote lifecycle, never a raw client-supplied value) from the decoded body in every `Create<Type>` handler before calling the manager, mirroring `Code`'s existing server-side-overwrite pattern, and drop `draft_id` from `Update<Type>`'s writable column set wherever it's present today. | — |

Cross-repo follow-up for AG20 (done — see [todo_done.md](todo_done.md)),
filed here per dev-research.md's "a row for
another board is still filed here first" convention:
- `mwanachama-wakala-studio`: now that AG20's `agency_list_issues` exists,
  give Issues their own hand-written dashboard section (like
  `ShortcutsMenu.tsx`'s Roles/Goals/Objectives buttons and
  `AgencyDashboardDetail.tsx`'s matching panels) — explicitly **not** a
  generic `OTHER_MODULES`/`SEARCH_MODULES` row, which is what every other
  non-hand-written module falls back to today.

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
  **AG20 adds one more route to the same already-failing list**: `GET
  /v1/agency/drafts/{draftID}/issues` (confirmed via `go test
  ./internal/api/http/...` in that repo — 17 uncovered routes now, was 16
  before AG20; `TestPostmanCollectionHasNoOrphanRequests`'s one orphan
  request is unrelated, predates AG20 too). Not fixed here, same reasoning
  as AG12's own deferral above — this is that same pre-existing gap, not a
  new one.
