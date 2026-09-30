# QA

Test coverage for `mwanachama-backend-agency`. Tests live beside the code
they cover; there is no separate test corpus.

```sh
go test ./...        # in-memory SQLite (glebarez/sqlite), no database needed
```

`go test ./...` is green apart from the deliberate red listed under Red by
design. As of 2026-09-29 (confirmed by the integration sweep with `grep -rc
"^func Test" *.go mcp/*.go routes/*.go`) there were
135 test functions in the root package (across 25 `*_test.go` files:
`agency.go`'s own lifecycle, `drafts.go`'s fork/promote, RACI, Value/Rule
attachments, the Issue scan, publication/document, `filters.go`'s per-module
filters, `security_wk17_test.go`), 20 in `mcp/`, 60 in `routes/`.

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
- **Every draft door refuses a draft that does not exist, and a draft's own
  id, status and clock are the server's** — pinned by
  `routes/draft_doors_security_test.go`: publish, fork, update, archive and a
  draft-scoped create each answer `404` for an unknown draft id, and a body
  `id`/`status`/`created_at` on `POST /drafts` or `PATCH /drafts/{id}` is
  ignored. The issue scan is the one door that does not hold yet — see Red by
  design (AG50).
- **The composite tool never reaches across drafts** — pinned by
  `mcp/composite_draft_scope_security_test.go`: an `agency_design_from_spec`
  `raci` entry naming another draft's Role or Objective by id is refused
  (`role not found` / `objective not found`) and nothing is stored in either
  draft. Mutation-checked against `validateRoleAssignment`'s draft scope.
- **A publication is the server's, and only its channel moves** — pinned by
  `routes/publication_forged_fields_security_test.go` (2026-09-29,
  twenty-third sweep): `POST /publications` carrying `version: 99`,
  `content_hash`, a `document`, `id` and `created_at` stores version 1, the
  draft's own hash and document, and a server id and clock; `PATCH
  /publications/1` carrying `tag`/`content_hash`/`document` changes only the
  channel; a version or fork tag that does not exist answers 404. Not a
  bounded door: `GET /publications` returns every publication's whole
  document and drops `limit` — 200 publications answered 4,960,632 bytes in
  127 ms with or without `?limit=1` (the uncapped-list class tracked as
  wakala-api's WK36).

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
- `publication_race_security_test.go`'s
  `TestAG52_OpenHole_TwoPublicationsHoldOneVersionAndOneChannel` (AG52, P2) —
  left failing on purpose by the 2026-09-30 security sweep: concurrent
  `PublishAgency` calls on Postgres each take `max(version)+1` and each hold
  the channel (through `mwanachama-wakala-api`'s mount, three `201`s at
  version 3 and two publications left on `production`). SQLite serialises the
  writers, so the test writes the losing writer's row directly and turns green
  once the store refuses a second publication at one version.
- `roleassignment_race_security_test.go`'s
  `TestAG53_OpenHole_TwoAccountablesHoldOneObjective` (AG53, P2) — left
  failing on purpose by the 2026-09-30 security sweep (thirty-eighth run):
  eight concurrent `CreateRoleAssignment(label: accountable)` calls on one
  Objective over Postgres stored two, three or four Accountables in three of
  twenty-five rounds, the Issue scan reported nothing and the draft still
  publishes. SQLite serialises the writers, so the test writes the racing
  row directly and turns green once the store refuses it or the publish
  gate refuses an Objective with more than one Accountable.

## What this package does not cover

This library has no server and no session of its own (see CLAUDE.md's "no
auth model" note) — reachability, cross-Instance scope, and every other
caller-identity concern are `mwanachama-wakala-api`'s to test, against its
own real router. That repo's own `documentation/4. qa/` and
`cmd/server/selftest/security/` are where those checks live; this folder is
about correctness of the domain logic alone.
