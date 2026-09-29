# QA

Test coverage for `mwanachama-backend-agency`. Tests live beside the code
they cover; there is no separate test corpus.

```sh
go test ./...        # in-memory SQLite (glebarez/sqlite), no database needed
```

`go test ./...` is green apart from the deliberate red listed under Red by
design. As of 2026-09-25 there were
131 test functions in the root package (across 24 `*_test.go` files:
`agency.go`'s own lifecycle, `drafts.go`'s fork/promote, RACI, Value/Rule
attachments, the Issue scan, publication/document, `filters.go`'s per-module
filters, `security_wk17_test.go`), 15 in `mcp/`, 54 in `routes/`.

## What is guarded, and how

- **No live row to protect** (AGD-006) — `ErrCannotEditLiveRACI` and
  `PromoteDraft` are gone; every row belongs to a Draft and is edited there.
  The one immutability left is `ErrCannotEditLive`: an Agency-wide
  Value/Rule (attached to nothing) refuses `Update`/`Delete`.
- **Attachments resolve by `Code`, not row id**, so a Value/Rule attached
  in one Draft still resolves in a Draft forked from it, where every row
  has a fresh id. Covered by `value_test.go`/`rule_test.go`.
- **`security_wk17_test.go`**: every `Create<Type>` clears a caller-supplied
  `id`/`created_at` rather than honouring it, and a non-empty `draft_id`
  naming an unknown or archived Draft is refused
  (`ErrDraftNotFound`/`ErrDraftNotOpen`) rather than silently attached to a
  Draft the write could never reach.
- **The publish gate** (`AGD-004`): `PublishAgency` runs the Issue scan —
  dangling references, an Objective with no Accountable, a Goal with no
  Objectives, a Workflow with no WorkItems — before hashing, and refuses
  with `ErrPublishValidationFailed` if any check family reports `error`
  severity. `SeedSWELab` publishing clean on the first try is the check
  set's own calibration test.
- **Delete cascades** (`DeleteGoal`/`DeleteObjective`/`DeleteWorkflow`/
  `DeleteRole`) clean up every row a delete would otherwise strand —
  `WorkItem`, `RoleAssignment` — closing the exact orphan shapes the Issue
  scan's dangling-reference checks exist to catch.
- **A row's draft is fixed on update** — pinned for Roles by
  `routes/draft_scope_security_test.go`'s
  `TestRoleUpdateIgnoresABodyDraftID`: a body `draft_id` on `PATCH /roles/{id}`
  is ignored. Goals, Objectives, Workflows and Work Items do not hold it yet —
  see Red by design.

## Red by design

- `routes/draft_scope_security_test.go`'s
  `TestAG48_OpenHole_UpdateMovesRowIntoAnotherDraft` (AG48, P1) — left
  failing on purpose by the 2026-09-28 security sweep: `PATCH` on a Goal,
  Objective, Workflow or Work Item moves it into whatever draft the body
  names, archived ones included. It turns green when the four `Update<Type>`
  methods pin the stored draft, as `UpdateRole` does.
- `routes/issue_draft_security_test.go`'s
  `TestAG50_OpenHole_IssueScanWritesUnderADraftThatDoesNotExist` (AG50, P2) —
  left failing on purpose by the 2026-09-29 security sweep:
  `GET /drafts/{id}/issues` answers `200` for an id no draft carries and stores
  the scan's findings under it, where `GET /drafts/{id}` and every create
  answer `404`. It turns green when `ListIssues` refuses an unknown draft
  before `rescanIssues` writes.

## What this package does not cover

This library has no server and no session of its own (see CLAUDE.md's "no
auth model" note) — reachability, cross-Instance scope, and every other
caller-identity concern are `mwanachama-wakala-api`'s to test, against its
own real router. That repo's own `documentation/4. qa/` and
`cmd/server/selftest/security/` are where those checks live; this folder is
about correctness of the domain logic alone.
