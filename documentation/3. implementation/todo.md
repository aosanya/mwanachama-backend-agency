# mwanachama-backend-agency (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

| ID | Pri | Status | Task | Depends on |
|---|---|---|---|---|
| AG35 | Medium | 📋 | 🐞 **BUG — `NextCode` (`gormstore/codesequence.go`) reads the counter row then writes `next_number+1` with no CAS guard and no row lock, so two truly concurrent `Create<Type>` calls for the same entity type can fail outright with a raw, unclassified database error instead of one of them simply succeeding after the other.** `NextCode` does `SELECT ... WHERE entity_type = ?` (no `FOR UPDATE`/lock) to read `n`, computes `n+1` in Go, then `UPDATE ... SET next_number = <literal n+1> WHERE entity_type = ?` — no `WHERE next_number = ?` precondition, so nothing detects or serializes a second concurrent reader of the same `n`. Demonstrated directly against `AgencyManager.CreateGoal` (every `Create<Type>` shares this same helper via its own transaction — `goal_impl.go:24-32`), firing 2 goroutines concurrently per iteration across 30 iterations on a shared-cache multi-connection in-memory sqlite database (`file::memory:?cache=shared`, `SetMaxOpenConns(4)` — the plain `:memory:`+`SetMaxOpenConns(1)` pattern this repo's own tests otherwise use fully serializes `CreateGoal`'s whole transaction per connection and cannot expose this race at all, since only one transaction can even begin at a time). Real output: **15/30 iterations (50%) had one of the two concurrent `CreateGoal` calls fail** with `CreateGoal: NextCode: database table is locked: database is deadlocked (6)` — a raw SQLite driver error propagated completely unwrapped to the caller, not any of this repo's own sentinels, for an operation (creating two unrelated Goals) that has no legitimate reason to fail at all. Two things worth separating: (1) **demonstrated fact** — under sqlite's own locking model, this missing guard causes real, frequent (50% in this probe), ungraceful failures of ordinary concurrent writes, for every one of the ~9 entity types `NextCode` mints a Code for (Goal/Objective/Role/Workflow/WorkItem/Value/Rule/Department/Level); (2) **not demonstrated, but a real risk from reading the code** — on Postgres, an `UPDATE` with no `WHERE next_number = ?` precondition does not re-validate what it read before writing, so the same unguarded gap plausibly manifests there as a *silent duplicate Code* (two rows sharing e.g. `"G-5"`) rather than an error, since Postgres's row-lock semantics would let the second writer proceed once the first commits rather than detecting a cross-transaction conflict the way sqlite's shared-cache locking incidentally does — this repo's own tests never stand up real Postgres, so this half is flagged as risk, not filed as a separate confirmed finding. Same unguarded-counter shape actor's own ACT1 board row already flagged in its sibling `codesequence.go` as "unreachable today" (no `Unique` attribute exists there yet to race on) — this repo's version is the opposite: reachable on every single `Create<Type>` call. Fix location: `gormstore/codesequence.go`'s `NextCode` — either a CAS write (`UPDATE ... SET next_number = ? WHERE entity_type = ? AND next_number = ?`, retried on `RowsAffected == 0`) or wrap the read+write in a row lock (`SELECT ... FOR UPDATE` on Postgres; sqlite has no direct equivalent, which is exactly why this shows up as a locking error there instead). Found 2026-09-21 by the fleet integration-test sweep, widening actor's ACT1 finding (`gormstore/codesequence.go`'s identical unguarded read-modify-write, noted there as unfiled/unreachable) into agency's own copy of the same helper, which — unlike actor's — is reachable on every Create call. No committed pinning test — the probe needed a non-standard shared-cache multi-connection sqlite setup (`file::memory:?cache=shared`, `SetMaxOpenConns(4)`) that diverges from this repo's own single-connection `newTestManager`/`newTestDB` convention used everywhere else, and its result (a 50% failure *rate*, not a deterministic one-shot reproduction) is a probe finding rather than a stable regression-guard shape; the real output quoted above is the reproduction. Probe file (`zz_probe_nextcode_race_test.go`) deleted per the sweep's own rule against leaving a probe file behind. | — |
| AG34 | High | 📋 | 🐞 **BUG — `UpdateRole` can report `200 OK` for a Role that `DeleteRole` concurrently removed, returning a "successfully renamed" object for a row that no longer exists.** `role_impl.go`'s `UpdateRole` reads the row (`findRoleRow`), checks `existing.DraftID != nil` ("is this draft-scoped, i.e. editable"), then — outside any transaction, with no lock held across that gap — runs a separate `Updates(cols).Where("id = ?", r.ID)`. `DeleteRole` does the identical read-check-then-write, wrapping only its own delete in a transaction. Nothing serializes the two: `DeleteRole`'s transaction can commit in the window between `UpdateRole`'s read and its `Updates` call, so `UpdateRole`'s `Where("id = ?", ...)` matches zero rows — GORM reports `RowsAffected: 0` with `err == nil` for that, so `UpdateRole` returns its usual `200` with the caller's requested `models.Role` echoed back, even though the row is gone. Demonstrated over the real HTTP route table (`routes.Routes`, real `http.ServeMux` + `httptest.NewServer`, real `http.Client` PATCH/DELETE/GET — not a direct manager call): created a draft-scoped Role, fired concurrent `PATCH /roles/{id}` and `DELETE /roles/{id}`, then `GET /roles/{id}` whenever the PATCH answered `200`. Real output: `UpdateRole returned 200 concurrently with DeleteRole succeeding, and the role no longer exists afterward: 91/200 iterations`. This is the same TOCTOU shape already filed and pinned elsewhere in the fleet — `mwanachama-backend-taskmanager`'s W19/W20/W21 and `mwanachama-backend-assetmanager`'s A14 — now confirmed present in this repo too. **The identical unguarded read-check-then-write shape recurs on every one of this repo's other Update/Delete pairs** (`UpdateGoal`/`DeleteGoal`, `UpdateObjective`/`DeleteObjective`, `UpdateWorkflow`/`DeleteWorkflow`, `UpdateWorkItem`/`DeleteWorkItem`, `UpdateRoleAssignment`/`DeleteRoleAssignment`, `UpdateValue`/`DeleteValue`, `UpdateRule`/`DeleteRule` — each `*_impl.go`) — Role was chosen for the demonstration because it is the one pair guarded by a real precondition (`ErrCannotEditLiveRACI`) reachable over a real REST route on both sides (`PATCH`/`DELETE /roles/{id}`); the others are exposed to the same class but weren't each individually driven this pass. Fix location: give each Update/Delete pair a shared lock across the check-then-write gap — e.g. wrap `Update<Type>` in the same kind of transaction `Delete<Type>` already uses and re-check existence/scope inside it immediately before the `Updates` call, or add a `RowsAffected == 0` check to every `Update<Type>` and return a not-found sentinel instead of silently reporting success. Pinned by `TestPinsAG34_UpdateRoleCanReport200OverARoleDeleteConcurrentlyRemoved` in `routes/ag34_role_update_delete_race_test.go`, asserting current (broken) behavior (`staleUpdateSucceeded > 0`) — goes red once the race is closed. Not fixed here per this sweep's rule (file + pin only). | — |
| AG33 | Medium | 📋 | 🐞 **BUG — `agency_design_from_spec`'s per-section key maps silently misattribute a reference when two spec entries reuse the same `key`, with zero error reported.** `mcp/composite.go`'s `apply` writes each created row straight into a plain `map[string]string` (`out.Roles[r.Key] = row.ID`, and identically for `out.Goals`/`out.Objectives`/`out.Workflows`/`out.WorkItems`, plus the local `workItemIDs`) with no check for a key already present — a second spec entry that reuses an earlier entry's `key` (a realistic authoring slip, especially for an LLM-composed spec) overwrites the first entry's id, so every later reference to that key (`objective.goal_key`, `work_item.workflow_key`/`objective_key`, `raci.role_key`) silently resolves to whichever entry was created *last*. Demonstrated end-to-end through the real MCP surface (a real `mcp.NewClient` over `httptest.NewServer`, not a direct manager call): called `agency_design_from_spec` with two Goals both keyed `"g1"` (`{"key": "g1", "title": "Goal One"}` / `{"key": "g1", "title": "Goal Two"}`) and one Objective with `"goal_key": "g1"`. Result: `IsError=false`, `structured=map[... goals:map[g1:910556ca-...] objectives:map[o1:eb3f9cc4-...] ...]` — both Goals really were created (two distinct real ids exist via `ListGoals`), but the Objective's `GoalID` resolved to Goal Two's id, not Goal One's, and nothing in the response signals the collision happened. This is the exact same loophole class independently found in `mwanachama-wakala-api`'s `POST /instances/import` the same sweep (filed there as WK23) — that endpoint's `refMap.add` has an identical unguarded overwrite — except this is the shared library's own `agency_design_from_spec` composite tool, used directly by any MCP caller (not just wakala-api's importer). Fix location: `mcp/composite.go`'s `apply`, everywhere it writes `out.<Section>[x.Key] = row.ID` (roles, goals, objectives, workflows, work items) — needs to check for an already-present key first and record a failure via `out.Errors = append(...)` instead of overwriting silently. | 2026-09-18, integration-test-sweep fleet pass, widening WK23's finding (loophole catalogue #1: a key that does not identify the row) into the shared `agency_design_from_spec` composite tool this repo owns | mwanachama-backend-agency, mwanachama-wakala-api | — | Pinning test: `mcp/mcp_test.go`'s `TestDesignFromSpec_DuplicateKeySilentlyMisattributesReference`, committed and passing — asserts the *current* (broken) behaviour (`len(errors) == 0`, `objectives[0].GoalID == goalTwoID`) with a comment stating both assertions should invert once AG33 is fixed. `go build`/`go vet`/`gofmt -l .`/`go test ./...`/`go test -tags=integration ./...` all clean. |
| AG36 | Low | 📋 | 🧹 **CLEANUP — `AgencySnapshot` is now fully dead code.** `PromoteDraft` (`draft_impl.go`) was its only writer, ever — confirmed by grep: nothing else calls `AgencySnapshotToRow` or writes to `m.tables.AgencySnapshots`. AG33–35 (AGD-006) deleted `PromoteDraft` outright, so the table still migrates (`gormstore/tables.go`'s `AutoMigrate(&AgencySnapshotRow{})`) and the model/gormstore files still exist, but nothing will ever insert a row into it again. Not removed here — AGD-006 never decided to retire the type itself, only the promote step that fed it, and AGD-005 already noted it "sounds like it solves \[versioning\] but is an empty stub" even before this. Follow-up: either retire it properly (model, gormstore, migration, `TableNames.AgencySnapshots`) the way AGD-005 retired WorkPlan/ContextSource, or repurpose it as a real per-publish audit record if that's wanted — a decision, not a mechanical fix. | — |

✅ **The AG20 cross-repo follow-up is closed.** It asked
`mwanachama-wakala-studio` for a hand-written Issues dashboard section
rather than a generic `OTHER_MODULES`/`SEARCH_MODULES` fallback row.
Verified 2026-09-22 by `/compile-todos`: `IssuesPanel` is rendered from
`AgencyDashboardDetail.tsx` under its own `'issues'` panel kind, reached
from a dedicated Issues row in `PublishingMenu.tsx` calling
`dashboard.openIssues()`, and `issue` no longer appears in
`search-modules.ts` at all — exactly the shape this asked for. Trimmed
rather than left reading as open work.

Cross-repo follow-ups from AGD-001/AG12 (not this repo's own board — filed
here per dev-research.md's "a row for another board is still filed here
first" convention):
- ✅ **The two Mission/Vision follow-ups are closed** (wakala-api's
  handlers/MCP tools/self-test, and the Studio's identity UI) — closed by
  AG13 and verified again 2026-09-22 by `/compile-todos`: the named files
  `mcp_tools_agency.go` and `selftest_create_agency_mission_vision.go` no
  longer exist, and every surviving `Mission`/`Vision` match in either repo
  is a historical doc comment recording that the fields *were* removed.
  Trimmed from this list rather than left reading as open work.
- `mwanachama-backend-api-kazi`: K6's RACI lookup ("who to notify for a
  WorkItem") must resolve via the new `WorkItem.ObjectiveID` →
  `ListRoleAssignmentsForObjective`, not a direct WorkItem-scoped call.
  ⚠️ **Re-checked 2026-09-22 by `/compile-todos`, the third consecutive
  compile to do so: neither `ListRoleAssignmentsForObjective` nor
  `ObjectiveID` appears in any non-test `.go` file in that repo at all.**
  There is no WorkItem-scoped RACI call there to migrate — so this bullet
  may describe work that was never built rather than work left undone.
  That repo's owner should either scope it as a real board row or drop it;
  it has been carried forward unchanged since AG12.
- `mwanachama-backend-api-gateway`: this repo's route changes keep landing
  on that repo's two Postman gates, which have been failing since AG12 and
  are now failing in **both** directions. Measured there 2026-09-22 by
  `/compile-todos` (`go test ./internal/api/http/... -run
  'TestPostmanCollectionCoversEveryRoute|TestPostmanCollectionHasNoOrphanRequests'`),
  superseding the running "17 → 23 uncovered" tally this bullet used to
  carry — **do not trust an uncovered-route count in this prose without
  re-running that command**, it has drifted every compile:
  - `TestPostmanCollectionCoversEveryRoute`: **29 uncovered of 361
    registered routes** (597 requests parsed). The uncovered count has not
    moved since the previous measurement, but the route total fell from
    373 to 361 — routes were removed without the collection following.
  - `TestPostmanCollectionHasNoOrphanRequests`: **16 orphan requests**, up
    from 1. AGD-006/AG33–35 removed the Department/Level surface and
    `POST /drafts/{draftID}/promote`, so the whole
    `/v1/agency/departments*`, `/v1/agency/levels*`,
    `/v1/agency/work-items/{id}/role-assignments` and promote family in
    `operator-capabilities.postman_collection.json` now points at routes
    that no longer exist. **These need deleting, not covering** — the
    opposite remedy from the uncovered half, which is why they are worth
    separating here.
  Still not fixed from this repo, same reasoning as AG12's original
  deferral — but the orphan half is new damage caused by AGD-006, not the
  pre-existing coverage gap, and is mechanical to clear.

Cross-repo follow-ups from AGD-006 (not this repo's own board — filed here
per dev-research.md's "a row for another board is still filed here first"
convention; none of these are closed by AG33–AG35):
- `mwanachama-wakala-api`: any HTTP handler or MCP tool that calls a
  `List*`/`Get*`/`Update*`/`Delete*` with a bare `""` `draft_id` expecting
  "the live scope" needs a real Draft id instead — grep
  `draft_id.*""`/`DraftID: ""` across `internal/api/http` and any
  `mcp_tools_*.go` that still exists there. `PromoteDraft`'s REST/MCP
  surface is gone; anything calling it needs removing, not just updating.
  **Now filed on that repo's own board as WK29** (2026-09-21, 🚀) with the
  concrete symptom attached — 7 of its `internal/api/http` tests fail at
  HEAD with `a draft id is required` — so track it there, not here.
- `mwanachama-wakala-studio`: this is the actual fix for the "give me an
  Edit button on the drafts page" ask that led to this design doc —
  `app/api/agency.ts`'s `getAgency`/`createRole`/`updateRole` read/write
  the (now-nonexistent) live scope with no `draft_id` parameter at all;
  `app/routes/agency-canvas.tsx` has no notion of "which draft" in its
  loader/action; `app/routes/agency-drafts.tsx` has no Edit link per row.
  Once AG33/AG34 land, there is no "edit live" case left to special-case —
  every edit is draft-scoped, so this is a straightforward threading job,
  not a UI decision. ⚠️ **The Publish half of this bullet is out of date**:
  it says the Studio's Publish button is "still a hardcoded `disabled`
  placeholder per AGD-004's own note", but WKS37 landed 2026-09-16 and
  replaced that placeholder with a real `PublishPanel` that publishes end
  to end (and WKS39 added a Tags panel beside it). The threading job on
  `agency.ts`/`agency-canvas.tsx`/`agency-drafts.tsx` is the part still
  open.
- `mwanachama-backend-api-kazi`/`mwanachama-backend-api-shared`: K6's RACI
  lookup and the approval-gate `resolveAccountable`
  (`mcp_tools_approval.go`) both currently resolve against the live scope
  (`draft_id ""`) at request time. AGD-005 decision #2 already said
  production should read the published tag's JSON document, not rows —
  this task makes that the *only* option, since there's no live row-scope
  left to fall back on. Needs its own design pass per AGD-005's own
  "consumer cutover... needs its own pass per consumer" note; not
  mechanical.
- `mwanachama-backend-api-gateway`: boot-time module gating
  (`Module.Enabled`, AGD-002) reads rows directly too — same cutover
  question as api-kazi/api-shared above, and the same "not mechanical"
  caveat. Separately, AG33–35 removed the top-level `GET /issues` route
  (`routes/issue.go`) — there was never a real "live" scope left to scan
  once every row must belong to a Draft, so the route was dead weight.
  Only the nested `GET /drafts/{draftID}/issues` remains. The gateway's
  Postman collection has a request for the removed route, which now fails
  `TestPostmanCollectionHasNoOrphanRequests` — ⚠️ **but "one request" is an
  understatement**: measured 2026-09-22, that gate reports **16** orphan
  requests, because AGD-006 also removed the whole Department/Level surface
  and `POST /drafts/{draftID}/promote`, whose requests are all still in
  `operator-capabilities.postman_collection.json`. See the AG12 gateway
  bullet above for the full current numbers. All 16 want deleting rather
  than covering.
