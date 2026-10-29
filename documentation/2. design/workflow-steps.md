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
| `kind` | `fill_form`, `send_form`, `notify`, `yes_no`, `route` or `run_workflow`. Empty is an instruction-only step, which is every step written before this change. |
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

## Not here

Scenarios, their actors and their test runs are HS260. The SQL mirror for the
gateway's migrations is that repo's: this repo creates and alters its tables
from the blueprint. The links between steps (recipient, sent form, routes) and
a workflow's records are in [workflow-records.md](workflow-records.md).
