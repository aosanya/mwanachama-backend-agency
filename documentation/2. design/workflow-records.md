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
| `code` | A readable label made from the name, such as `tender` (see A record's code and key). Unique in the draft and renamable. Roles carry permissions on it: `record.tender.edit.*`. |
| `key` | The hidden identity, minted once and never edited: a UUID for a record made since ISS-201 (D61), the counter code `RC-1` for an older one. Rows, steps, generated workflows and a workflow's subject name the record by it, and it stays the same in every later version. The counter code `RC-n` is deprecated as a code (D58) and lives on only as an older record's key. |
| `workflow_id` | The workflow that keeps the record, in the same draft. |
| `name` | What the record is called, such as Payments. Unique inside its workflow, ignoring case. |
| `form_name`, `questions` | The form each row is added with, in the same shape as a step's form. Each question of a record is a field with a `key` and a `code` (see A field's code and key). |
| `item_name` | What one row is called, singular, such as Tender or Office holder. Optional; the generated workflows are named from it (see Generated workflows). |
| `add_replaced_by`, `edit_replaced_by`, `delete_replaced_by` | The code of a designed workflow of the same draft that stands in for the generated add, edit or delete workflow. Optional; it must exist and must not be generated. |

A record has no role of its own. Who adds, edits, sees and deletes its rows is
given to roles as permissions on the record's code (see Record permissions
below); the four fields `role_id`, `edit_role_id`, `viewer_role_id` and
`delete_role_id` that a record once carried are gone (ISS-155, decisions D1, D6,
D35 and D36). See A record carries no role below.

Create and update refuse with `ErrInvalidRecord` (400) a record with no name,
no workflow, no form name, no question, an invalid question or a name another
record of the workflow already has. A workflow that is not in the draft is
`ErrWorkflowNotFound` (404). The update replaces the whole record and does not
merge into it, so a client sends `item_name` and the three `*_replaced_by`
fields again to keep them.

Storing nothing about a viewer, the sibling does not decide who may read a row.
The live read in wakala-api (`agency_list_live_record_rows`) decides it from the
`record.<code>.view.*` and field codes of the person's roles.

A fork copies the records onto the forked workflows. A publication
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
| `GET /record-rows?draft_id=` | `agency_list_record_rows` | List rows, narrowed by `record_id` or by `record_code`, and by `tags`. |
| `POST /record-rows/{row_id}/tags` | `agency_add_record_row_tag` | Add one tag id to a row. Nothing takes one off. |

A row names the draft whose record it was added to (any draft that exists, not
only an open one, because the app serves the published draft), the `record_id`,
the `role_id` the person acted as (the role of the row, which stays), who added it, and the `answers` by the
field's key. The row keeps the record's `record_code`, so it can be read across
versions.

A row is refused with `ErrRecordRowRefused` (403) unless its `role_id` is a role
of the draft whose permissions carry `record.<code>.add.*` or a
`record.<code>.add.<field code>` of the record (`roleAddsRows` in
`role_record_permissions.go`). A group, an empty role and a role that carries
only view, edit or delete codes are refused. Answers are refused with `ErrInvalidRecordRow` (400) when
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
| `agency_list_live_records` | Each record with its form and `can_add_rows`, true for the agency owner and for a person whose seat carries an add permission of the record. |
| `agency_list_live_record_rows` | The current rows of one record by `record_code`, oldest first, narrowed by `order_id`; `history` true reads the rows a replacing row corrected instead. A role that carries the view codes sees the fields they name, anyone else only the rows whose picked-from-a-list answer names them. |
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
| `row_record` | a record key (a UUID, or `RC-1` for an older record; a readable code is saved as the key) | The record whose row the step writes. |
| `row_action` | `add`, `edit`, `delete` | What the step does to it. |

Create and update refuse with `ErrInvalidWorkItem` (400): one of the two without
the other, an unknown action, a step that is not `fill_form`, a record code that
is not a record of the same draft, an
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

## Sending a step back (ISS-139)

Hub Steward decision 17 (IAS, 2026-11-23). A step that reviews earlier work may
be sent back: the run reopens the previous step for the role that holds it, with
the sender's reason. For now there is one target, the previous step. Where work
goes after a send-back may branch later, so the target is chosen by the caller
and never by the sibling.

### How a step declares its decision and reason

Two optional fields on `models.WorkItem`, set together:

| Field | Meaning |
|---|---|
| `send_back_decision` | The prompt of the yes or no question of the step's own form that holds the pick. Yes is Approve, no is Send back. |
| `send_back_reason` | The prompt of the free text question of the step's own form that holds the reason. |

They are the two prompts the open steps reply carries as `send_back {decision,
reason}`. They are columns of `work_item` in `agency.blueprint.json`, fields of
`models.WorkItem` and `models.DocumentWorkItem`, written by the fork, the
import, the publication document and the restore of a publication, and in the
content hash only when set (record `WorkItemSendBack`), so an agency without
them hashes as before. An update that leaves them out keeps the stored values,
as for `row_record`; sending them empty clears them.

Create and update refuse with `ErrInvalidWorkItem` (400): one prompt without
the other, a step that is not `fill_form`, a step no role acts on, a step that
writes a row (`row_record`), a decision prompt the step's own form does not ask
or that is not a `yes_no` question, and a reason prompt the form does not ask or
that is not a `free_text` question. Prompts are matched exactly. Code:
`send_back_step.go`, called from `validateStepShape`.

The sibling does not check that a previous step exists, because a workflow is
built step by step; the previous step is the nearest earlier `fill_form` step of
the same workflow that has a record, as the run already orders them, and the
caller finds it (see below).

