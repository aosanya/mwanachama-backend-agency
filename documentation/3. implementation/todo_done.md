# mwanachama-backend-agency — completed tasks

| Task | Title | Completed | Notes |
|------|-------|-----------|-------|
| AG1 | Bootstrap repo: `git init`, four-phase `documentation/` skeleton, `requirements.md`, `todo.md`/`todo_done.md` | 2026-09-09 | New repo, scoped from `CodeValdAgency` (a standalone gRPC microservice, ArangoDB/entitygraph-backed, ~10,150 non-generated lines) the way `mwanachama-backend-taskmanager`/`-git` were scoped from their own `CodeVald*` sources. Decisions and open questions recorded in [../1. requirements/requirements.md](../1.%20requirements/requirements.md). No `go.mod` yet (AG2). |
| AG2 | Scaffold the module: `go.mod`, `.gitignore`, `Makefile` | 2026-09-09 | `go.mod` declares `github.com/aosanya/mwanachama-backend-agency`, `go 1.25.0`, no dependencies yet — nothing to import until AG3 lands real `.go` files. `.gitignore`/`Makefile` copied verbatim from `mwanachama-backend-assetmanager` (bin/env/test-cache ignores; `build`/`test`/`test-pg`/`vet`/`clean` targets — actor itself predates this Makefile convention and has none). `make build`/`go vet ./...`/`go test ./...` all report "matched no packages" until AG3 adds source — expected for a module with zero `.go` files, not a scaffold defect. |

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
