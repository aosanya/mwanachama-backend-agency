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

There is no "live" scope any more (AGD-006/AGD-007, AG31–AG42,
2026-09-19–24) — every Draft-scoped row requires a `DraftID`
(`ErrDraftRequired`), `PromoteDraft` is gone outright, and `PublishAgency`
reads a named Draft's own rows directly rather than merging them into a
live copy. What this repo still guards, checked against the current
`errors.go`:

- **Agency-wide Value/Rule permanent immutability** (`ErrCannotEditLive`)
  — a Value/Rule with zero attachments can never be edited or forked
  (`TestValue_AgencyWideNeverForkable`/`TestRule_AgencyWideNeverForkable`);
  only an *attached* one is Draft-scoped and editable
  (`TestValue_AttachedForksAndCanBeEdited`/`TestRule_AttachedForksAndCanBeEdited`).
- **Cross-Draft id integrity**: Value/Rule attachments resolve by `Code`,
  not row id, because `CreateDraft`'s fork (`forkedFromID`/`forkedFromType`
  — a Draft or a Publication tag) mints a fresh id for every row it
  copies — an id-keyed attachment would go stale the moment its target is
  forked again. Covered by `value_test.go`/`rule_test.go`'s
  fork-then-resolve cases.
- **`security_wk17_test.go`**: every `Create<Type>` clears a caller-supplied
  `id`/`created_at` rather than honouring it, and a non-empty `draft_id`
  naming an unknown or archived Draft is refused
  (`ErrDraftNotFound`/`ErrDraftNotOpen` — `Draft.Status` is only `open` or
  `archived`) rather than silently attached to a Draft the write could
  never reach.
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