### How a row is marked returned

Three new columns of `record_row` and fields of `models.RecordRow`, written
together and by nothing else, the way `deleted_by`, `deleted_reason` and
`deleted_on` are:

| Field | Meaning |
|---|---|
| `returned_by` | Who sent the row back, as the app names them. Empty for a row that was never returned. |
| `returned_reason` | The sender's reason, in her own words. Empty for a row that was never returned. |
| `returned_on` | When it was sent back. A row is returned exactly when this is not empty. |

`ReturnRecordRow(ctx, rowID, returnedBy, reason)` on the manager, tool
`agency_return_record_row`, route `POST /record-rows/{row_id}/return` with a
`reason` in the body and the caller as `returned_by`. It marks one current row
returned and writes the three fields and `last_updated`. It refuses with
`ErrInvalidRecordRow` (400) a row that does not exist, one already returned, one
retired, one replaced by a correction and a blank reason.

A returned row:

- no longer counts as done: `ListRecordRows` hides it unless
  `RecordRowFilter.IncludeReturned` (`include_returned` on the route) is true,
  so a run that reads its rows sees the step as not done and reopens it;
- frees its send key: the row's `send_key` is cleared in the same write, so the
  unique `send_key_unique` index lets the step be sent again for the same run.
  Clearing the key needs no change to a database that already holds the index,
  and the row still says which record and order it belonged to;
- stays in history with its answers, who returned it, why and when, and reads
  with `include_returned`; the reopened step prefills from those answers;
- no longer counts as a duplicate of an equal row added afterwards, and can
  no longer be replaced, retired or returned again.

The tool takes a row id and nothing about where the work goes next. Naming the
previous step, and later another target, is the caller's rule, so a branching
workflow changes the caller and not the row. A row that is both retired and
returned is refused, so the two reasons never share one row.

Code: `record_row_return.go`.

### Where a return is written and read (ISS-247, decision D85)

Writers of `returned_by`:

- `ReturnRecordRow` in `record_row_return.go`, the only writer: the update map
  and the returned struct. It is the place the sender's id is set.
- The route `POST /record-rows/{row_id}/return` and the tool
  `agency_return_record_row` reach it with the caller as `returned_by`.

Readers of the returned fields:

- `models.RecordRow` and the `record_row` columns in `agency.blueprint.json`,
  read whole by `ListRecordRows` with `include_returned`, and by the row readers of the document, the fork and the publication restore.
- `record_row_correction.go` and `record_row_retire.go` read only `returned_on`
  to refuse a replace or a retire of a returned row.
- `notReturned` in `record_row_return.go` filters on `returned_on`.

The sender's id (ISS-247, decision D85): `returned_by_id`, a fourth column of
`record_row` and field `ReturnedByID` of `models.RecordRow`, written in the same
update as `returned_by` and by nothing else. `returned_by` keeps the sender's
display name; the id is the sender's account id, so a reader can tell two people
with one name apart and find the account. A host that knows the account sets it
with `WithReturnedByID(ctx, id)` on the context it passes to `ReturnRecordRow`.
The route and the tool carry only the caller's name, so a return made through
them leaves the id empty. An empty id is a valid value: rows returned before
the field existed, and returns made without an id, read exactly as they did,
and a send that follows a return neither copies nor clears the id of the
returned row. The field is read wherever the row is, with `include_returned`.

## A form step names its record (ISS-140)

One optional field on `models.WorkItem` and `models.DocumentWorkItem`:
`record_code`, the code of the record the step's form writes, such as
`bid_decision`. It is a column of `work_item` in `agency.blueprint.json`,
written by the fork, the import, the publication document and the restore of a
publication, and in the content hash only when set (record
`WorkItemRecordCode`), so an agency without it hashes as before. An update that
leaves it out keeps the stored value; sending it empty clears it.

Create and update refuse with `ErrInvalidWorkItem` (400) a `record_code` on a
step that is not `fill_form`. The sibling does not check the format of the
code, that a record with that code exists, or that no other step names it: the
platform that imports the file owns those checks and derives the record. Code:
`record_code_step.go`, called from `validateStepShape`.

## Generated workflows (ISS-121)

An agency has designed workflows, written in its file, and generated workflows,
made by the sibling from its records and never authored. Every live record has
three, one step each, all `fill_form` steps with `row_record` set to the
record's code and `row_action` set:

| Workflow code | `row_action` | Role of the step | Form |
|---|---|---|---|
| `record.<code>.add` | `add` | none | the record's form and questions |
| `record.<code>.edit` | `edit` | none | the record's form and questions |
| `record.<code>.delete` | `delete` | none | `Why delete this <item>` asking `Why is it removed` |

The workflow name is the verb and the item with its article: `Add a tender`,
`Edit an office holder`. The item is `item_name`, else the record's
`form_name`, else its name. The step is titled like its workflow, has the
ordinality 1 and the code `record.<code>.add.1` (D75). The workflow code is built by
`generatedCode` in `generated_workflows.go` through `RecordPermissionCode`, so for
the record `tender` the workflows are `record.tender.add`, `record.tender.edit` and
`record.tender.delete`, and the add step is `record.tender.add.1`. `<code>` is the
record's current code. The step code is a name and not a permission: the
permissions of the record are `record.tender.add.<field>` and
`record.tender.add.*`, so no permission equals a workflow code. Which role holds a
generated step is decided by the hub (ISS-157): a role holds the add step when it
holds add for every required field of the record (`add.*`, or an `add.<field>` for
each required field), and for a record with no required field when it holds
`add.*` or at least one `add.<field>`.

