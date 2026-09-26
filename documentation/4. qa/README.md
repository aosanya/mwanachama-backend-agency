# QA

Test coverage for `mwanachama-backend-agency`. Tests live beside the code
they cover; there is no separate test corpus.

```sh
go test ./...        # in-memory SQLite (glebarez/sqlite), no database needed
```

As of 2026-09-26: `go test ./...` and `go test -tags=integration ./...` are
both fully green across all four packages — 132 test functions in the root
package (across 24 `*_test.go` files: `agency.go`'s own lifecycle,
`drafts.go`'s fork, RACI, Value/Rule attachments, the Issue scan,
publication/document, `filters.go`'s per-module filters,
`security_wk17_test.go`), 19 in `mcp/`, 54 in `routes/`.

## What is guarded, and how

- **`ErrCannotEditLive` is permanent, Agency-wide-only immutability, not a
  live/draft split.** AGD-006/AGD-007 (2026-09-19 through 09-24) deleted
  `PromoteDraft` and the old live-row/`ErrCannotEditLiveRACI` model outright
  — "any draft can be edited and published," so `Role`/`RoleAssignment` carry
  no such check any more. What remains: `UpdateValue`/`DeleteValue`/
  `UpdateRule`/`DeleteRule` refuse with `ErrCannotEditLive`
  (`value_impl.go`/`rule_impl.go`) when the target has **no attachment** at
  all (`hasAnyAttachment`) — an Agency-wide Value/Rule is never forked into a
  Draft the way an attached, target-scoped one can be, so this is permanent,
  not "until archived."
- **Value/Rule attachments resolve by `Code`, not row id** — a target's row
  id is not guaranteed stable across every write path this package offers
  (e.g. `import.go`'s document import mints fresh rows), so an id-keyed
  attachment would risk going stale; Code survives. Covered by
  `value_test.go`/`rule_test.go`.
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
