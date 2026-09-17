# mwanachama-backend-agency (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

| ID | Pri | Status | Task | Depends on |
|---|---|---|---|---|
| AG32 | Medium | 📋 | 🐞 **BUG — every `List*` filter built on `likeFilter` treats a caller-supplied `%`/`_` as a SQL LIKE wildcard instead of a literal character, so a one-character filter value can match every row instead of none/one.** `draft_impl.go`'s `likeFilter(q, column, value)` builds its pattern as `"%"+strings.ToLower(value)+"%"` with no escaping of `%`/`_` already present in `value` — both are live SQL `LIKE` metacharacters (`_` matches any one character, `%` matches any run), so a filter value that happens to be exactly `"_"` or `"%"` degenerates to the unconstrained pattern `%_%`/`%%%`, matching everything. Demonstrated directly against `ListGoals` (sqlite, in-process): seeded 4 Goals titled `"Alpha Beta"`, `"Gamma Delta"`, `"Contains_Underscore"`, `"Has%Percent"`; `ListGoals(ctx, "", GoalFilter{Title: "_"})` and `GoalFilter{Title: "%"}` each return **all 4** goals instead of the 1 whose title genuinely contains that character. First surfaced downstream in `mwanachama-wakala-api`'s WK22 (filed 2026-09-17, same fleet sweep) via its own `?title=` query param, but the defect is here: every one of the 17 `List*`/`ListDepartmentChildren`/`ListRoleAssignmentsForObjective` methods AG18 gave a `likeFilter`-based text field shares this gap — a UI or MCP caller typing a bare underscore or percent sign as a search term (plausible free text, not an attack) silently gets an unfiltered result set instead of the narrow one they asked for. **Fix location**: `draft_impl.go`'s `likeFilter` — escape `\`, `%`, and `_` in `value` (in that order, so the escape character itself doesn't get double-escaped) before wrapping it in `%...%`, and add `ESCAPE '\\'` to the `LIKE` clause; both Postgres and sqlite support the same `ESCAPE` syntax, matching this function's own existing ILIKE-avoidance portability note. Pinned by `TestListGoals_PinsLikeFilterWildcardInjection` in `ag32_likefilter_wildcard_test.go`, asserting current (broken) behavior — goes red (all 4 seeded goals match, want 1) once fixed. Not fixed here per this sweep's rule (file + pin only). | — |
| AG31 | High | 📋 | 🐞 **BUG — deleting an Objective leaves its ObjectiveModule links dangling, so a Module can stay permanently "enabled" with no Objective backing it, undetected by the publish-validation scan.** Driven through the real `mcp/` surface (a real MCP client over `httptest.NewServer`, `agency_create_agency`/`_goal`/`_objective`/`_attach_module`/`_delete_objective`/`_list_modules`/`_list_objective_modules`/`_list_issues`): created an Objective, linked catalog Module `git` to it via `agency_attach_module` (module read back `"enabled":true`), then called `agency_delete_objective`. Confirmed the Objective is really gone (`agency_get_objective` on it returns a tool error). `agency_list_modules` right after still reports `"code":"git", ...,"enabled":true`, and `agency_list_objective_modules` for the now-deleted objective id still returns the stale link row (`{"module_id":"...","objective_id":"<deleted>", ...}`). `agency_list_issues` (AG26's own scan, the thing `PublishAgency` gates on) reports only the unrelated "Goal G-1 has no Objectives" finding — nothing about the dangling `ObjectiveModule` row, so a publish would go through with a Module gate that no longer has a real Objective behind it. This breaks `agency_attach_module`'s own documented contract ("a Module is enabled exactly while some Objective links it", `mcp/module.go`) and matters in production per this repo's own `gormstore/module.go` doc comment on `BackfillModules` ("a gateway gating real routes on `Module.Enabled`"): a mounting gateway could keep an entire domain's routes live indefinitely after every trace of why was deleted. Root cause: `objective_impl.go`'s `DeleteObjective` (and `goal_impl.go`'s `DeleteGoal`, which cascades through it) only cascades into `RoleAssignment`/`WorkItem` (AG27) — never `ObjectiveModule`. `module_impl.go`'s `linkedModuleIDs` derives "linked" purely from `ObjectiveModule` row existence in scope, never checking the `ObjectiveID` it names still resolves. `issue_impl.go`'s scan (`checkStructure`/`checkRACI`/`checkWorkItems`/`checkEventFlowSteps`/`checkAttachments`) has no check family for a dangling `ObjectiveModule → Objective` reference, unlike the four dangling-reference families it does catch. Pinned by `TestDeleteObjective_LeavesModuleLinkDangling` in `objectivemodule_test.go`, asserting current (broken) behavior — goes red once fixed. Not fixed here per this sweep's rule (file + pin only). | — |

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