A generated workflow belongs to its record by `generated_for`, the record's key,
and to its action by the last part of its code. Generation matches an existing
workflow by those two, not by its code. A draft that holds the old codes
(`<key>.add`, `<key>.add.1`, the key being a UUID or `RC-1`) and a record whose
code is renamed are therefore renamed in place: the workflow row and its step row
are updated, so their ids, the role assignments, scenario inputs and `run_workflow`
steps that name them, and the replaced-by link stay, and the rename never goes
through `removeGeneratedWorkflow`. A rename of the record's code renames its
three workflows and their steps at the same time, and the Issue scan names a
generated workflow or step whose code is not the one the record gives now.

On a workflow, three fields: `generated` (true), `generated_for` (the record's
key) and `replaced_by` (the code of the designed workflow that stands in for
it, copied from the record's `*_replaced_by`; the generated workflow stays in
the agency and the hub hides it from Start). They are columns of `workflow` and
fields of `DocumentWorkflow`.

When they are made: on every fork of a draft or of a publication, on every
import, on publish (before the scan and the hash), and whenever a record is
created, updated or deleted or a workflow is deleted (`ensureGenerated`, also
the public `EnsureGeneratedWorkflows(ctx, draftID)`). Generation reuses the row
of the workflow of the same record and action, so ids stay the same across regenerations
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

## A record carries no role (ISS-155, decisions D1, D6, D35, D36)

`role_id`, `edit_role_id`, `delete_role_id` and `viewer_role_id` left
`models.Record`, `models.DocumentRecord`, the `record` object of
`agency.blueprint.json`, the fork, the publication document and the content
hash. Nothing reads them: the Issue scan no longer names a record whose role is
gone, and a record without an adding role is no longer a finding.

| Field that is gone | The codes that replace it |
|---|---|
| `role_id` | `record.<code>.add.*` or `record.<code>.add.<field>` |
| `edit_role_id` | `record.<code>.edit.*` or `record.<code>.edit.<field>` |
| `viewer_role_id` | `record.<code>.view.*` or `record.<code>.view.<field>` |
| `delete_role_id` | `record.<code>.delete` |

- **Refused, not folded.** A document, a publication being forked, or a request
  body that still carries one of the four with a value is refused with
  `ErrInvalidRecord` (400), naming the field and the codes above for that
  record: `role_id is gone from a record: give a role record.tender.add.* or
  record.tender.add.<field> instead`. A field sent empty is not a carry and is
  accepted. Nothing of an import is kept when one record is refused. There is no
  fold of the old fields into role permissions: the roles are the owner's to
  give.
- **How the refusal is read.** `models.Record` and `models.DocumentRecord` read
  the four names in `UnmarshalJSON` into an unexported list
  (`CarriedGoneRoleFields`) and drop the values; `refuseGoneRoleFields` in
  `record_role_fields.go` makes the refusal. `CreateRecord`, `UpdateRecord` and
  `applyDocumentToDraft` (import and fork from a publication) call it. Reading a
  stored publication that still holds the fields works, so older versions can be
  listed; forking one of them is refused until its records are rewritten.
- **The generated step holds no role.** `generatedPlansOf` leaves `RoleID`
  empty on the add, edit and delete steps, and `RoleOfRowAction` is removed.
  Who may do the step is the permission codes. A designed row step keeps the role
  it names (D12) and is no longer checked against the record.
- **A row's role stays.** `RecordRow.role_id` is the role the person acted as.
  `CreateRecordRow` accepts it only when that role carries an add code of the
  record.
- **The hash.** A record hashes its id, workflow, name, form, questions and
  deleted flag, and the optional code, item name and stand-ins. Every agency
  with a record therefore reads as changed once, which is the breaking change.
- **Fixtures.** The hub-demo files carry the add code on the role that used to
  be the record's `role_id`.

## Record permissions (ISS-150, ISS-182, decisions D20, D21, D29, D30, D36, D37)

Every record gives its permissions, derived by `RecordPermissionsOf(record)` in
`record_permissions.go`. ISS-150 gave four plain codes; ISS-182 replaced them with
field-level codes, so the plain `record.<record code>.add`, `.edit` and `.view` no
longer exist (D36). The code is built only by
`RecordPermissionCode(recordCode, action, field...)`, which the generated workflow
naming reuses later so that the two never drift apart. The star and the field part
are added there and nowhere else, and a `delete` code drops any field part.

| Code form | Meaning |
|---|---|
| `record.<record code>.add.<field code>` | May fill that field when adding a row. |
| `record.<record code>.add.*` | May fill every field the record has at the time of the check. |
| `record.<record code>.edit.<field code>` | May change that field of a row. |
| `record.<record code>.edit.*` | May change every field the record has at the time of the check. |
| `record.<record code>.view.<field code>` | May see that field of a row. |
| `record.<record code>.view.*` | May see every field the record has at the time of the check. |
| `record.<record code>.delete` | May delete a row. No field part and no star (D29). |

A tender record has ten fields, so it gives ten add, ten edit and ten view
field codes, the three stars and `record.tender.delete`: 34 codes. Add, edit and
view come first by action, each action listing the fields in the record's order and
then its star; delete comes last. A record with no fields gives the three stars and
delete.

- **The field part is the field's stored code** (ISS-178, D37), for example
  `closing_date`, read from `StepQuestion.Code`. It is never worked out from the
  prompt when the permissions are derived, so rewording a prompt changes no code. A
  field with no stored code gives no field code (a saved record always has codes; a
  document that carries none is given the same codes the import stores, see below).
