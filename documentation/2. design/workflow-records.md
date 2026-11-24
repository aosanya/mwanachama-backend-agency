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
| `form_name`, `questions` | The form each row is added with, in the same shape as a step's form. Each question of a record is a field with a `key` and a `code` (see A field's code and key). |
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
the `role_id` the person acted as, who added it, and the `answers` by the
field's key. The row keeps the record's `record_code`, so it can be read across
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

The IAS tender record has ten fields, so it gives ten add, ten edit and ten view
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
| `record.tender.add.closing_date` | `Add a tender: Closing date` | Fill that field when adding. |
| `record.tender.add.*` | `Add a tender: every field` | Fill every field when adding. |
| `record.tender.edit.closing_date` | `Edit a tender: Closing date` | Change that field. |
| `record.tender.view.closing_date` | `See a tender: Closing date` | See that field. |
| `record.tender.delete` | `Delete a tender` | Delete a row. |

The names are built in one function, `recordPermissionName`; what a field code and a
star should read on the role page and the Role card is an open owner question, so
the wording above is provisional and the answer changes that one function only. The
item in the name is the record's `item_name`, else its `form_name`, else its name,
lower-cased, with `a` or `an` as in the generated workflow names (`Add an office
holder`); the field in the name is its prompt, else its code. Each permission is not
locked, has an empty `actions` list (an empty list, not none) and no module. The
static catalogue of 109 is unchanged; no catalogue code starts with `record.`, so a
derived code never equals one, and `record_permissions_test.go` pins both.

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
"Clerk" carries permission "record.RC-1.edit.closing_date", which no record gives`, so a
publish is refused and names the role and the code (D9); the same happens when a
field is removed from a record while a role names it, and a role that holds the star
is unaffected (D30). The scan's other
refusals (D18) are built in their own issue. A code of a record that exists is
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
