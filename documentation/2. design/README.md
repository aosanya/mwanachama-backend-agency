# Design

This module's design is a series of decision records, numbered `AGD-XXX` and
kept one per file under [todo_details/](todo_details/). [todo.md](todo.md) is
the design board that numbers them and [todo_done.md](todo_done.md) is where
a decided row goes. Read them in order — each later one reshapes what an
earlier one built, and three of them retire whole surfaces:

| Record | What it decided |
|---|---|
| [AGD-001](todo_details/AGD-001.md) | `Objective` is first-class, RACI is Objective-scoped, Values/Rules replace Mission/Vision |
| [AGD-002](todo_details/AGD-002.md) | the Module catalog and the Objective↔Module cross table |
| [AGD-003](todo_details/AGD-003.md) | a Value or Rule attaches to many targets, by `Code`, outside the draft lifecycle |
| [AGD-004](todo_details/AGD-004.md) | publishing is gated by the Issue scan and takes an operator tag |
| [AGD-005](todo_details/AGD-005.md) | a publication carries the whole Agency; production reads a channel on a tag |
| [AGD-006](todo_details/AGD-006.md) | there is no live — every row belongs to a Draft, and publish reads one directly |
| [AGD-007](todo_details/AGD-007.md) | the objects are declared in `agency.blueprint.json`, the routes and tools in `agency.operations.json` |

The decisions that matter to anyone checking an authorization boundary:

- **There is no live row to protect** (AGD-006). Every row carries a
  non-empty `draft_id`; the one immutability left is an Agency-wide
  Value/Rule (attached to nothing). An archived draft is closed to creates.
- **Attachments are not draft-scoped** (AGD-003). Attaching across drafts,
  or onto an archived draft's Value, is the design rather than a gap.
- **This module has no caller model.** Who may call an operation is the
  mount's gate, keyed on each operation's declared `action`; in production
  that mount is `mwanachama-wakala-api`, which adds instance membership.
- **A listing has no page of its own.** Listings run through
  `mwanachama-backend-shared`'s `specstore.List`, which applies no default cap
  (S24 on that repo's board), and no agency operation declares a `limit`, so
  `GET /roles?limit=5` over 300 Roles answers all 300.

[generated-tool-text.md](generated-tool-text.md) records how a generated MCP
tool answer is worded; the open tool-text forging rows on the board (AG59,
AG66, AG68–AG71) are each a place that wording repeats a stored value
unquoted. The engine underneath is documented in
`mwanachama-backend-shared`'s own `2. design/`.

The security sweep checks these decisions against the shipped code every
run; what it has pinned is listed in [`4. qa/README.md`](../4.%20qa/README.md).