- **The star is stored as written** in a role's list and means every field the
  record has when the check runs, so a field added later is covered by a star and
  never by a role that names fields (D30). A named field that the record no longer
  has leaves its code on the role until the scan refuses the publish.
- **Add and edit need no view** (D43, D44): a role may carry `edit.<field>` or
  `add.<field>` without any view code, and nothing refuses it.
- **No tag, group or seat** appears in a code (D24, D35).

| Code | Name | Meaning |
|---|---|---|
| `record.tender.add.closing_date` | `Add the closing date of a tender` | Fill that field when adding. |
| `record.tender.add.*` | `Add every field of a tender` | Fill every field when adding. |
| `record.tender.edit.closing_date` | `Edit the closing date of a tender` | Change that field. |
| `record.tender.view.closing_date` | `See the closing date of a tender` | See that field. |
| `record.tender.delete` | `Delete a tender` | Delete a row. |

The names are built in one function, `recordPermissionName` (ISS-205, D66). The role
page and the Role card show the long wording, a full sentence, with the permission's
code beside it. A field permission reads `<Verb> the <field> of <a item>`, a star reads
`<Verb> every field of <a item>`, and delete reads `Delete <a item>`. The verbs are
`Add`, `Edit` and `See` (the view permission). The item in the name is the record's
`item_name`, else its `form_name`, else its name, lower-cased, with `a` or `an` as in the
generated workflow names (`Add an office holder`). The field in the name is its prompt
with the first letter lowered (`Closing date` reads `the closing date`), kept as
written when the second letter is a capital (`USD amount`), and its code when it has no
prompt. The name carries no code; the code is the permission's own `code` field, and
the page that lists permissions shows it beside the name. Each permission is not
locked, has an empty `actions` list (an empty list, not none) and no module. The
static catalogue of 109 is unchanged; no catalogue code starts with `record.`, so a
derived code never equals one, and `record_permissions_test.go` pins both.

### A publication lists the record permissions its records give (ISS-152, decisions D24, D30, D36, D38)

A publication document carries, beside its records and roles, `record_permissions`:
the permissions every record of the draft gives, so a version states what it grants
and the hub reads the version, not the draft. It is a list of `models.Permission`
(`code`, `name`, `locked`, `actions`), in the order of the records and, inside a
record, in the order `RecordPermissionsOf` gives them: the add, edit and view code of
each field, each action closed by its star as written, then `record.<record code>.delete`.
A tender with `title`, `issuer` and `closing_date` lists 13 codes; a record with one
field lists 7.

- **It is derived, never a source.** `buildDocument` appends `RecordPermissionsOf(record)`
  for each record when a draft is published. A fork of a draft or of a publication
  restores the records and ignores the list; the next publish derives it again from the
  records the fork holds, so a record edited in the fork lists its new codes.
- **It holds no tag, group or seat** (D24, D35): those are set after the publish. It
  holds no plain `add`, `edit` or `view` code either (D36).
- **It is omitted when no record exists** (`omitempty`), so the document of an agency
  without records is the document it was before.
- **The content hash adds it only when records exist.** `contentHashFor` writes one
  `RecordPermissions` entry per record, so an agency without records hashes as before.
  A renamed field code, a new field or a changed item name is a change to publish,
  because the live Instance takes the new codes at the next publish (D38).

Code: `buildDocument` in `document.go`, `contentHashFor` in `publication_impl.go`;
tests in `record_publication_permissions_test.go`.

### Record permissions in a role's list (ISS-151, decisions D9 and D14)

A role's `permissions` list takes the generated codes beside the catalogue
codes, in the same list. Only a code that starts with `record.` is judged
against the draft's records; every other code is judged as before, so a role
that carries catalogue codes only behaves exactly as it did.

A `record.` code is accepted when `RecordPermissionsOf` gives it for a record of
the same draft, and refused otherwise (`ErrUnknownPermission`, naming the role
and the code), for example `record.nothing.add.closing_date`, `record.tender.sell`,
the plain `record.tender.add`, a field the record does not have, or
`record.tender.delete.closing_date`. The refusal is made by:

- `CreateRole`, and `UpdateRole` when the update carries `permissions`;
- `SetRolePermissions`;
- `ImportDocument` and a fork from a publication, which both apply a document:
  the codes are checked against the document's own records, with their fields
  settled the way the import settles them (`keyedFields`), so a document written
  before field codes existed is judged by the codes the import will store;
- a fork from a draft copies the roles and the records together, so the codes
  stay valid.

The code that checks is `role_record_permissions.go`; the codes come from
`RecordPermissionsOf` only. A code of a record of another draft gives nothing.

Deleting a record does not touch the roles that still carry its codes. The Issue
scan (`checkPermissions`) raises an `error` for each such code, `Role R-4
"Clerk" carries permission "record.tender.edit.closing_date", which no record gives`, so a
publish is refused and names the role and the code (D9); when a field is removed from
a record while a role names it, the finding reads `Role R-4 "Clerk" carries permission
"record.tender.edit.closing_date", but record "tender" has no field "closing_date"`, and a
role that holds the star is unaffected (D30). A role save that carries a plain
`record.tender.edit` is refused with `which is now record.tender.edit.* for every field
or record.tender.edit.<field> for one` (D36); the wording is built once, from the code,
in `role_record_permissions.go`, and the scan reads it from there. The scan's other
refusals (D51) are a record no role may add to and a required field no role holds
an add code for, listed under What the Issue scan flags. A code of a record that exists is
accepted by the scan and carried into the publication document with the role.

## A field's code and key (ISS-178, decisions D37 and D38)

