# mwanachama-backend-agency (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

| ID | Pri | Status | Task | Depends on |
|---|---|---|---|---|
| AG35 | Low | 📋 | 🐞 **BUG, half closed — a concurrent `Create<Type>` can still fail outright on SQLite with a raw, unclassified driver error, though it can no longer mint a duplicate code.** AG39 replaced the unguarded read-modify-write with a compare-and-set claim (`update ... where entity_type = ? and next_number = ?`, retried up to five times), which closes the half this row called a real risk rather than a demonstrated fact: the silent duplicate `Code` two writers could produce on Postgres. `TestConcurrentCreatesNeverShareACode` pins it — 60 concurrent `CreateGoal` calls across 30 iterations on a shared-cache multi-connection SQLite database, asserting no code is ever minted twice. **What is left is the failure the original probe actually saw**: 25 of those 60 writes were refused by SQLite's own locking (`database table is locked`), and the caller still sees that raw driver error rather than one of this package's sentinels. That is a property of SQLite under `file::memory:?cache=shared` with several connections, not of the counter, and Postgres serializes on the row instead — so the fix is either to classify the error (a `database is busy` sentinel every write path maps) or to retry the whole transaction, and neither is the code sequence's own business. Filed down to Low: nothing in production runs on SQLite. | — |
| AG33 | Medium | 📋 | 🐞 **BUG — `agency_design_from_spec`'s per-section key maps silently misattribute a reference when two spec entries reuse the same `key`, with zero error reported.** `mcp/composite.go`'s `apply` writes each created row straight into a plain `map[string]string` (`out.Roles[r.Key] = row.ID`, and identically for `out.Goals`/`out.Objectives`/`out.Workflows`/`out.WorkItems`, plus the local `workItemIDs`) with no check for a key already present — a second spec entry that reuses an earlier entry's `key` (a realistic authoring slip, especially for an LLM-composed spec) overwrites the first entry's id, so every later reference to that key (`objective.goal_key`, `work_item.workflow_key`/`objective_key`, `raci.role_key`) silently resolves to whichever entry was created *last*. Demonstrated end-to-end through the real MCP surface (a real `mcp.NewClient` over `httptest.NewServer`, not a direct manager call): called `agency_design_from_spec` with two Goals both keyed `"g1"` (`{"key": "g1", "title": "Goal One"}` / `{"key": "g1", "title": "Goal Two"}`) and one Objective with `"goal_key": "g1"`. Result: `IsError=false`, `structured=map[... goals:map[g1:910556ca-...] objectives:map[o1:eb3f9cc4-...] ...]` — both Goals really were created (two distinct real ids exist via `ListGoals`), but the Objective's `GoalID` resolved to Goal Two's id, not Goal One's, and nothing in the response signals the collision happened. This is the exact same loophole class independently found in `mwanachama-wakala-api`'s `POST /instances/import` the same sweep (filed there as WK23) — that endpoint's `refMap.add` has an identical unguarded overwrite — except this is the shared library's own `agency_design_from_spec` composite tool, used directly by any MCP caller (not just wakala-api's importer). Fix location: `mcp/composite.go`'s `apply`, everywhere it writes `out.<Section>[x.Key] = row.ID` (roles, goals, objectives, workflows, work items) — needs to check for an already-present key first and record a failure via `out.Errors = append(...)` instead of overwriting silently. | 2026-09-18, integration-test-sweep fleet pass, widening WK23's finding (loophole catalogue #1: a key that does not identify the row) into the shared `agency_design_from_spec` composite tool this repo owns | mwanachama-backend-agency, mwanachama-wakala-api | — | Pinning test: `mcp/mcp_test.go`'s `TestDesignFromSpec_DuplicateKeySilentlyMisattributesReference`, committed and passing — asserts the *current* (broken) behaviour (`len(errors) == 0`, `objectives[0].GoalID == goalTwoID`) with a comment stating both assertions should invert once AG33 is fixed. `go build`/`go vet`/`gofmt -l .`/`go test ./...`/`go test -tags=integration ./...` all clean. |
| AG36 | Low | 📋 | 🧹 **CLEANUP — `AgencySnapshot` is now fully dead code.** `PromoteDraft` (`draft_impl.go`) was its only writer, ever — confirmed by grep: nothing else calls `AgencySnapshotToRow` or writes to `m.tables.AgencySnapshots`. AG33–35 (AGD-006) deleted `PromoteDraft` outright, so the table still migrates (`gormstore/tables.go`'s `AutoMigrate(&AgencySnapshotRow{})`) and the model/gormstore files still exist, but nothing will ever insert a row into it again. Not removed here — AGD-006 never decided to retire the type itself, only the promote step that fed it, and AGD-005 already noted it "sounds like it solves \[versioning\] but is an empty stub" even before this. Follow-up: either retire it properly (model, gormstore, migration, `TableNames.AgencySnapshots`) the way AGD-005 retired WorkPlan/ContextSource, or repurpose it as a real per-publish audit record if that's wanted — a decision, not a mechanical fix. | — |

Cross-repo follow-ups from AGD-001/AG12 (not this repo's own board — filed
here per dev-research.md's "a row for another board is still filed here
first" convention; the AG20 and Mission/Vision follow-ups that used to be
listed here are closed — see `todo_done.md`'s `AG20-followup`/
`AGD-001-followup` rows):
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
