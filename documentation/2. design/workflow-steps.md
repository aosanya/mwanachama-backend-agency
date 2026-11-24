# Workflow steps (HS288)

Hub Steward decisions 84, 87 and 89 (2026-10-29): a workflow step says what it
is, who acts on it and what goes into it. A test scenario's blocks mirror these
later (HS260), so the shape is kept small and stable.

## Where it lives

A step is a `work_item`. It keeps its title, description, ordinality and
`prompt` (the instruction it has always carried) and gains five stored fields
in `agency.blueprint.json`, carried by `models.WorkItem`:

| Field | Meaning |
|---|---|
| `kind` | `fill_form`, `send_form`, `notify`, `yes_no`, `route`, `run_workflow` or `chat`. Empty is an instruction-only step, which is every step written before this change. |
| `role_id` | The one role that acts, in the same draft. Empty means the agency takes the step. |
| `run_workflow_id` | The workflow a `run_workflow` step starts, in the same draft. |
| `form_name` | What the step's form is called. |
| `questions` | The form's questions in the order they are asked. |

`action` is derived, never stored: `models.ActionOf(kind, form name, workflow
name)` gives "Fills the Group order form", "Sends the invitation", "Gives the
supplier's answer", "Notifies with the order", "Routes the answers" and
"Starts Cancel after the Supplier says yes". It is declared `spec:"-"` on the
model and filled by every create, get, list and update.

A question is `models.StepQuestion`: a `prompt`, an `answer_type` (`yes_no`,
`single_choice`, `multi_select`, `free_text`, `date`, `time`, `number`,
`currency`, `photo` or `file`, the Forms module's ten names), whether it is
`required`, a `helper` line, the typed `options` or the named list they come
from (`options_from`) for a `single_choice` or `multi_select` question, and
a `fixed` value for a question the step always answers the same way, which is
how a notify step holds its message.

## What create and update refuse

All with `ErrInvalidWorkItem` (400) unless noted:

- more than one role: `role_id` is one id, so a list or ids joined by a comma
  or a space is refused (a JSON list in the body is refused by the route as a
  400 before it gets here);
- a `send_form` or `route` step that names a role, since both are always the
  agency's (decision 89 b and c);
- a role, form or workflow on a step with no kind;
- an unknown kind;
- a `run_workflow` step that names no workflow, a workflow on any other kind,
  or a step that runs the workflow that owns it;
- a form on a `route` or `run_workflow` step; a form with no name or no
  question on the other four kinds; a question with no prompt, an unknown
  answer type, a `single_choice` or `multi_select` with neither options nor a list, or
  options on any other answer type;
- a role that is not in the step's draft: `ErrRoleNotFound` (404);
- a workflow to run that is not in the step's draft: `ErrWorkflowNotFound`
  (404).

Update keeps the step in its draft whatever `draft_id` the body carries.

## Everything that copies a step

- A draft fork remaps `role_id` and `run_workflow_id` onto the roles and
  workflows it minted, so the roles are copied before the steps.
- A publication document carries the five fields on `DocumentWorkItem`, and
  forking a draft from a publication restores them, remapping both ids.
- The Issue scan raises an `error` for a step whose role or run workflow no
  longer exists in the draft, which is what deleting a role or a workflow now
  leaves behind. Neither delete cascades into the steps.

## Objectives and the accountable role of a step (AGD-009, ISS-179)

A step has no `objective_id`. A workflow names the objectives it delivers
through `workflow_objective` rows (`agency_link_workflow_objective`), many to
many, both in one draft; a step serves its workflow's objectives. A workflow
with none is a blocking `error` in the scan.

A step carries one plain field, `accountable_role_id`: the one role of the same
draft that answers for it. It may be set on a step of any kind, and on a step
with no role of its own. There is no row, label or vocabulary beside it; the
design is in [accountable-role.md](accountable-role.md).

The field drives nothing. Who acts is `role_id`, who is told is
`recipient_role_id`, the answer comes from the step's kind and the limit is
`time_limit_minutes`; the approval gate reads those. The scan raises an `error`
for an accountable role that no longer exists. No role to objective link is
stored: a role's objectives are read through the workflows whose steps it
answers for, which is how the role brief ([role-brief.md](role-brief.md)) finds
them.

Deleting a role empties `accountable_role_id` on the steps that named it;
deleting a workflow removes its `workflow_objective` rows; deleting an objective
removes its `workflow_objective` rows. Fork remaps both the joins and the
accountable role, the publication document carries `workflow_objectives` and
`accountable_role_id` on each step, and import seeds one `workflow_objective`
per distinct (workflow, objective) of an older `work_items[].objective_id` while
ignoring an older `role_assignments` entry that names an objective and no step.

## The pickers (HS299)

`agency_list_pickers` (`GET /pickers?draft_id=`, action `agency.picker.list`)
is the one list the workflow editor's Who does it, Who is told, Who adds rows
and Picks from pickers read (decision 94), and the Test page's actors and the
app's role chips read it too. It returns `actors`, each `{id, name, kind}` with
`kind` one of `agency`, `role` or `group`, and `answer_types`, the ten above.
The agency comes first, named "The agency", with the agency's id (empty when
none is written yet). Roles are the asked draft's. Groups belong to the Actor
module, so the manager reads them through `WithGroupLister`, which the host
sets; a host that sets none lists no groups, and a source that fails fails the
call. Step kinds, operators and attachments stay a fixed vocabulary of the
framework and are not in this list. Who holds a role stays Q72.