A record's fields are its `questions`, a list of `models.StepQuestion` stored as
one json column of `record`; a step's form uses the same type. Every field of a
record has two names of its own besides its prompt:

| Name | Meaning |
|---|---|
| `key` | A hidden, stable identity such as `f_3b5f1a39c58f1114`. The agency mints it when the field is made and nothing edits it. Answers are saved under it. |
| `code` | A readable label on the key, such as `closing_date`: lower case letters, digits and underscores, starting with a letter, at most 40 characters when proposed. Unique inside the record. It may be renamed. |
| `prompt` | What the person is asked. It may be reworded; the field stays the same field. |

A step's questions leave `key` and `code` empty. They only matter for a record.

### How a field gets its key and code

`record_field.go` settles the fields of every record that is written:

- **Create.** Each field gets a new random key. A code that is given must follow
  the rule above and must not repeat inside the record, or the create is refused
  (`ErrInvalidRecord`, 400). A field with no code gets one proposed from its
  prompt (`Closing date` gives `closing_date`, `Amount (USD)!` gives
  `amount_usd`), with `_2`, `_3` added when the record already has it. A code that
  looks like a key (`f_` and sixteen hex digits) is refused so a code can never
  be read as a key.
- **Update.** The update replaces the whole record, so it sends the fields again.
  A field that sends its `key` is that field: its prompt, code and the rest may
  change and the key stays. A key the record never had is refused
  (`ErrInvalidRecord`). A field that sends no key is matched to an existing field
  by its code, then by its prompt, and takes that field's key (and its code, when
  it sends none); a field that matches nothing is a new field and gets a new key.
  A client that does not know about keys therefore loses nothing.
- **Fork, publication and restore.** The fields are copied whole, key and code
  with them, so a row added to one version reads the same field in every later
  version (rows name their record by `record_code`).
- **Import, and a fork from a publication made before this change.** A field with
  no key gets one worked out from the record's code, the prompt and the number of
  earlier fields with the same prompt (`legacyFieldKey`), and a code proposed from
  its prompt. Because it is worked out and not drawn at random, the same old file
  imported twice, the same field read from an old publication and the same field
  migrated in a stored database all get the same key. A file whose fields repeat a
  code is refused.

### Proposing codes for a list of prompts

`ProposeFieldCodes(prompts []string) []string` in `record_field.go` is the one
place the numbering rule lives. It gives each prompt the code `FieldCodeOf` makes
from it and adds `_2`, `_3` when an earlier prompt of the list already took the
code: `Closing date`, `closing date` and `Closing date` give `closing_date`,
`closing_date_2` and `closing_date_3`; a prompt with no letters or digits gives
`field`, then `field_2`. The create path (`settleCodes`) numbers the fields that
sent no code through the same loop, with the codes that were given already taken,
so a record whose fields send no code gets exactly the codes the function
returns for its prompts. A caller outside this repo that has to know the codes
before the record is made, such as the hub's import checking role permissions,
calls this function and keeps no copy of the rule.

### Renaming a code

A code is a label, so renaming it changes nothing that is saved: the key stays,
the rows keep their answers under it, and the next generated form carries the new
code. A rename is a change to the record, so it is a change to publish; the live
Instance takes the new code at the next publish. The old code stops naming the
field at once (a row written with it is refused as not a question of the form).

Field permissions exist since ISS-182 (`record.<record code>.<action>.<field code>`,
D36 and D37), so a role can now carry a field's code. Following a rename, so that a
role that carries the old code takes the new one in the same save (D38), is **not
built yet** and is a follow-up issue: today the old code stays on the role until the
scan raises the error above, and the role is edited by hand. The star needs no
follow-up.

### What reads a field, and by which name

| Where | How it names a field |
|---|---|
| `models.RecordRow.Answers`, `PlaceRefs` | `map[string]string` keyed by the field's `key`; the `answers` and `place_refs` columns of `record_row`. |
| `validateRowAnswers` in `record_impl.go`, called by `CreateRecordRow` | Takes each name of a row's answers as a key, else a code, else a prompt (the last so a caller that has not moved yet is still understood), and saves it under the key. Naming one field twice, or a field the record does not have, is refused with `ErrInvalidRecordRow`. A blank required field, and an answer that does not fit its type, are refused as before. `PlaceRefs` are turned into keys the same way. |
| `RowAllowance.Extra` and `Waived` in `row_allowance.go` | An extra detail, and a waived required question, are named by prompt; they are not fields of the form. |
| `approval_stamp.go`, and the `Changed: ` prefix in `record_impl.go` | The four stamp prompts and the amendment lines are answers that are not fields of the form; they are kept under their own text. A key never looks like one of them. |
| `refuseDuplicate` in `record_row_correction.go` | Compares two rows' keyed answers with `maps.Equal`. |
| `ListRecordRows`, `record_row_retire.go`, `record_row_return.go` | Read and keep the answers as stored, by key. |
| `rowStepFormFits` in `row_step.go` | A designed row step must take the record's form; keys and codes are left out of the comparison, so a stand-in written without them still fits. |
| `sameQuestions` in `generated_workflows.go` | Compares the generated step's questions with the record's whole, keys and codes included, so a rename rewrites the generated steps. |
| `document.go`, `publication_impl.go` | The publication document carries `key` and `code`. The content hash leaves keys out, and carries a code only when it is not the one the prompt proposes, so an agency published before this change does not read as changed, and a renamed code does. |
| `ListInputSources` in `input_source_impl.go`, `recordSourceType` and `recordSource` in `compute_step.go` | Still name a record field by prompt (`StepBinding.Question`). Moving a binding to the key is a separate issue. |

