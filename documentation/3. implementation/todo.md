# mwanachama-backend-agency (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

| ID | Pri | Status | Task | Depends on |
|---|---|---|---|---|
| AG22 | 📋 | Not Started | Value/Rule multi-attach: Role/Goal/Objective/Workflow/WorkItem, live-editable attachments | — |

AG22 implements [AGD-003](../2.%20design/todo_details/AGD-003.md) — widen
`Value`/`Rule` scope from Objective-only to all five content types, and
allow attaching to more than one target at once via a new join-table
layer instead of `ValueRow`/`RuleRow`'s current `ObjectiveID` column.
Attach/detach is live-editable, not Draft-gated (unlike a `Value`/`Rule`'s
own `Text`, which keeps today's strict fork→edit→promote rule). See the
design doc's "Open, not decided here" section for the real unresolved
calls (ten narrow join tables vs. two polymorphic ones; existence
validation; write-authorization; create-time convenience; List
filtering) — pick those up when this row is actually started.

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
