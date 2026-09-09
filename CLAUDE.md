# CLAUDE.md

Guidance for Claude Code (claude.ai/code) working in this repository.

## Project: mwanachama-backend-agency

A standalone Go library, sibling to
[`mwanachama-backend-actor`](../mwanachama-backend-actor), modeling an
Agency: departments under one Agency, Goals/Workflows/WorkItems, and a RACI
matrix (Responsible/Accountable/Consulted/Informed) assigning Roles to
WorkItems. Module path `github.com/aosanya/mwanachama-backend-agency` (not
yet scaffolded). Ported from the standalone `CodeValdAgency` gRPC
microservice at `/Users/tony/Agents/workdir/CodeVald-AIProject/CodeValdAgency` —
see [documentation/1. requirements/requirements.md](documentation/1.%20requirements/requirements.md)
for the full decision record from the 2026-09-09 scoping session.

Scoped 2026-09-09. No code yet — this session produced the requirements
record and the initial task board; the actual port from `CodeValdAgency`,
the GORM conversion, and the new RACI/Role work are follow-up work (see
[documentation/3. implementation/todo.md](documentation/3.%20implementation/todo.md)).

## Porting notes

- **AG3 landed 2026-09-09: `models/`, `gormstore/`, root `doc.go`/
  `tables.go` for Agency/Draft/Goal/Workflow/WorkItem.** Package name is
  `mwanachamaagency`. No `AgencyID` field exists on any of these five
  types — confirmed with the user this is a single-agency deployment for
  now, so the todo's "drop AgencyID-style single-tenancy in favor of AG5's
  department scoping" is satisfied by simply not adding the field yet, not
  by inventing a placeholder. Revisit when AG5's department type lands and
  a real scoping column is needed.
- `Draft` ports `CodeValdAgency`'s `AgencyDraft` as a header/pointer type
  only (`ID`, `Description`, `ForkedFromID`/`ForkedFromType`, `Status`,
  timestamps) — it does not carry a copy of the forked sub-graph.
  `AgencySnapshot`, `AgencyPublication`, and the `Draft*` sub-entity shadow
  types (`DraftGoal`, `DraftWorkflow`, etc. — which don't exist as Go
  structs in the source either, only as `schema.go` entitygraph property
  lists) are AG4's job, produced by the edit-lifecycle behavior it ports.
- `ConfiguredRole` was **not** ported — AG6 is building a proper `Role`
  model for RACI, and porting `ConfiguredRole` now would just be thrown
  away.
- `WorkItem` is owned by exactly one `Workflow` (`WorkflowID`) and
  optionally tagged to a `Goal` (`GoalID`, `""` = untagged) — the source's
  own schema notes WorkItem is Workflow-owned but goal-taggable, not
  strictly single-parent.
- `Agency.EventFlows`/`Workflow.EventFlows` stay opaque JSON strings, ported
  as-is — no typed parsing yet (AG4's `EventFlowStep` lookup is what reads
  them).
- Timestamps are hand-formatted RFC3339-nano strings (`models/time.go`,
  ported verbatim from actor's), not GORM-managed `time.Time` columns —
  same reasoning as actor's own `models/time.go` doc comment.
- No manager interface, `*_impl.go`, or `errors.go` exist yet — AG3 was
  scoped to data shape and migration plumbing only. AG4+ adds behavior.
- Worth a closer read before AG8: a shallow pass over `CodeValdAgency/
  raci.go` suggested it might be a live, non-proto RACI implementation
  rather than purely the WorkPlan/`ContextSource` event-routing logic this
  repo's requirements.md describes it as. Not resolved, not blocking AG3.

## Conventions

- Follow `mwanachama-backend-actor`'s shape as closely as possible: `models/`
  (one file per domain type), `gormstore/` (row structs, `BeforeCreate` UUID
  minting, `XToRow`/`XFromRow`, `Migrate`/`TableNames`), a `routes/` package,
  and root `doc.go`/`tables.go`/`*_impl.go` files. Read
  `mwanachama-backend-actor/CLAUDE.md` first — it documents these decisions
  in detail (multi-tenancy dropped, recursive tree walks stay Go loops not
  CTEs, `group_impl.go`/`hierarchy_impl.go` as the pattern for this repo's
  department modeling).
- Documentation follows the workspace-wide four-phase layout
  (`1. requirements/2. design/3. implementation/4. qa`) — see
  [documentation/README.md](documentation/README.md).
- Read `CodeValdAgency`'s `agency.go`, `models.go`, `schema.go`, `drafts.go`,
  `event_flow_steps.go` before porting — they carry the domain logic.
  `raci.go` does NOT implement RACI despite its name (it's WorkPlan/
  ContextSource event-topic routing, stranded under the wrong filename by a
  2026-05-06 rename in the source repo); port it separately from the new
  RACI matrix work, under a name that matches what it does.
