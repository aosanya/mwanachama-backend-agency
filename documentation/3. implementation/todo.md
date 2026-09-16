# mwanachama-backend-agency (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

| ID | Pri | Status | Task | Depends on |
|---|---|---|---|---|
| AG28 | P1 | 🚀 | Publication carries the whole Agency, and `Status` becomes `Channel`, per [AGD-005](../2.%20design/todo_details/AGD-005.md) decisions #1–#4. (a) `AgencyPublication` gains a document field holding the full Agency serialized in `mwanachama-wakala-api`'s import/export shape (its decision #19 — reuse that shape exactly so publish↔import round-trips; the 52 mined templates are already written in it). `PublishAgency` serializes the working set into it; `ContentHash` stays as the no-op-publish guard. (b) `Status` → `Channel`: a name (`"production"`, `"staging"`, …) or `""` for archived, **unique across tags** — several tags may be live at once, at most one per channel. `PublishAgency` takes the target channel (default `"production"`) and archives whichever tag held it. (c) Relax `UpdatePublicationStatus` — today it allows only one-way `active → archived`; re-living an archived tag is the rollback path (AGD-005 #4), so it must allow setting a channel back. (d) Retire the word "live" for the `draft_id IS NULL` row set across doc comments, `mcp/catalog.go`, routes and the Studio — it is the **working set**; "live" now names a channel on a tag. Existing publications keep `Channel: ""` and stay readable as history; they cannot be back-filled with a document (the rows they described have moved on). | — |
| AG29 | P1 | 🚀 | `CreateDraft` can fork from a published tag, per [AGD-005](../2.%20design/todo_details/AGD-005.md) decision #5. `forkedFromType` gains a third value alongside `"live"`/`"draft"` — fork from a tag, deserialising that publication's document back into draft rows (the inverse of AG28's serializer, and the same remapping `instance_import.go` already does: ids and Codes in the document are reference keys, remapped onto freshly-minted rows). This is what makes "go back to what v2 was and change one thing" possible, and it pairs with re-living an archived tag rather than replacing it. Sequence after AG28 — there is no document to fork from until it lands. | AG28 |
| AG30 | P1 | 🚀 | Decommission `WorkPlan` and `ContextSource` outright — remove the code and drop the tables, per [AGD-005](../2.%20design/todo_details/AGD-005.md) decision #6 ("they are currently useless", user's call). Delete `models/workplan.go`/`contextsource.go`, `gormstore/workplan.go`/`contextsource.go` + their `TableNames` entries (with a migration dropping the tables), `workplan_impl.go`/`contextsource_impl.go`, the 8 `AgencyManager` methods (`CreateWorkPlan`/`GetWorkPlan`/`ListWorkPlans`/`UpdateWorkPlan`/`MatchWorkPlans`/`AddContextSource`/`ListContextSources`/`RemoveContextSource`), `routes/contextsource.go` + the WorkPlan routes, `mcp/workplan.go`, both filter structs, the `ErrWorkPlanNotFound`/`ErrInvalidRegex`/`ErrContextSourceNotFound` sentinels, both `mcp/catalog.go` entries, and the tests. Cross-repo: `mwanachama-wakala-studio`'s `search-modules.ts` `work_plan` entry and `ShortcutsMenu.tsx:25`'s exclusion of it both go (it is already hidden there); `mwanachama-backend-api-gateway`'s `selftest_agency_routes_3.go` and its Postman coverage shrink. **`mwanachama-backend-taskmanager/events.go:183`'s `WorkPlanID`/`WorkPlanCode` event-payload fields become dead** — they reference the concept but never call this library; file a row on that repo's own board rather than editing it from here. `EventFlowStep` is explicitly **out** of scope and stays (AGD-005 #7). | — |

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
  **AG22 adds six more** (its new `/{id}/attachments` sub-resource for
  both Value and Rule): 23 uncovered routes now, was 17. Same deferral.
