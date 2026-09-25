# QA

Test coverage for `mwanachama-backend-agency`. Tests live beside the code
they cover; there is no separate test corpus.

```sh
go test ./...                    # in-memory SQLite (glebarez/sqlite), no database needed
go test -tags=integration ./...   # same suite, integration-tagged files included
make test-pg                       # -tags=integration, real Postgres, needs POSTGRES_URL
```

## Where the suite stands (2026-09-25, integration-test sweep)

200 `TestXxx` functions across three packages, all green in both untagged
and `-tags=integration` runs: 131 in the root package (`agency.go` and every
`*_impl.go`/`*_test.go` pair — Department/Level, Goal/Objective, Role/
RoleAssignment, Value/Rule + attachments, Workflow/WorkItem, WorkPlan/
ContextSource before AG30 retired them, EventFlowStep, Draft/Publication,
the Issue scan, `SeedSWELab`, the WK17-era `security_wk17_test.go`), 54 in
`routes/` (one `<domain>StatusFor`/`write<Domain>Err` pair per domain,
exercised through a real `httptest.NewServer`/`http.ServeMux` built from
`Routes(m, names)` — never a hand-listed subset), 15 in `mcp/` (a real MCP
client session against `mcp.AddTool`-registered tools, not direct handler
calls). No suite in this repo is currently red.

## What is declared rather than hand-tested per field

Since AGD-007 (2026-09-24), the objects this repo stores are declared in
`agency.blueprint.json`/`agency.agency.json` and migrated by
`mwanachama-backend-shared/spec`, not by GORM row structs — `gormstore/` is
gone. That shifts what a new field's test has to prove: not "does the
column exist" (the blueprint/type agreement check inside
`NewAgencyManager` already refuses a mismatch at construction) but whether
the declared rule (`required`, an enum's `values`, `matches`) actually
reaches `validate.go` on the way in, the same shape
`mwanachama-backend-catalog`'s `TestVocabularyComesFromTheSpec` proved
first.

## What only a real Postgres covers

`make test-pg` needs `POSTGRES_URL`; this routine never sets it (its own
rule: no Postgres containers, `POSTGRES_URL` tests stay opt-in and
skipped), so whatever this suite depends on jsonb/identifier-truncation
semantics for is not re-verified by this sweep on every run — the same
gap `mwanachama-backend-catalog`'s CAT2 found and fixed for its own
`sameDocument` comparison. Check this repo's own `-tags=integration` files
for whether an equivalent silent-behaviour-change risk exists here before
assuming SQLite green means Postgres green.

## What this repo depends on to build at all

Consumers (`mwanachama-wakala-api`, and this sweep's own checkout) resolve
this module via a `replace` onto a sibling directory — no version pin to
chase, but a sibling checkout has to physically exist. This repo itself has
no outgoing `replace` beyond `mwanachama-backend-shared`, which this sweep's
repo set already carries.
