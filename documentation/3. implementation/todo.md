# mwanachama-backend-agency (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

| ID | Pri | Status | Task | Depends on |
|---|---|---|---|---|
| AG18 | P2 | Not started | 🐞 **SECURITY — any caller with reach into an Agency Instance can force an unbounded read on any of ~14 List-shaped methods (ListGoals/ListWorkflows/ListWorkItems/ListRoles/ListDepartments/ListLevels/ListValues/ListRules/ListDrafts/ListPublications/ListRoleAssignments/ListWorkPlans/ListObjectives/ListContextSources/ListEventFlowSteps), all exposed as `mwanachama-wakala-api` `agency_list_*` MCP tools and `routes/` GET endpoints.** Did: found by the 2026-09-13 scheduled security-QA sweep's cross-repo replication pass (mwanachama-backend-api-kazi was this run's primary; this repo was checked because K12 — mwanachama-backend-taskmanager/-git's own identical "unbounded list" gap, found by an earlier sweep — is exactly the shape to check for here too). `grep -rn "Limit(" --include="*.go" .` (excluding `_test.go`/`mcp/`/`routes/`) returns zero hits: every `List<Type>` implementation (`goal_impl.go`, `workflow_impl.go`, `workitem_impl.go`, `role_impl.go`, `department_impl.go`, `level_impl.go`, `value_impl.go`, `rule_impl.go`, `draft_impl.go`, `publication_impl.go`, `roleassignment_impl.go`, `workplan_impl.go`, `objective_impl.go`, `contextsource_impl.go`, `eventflowstep_impl.go`) runs an unconditional GORM `.Find(&rows)` with no `.Limit(...)` anywhere in the call chain, and neither this repo's own `mcp/*.go` (AG13) nor `routes/*.go` (AG9) tool/handler layer adds a page or limit parameter on top. Observed: contrasted directly against `mwanachama-backend-taskmanager`/`mwanachama-backend-git`, which hit this exact gap (board item K12 on `mwanachama-backend-api-kazi`'s board) and fixed it with a hard `maxListPage = 500` applied unconditionally inside every list query — this repo never got that fix, because K12 only touched the two repos it found the gap in. Threat: one MCP tool call or one `routes/` GET request, from any caller who can reach a registered Agency Instance (per decision #13, `mwanachama-wakala-api` is explicitly not multi-tenant — any signed-in operator session or API key reaches every Instance) forces a full, unbounded table scan and unbounded response payload; because every Instance shares one physical Postgres database (also decision #13), one caller flooding reads against one Instance's Goal/WorkItem/Role graph can degrade every other Instance's callers too. Control: none — no cap exists anywhere in this repo's list path, unlike its sibling repos' fixed version of the identical gap. Verdict: Case B, real and still open — not filed as a self-test (would need a product-code fix that doesn't exist yet). Fix should mirror K12's own shape: a `maxListPage`-style hard cap (this repo has no existing constant of that name yet) applied inside every `List<Type>` implementation above, regardless of caller input, plus (optional follow-up) a caller-facing `limit`/`cursor` param on the params structs where filtering already exists cleanly. See `mwanachama-backend-taskmanager`/`mwanachama-backend-git`'s own K12 fix for the exact pattern to copy. | — |

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
