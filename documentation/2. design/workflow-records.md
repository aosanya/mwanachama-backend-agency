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
| `item_name` | What one row is called, singular, such as Tender or Office holder. Optional; the generated workflows are named from it (see Generated workflows). |
| `edit_role_id`, `delete_role_id` | The one role that edits, and the one that deletes, a row through the generated workflows. Optional; empty means `role_id`. In the same draft. |
| `add_replaced_by`, `edit_replaced_by`, `delete_replaced_by` | The code of a designed workflow of the same draft that stands in for the generated add, edit or delete workflow. Optional; it must exist and must not be generated. |

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

## Row steps (ISS-121)

A `fill_form` step may write a row of a record. Two fields on `models.WorkItem`,
columns of `work_item` in `agency.blueprint.json`, in `DocumentWorkItem`, in
the fork, the import, the publication document, and in the content hash only
when set (record `WorkItemRow`, so an agency without them hashes as before):

| Field | Values | Meaning |
|---|---|---|
| `row_record` | a record code such as `RC-1` | The record whose row the step writes. |
| `row_action` | `add`, `edit`, `delete` | What the step does to it. |

Create and update refuse with `ErrInvalidWorkItem` (400): one of the two without
the other, an unknown action, a step that is not `fill_form`, a record code that
is not a record of the same draft, a step whose role is not the record's role
for that action (`role_id`, or `edit_role_id` and `delete_role_id` when set), an
`add` or `edit` whose form name and questions differ from the record's, a
`delete` that does not ask exactly one required `free_text` question (the
reason), and a workflow that holds a row step beside any other step. An update
that leaves `row_record` or `row_action` out keeps the stored value, as
`tool` is kept; sending them empty clears them. The Issue scan names a row step
whose record is gone. Code: `row_step.go`, `models/rowstep.go`.

## Retiring a row (ISS-121)

`RetireRecordRow(ctx, rowID, deletedBy, reason)` on the manager, tool
`agency_retire_record_row`, route `POST /record-rows/{row_id}/retire` with a
`reason` in the body and the caller as `deleted_by`. It marks one current row
`deleted` and writes `deleted_by`, `deleted_reason` and `deleted_on` (new
columns of `record_row`) and `last_updated`. Nothing is physically deleted and
no audit entry is written; the row keeps its answers. It refuses with
`ErrInvalidRecordRow` (400) a row that does not exist, one already retired, one
replaced by a correction (history, not a current row) and a blank reason. A
retired row can no longer be replaced, and does not count as a duplicate of an
equal row added afterwards.

`ListRecordRows` hides retired rows unless `RecordRowFilter.IncludeRetired`
(`include_retired` on the route) is true; a row that was never retired lists as
before. The route is a POST and not a DELETE because a reason travels in a body.
Code: `record_row_retire.go`.

## Generated workflows (ISS-121)

An agency has designed workflows, written in its file, and generated workflows,
made by the sibling from its records and never authored. Every live record has
three, one step each, all `fill_form` steps with `row_record` set to the
record's code and `row_action` set:

| Workflow code | `row_action` | Held by | Form |
|---|---|---|---|
| `RC-1.add` | `add` | `role_id` | the record's form and questions |
| `RC-1.edit` | `edit` | `edit_role_id`, else `role_id` | the record's form and questions |
| `RC-1.delete` | `delete` | `delete_role_id`, else `role_id` | `Why delete this <item>` asking `Why is it removed` |

The workflow name is the verb and the item with its article: `Add a tender`,
`Edit an office holder`. The item is `item_name`, else the record's
`form_name`, else its name. The step is titled like its workflow, has the
ordinality 1 and the code `RC-1.add.1`.

On a workflow, three fields: `generated` (true), `generated_for` (the record's
code) and `replaced_by` (the code of the designed workflow that stands in for
it, copied from the record's `*_replaced_by`; the generated workflow stays in
the agency and the hub hides it from Start). They are columns of `workflow` and
fields of `DocumentWorkflow`.

