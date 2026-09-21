# Requirements — mwanachama-backend-agency

## Problem

Mwanachama needs a domain repo modeling an **Agency** — not as a live
operational store of departments/membership, but as **a declarative
document**: an org chart, a strategy (Mission/Vision/Goals), and a RACI
matrix (Responsible/Accountable/Consulted/Informed) recording the rules for
how entities are meant to interact. It is self-contained data other systems
may read as reference; it does not call out to, provision, or stay in sync
with any other repo (confirmed 2026-09-09 — see decision #8). This exists
today as a standalone service — a Go gRPC
microservice (port 50053, ArangoDB/entitygraph storage, ~10,150 lines of
non-generated Go) built on the premise "one running instance = one agency."
This repo is the mwanachama-native home for that domain, built the way
`mwanachama-backend-actor` and `mwanachama-backend-taskmanager` were: a
plain Go library the gateway imports, not a network service of its own.

**RACI is genuinely new**: the source service's proto defines a `RACILabel`
enum and `RoleAssignment` message, but no Go code implements it anywhere —
it's dead proto surface. A workspace-wide grep across all of mwanachama
found zero other references to RACI. This repo is where it gets built for
real, for the first time.

## Scope decisions (session of 2026-09-09)

| # | Question | Decision |
|---|---|---|
| 1 | Transport | Plain Go library imported by the gateway, not a standalone gRPC service. Drop the source's `proto/`, `gen/`, `internal/server/`, `cmd/`, `Dockerfile.server` entirely (~9,500 of its ~10,150 non-domain lines, most of it generated). Matches the precedent both `mwanachama-backend-git` and `mwanachama-backend-taskmanager` set — their `CodeVald*` sources were also gRPC services, converted to libraries during their own ports. |
| 2 | Storage & package shape | GORM, following `mwanachama-backend-actor`'s shape as closely as possible — explicit user instruction. `models/` (one file per domain type), `gormstore/` (row structs + `Migrate`/`TableNames`), `routes/`, root `doc.go`/`tables.go`/`*_impl.go`. Replaces the source's ArangoDB/entitygraph storage (`storage/arangodb/storage.go`). |
| 3 | RACI matrix | Build for real: Responsible/Accountable/Consulted/Informed assignment of a Role to a WorkItem. Net-new capability, not a port — see Problem above. |
| 4 | Role taxonomy | Seeded from all 21 personas in NeoLabHQ/context-engineering-kit's `agents/` directory: `bug-hunter`, `business-analyst`, `change-expectation-agent`, `change-failure-agent`, `change-impact-agent`, `change-story-agent`, `code-explorer`, `code-quality-reviewer`, `code-reviewer`, `contracts-reviewer`, `developer`, `fpf-agent`, `historical-context-reviewer`, `judge`, `meta-judge`, `researcher`, `security-auditor`, `software-architect`, `tech-lead`, `tech-writer`, `test-coverage-reviewer` — unfiltered, no curated subset. RACI labeling (Responsible/Accountable/Consulted/Informed) happens per Role-to-WorkItem assignment, not as a fixed property of the Role, so seeding every persona costs nothing up front. That kit is a Claude Code plugin marketplace (markdown skills/agent prompts across multiple AI tools) — not a Go library, no `go.mod` dependency results from this. Its role names/definitions are **seed data** for this repo's own `Role` model, nothing more. Note: every persona here is software-development-specific — this catalog seeds a software-type department only; a "mechanical" department (decision #7's other example) has no equivalent source yet. |
| 5 | Test fixture | "SWELab" is itself a test Agency — seeded with Roles drawn from that kit's taxonomy, to prove the Agency/Role/RACI model end-to-end. Not a real production tenant. |
| 6 | Workflow/WorkItem vs. `mwanachama-backend-taskmanager` | Kept separate — and now with a principled reason, not just precedent: per decision #8, Agency's Workflow/WorkItem is a **declarative plan** ("how work is meant to flow"), while taskmanager's Project/Task/WorkflowRun is **live execution state**. Same shape in spirit, different purpose; the two domains do not share types or import each other. |
| 7 | Agency cardinality & departments | One Agency can contain multiple separated departments (e.g. "software" and "mechanical") in one shared database. Modeled as a **local, self-contained org-chart type** shaped like `mwanachama-backend-actor`'s Group/Hierarchy pattern (parent/child nodes, named levels) — but NOT imported from actor (see decision #8: no code dependency, no live coupling). One Agency's org chart is its own record, not a live pointer into actor's real `Hierarchy`/`Group` tables. |
| 8 | Agency's relationship to other repos | **Declarative document only.** Confirmed 2026-09-09: Agency does not call actor's, taskmanager's, or any other repo's API to provision or synchronize state, and nothing here is kept in sync with what those repos actually store. Other systems may read an Agency's published document as a reference, but Agency does not write to them. This settles decision #7's data-model question (no import, no dependency) and sharpens decision #6's rationale. |
| 9 | Auth/session model & deployment topology | No new design needed — both fall out of decision #1. As a plain Go library the gateway imports, Agency has no deployment of its own (no separate namespace/ingress) and no bespoke auth model: its future `routes/` package (AG9) is capability-gated the same way actor/taskmanager/git's routes already are, under the gateway's existing session middleware. |

## What ports vs. what's new

- **Ports from the source service** (domain logic only, pre-GORM-conversion):
  `Agency` (Mission/Vision, `Draft → Active → Achieved`), `Drafts`
  (immutable-once-published edit lifecycle — `CreateDraft` deep-copies a
  sub-graph, `PromoteDraft` writes an immutable `AgencySnapshot`,
  `PublishAgency` produces a SHA-256 `AgencyPublication`), `EventFlowStep`
  (per-workflow ordered lookup into a workflow's `event_flows` JSON). The
  file currently named `raci.go` (485 lines) does not implement RACI — it's
  WorkPlan/`ContextSource` event-topic-matching-and-dispatch logic
  (`CreateWorkPlan`/`MatchWorkPlans`), stranded under the wrong filename by
  a 2026-05-06 rename in the source repo. Port that logic under a name that
  matches what it does, separately from decision #3's new RACI work.
- **Net new**: the RACI matrix itself, the `Role` model and its
  SWELab-sourced seed taxonomy, and the local declarative org-chart type
  (departments, shaped like actor's Group/Hierarchy but with no dependency
  on it — decision #8).
- **Dropped**: `proto/`/`gen/`/`internal/server/`/`cmd/`/`Dockerfile.server`
  (decision #1); ArangoDB/entitygraph storage (decision #2, replaced by
  GORM); the source's one-instance-per-agency model (decision #7).

## Open questions

None — all questions raised in the 2026-09-08 and 2026-09-09 scoping
sessions are resolved above. Next open items belong on
[../3. implementation/todo.md](../3.%20implementation/todo.md) as
implementation-time decisions (e.g. exact GORM column choices), not as
scope gaps.
