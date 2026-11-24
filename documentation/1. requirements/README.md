# Requirements

Two pages live here.

| Page | What it is |
|---|---|
| [requirements.md](requirements.md) | the scoping record of 2026-09-08/09: the problem, nine scope decisions, what ported from the source gRPC service and what was new |
| [insights-strategy-framework.md](insights-strategy-framework.md) | a strategy/goal ontology kept as reference for later scoping, not a decision |

## Read requirements.md as history

It is the record of what was decided on 2026-09-09. Several of its decisions
were later superseded, and the page itself was not rewritten. Where it and the
code disagree, the code and the design records under
[`2. design/`](../2.%20design/README.md) win.

| Decision | Says | What ships instead |
|---|---|---|
| #2 Storage | GORM row structs in `gormstore/` | Objects are declared in `agency.blueprint.json` and stored through `mwanachama-backend-shared/spec`. There is no `gormstore/` directory (AGD-007) |
| #3 RACI | a Role is assigned to a WorkItem | one Role is named accountable for a WorkItem, as the step's `accountable_role_id`; there are no labels and no row. A Workflow, not a step, carries its Objectives. The blueprint's work item has no `objective_id` (AGD-009, which reverses AG12/AGD-001's Objective-scoped RACI; ISS-179 and decisions D39 and D49 drop the labels) |
| #7 Departments | an org chart of departments and levels | there is no `Department` or `Level` model, route or tool. `models/` names neither |
| #9 Auth | gated under the API gateway's session middleware | mounted by `mwanachama-wakala-api`. The boundary is the Instance, enforced by that host's membership check (its decision #21) |
| Ports | `PromoteDraft`, an immutable snapshot, a live row set | there is no live row set and no promote. Every row belongs to a Draft, and publish reads the named Draft (AGD-006) |

## Security

This module has no authentication or authorization of its own. A route or
tool trusts whatever mounted it to have decided the caller may be there. What
the security sweep has pinned about the module's own invariants (draft scope,
server-owned fields, exact-match filters) is listed in
[`4. qa/README.md`](../4.%20qa/README.md).