When they are made: on every fork of a draft or of a publication, on every
import, on publish (before the scan and the hash), and whenever a record is
created, updated or deleted or a workflow is deleted (`ensureGenerated`, also
the public `EnsureGeneratedWorkflows(ctx, draftID)`). Generation reuses the row
of a workflow with the same code, so ids stay the same across regenerations
inside a draft, updates the workflow and its step in place when the record's
questions, role, names or stand-ins changed, removes a second copy and a
workflow whose record is gone, and never changes a designed workflow. A draft,
publication or import written before this existed has none, and gets them at
its next fork, import or publish. Nothing is edited by hand for that.

They are ordinary workflows to everything else: a `run_workflow` step of a
designed workflow may name one (that is how a designed workflow uses
`Add a client` as a sub-workflow), reads list them with `generated: true`, and
they are started and run like any workflow. A run of one step is complete the
moment it is sent and waits on no second actor.

They are refused with `ErrGeneratedWorkflow` (409) on `UpdateWorkflow`,
`DeleteWorkflow`, and on creating, updating, moving into or deleting a step of
one. `CreateWorkflow` and `UpdateWorkflow` force `generated` false and the two
other fields empty. A record cannot be kept by a generated workflow.

The content hash leaves out generated workflows and their steps, so a record
counts once and a record change is the change that regenerates them.

The publish guard is `checkGeneratedWorkflows` in `generated_workflows.go`,
one of the Issue scan's checks that `validationErrors` runs before a publish.

## A workflow's subject and the row a run is started on

A workflow may name a **subject record**: `subject_record` on `Workflow` and on
the document workflow holds a record code, such as `RC-1`, of the same draft. It
says the workflow acts on one row of that record, so a run of it is started on
a row. An empty `subject_record` means no subject, and the workflow behaves as
every workflow did before the field existed.

`CreateWorkflow` and `UpdateWorkflow` refuse a code the draft does not keep with
`ErrInvalidWorkflow`, naming it; an update that leaves the field out keeps it,
and one that leaves it unchanged is not checked again. `ImportDocument`, and a
fork from a publication, refuse a workflow whose subject is not among the
document's records, in the same words, and nothing of the import is kept. A record
cannot be named by a workflow created before it exists, so an importer that
creates workflow by workflow creates the records first and then sets the
subject with `UpdateWorkflow`. The Issue scan reports a workflow whose subject
record is gone. The field is in the publication document, in the content hash
(naming a subject is a change to publish), and in a fork.

A run is the register rows that share an `order_id`. `RecordRow` gains
`subject_row_id` beside it: the row of the subject record the run was started
on. `CreateRecordRow` takes it from the caller and:

- refuses an id that is not a current row of the draft, naming it;
- gives a row of a run the `subject_row_id` its run already carries when it
  names none, and refuses one that names another;
- gives a replacing row the one of the row it replaces.

`ListRecordRows` narrows by `subject_row_id`, so every row of the runs started
on one row reads back together. A row without one, and every row written before
the column existed, behaves as before; no row is migrated, and the column is
added to an existing table by `Provision`.

## What the Issue scan flags

All are `error`, so none can be published:

- a `route` step that names no sent form, or has no route;
- a sent form, a route target or an otherwise target that no longer exists, or
  that now belongs to another workflow;
- a recipient role that no longer exists;
- a record whose role, viewer role or workflow no longer exists;
- a row step whose record no longer exists;
- a workflow whose subject record no longer exists;
- a live record without an adding role;
- for a live record, a missing, duplicated, altered or stale generated
  workflow (different name, stand-in, step, role, form or questions than the
  record now gives it), naming the record and the workflow code, and a
  generated workflow whose record is gone;
- a record whose `*_replaced_by` names no designed workflow of the draft.

A designed workflow that stands in for a generated one but has no
`run_workflow` step naming it is a `warning`.

None of the deletes cascade into the steps that point at what they remove.

## Not here

The Test pages' blocks over a record's rows are not settled (decision 89).

There is no SQL to mirror. The tables, the `order_id`, `replaces` and
`corrected_on` columns and the `record_order` index come from the blueprint when
an agency is provisioned (`Provision`, through `spec.Migrate`).