### Groups as actors (HS364)

A step's `role_id` and `recipient_role_id` and a record's `role_id` accept a
role of the step's draft or a group the same `WithGroupLister` offers, so
create and update agree with the picker. A role is looked up first; only an id
that is no role of the draft asks the group source. An id that is neither is
`ErrRoleNotFound`, a host with no group source accepts roles only, and a source
that fails fails the write with its own error rather than reading as not found.
The draft scan and the document import treat an offered group the same way: the
scan does not flag it as a role that does not exist, and an imported step keeps
the group id as it is, since groups belong to the agency and not to a draft. The
agency in the picker is not an actor id here: an agency-only step names no role.

A group is kept on a step or record when it is read back, but a test scenario cannot name it: `scenarioView` and `actorFor` (`scenario_impl.go`) look the actor up among the draft's roles only, so a scenario block whose step a group acts on shows an empty `actor`, and a group is never one of the scenario's actors.

## A workflow that opens in a chat (HS640, Hub Steward decision 123)

A workflow says where it starts for the person with `opens_in`: `chat` or
`form`. Empty means a form, and `form` is stored as empty, so a workflow that
never says is exactly what it was before: the publication document leaves the
key out, the content hash leaves the record out, and a workflow saved with
`form` hashes the same as one saved with nothing. Any other value is refused
with `ErrInvalidWorkflow` (400).

A chat workflow's step 1 is a step of the new kind `chat`, so Publish still
counts a step (the "workflow with no steps" error in the scan is unchanged).
More steps may follow it. The agency does not create or lock the chat step: the
console draws it as the locked step and creates it.

The chat step reuses `role_id` for the role that opens it and adds five fields
on `work_item`, carried by `models.WorkItem`:

| Field | Meaning |
|---|---|
| `agent_role_id` | The role, in the same draft, that the chat is with. |
| `topic` | What the chat talks about; the agent steers back to it. |
| `opening_line` | What the chat says first. |
| `prompts` | Up to three lines offered to start with. |
| `tools` | The tools it may call, stored as the plain names the manager picked them by. Nothing checks them against a list: where the list comes from is open (decision 123). |
| `start_workflow_ids` | The workflows, in the same draft, that the chat may lead the person to. |

Create and update refuse, with `ErrInvalidWorkItem` unless noted:

- a chat step in a workflow that opens in a form, or a second chat step in one
  workflow;
- any chat field on a step that is not a chat step, or on a step with no kind;
- a `tool` (the single tool of a role's step) on a chat step, which lists its
  tools in `tools`;
- more than three prompts, or a blank prompt, tool or workflow to start;
- a start workflow that is the workflow that owns the step;
- an agent role that is not in the draft: `ErrRoleNotFound` (404); a workflow to
  start that is not in the draft: `ErrWorkflowNotFound` (404).

A workflow cannot go back from `chat` to a form while it holds a chat step:
`ErrInvalidWorkflow`.

What is left for Publish rather than for each save, because the console fills
the step in after creating it: the Issue scan raises an `error` for a chat step
that names no opening role, no agent or no topic, whose agent or start workflow
no longer exists, and for a workflow that opens in a chat and has no chat step.

Everything that copies a step carries the fields: a draft fork and a fork from a
tag remap `agent_role_id` and `start_workflow_ids` onto the new draft's roles and
workflows; the content hash adds each non-empty field to the step's record and
`opens_in` to the workflow's, the same way `hazard_code` joined a risk's.

`chat` is one of the `comm` module's `step_kinds` in `models/modules_catalog.json`, so
a chat step keeps comm running while its switch is off, the way a notify step does.

The wakala-api reviewer that renames a workflow or a step rewrites the fields it
knows (`reviewruns/workflows.go`), so it has to keep `opens_in` on a workflow and
`agent_role_id`, `topic`, `opening_line`, `prompts`, `tools` and
`start_workflow_ids` on a step, or a review's rename blanks them.

## A workflow that cancels an order (HS674, Hub Steward decisions 144 and 146)

A workflow says it cancels an order with `cancels_order`, a boolean that is false
unless the designer ticks Cancels an order on the workflow's Workflow tab. The
agency stores it and returns it on every read; nothing in the agency reads it
when a step runs. It exists for the picker on a waiting step, "When it passes,
run", which lists the workflows that cancel and the built-in Cancel and offers
no other, so a workflow such as Check the delivery is never offered as a cancel.
The built-in Cancel stays and is what a step with no pick still does.

It follows `opens_in` through every copy. A true value is written to the
publication document and added to the workflow's record in the content hash; a
false value leaves both as they were, so an existing publication hashes the same.
A draft fork and a fork from a tag carry it.

An update changes only the workflow fields the request carries. A body that
leaves out `ordinality`, `event_flows`, `opens_in` or `cancels_order` keeps the
stored value, and a body that sends one of them empty clears it. A step's `tool`
works the same way. A caller that builds the struct in code carries every field,
so it still replaces the whole row. See
[update-keeps-unsent-fields.md](../3.%20implementation/update-keeps-unsent-fields.md).

## Not here

Scenarios, their actors and their test runs are HS260. The SQL mirror for the
gateway's migrations is that repo's: this repo creates and alters its tables
from the blueprint. The links between steps (recipient, sent form, routes) and
a workflow's records are in [workflow-records.md](workflow-records.md).