### Rows already stored

`Provision` runs `carryFieldKeysForward` each time it starts. It reads every
record, gives the fields that have no key the key `legacyFieldKey` works out and
the code the prompt proposes, rewrites the answers and place references of that
record's rows from prompt to key in the same transaction, and then regenerates
the generated workflows of the drafts it changed so their forms carry the keys
(the same pass that every record save makes, so it also settles a generated
workflow that was missing or doubled). Stamps and amendment lines stay as they
were. A record whose fields all have keys is not touched, and a second run changes
nothing. Every run also reads every row, in batches of 500, and moves a row that
still names a field by prompt (a row an older process wrote after the record was
keyed); a row that is already keyed is left alone. The statements run inside one
transaction and have been run on SQLite only.

Outside this repo the hub's `agency_add_live_record_row` and
`agency_list_live_record_rows` write and read `answers`, and the app shows them.
A write by prompt is still understood and saved by key; a read now returns keys,
so the hub and the app must move to the field list's keys (or codes) when the hub
takes the new sibling tag.

## A record's code and key (ISS-195, decision D58)

The counter code `RC-n` is deprecated as a record's code. A record has two names
of its own, set the same way as a field's code and key (see A field's code and
key above):

| Name | Meaning |
|---|---|
| `key` | The hidden identity, minted once and never edited: a UUID from `newID()` for a record made since ISS-201, or the counter code (`RC-1`, `RC-2`) an older record was given. Everything that points at a record points at the key. |
| `code` | A readable label made from the name, such as `payments` for `Payments` or `tender_register` for `Tender register`. Lower case letters, digits and underscores, starting with a letter. Unique in the draft (not only in the workflow). It may be renamed. |

A code never looks like a key: a code that is `RC-9`, that has a capital, a space,
a dash or a dot, that starts with a digit or that has the shape of a field key
(`f_` and sixteen hex digits) is malformed.

### How a record gets its code and key

`record_code.go` settles every record that is written:

- **Create.** The key is a new UUID (ISS-201, D61); the `record` counter is no longer drawn from. A code that is given must be well
  formed and not held by another record of the draft, or the create is refused
  with `ErrInvalidRecord` (400). A record with no code gets one proposed from its
  name (`FieldCodeOf`'s rule: `Payments` gives `payments`, a name that starts
  with a digit gives `record_2024`), with `_2`, `_3` added when the draft already
  has it. A second writer racing for the same code meets the unique
  `draft_code_unique` index and is answered `ErrConflict`.
- **Update.** The update replaces the whole record. A code left out keeps the
  record's code; a code sent that differs renames the record. The key never
  changes, whatever is sent.
- **Fork from a draft.** The records are copied whole, key and code with them.
- **Import and fork from a publication.** `settleDocumentRecords` runs first. A
  record that has a key keeps it, and its code must be well formed and unique in
  the file (`ErrInvalidRecord`, nothing kept). A record with no key, as every
  record of a file or publication made before ISS-195, takes its old code as its
  key and gets a code proposed from its name; a record that carries a well formed
  code and no key (a hand written file) keeps the code and takes it as the key
  too. Records are settled in the order of the file, the ones that already hold a
  well formed code first, so the same old file imported twice gives the same codes
  and keys. The roles of the file have their `record.RC-1.add` codes moved to the
  readable code, a workflow's `subject_record` and a step's `row_record` written
  as a readable code are saved as the key, and `advanceCodeSequences` raises the
  `record` counter past every key the file brought so a new record never takes a
  key that is held.
- **Provision.** `carryRecordKeysForward` (in `provision_record_codes.go`, run at
  the start of `carryFieldKeysForward`) settles every stored record that has no
  key the same way, writes the code and the key columns, and moves the
  permissions of the roles of that draft in the same transaction. It reads and
  writes no row of `record_row`; the generated workflows are named from the key,
  so they are left as they are. A second run changes nothing.

### Renaming a code

