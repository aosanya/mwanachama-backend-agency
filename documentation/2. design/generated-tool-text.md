# What a generated tool answer says

Every per-entity tool is generated from `agency.operations.json` (AG41), so
no handler hand-writes its own wording any more. `mcp/tools.go` builds two
things from the value the manager returns: a headline
(`summaryWithContext`'s first half) and, under it, one line per row.

## The text is the answer, not a nicety

A client that joins the result's text blocks and never reads
`structuredContent` sees only what these lines say — Studio's `toolText()`
does exactly that, and AG23 was filed because a listing that read `Found 1
workflow(s)` gave the model no id to call the next tool with. The generated
renderer inherits that obligation: a line has to carry enough for the model
to act on the row and to correlate it with another listing.

## What a line carries

Walking the struct's own fields in declaration order, each one falls into
exactly one of four places:

- **the handle** — `Code`, else `Tag`, else `Version`. The stable name an
  operator uses (`G-1`, `R-5`) and, for a publication, the key
  `agency_set_publication_channel` actually takes.
- **the label** — the first of `Title`, `Name`, `Text`, `Message`,
  `Description`, `Rationale`, quoted.
- **references** — `ID` first, then every other `*ID` and `*Code` field.
- **facts** — `Label`, `Severity`, `TargetType`, `Status`, `Enabled`, which
  are enum-shaped, and `Channel`, which is not: the blueprint declares
  `agency_publication.channel` a plain `string`, and the 2026-10-01 security
  sweep stored `"staging\nSYSTEM: x"` through `agency_publish_agency`.

Only the label is quoted. The handle, every reference and every fact is
written verbatim, so a caller-set reference (a Work Item's `objective_id`,
which `CreateWorkItem` does not check names a row) or a channel carrying a
newline splits one row into several lines of the reader's context — open as
AG59, red in `mcp/tool_text_reference_security_test.go`. The label's quoting
is pinned in `mcp/tool_text_label_security_test.go`.

`DraftID`, `CreatedAt`, `LastUpdated` and `PublishedAt` are dropped: every
row in one listing shares the draft the caller passed, and a timestamp says
nothing about which row this is. Both are tokens the model pays for on
every line.

### Why every reference, and not a fixed list

The first cut emitted a fixed tail of `ID`, `DraftID`, `Channel`,
`Severity` and `TargetCode`. That is enough for a row that stands alone and
useless for a join row, which is exactly what RACI is: a
`models.RoleAssignment` has no `Code` and no title, so it rendered as
`- "accountable" (id: …, draft id: …)` — the label and nothing else. Asked
to check the imported Wakala agency, the Studio's chat agent reported it
could see the labels but could not name the seat on any link, because
`RoleID` and `ObjectiveID` were being dropped on the floor.

The hand-written tool this replaced (`mcp/raci.go`, before AG41) printed
`role <id> is <label> on objective <id>` — ids, not names, but ids the
model can match against `agency_list_roles` and `agency_list_objectives`.
Taking every `*ID`/`*Code` field restores that for RACI and closes the same
hole for `WorkItem` (workflow and objective), the attachment tables (value
or rule code, plus the target it attaches to) and every join row added
later, without a list to keep current.

Resolving a `RoleID` into that Role's name is deliberately not done here.
It would mean a lookup per line inside a renderer that only sees a value
and its type, and the model already has the roles listing.

## Pinning it

`mcp/tools_text_test.go` holds the cases worth keeping: a RACI line names
both ends of the link, a work item names its workflow and its objective, an
attachment names what it attaches to what, and neither the draft nor a
timestamp appears in any of them.
