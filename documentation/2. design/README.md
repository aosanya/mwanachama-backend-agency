# Design

This module's design lives in twelve decision records, one per design
session, under [todo_details/](todo_details/). Each is the page that carries
what would otherwise be comments in the source. Read the latest one that
touches what you are changing. Later records supersede earlier ones where
they disagree.

| Record | What it decided |
|---|---|
| [AGD-001](todo_details/AGD-001.md) | `Objective` is first-class, Values and Rules make tacit rules explicit, and `Mission`/`Vision` are gone (its Objective-scoped RACI was reversed by AGD-009) |
| [AGD-002](todo_details/AGD-002.md) | the Module catalog (its Objective↔Module link was removed by AGD-008) |
| [AGD-003](todo_details/AGD-003.md) | a Value or Rule attaches to many targets, referenced by Code so promotion never strands it |
| [AGD-004](todo_details/AGD-004.md) | publishing is gated by the Issue scan, and takes an operator tag |
| [AGD-005](todo_details/AGD-005.md) | a publication carries the whole Agency as a document, and production reads a channel on a tag |
| [AGD-006](todo_details/AGD-006.md) | there is no live row set: every row belongs to a Draft, and publish reads that Draft directly |
| [AGD-007](todo_details/AGD-007.md) | the objects are declared in `agency.blueprint.json`, not written as row structs |
| [AGD-008](todo_details/AGD-008.md) | a module is switched on by a switch or a workflow step, not by an objective link |
| [AGD-009](todo_details/AGD-009.md) | objectives attach to workflows, RACI is a label on a step with no limit on any label, and a step has no objective |
| [AGD-011](todo_details/AGD-011.md) | a control replaces rule and value: advisory or enforced, attached to a pathway, managed through permissions |
| [AGD-012](todo_details/AGD-012.md) | a library of hazards, threats, consequences and controls; a bowtie is composed from it and controls stand on library parts |

AGD-006 and AGD-007 describe the code as it ships today. The porting notes in
the repo's `CLAUDE.md` up to AG30 describe a nullable `DraftID`, a live row
set and `gormstore/`. None of those exist any more.

[workflow-steps.md](workflow-steps.md) covers what a workflow step carries: its
kind, its one role, its form and questions, its derived action, and the RACI labels that put roles on it. A workflow's objectives are in the same page.

[workflow-records.md](workflow-records.md) covers what a workflow keeps beside
its steps: records and their rows, a Send form step's recipient, the step that
answers a sent form, and the Route step.

[compute-settings.md](compute-settings.md) covers where an agency keeps the settings the calculating steps need (the decimal places of its currency, the rounding rule, the digits kept in a division and the most rows one batch may hold), who changes them and what a change does to a publication that already exists.

[compute-step.md](compute-step.md) covers the calculating step kind: where each
input comes from, who runs the step, and whether a confirming role checks the
figure first (ISS-19).

[compute-step-failure.md](compute-step-failure.md) covers what a calculating step
does when it cannot work out a result: stop, or go to a named step (ISS-6).

[posting-step.md](posting-step.md) covers the posting step, which writes a
calculated amount to a member's ledger account as an appended entry (ISS-7).

[held-figures.md](held-figures.md) covers a calculated figure that waits for
approval before it becomes real, and what happens when its inputs move first
(ISS-22).

[library-versioning.md](library-versioning.md) covers the library versions a
publication carries, what the Issue scan reports about them and what the agency
never does to them (ISS-18).

[test-scenarios.md](test-scenarios.md) covers test scenarios, their actors and
inputs, test runs, the send-back rule and the publish gate (HS260).

[background-runs.md](background-runs.md) covers how a test run is started in the
background, held Waiting under the hub's Scenarios at once setting, stopped by
the designer and picked up after a restart (HS330).

[review.md](review.md) covers the seven operations of the Review feature, how a
run starts and a part is recorded, the carry-over of a designer's answers and
the `then_steps` column (HS465-HS468).

[module-mounts.md](module-mounts.md) covers the names an agency gives each mount
of a module (HS271), their rules, and which modules can be mounted by name.

[objective-measures.md](objective-measures.md) covers the measures an objective
carries, the two operations that list and set them, and how they travel through
fork and publish (HS427).

[generated-tool-text.md](generated-tool-text.md) covers what a generated MCP
tool's answer says. It is the design behind every `agency_*` tool's text,
which a client that never reads `structuredContent` sees as the whole answer.

[todo.md](todo.md) is the open design board (`AGD-XXX`), and
[todo_done.md](todo_done.md) holds the closed rows. Implementation work these
decisions caused is tracked on
[`3. implementation/todo.md`](../3.%20implementation/todo.md).

**The draft is not a security boundary.** Every member of an Instance
reaches every draft in it. The boundary is the Instance, and the host
enforces it (`mwanachama-wakala-api`'s decision #21). Draft scoping is still
held as a correctness property, and the security sweep pins it. What it has
pinned is listed in [`4. qa/README.md`](../4.%20qa/README.md).
