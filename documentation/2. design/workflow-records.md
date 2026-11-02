# Workflow records, Send form and Route (HS295)

Hub Steward decisions 87 and 89 (2026-10-29), 110 and 111 (2026-11-01). A
workflow keeps records beside its steps; a Send form step goes to a recipient
and a later step answers it; a Route step sorts the answers. This extends
[workflow-steps.md](workflow-steps.md).

## Records

A record is `models.Record`, stored in the `record` object of
`agency.blueprint.json` and scoped to a draft like a workflow or a role.

| Field | Meaning |
|---|---|
| `code` | `RC-1`, minted once and never edited. Rows quote it, and it stays the same in every later version. |
| `workflow_id` | The workflow that keeps the record, in the same draft. |
| `name` | What the record is called, such as Payments. Unique inside its workflow, ignoring case. |
| `role_id` | The one role that adds rows, in the same draft. |
| `viewer_role_id` | The one role whose members each see their own row of the record (decision 90, HS675), in the same draft. Empty when no role but the one that adds rows sees them. A group is accepted where a role is, as for `role_id`. |
| `form_name`, `questions` | The form each row is added with, in the same shape as a step's form. |

Create and update refuse with `ErrInvalidRecord` (400) a record with no name,
no role, two roles, two viewer roles, no workflow, no form name, no question, an
invalid question or a name another record of the workflow already has. A role
(adding or viewing) or a workflow that is not in the draft is `ErrRoleNotFound`
or `ErrWorkflowNotFound` (404). The viewer role is optional, and an update that
sends it empty, or leaves it out, clears it, as the update replaces the whole
record and does not merge into it. `agency_update_record` says so in its
description and `routes/record_viewer_test.go` pins it. Whether an update should
keep what it leaves out is the owner's call (HS672); until then a client that
means to keep the viewer sends it again.

`viewer_role_id` is declared the way `role_id` is: a column of `record` in
`agency.blueprint.json`, a field of `models.Record` and `models.DocumentRecord`,
checked in `validateRecord`, remapped onto the forked role in `CreateDraft` and
in restoring a publication, and named by the Issue scan when its role is gone.
`create_record`, `update_record` and `get_record` in `agency.operations.json`
serve it as `viewer_role_id` on the tool and on `/records`; no handler is
hand-written. The content hash adds it only when it is set, so an agency
published before the field existed does not read as changed.

Storing the field does not yet change who may read a row. The live read in
wakala-api (`agency_list_live_record_rows`) still shows every row to the role
that adds rows and to anyone else only the rows whose picked-from-a-list answer
names them; making `viewer_role_id` the gate is open for the owner.

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

### Order, corrections and double saves

A row may carry `order_id`, the one group order it was added from (decision
110); a row that belongs to no order leaves it empty. It is a stored column of
`record_row`, not an answer, with the index `record_order` on `record_id` and
`order_id`, and `agency_list_record_rows` can be narrowed by `order_id`. The
agency does not check that the order exists or that the person named in the
row belongs to it (open question Q100).

A saved row is never edited. A correction is a new row whose `replaces` names
the row it corrects (decision 111). The replaced row stays and reads as
history; the replacing row carries `corrected_on`, the moment it was added.
The replacing row takes the order of the row it replaces, and one that names
another order is refused. A row can be corrected once, so a second correction
names the replacing row. A row that does not exist, belongs to another record
or was already corrected is refused. All of these are `ErrInvalidRecordRow`
(400).

A row whose answers equal those of a current row of the same record and order,
saved less than 60 seconds earlier, is refused with `ErrDuplicateRecordRow`
(409). A row that was corrected away no longer counts as current.

## What the app reads

wakala-api serves these to the member app from the draft behind the latest
version published to the production channel (decisions 110 to 112). None of
them is a studio tool. A draft that is still open and edited after publishing
shows its edits at once (open question Q98).

| Tool | Meaning |
|---|---|
| `agency_list_live_records` | Each record with its form and `can_add_rows`, true for the agency owner and for the person seated in the record's role. |
| `agency_list_live_record_rows` | The current rows of one record by `record_code`, oldest first, narrowed by `order_id`; `history` true reads the rows a replacing row corrected instead. A role that adds rows sees every row, anyone else only the rows whose picked-from-a-list answer names them. |
| `agency_add_live_record_row` | Adds a row with `answers`, an optional `order_id` and an optional `replaces`. A repeat within 60 seconds is refused, 409 on the route. |
| `agency_get_my_roles` | The caller's `membership` (`owner`, `agent` or `resident`) and the one agency role her seat holds, if any. |
| `agency_list_my_open_steps` | The workflow steps of the roles the caller holds, with each step's kind, role, form name and questions. It does not know whether a step has been done. A caller who holds no role gets none. |

The Home card, the Inbox row, the Apps tile and the Start form read
`agency_list_my_open_steps`; the order page shows Send only when
`agency_get_my_roles` gives `owner`. For now an app is a workflow step, and
every member gets the Apps tab (decision 113).

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
- a record whose role, viewer role or workflow no longer exists.

None of the deletes cascade into the steps that point at what they remove.

## Not here

The Test pages' blocks over a record's rows are not settled (decision 89).

There is no SQL to mirror. The tables, the `order_id`, `replaces` and
`corrected_on` columns and the `record_order` index come from the blueprint when
an agency is provisioned (`Provision`, through `spec.Migrate`).
