# QA

Test coverage for `mwanachama-backend-agency`. Tests live beside the code
they cover; there is no separate test corpus.

```sh
go test ./...        # in-memory SQLite (glebarez/sqlite), no database needed
```

As of 2026-09-27: `go test ./...` **and** `go test -tags=integration ./...`
are both fully green across all four packages — 132 test functions in the
root package (across 24 `*_test.go` files: `agency.go`'s own lifecycle,
RACI, Value/Rule attachments, the Issue scan, publication/document,
`filters.go`'s per-module filters, `security_wk17_test.go`), 19 in `mcp/`,
54 in `routes/`.

## What is guarded, and how

- **Unattached Value/Rule immutability** (`ErrCannotEditLive`,
  `errors.go:77`) — `UpdateValue`/`DeleteValue`/`UpdateRule`/`DeleteRule`
  refuse outright on a Value/Rule with zero attachments (Agency-wide);
  once it carries at least one `ValueAttachment`/`RuleAttachment` it can be
  edited directly. There is no equivalent restriction on Role or
  RoleAssignment — `UpdateRole`/`DeleteRole`/`UpdateRoleAssignment`/
  `DeleteRoleAssignment` (`role_impl.go`, `roleassignment_impl.go`) carry no
  `ErrCannotEdit*` check at all, and `ErrCannotEditLiveRACI` (the old
  pre-AGD-006 name) no longer exists in this codebase — confirmed by
  `grep`, 2026-09-27. **There is no more "live" row to protect**: AGD-006
  (CLAUDE.md's AG33–35 note) made every Draft-scoped table's `DraftID`
  non-nullable and deleted `PromoteDraft` outright, so what this file used
  to call "live" is simply an ordinary open Draft now.
- **Value/Rule attachments resolve by `Code`, not row id** — they're
  never forked or remapped themselves (AGD-003), so a Code that outlives
  whichever Draft its target currently lives in is what keeps
  `ListValueAttachments`/`ListRuleAttachments` resolving correctly. Covered
  by `value_test.go`/`rule_test.go`.
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
