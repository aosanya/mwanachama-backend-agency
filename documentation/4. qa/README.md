# QA

Test coverage for `mwanachama-backend-agency`. Tests live beside the code
they cover; there is no separate test corpus.

```sh
go test ./...        # in-memory SQLite (glebarez/sqlite), no database needed
```

As of 2026-09-27: `go test ./...` is fully green across all four packages —
132 test functions in the root package (across 24 `*_test.go` files:
`agency.go`'s own lifecycle, `draft_impl.go`'s fork/archive, RACI, Value/Rule
attachments, the Issue scan, publication/document, `filters.go`'s per-module
filters, `security_wk17_test.go`), 19 in `mcp/`, 57 in `routes/` (across 17
files).

Since AGD-006/AGD-007 (2026-09-19 through 2026-09-24) there is no more live
row-scope, no `gormstore/` row structs, and no `PromoteDraft` — every row
belongs to a real, non-nullable `Draft.ID`, `Draft.Status` is `open` or
`archived` only, storage is declared in `agency.blueprint.json` and
provisioned by `mwanachama-backend-shared/spec`, and `routes/`/`mcp/` are
generated from `agency.operations.json` via that same shared package's
`dispatch`. See CLAUDE.md's AG31–AG42 note for the full record.

## What is guarded, and how

- **Agency-wide Value/Rule immutability** (`ErrCannotEditLive`) — a
  Value/Rule with no attachment to any target (Role/Goal/Objective/
  Workflow/WorkItem) is Agency-wide and permanently unwritable; one with at
  least one attachment can be edited. `ErrCannotEditLiveRACI` and the old
  "live Role/RoleAssignment" immutability rule it guarded were deleted
  outright by AGD-006 — a Role/RoleAssignment in any *open* Draft is always
  editable now, there is no privileged row-scope left to protect.
- **Cross-repo id integrity across a Draft fork**: Value/Rule attachments
  resolve by `Code`, not row id, because every draft-scoped type mints a
  fresh id when copied into a new Draft (fork-from-draft or
  fork-from-publication) — an id-keyed attachment would go stale on the
  first fork. Covered by `value_test.go`/`rule_test.go`'s
  fork-then-resolve cases.
- **`security_wk17_test.go`**: every `Create<Type>` clears a caller-supplied
  `id`/`created_at` rather than honouring it, and a non-empty `draft_id`
  naming an unknown or archived Draft is refused
  (`ErrDraftNotFound`/`ErrDraftNotOpen`) rather than silently attached to a
  Draft the write could never reach — on `Create<Type>` only; AG48 tracks the
  matching gap on `Update`/`Delete`.
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

## What this package does not cover

This library has no server and no session of its own (see CLAUDE.md's "no
auth model" note) — reachability, cross-Instance scope, and every other
caller-identity concern are `mwanachama-wakala-api`'s to test, against its
own real router. That repo's own `documentation/4. qa/` and
`cmd/server/selftest/security/` are where those checks live; this folder is
about correctness of the domain logic alone.
