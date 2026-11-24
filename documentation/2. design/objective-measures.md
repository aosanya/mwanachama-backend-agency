# Objective measures (HS427)

An objective carries measures: the figures it is judged by. The console's
Objectives page lists them on each objective card and sets them from a small
form. This page is what the agency stores and what the two tools answer.

## The measure

A measure belongs to one objective in one draft. It has:

| Field | Set by the tool | Notes |
|---|---|---|
| `id` | no | Minted on the first set and kept when the measure is set again. |
| `draft_id` | no | The objective's own draft. A caller never supplies it. |
| `objective_id` | yes, required | An objective that exists. |
| `name` | yes, required | What is measured. Trimmed. |
| `target` | yes, required | Where the figure should get to, kept as written (`40`, `95%`). Trimmed. |
| `unit` | yes, optional | What the figure counts. |
| `status` | yes, optional | A word or two on how it is going. Free text; no vocabulary is declared. |
| `current` | no | Where the figure stands now. Empty until something else writes it. |

`current` is a string like `target`, so the console shows `current / target unit`
without parsing either. The tool does not offer `current`, and the manager
ignores one that arrives anyway.

## The operations

| Action | Tool | Route | Arguments |
|---|---|---|---|
| `agency.measure.set` | `agency_set_measure` | `PUT /measures` | `objective_id`, `name`, `target`, `unit?`, `status?` |
| `agency.measure.list` | `agency_list_measures` | `GET /measures` | `draft_id`, `objective_id?` |

Both tools also take `instance_id`, like every `agency_*` tool.

Setting is the only write. There is no separate create, update or delete:

- A second set with the same name on the same objective replaces the first. The
  names are compared without regard to case or surrounding spaces, so `Orders per
  week` and ` orders PER week ` are one measure. The row, its id and its `current`
  are kept; `name`, `target`, `unit` and `status` take the new values, and a
  `unit` or `status` left out of the second set is cleared. That is what
  "replaces" means: the second call states the whole measure.
- The same name on two different objectives is two measures.
- An unknown objective, or none, is refused with `ErrObjectiveNotFound` (404).
- A blank name or target is refused with `ErrInvalidMeasure` (400).
- An archived draft is refused with `ErrDraftNotOpen` (409), as for every other
  write. The draft is read off the objective, so a caller never names it.

The list is scoped to one draft like every other list, ordered by when each
measure was first set, and narrowed to one objective with `objective_id`.

## Travelling with the objective

A measure follows its objective everywhere an objective goes:

- **Fork from a draft** copies each measure onto the copied objective, with a new
  id, so editing the fork leaves the source alone.
- **Publish** writes the measures into the publication's document, under
  `measures`, each naming its objective by the objective's id in that document.
  The content hash covers every field, so retargeting a measure is a change that
  a republish accepts.
- **Fork from a publication** reads them back onto the objectives it creates.
- **Removing an objective** removes its measures with it, in the same transaction
  as its work items. Removing a goal
  removes the measures of every objective under it. Nothing refuses, because
  nothing in the neighbouring removals refuses either.

## Not here

The applier that turns a proposed `agency_set_measure` into a call, and the
console, are separate rows. Nothing writes `current` yet; when something does, it
writes the stored column, and a replaced measure keeps what it wrote.