The key stays and the roles follow in the same save (D38): `UpdateRecord` moves
every permission `record.<old code>.<action>` (and the field part after it, when
field permissions exist) of every role of the draft to the new code, in the
transaction that writes the record. A permission of another record whose code
starts with the old one (`payments_extra` beside `payments`) is not touched. A
rename that is refused changes nothing. The old code stops naming the record at
once, so a role created afterwards that carries it is refused. A rename is a
change to the record, so it is a change to publish: the record's code is part of
the content hash unless it is the one its name proposes (so an agency published
before ISS-195 does not read as changed because of the code alone, though its
roles' permission codes are new).

### What reads the code and what reads the key

| Where | Names the record by |
|---|---|
| `RecordPermissionCode` and `RecordPermissionsOf` in `record_permissions.go`; `recordPermissionCodesOf` and `validateDocumentRolePermissions` in `role_record_permissions.go` | The code: `record.tender.edit`. |
| `ListRecords` (the `code` filter), the Issue scan's messages | The code. |
| `models.RecordRow.RecordCode` (`CreateRecordRow`), and the duplicate and replace checks in `record_row_correction.go` | The key, so a row keeps naming its record after a rename and the rows of an older database are left as they were. |
| `ListRecordRows` with `record_code` | The code or the key: a code is turned into the key of the record of the draft that holds it, and a value that is no code is read as a key. |
| `generatedCode`, `generatedPlansOf` and the Issue scan in `generated_workflows.go` | The code for the workflow and step codes (`record.<code>.add`, `record.<code>.add.1`); the key for `generated_for` and the step's `row_record`. |
| `row_record` of a step (`validateRowStepReferences`), `subject_record` of a workflow (`validateWorkflowSubject`), `checkRowSteps`, `checkWorkflowSubjects`, `validateDocumentSubjects` | The key. A readable code written there is saved as the key (`recordKeyOfRef`, and `settleDocumentRecords` for a file), so a rename never leaves them behind. |
| `HomePartsOf` in `models/home_refs.go` | Both: a home entry's `target` or `page_of` resolves by the code or the key. Since ISS-201 a write turns the code into the key (see below), so a rename leaves it whole; an entry written by code before ISS-201 still resolves by the code until the code is renamed. |
| `keyedFields`, `recordFieldsOf`, `legacyFieldKey` | The key (`Record.Identity()`, the key or, for a record that has none, the code), so the field keys worked out for an old file and for a migrated database stay the same. |
| `StableIdentity` and `Resolve` in `identity_owner.go`; `advanceCodeSequences` in `import.go` | The key. |
| `WorkItem.RecordCode` | Unchanged: a `fill_form` step names a record the platform derives; the sibling does not look it up. |
| The hub's imports | Outside this repo: they read a row's `record_code` and a step's `row_record`, which are keys, and a record's `code` and `key`. |

## A record's key is a UUID, and a home entry names it by the key (ISS-201, decisions D61 and D62)

The hidden key of a newly made record is a UUID, not the next `RC-n` and not a
hash (the issue first said a hash; the owner's answer is the UUID). Old records
keep the key they hold. A home entry that names a record by its code is saved as
the key. This section is the inventory the change was made from and what it did.

### Where the key is minted

| Where | What it does |
|---|---|
| `insertRecord` in `record_code.go` | The only place a key is minted for a record the platform makes. It is `newID()` (`store.go`, a UUID). Before ISS-201 it was `claimCode`, the next `RC-n` of the `record` counter. |
| `settleRecordIdentities` in `record_code.go`, for import, fork from a publication and `carryRecordKeysForward` | Unchanged. A record that arrives without a key takes its own old code as the key (`RC-1` stays `RC-1`). It mints nothing. |
| `advanceCodeSequences` in `import.go` and `numberIn` | Unchanged. It raises the `record` counter past the highest `RC-n` key a file brought, so the counter stays past every old key. A UUID has no `RC-` part, so it counts as 0 and moves nothing. |
| `randomFieldKey` and `legacyFieldKey` in `record_field.go` | The model for the field, and untouched. A new field gets a random `f_` plus sixteen hex digits; only a field read from an old file gets a worked out one (sha256 of the record's key, the prompt and the number of earlier fields with that prompt). |

What the new key is made of: nothing worked out. A record has no property that is
stable (its name and code are both editable), so a hash of them would repeat for
two records with the same name or move on a rename. The key is drawn once at
create, the way a new field's key is, from the repo's row id generator. A hash is
kept only where a key must come out the same on a second read of old data
(`legacyFieldKey`), and a record that has no key is given its old code, not a
hash or a UUID, so the same old file imported twice gives the same keys.

### Where the key is read

| Where | Reads the key as |
|---|---|
| `Record.Identity()` and `DocumentRecord.Identity()` in `models/record.go` | The key, else the code. |
| `generatedCode`, `generatedPlansOf` and the Issue scan in `generated_workflows.go` | A generated workflow's `generated_for` and the step's `row_record`. Since ISS-153 the workflow code is `record.<code>.add` and the step code `record.<code>.add.1`, built from the code. A generated workflow is found by its `generated_for` and the action at the end of its code, so a UUID resolves as `RC-1` did and a rename of the record's code renames the workflows and steps in place. |
| `CreateRecordRow` in `record_impl.go` and `record_row_correction.go` | `RecordRow.RecordCode` is the key. |
| `row_step.go`, `workflow_subject.go`, `document.go` | `row_record`, `subject_record` and the document's records, by the key. |
| `recordFieldsOf`, `keyedFields`, `legacyFieldKey`, `fieldSettlement` | The key, as the seed of an old field's worked out key. |
| `StableIdentity` and `Resolve` in `identity_owner.go` | `StableIdentity` reads the `key` column. `Resolve` cuts the identity at the first `-` and reads the front as an object prefix (`RC`); an identity that is a UUID has no such prefix, so `Resolve` reads it as the key of a record of the open draft. |
| `carryRecordKeysForward` in `provision_record_codes.go` | Only a stored record with no key. A stored key, `RC-n` or a UUID, is never touched. |
| The `record` entry in `codedRoles` and `backfillCodes` in `provision.go` | The `record` counter and prefix stay, for the old keys. |
| `agency.blueprint.json` (the `key` column of `record`) and `agency.operations.json` | Their descriptions name the key as a UUID, or `RC-1` for an older record. |

A permission code (`record.<code>.add`) and the `code` filter use the readable code,
and nothing in the repo splits a key at a dot or a dash except `Resolve`.

### A home entry names a record by its key

| Where | What it does |
|---|---|
| `HomeEntry.PageOf` and `HomeEntry.Target` (with `TargetType` `record`), `DocumentHomeNode.PageOf` and `Target.Ref` in `models/home.go` | A free string up to 200 characters. |
| `CreateHomeEntry` and `SetHome` in `home_impl.go`, and `ImportDocument` and a fork from a publication in `document.go` | Resolve the `page_of` and the `target` of a record entry through `HomeParts.KeyRecordRefs` and save the key. The reference may be the record's code, its key or its name (`ResolveHomePart`'s order: code or key first, then a name that only one record has). |
| `HomePartsOf`, `ResolveHomePart` and `checkHomeNode` in `models/home_refs.go` | Resolve a reference when the home is checked, by code, key or name. |

A reference that matches no record, or a name that matches more than one, is saved
as it was written, since a home may be written before its records are, and
`HomeParts.CheckHome` still reports it. A file is resolved against its own records
after their keys are settled, so an old file whose record has no key saves the
old counter code as the key. A rename of the record's code never touches a home
entry, because the entry holds the key. An entry saved before ISS-201 holds the
code it was written with and is not migrated: it still resolves until that code is
renamed.

## A workflow's subject and the row a run is started on

A workflow may name a **subject record**: `subject_record` on `Workflow` and on
the document workflow holds a record key (a UUID, or `RC-1` for an older record) of the same draft (the
record's readable code is understood too and is saved as the key). It
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

## Row tags (ISS-186, decisions D23, D27 and D28)

The row dimension of record permissions is the tag. `RecordRow` carries `tags`,
a json list of tag ids kept as plain strings, a column of `record_row` in
`agency.blueprint.json`. The agency knows no tag table and does not check that a
tag exists: the hub keeps the tag list, and a row holds only the ids, so
renaming a tag in the hub moves nothing here. The sibling stores no owner of a
row for reach; who added a row or who it names never decides who reaches it
(D47).

| Case | What the agency does |
|---|---|
| Add a row | `CreateRecordRow` keeps the `tags` it is given (the hub passes the tags of the group the person assumed, D27), without duplicates and in the order given. A blank tag is refused with `ErrInvalidRecordRow` (400). A row given none carries an empty list. |
| Add a tag | `AddRecordRowTag(ctx, rowID, tag)`, tool `agency_add_record_row_tag`, route `POST /record-rows/{row_id}/tags` with `tag` in the body. It appends the tag, and a tag the row already carries changes nothing. It refuses with `ErrInvalidRecordRow` (400) a blank tag and a row that does not exist, is retired, was returned or was replaced by a correction, as the retire and return operations do, because a tag added to a history row would never be seen. The row is locked in the transaction, so two adds of different tags both stay. |
| Take a tag off | No operation does. The manager has no method for it and the routes have no verb for it (`TestRecordRowTags_NothingTakesATagOffARow`, `TestRecordRowTagRoutes`). After the add, tags change only by adding a further tag (D27). |
| Edit a row | An edit is a replacing row (`replaces`). `settleReplacement` gives it the tags of the row it replaces and ignores any the replacing request names, so an edit never adds or drops a tag (D27, D53). |
| Rows from before | A row written before the column existed has no tags and lists as an empty list; no row is migrated, and `Provision` adds the column to an existing table. |

`ListRecordRows` takes `RecordRowFilter.Tags` (`tags` on the route, repeated
for several) and keeps the rows that carry any one of them, so the hub does not
read every row of a publication to filter in memory. A row is reached when it
and the group share a tag; the system tag `all` reaches every row (D28), so a
filter that is empty or names `all` keeps every row. A filter that names only a
blank tag keeps none, so a malformed call never widens the list. The filter
adds to the other filters (`record_id`, `order_id` and the rest) and to the
default hiding of retired and returned rows.

The filter reads the json column with a `LIKE` on the tag as encoded, then
checks each row's tag list exactly, so `north` does not match `north-east`. A
json text column has no portable index a contains test could use, so the filter
narrows by draft, record and order first and scans the rest of the rows of the
draft; a table of tagged rows that is too large for that is the case for a row
tag table, and the hub's choice of reach does not depend on which holds the tags.

A `RecordRow` always serialises `tags` as a list, an empty one when it carries
none, never `null`. Code: `record_row_tags.go`, `settleReplacement` in
`record_row_correction.go`, the `Tags` filter in `filters.go`.

## What the Issue scan flags

All are `error`, so none can be published:

- a `route` step that names no sent form, or has no route;
- a sent form, a route target or an otherwise target that no longer exists, or
  that now belongs to another workflow;
- a recipient role that no longer exists;
- a record whose workflow no longer exists;
- a row step whose record no longer exists;
- a workflow whose subject record no longer exists;
- for a live record, a missing, duplicated, altered or stale generated
  workflow (different name, stand-in, step, form or questions than the
  record now gives it), naming the record and the workflow code, and a
  generated workflow whose record is gone;
- a record whose `*_replaced_by` names no designed workflow of the draft;
- a record that no role may add to (ISS-154, D51): no role of the draft carries
  `record.<code>.add.*` or any `record.<code>.add.<field>`. The finding names the
  record by code and name;
- a required field that no role holds an add code for (ISS-154, D51): no role
  carries `record.<code>.add.*` or `record.<code>.add.<field>` of that field. The
  finding names the record and the field code. It is raised only for a record
  that some role may add to, so a record no role may add to is one finding and
  not one more per field.

The two findings read the permission codes of the roles only (`record_add_scan.go`,
`checkRecordAdds`). They do not check that a role that may add can also view
(D44), a role that may edit and cannot view (D43), a delete without view, or
anything about groups or tags, which exist only after the publish (D24, D35). An
agency without records scans as before. A template that holds a record therefore
publishes only once a role carries an add code for it.

A designed workflow that stands in for a generated one but has no
`run_workflow` step naming it is a `warning`.

None of the deletes cascade into the steps that point at what they remove.

## Not here

The Test pages' blocks over a record's rows are not settled (decision 89).

There is no SQL to mirror. The tables, the `order_id`, `replaces` and
`corrected_on` columns and the `record_order` index come from the blueprint when
an agency is provisioned (`Provision`, through `spec.Migrate`).
