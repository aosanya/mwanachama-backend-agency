# QA

Test coverage for `mwanachama-backend-agency`. Tests live beside the code
they cover; there is no separate test corpus.

```sh
go test ./...        # in-memory SQLite (glebarez/sqlite), no database needed
```

As of 2026-09-25: `go test ./...` is fully green across all four packages —
131 test functions in the root package (across 24 `*_test.go` files:
`agency.go`'s own lifecycle, `drafts.go`'s fork/promote, RACI, Value/Rule
attachments, the Issue scan, publication/document, `filters.go`'s per-module
filters, `security_wk17_test.go`), 15 in `mcp/`, 54 in `routes/`.

## What is guarded, and how

- **Draft-scoped live immutability** (`ErrCannotEditLiveRACI`,
  `ErrCannotEditLive`) — a live Role, RoleAssignment, or Agency-wide
  Value/Rule refuses `Update`/`Delete` outright; only a Draft-scoped copy
  can be edited, then promoted.
- **Cross-repo id integrity across a Draft promotion**: Value/Rule
  attachments resolve by `Code`, not row id, specifically because
  `PromoteDraft` mints a fresh id for every row it promotes — an id-keyed
  attachment would go stale on the first promotion. Covered by
  `value_test.go`/`rule_test.go`'s fork-then-promote-then-resolve cases.
- **`security_wk17_test.go`**: every `Create<Type>` clears a caller-supplied
  `id`/`created_at` rather than honouring it, and a non-empty `draft_id`
  naming an unknown, promoted or archived Draft is refused
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

## What this package does not cover

This library has no server and no session of its own (see CLAUDE.md's "no
auth model" note) — reachability, cross-Instance scope, and every other
caller-identity concern are `mwanachama-wakala-api`'s to test, against its
own real router. That repo's own `documentation/4. qa/` and
`cmd/server/selftest/security/` are where those checks live; this folder is
about correctness of the domain logic alone.
