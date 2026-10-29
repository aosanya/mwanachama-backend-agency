# Workflow records, Send form and Route (HS295)

Hub Steward decisions 87 and 89 (2026-10-29). A workflow keeps records beside
its steps; a Send form step goes to a recipient and a later step answers it; a
Route step sorts the answers. This extends [workflow-steps.md](workflow-steps.md).

## Records

A record is `models.Record`, stored in the `record` object of
`agency.blueprint.json` and scoped to a draft like a workflow or a role.

| Field | Meaning |
|---|---|
| `code` | `RC-1`, minted once and never edited. Rows quote it, and it stays the same in every later version. |
| `workflow_id` | The workflow that keeps the record, in the same draft. |
| `name` | What the record is called, such as Payments. Unique inside its workflow, ignoring case. |
| `role_id` | The one role that adds rows, in the same draft. |
| `form_name`, `questions` | The form each row is added with, in the same shape as a step's form. |

Create and update refuse with `ErrInvalidRecord` (400) a record with no name,
no role, two roles, no workflow, no form name, no question, an invalid question
or a name another record of the workflow already has. A role or a workflow that
is not in the draft is `ErrRoleNotFound` or `ErrWorkflowNotFound` (404).

A fork copies the records onto the forked roles and workflows. A publication
document carries them as `records` on `AgencyDocument`, and forking a draft from
a publication restores them. The content hash includes them, so changing only a
record is a change that can be published. The hash also now includes a step's
kind, role, form, questions and links; before this change only the title,
description, position and instruction were hashed.

Deleting a workflow deletes its records and their rows. Deleting a record
deletes its rows.

## Rows

Rows are made by the member app and kept in the agency, not in the catalog.
The studio has no screen for adding one. `models.RecordRow` is stored in the
`record_row` object and is not copied by a fork or carried by a publication.

| Operation | Tool | Meaning |
|---|---|---|
| `POST /record-rows` | `agency_create_record_row` | Add a row. |
| `GET /record-rows?draft_id=` | `agency_list_record_rows` | List rows, narrowed by `record_id` or by `record_code`. |

A row names the draft whose record it was added to (any draft that exists, not
only an open one, because the app serves the published draft), the `record_id`,
the `role_id` the person acted as, who added it, and the `answers` by question
prompt. The row keeps the record's `record_code`, so it can be read across
versions.

A row is refused with `ErrRecordRowRefused` (403) unless its `role_id` is the
record's `role_id`. Answers are refused with `ErrInvalidRecordRow` (400) when
they answer a question the form does not ask, leave a required question blank,
or do not fit the question: yes or no, a number for a number or an amount, a
date as `2006-01-02`, or one of the typed options of a pick-one question. The
list a pick-one question names with `options_from` is not checked here, because
the agency does not hold it.

## Send form, the step that answers it, and the Route step

Four fields join `models.WorkItem`:

| Field | Meaning |
|---|---|
| `recipient_role_id` | On a `send_form` or `notify` step: the one role it goes to, in the draft. Empty means everyone in the order's group. |
| `sent_step_id` | On a `yes_no` or `route` step: the `send_form` step of the same workflow whose form it answers or whose answers it sorts. |
| `routes` | On a `route` step, a list of routes. |
| `otherwise_step_id` | On a `route` step: where it goes when no route matches. Empty means the next step. |

A route is `match` (`all` or `any`), a list of rules, and `go_to_step_id`. A rule
is a `question` (a prompt of the sent form), an `operator` (`is`, `is_not`,
`contains`, `more_than` or `less_than`) and a `value`. The two numeric
operators need a number.

A `yes_no` step that has a `sent_step_id` takes no form of its own, because the
form is defined once, in the step that sends it. Its action reads "Answers the
invitation", derived from the sent form's name.

Create and update refuse with `ErrInvalidWorkItem` (400):

- a recipient on any kind but `send_form` and `notify`, or more than one;
- a `sent_step_id` on any kind but `yes_no` and `route`, a sent step that is in
  another workflow, is the step itself or is not a `send_form` step;
- routes or an otherwise on any kind but `route`;
- routes with no sent form, a match that is neither `all` nor `any`, a route with
  no rule or no target, a rule with no question, an unknown operator or no
  value, a question the sent form does not ask;
- a route or otherwise target in another workflow, or the step itself;
- a form on a step that answers a sent form.

A recipient that is not a role of the draft is `ErrRoleNotFound` (404). A sent
step or a route target that is not a step of the draft is `ErrWorkItemNotFound`
(404). A Route step names only steps that exist.

A route step may be saved with no sent form and no route. A workflow is built
step by step and a route points at steps after it, so the Route step is created
first and filled in once its targets exist. The Issue scan raises the error
that stops publishing.

## What the Issue scan flags

All are `error`, so none can be published:

- a `route` step that names no sent form, or has no route;
- a sent form, a route target or an otherwise target that no longer exists, or
  that now belongs to another workflow;
- a recipient role that no longer exists;
- a record whose role or workflow no longer exists.

None of the deletes cascade into the steps that point at what they remove.

## Not here

The Test pages' blocks over a record's rows are not settled (decision 89). The
SQL mirror for the gateway's active migrations is not written: that repo hand
maintains it, and the two new tables (`records`, `record_rows`) and the five new
columns of `work_items` still need to be mirrored there.
