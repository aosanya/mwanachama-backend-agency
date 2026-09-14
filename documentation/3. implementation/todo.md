# mwanachama-backend-agency (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

| ID | Pri | Status | Task | Depends on |
|---|---|---|---|---|
| AG20 | P1 | 🚀 | Add `Issue` — a Draft-scoped, system-generated diagnostics list (info/warning/error), its own dashboard section rather than a row in wakala-studio's generic Shortcuts menu. `models/issue.go`: `Issue{ID, DraftID (required — Issues only exist against an open Draft, never live), Severity (IssueSeverity: info\|warning\|error), Message, EntityType, EntityID, CreatedAt}` — `EntityType`/`EntityID` point at the specific Goal/Objective/Role/etc. the issue is about (`""` if none), so a UI can deep-link to what's wrong. No user-facing `Code` (matches AG16's exclusion list — Issues are regenerated, not referenced by handle). New `AgencyManager.ListIssues(ctx, draftID, filter IssueFilter) ([]Issue, error)` **re-scans on every call**: deletes the Draft's existing Issue rows, re-runs the rule checks below against current Draft-scoped data, inserts fresh rows, then returns them through `filter` (AG18's `<Module>Filter` shape — `Message` substring, `Severity`/`EntityType` exact match). No Create/Update/Delete surface — Issues are always system-derived, never manually authored, so nothing should be exposed to write one directly. First batch of rules to wire up: (1) an Objective with no RoleAssignment labeled Accountable → warning; (2) a Goal with no Objectives → info. The `error` severity ships in the enum now so a future blocking check (e.g. gating `PromoteDraft`/`PublishAgency` on unresolved Issues) has somewhere to land without a schema change later — don't force a rule to use it yet if none genuinely needs it. `routes/issue.go` (`IssueRoutes`, GET-only) and `mcp/issue.go` (`agency_list_issues`, Title "List Draft Issues") follow the existing one-file-per-domain shape. | — |

Cross-repo follow-up for AG20, filed here per dev-research.md's "a row for
another board is still filed here first" convention:
- `mwanachama-wakala-studio`: once AG20 lands, give Issues their own
  hand-written dashboard section (like `ShortcutsMenu.tsx`'s Roles/Goals/
  Objectives buttons and `AgencyDashboardDetail.tsx`'s matching panels) —
  explicitly **not** a generic `OTHER_MODULES`/`SEARCH_MODULES` row, which
  is what every other non-hand-written module falls back to today.

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
