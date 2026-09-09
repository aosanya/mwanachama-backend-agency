# mwanachama-backend-agency — completed tasks

| Task | Title | Completed | Notes |
|------|-------|-----------|-------|
| AG1 | Bootstrap repo: `git init`, four-phase `documentation/` skeleton, `requirements.md`, `todo.md`/`todo_done.md` | 2026-09-09 | New repo, scoped from `CodeValdAgency` (a standalone gRPC microservice, ArangoDB/entitygraph-backed, ~10,150 non-generated lines) the way `mwanachama-backend-taskmanager`/`-git` were scoped from their own `CodeVald*` sources. Decisions and open questions recorded in [../1. requirements/requirements.md](../1.%20requirements/requirements.md). No `go.mod` yet (AG2). |
| AG2 | Scaffold the module: `go.mod`, `.gitignore`, `Makefile` | 2026-09-09 | `go.mod` declares `github.com/aosanya/mwanachama-backend-agency`, `go 1.25.0`, no dependencies yet — nothing to import until AG3 lands real `.go` files. `.gitignore`/`Makefile` copied verbatim from `mwanachama-backend-assetmanager` (bin/env/test-cache ignores; `build`/`test`/`test-pg`/`vet`/`clean` targets — actor itself predates this Makefile convention and has none). `make build`/`go vet ./...`/`go test ./...` all report "matched no packages" until AG3 adds source — expected for a module with zero `.go` files, not a scaffold defect. |
| AG3 | Port Agency/Draft/Goal/Workflow/WorkItem onto GORM: `models/`, `gormstore/`, root `doc.go`/`tables.go` | 2026-09-09 | Package `mwanachamaagency`. Data shape and migration plumbing only — no manager interface, `*_impl.go`, or `errors.go` (that's AG4+). Confirmed with the user this is a single-agency deployment for now, so none of the five types carry an `AgencyID` field — the "drop AgencyID-style single-tenancy in favor of AG5's department scoping" instruction is satisfied by not adding the field yet, not by inventing a placeholder; revisit once AG5's department type lands. `Draft` ports `AgencyDraft` as a header/pointer type only (no `AgencySnapshot`/`AgencyPublication`/`Draft*` sub-entity shadow types — those are AG4's, produced by the lifecycle behavior it ports). `ConfiguredRole` was not ported (superseded by AG6's `Role`). `WorkItem` is Workflow-owned (`WorkflowID`) and optionally Goal-tagged (`GoalID`, `""` = untagged). `Agency`/`Workflow.EventFlows` stay opaque JSON strings. Timestamps are hand-formatted RFC3339-nano strings (`models/time.go`, ported verbatim from actor's). Added deps: `gorm.io/gorm`, `github.com/google/uuid`, `github.com/glebarez/sqlite` (test-only, pure-Go, no cgo/containers) — `gorm.io/datatypes` was NOT added, since none of these five types have a JSON/map column yet. `go build`/`go vet`/`go test ./...` all pass. Full decision record: this repo's `CLAUDE.md` "Porting notes". Flagged for AG8, not resolved here: a shallow pass over `CodeValdAgency/raci.go` suggested it might be a live, non-proto RACI implementation rather than purely the WorkPlan/`ContextSource` routing logic requirements.md describes it as. |

## Archived board context

### Why this repo exists

Sibling to `mwanachama-backend-actor`, modeling an Agency: departments
(e.g. software vs. mechanical) under one Agency in one shared database,
Goals/Workflows/WorkItems, and a real RACI matrix (Responsible/
Accountable/Consulted/Informed) assigning Roles to WorkItems — see
[../1. requirements/requirements.md](../1.%20requirements/requirements.md)
for the full decision record from the 2026-09-09 scoping session. Built as
a plain Go library the gateway imports, following `mwanachama-backend-actor`'s
GORM-backed shape as closely as possible, not as a standalone service the
way `CodeValdAgency` runs today.
