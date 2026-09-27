# QA

Test coverage for `mwanachama-backend-agency`. Tests live beside the code
they cover; there is no separate test corpus.

```sh
go test ./...        # in-memory SQLite (glebarez/sqlite), no database needed
```

As of 2026-09-27: `go test ./...` is green across all four packages except
one deliberate red — 134 test functions in the root package (across 25
`*_test.go` files: `agency.go`'s own lifecycle, `draft_impl.go`'s fork/
archive, RACI, Value/Rule attachments, the Issue scan, publication/
document, `filters.go`'s per-module filters, `security_wk17_test.go`), 19
in `mcp/`, 55 in `routes/` (across 17 files, one of them the AG48 red
below).

Since AGD-006/AGD-007 (2026-09-19 through 2026-09-24) **there is no more
live row-scope**: every draft-scoped row belongs to a real, non-nullable
`Draft.ID`, `Draft.Status` is `open` or `archived` only (there is no
"promoted" and no `PromoteDraft`), storage is declared in
`agency.blueprint.json` and provisioned through
`mwanachama-backend-shared/spec`, and `routes/`/`mcp/` are generated from
`agency.operations.json` via that package's `dispatch`. See CLAUDE.md's
AG31–AG42 note for the full record — this file used to describe the
architecture from before that rewrite (`ErrCannotEditLiveRACI`,
`PromoteDraft`, a "promoted" Draft status), none of which exists in the
code any more; corrected here against the tree as it stands today.

## What is guarded, and how

- **Agency-wide Value/Rule immutability** (`ErrCannotEditLive`) — a
  Value/Rule with no attachment to any target (Role/Goal/Objective/
  Workflow/WorkItem) is Agency-wide and permanently unwritable
  (`UpdateValue`/`DeleteValue`/`UpdateRule`/`DeleteRule` in `value_impl.go`/
  `rule_impl.go` check `hasAnyAttachment`); one with at least one attachment
  is freely editable. There is no separate "live Role/RoleAssignment"
  immutability rule any more — `ErrCannotEditLiveRACI` was deleted outright
  by AGD-006 (confirmed: `grep -rn ErrCannotEditLiveRACI --include=*.go .`
  finds nothing), since there is no privileged row-scope left to protect.
- **Cross-repo id integrity across a Draft fork**: Value/Rule attachments
  resolve by `Code`, not row id, because a fork mints a fresh id for every
  row it copies (from another Draft, or from a Publication) — an id-keyed
  attachment would go stale on the first fork. Covered by
  `value_test.go`/`rule_test.go`'s fork-then-resolve cases.
- **`security_wk17_test.go`**: every `Create<Type>` clears a caller-supplied
  `id`/`created_at` rather than honouring it, and a non-empty `draft_id`
  naming an unknown or archived Draft is refused
  (`ErrDraftNotFound`/`ErrDraftNotOpen`) rather than silently attached to a
  Draft the write could never reach. **This guard is Create-only** —
  `Update<Type>`/`Delete<Type>` do not call the same check, so a row
  belonging to an already-archived Draft can still be renamed or deleted;
  demonstrated through the real routes in
  `routes/ag48_archived_draft_writes_test.go`'s
  `TestAG48_OpenBug_UpdateAndDeleteRefuseArchivedDraft`, filed as board row
  AG48 and left red on purpose (see `documentation/3. implementation/
  todo.md`).
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
