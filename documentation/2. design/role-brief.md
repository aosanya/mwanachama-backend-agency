# A seat's brief

`get_role_brief` (REST `GET /roles/{role_id}/brief`, tool `agency_get_role_brief`, action `agency.role.brief`) gathers what one seat needs, in plain text and as named parts, so an AI role's instructions can come from the agency without the caller assembling them.

It reads the publication on the `production` channel (`GetPublicationForChannel`), never the working rows, so the brief changes only when a new tag is published. Nothing is stored and no model is involved; the same publication gives the same text.

Parts, all taken from the publication document:

- the seat's name and description (the seat is found by id or code);
- the objectives it serves: those of every workflow that has a step the seat holds a RACI label on (`role_assignments` by step, then `workflow_objectives` by workflow), each with its goal and the labels the seat holds there;
- the values and rules attached to the seat, to one of those objectives, to one of those goals, or to nothing at all (organization-wide). Anything attached only to another seat or objective is left out;
- the `prompt` of each work item of those workflows (items with no prompt are skipped).

An unknown seat answers `ErrRoleNotFound`; no production publication answers `ErrPublicationNotFound`.

## Open owner question: draft or published

The brief assumes the published agency. Whether a seat should also be briefable from a draft (to preview an edit before publishing) or from another channel is undecided and not built.

Tests: `role_brief_test.go`.
