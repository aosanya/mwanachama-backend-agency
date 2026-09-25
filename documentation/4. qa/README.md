# mwanachama-backend-agency — QA

This repo's `documentation/README.md` names this folder in its four-phase
layout, but nothing lived here until the 2026-09-25 integration-test sweep
created this page — there was no test-plan or QA-journey document, only the
tests themselves.

## Where QA actually lives

This library has no `selftest`/journey collection of its own (compare
`mwanachama-wakala-api`'s `cmd/server/selftest/`, driven by `make
self-test`) — verification is the Go test suite itself:

- `go test ./...` — 45 `_test.go` files at last count, covering the root
  package, `mcp/` and `routes/`, run against the in-memory sqlite harness
  (`newTestManager`/`newTestDB`), never a real Postgres.
- `go test -tags=integration ./...` — the `integration`-tagged tests that
  the untagged run skips; both modes are run together in CI-equivalent
  checks because a tagged file can break the untagged build.
- `routes/` is exercised through a real `http.ServeMux` built from the
  aggregator (`routes.Routes(...)`), not by calling handlers directly — see
  that package's own tests for the pattern.
- `mcp/` is exercised through a real MCP client session in its own tests,
  not by invoking tool funcs directly.

## Where the actual defect record lives

Known bugs and gaps are tracked on the task board, not here:
[`3. implementation/todo.md`](../3.%20implementation/todo.md) (open) and
[`todo_done.md`](../3.%20implementation/todo_done.md) (resolved) in the same
folder.
