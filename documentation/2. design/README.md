# Design

This module's design lives in seven decision records, one per design
session, under [todo_details/](todo_details/). Each is the page that carries
what would otherwise be comments in the source. Read the latest one that
touches what you are changing. Later records supersede earlier ones where
they disagree.

| Record | What it decided |
|---|---|
| [AGD-001](todo_details/AGD-001.md) | `Objective` is first-class, RACI is Objective-scoped, Values and Rules make tacit rules explicit, and `Mission`/`Vision` are gone |
| [AGD-002](todo_details/AGD-002.md) | the Module catalog, and the Objective↔Module link that switches a capability on |
| [AGD-003](todo_details/AGD-003.md) | a Value or Rule attaches to many targets, referenced by Code so promotion never strands it |
| [AGD-004](todo_details/AGD-004.md) | publishing is gated by the Issue scan, and takes an operator tag |
| [AGD-005](todo_details/AGD-005.md) | a publication carries the whole Agency as a document, and production reads a channel on a tag |
| [AGD-006](todo_details/AGD-006.md) | there is no live row set: every row belongs to a Draft, and publish reads that Draft directly |
| [AGD-007](todo_details/AGD-007.md) | the objects are declared in `agency.blueprint.json`, not written as row structs |

AGD-006 and AGD-007 describe the code as it ships today. The porting notes in
the repo's `CLAUDE.md` up to AG30 describe a nullable `DraftID`, a live row
set and `gormstore/`. None of those exist any more.

[workflow-steps.md](workflow-steps.md) covers what a workflow step carries: its
kind, its one role, its form and questions, and its derived action.

[workflow-records.md](workflow-records.md) covers what a workflow keeps beside
its steps: records and their rows, a Send form step's recipient, the step that
answers a sent form, and the Route step.

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
