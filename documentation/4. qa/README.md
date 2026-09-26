# QA

Test coverage for `mwanachama-backend-agency`. Tests live beside the code
they cover; there is no separate test corpus.

```sh
go test ./...        # in-memory SQLite (glebarez/sqlite), no database needed
```

As of 2026-09-26: `go test ./...` and `go test -tags=integration ./...` are
both fully green across all four packages — 131 test functions in the root
package (across 24 `*_test.go` files: `agency.go`'s own lifecycle,
`draft_impl.go`'s fork/archive, RACI, Value/Rule attachments, the Issue
scan, publication/document, `filters.go`'s per-module filters,
`security_wk17_test.go`), 15 in `mcp/`, 55 in `routes/` (54 plus AG46's
pinning test below).

## What is guarded, and how

- **There is no live row-scope any more (AGD-006).** Every business-entity
  row belongs to some real `Draft` — no nullable `DraftID`, no `PromoteDraft`
  (deleted outright), no `"live"` fork source. `Draft.Status` is only
  `open`/`archived`. `PublishAgency(ctx, draftID, tag, channel)` reads a
  named Draft's own rows directly and freezes them into a publication — no
  merge step. Confirmed on disk: `grep -rn "PromoteDraft\|gormstore"
  --include="*.go" .` (excluding stale comments) finds no such package or
  function; `ErrCannotEditLiveRACI` does not exist anywhere in this repo.
- **`ErrCannotEditLive`** still guards only the one case AGD-006 left
  standing: a Value/Rule with **zero** attachments is Agency-wide and
  permanently immutable (`UpdateValue`/`DeleteValue`/`UpdateRule`/
  `DeleteRule` refuse it) — `Role`/`RoleAssignment`/Goal/Objective/Workflow/
  WorkItem carry no such rule at all any more: "every Role belongs to some
  Draft and is freely editable there" (`agency.go`'s own `AgencyManager`
  doc comment).
- **Cross-repo id integrity across a Draft fork**: Value/Rule attachments
  resolve by `Code`, not row id, because every attachable type mints a
  fresh id on every `CreateDraft` fork — an id-keyed attachment would go
  stale the first time anyone forked a Draft touching either side. Covered
  by `value_test.go`/`rule_test.go`'s fork-then-resolve cases.
- **Known gap, not yet closed (filed as AG46, 2026-09-26):** `Create<Type>`
  refuses a `DraftID` naming an archived Draft (`ErrDraftNotOpen`), but no
  `Update<Type>`/`Delete<Type>` makes the same check — an archived Draft's
  rows stay fully editable and deletable through the real route table.
  `routes/ag46_archived_draft_still_editable_test.go` pins the current
  (broken) behaviour with a real `httptest.NewServer` + `http.Client`.
- **`security_wk17_test.go`**: every `Create<Type>` clears a caller-supplied
  `id`/`created_at` rather than honouring it, and a non-empty `draft_id`
  naming an unknown or already-archived Draft is refused
  (`ErrDraftNotFound`/`ErrDraftNotOpen`) rather than silently attached to a
  Draft the write could never reach — but see the AG46 gap above: that
  refusal only guards `Create<Type>`, not `Update`/`Delete`.
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
