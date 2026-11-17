# The calculating step (ISS-19)

Owner decisions of 2026-11-17: a builder adds a calculating step, says where each
input comes from and who runs it. Inputs are never matched by name on their own,
so the builder picks a source for each one from a list. Per step the builder
chooses who runs it: the agency, or a role who confirms the figure first. This
extends [workflow-steps.md](workflow-steps.md) and
[workflow-records.md](workflow-records.md).

Provisional: the formula language, the library's function signatures, the failure
choice, the posting step and where a result is kept are not written, so every
sentence here that leans on one of them contains the word Provisional. The
example throughout is one invented step, Work out the monthly payment, of the
cooperative "Lakeside Savings Circle", which calls `fin.pmt@2`.

The example loan: member Wanjiru Kamau asks for 12,000 on the earlier step "Apply
for a loan", whose form question "Amount requested" is a currency. At 12% a year
the monthly rate is 0.01, and over 24 months the payment is 564.88 a month.
Provisional: the input names `principal`, `monthly_rate` and `months`, and their
types, come from the library, not from this page.

## 1. Where it lives

A new step kind, `compute`, beside the seven kinds (`fill_form`, `send_form`,
`notify`, `yes_no`, `route`, `run_workflow`, `chat`). It adds three stored fields
to the work item, in `agency.blueprint.json`, carried by `models.WorkItem` and
`models.DocumentWorkItem`. All are empty on every other kind and are left out of
the document when empty.

| Field | Meaning |
|---|---|
| `function` | The library function, written `library.function@N`, such as `fin.pmt@2`. The version is required. |
| `bindings` | A list, one per input of the function. See below. |
| `result_name` | What the result is kept under, such as `Monthly payment`. Not blank, and unique inside its workflow, ignoring case. |

A binding is `input` (the function's input name), `source_type` and the fields
its source needs.

| `source_type` | Fields beside `input` |
|---|---|
| `answer` | `step_id` (an earlier step of the same workflow) and `question` (the question's prompt, its wording). |
| `record_field` | `record_id` (a record of the draft) and `question` (a prompt of the record's form). |
| `ledger_balance` | None. |
| `fixed` | `value`. |

`role_id` is reused. Empty means the agency runs the step. A role, or an offered
group as for any step (see Groups as actors in [workflow-steps.md](workflow-steps.md)),
is the confirming role, described in section 5.

The failure choice is one more field on the step. It is kept in the field that
[compute-step-failure.md](compute-step-failure.md) names, and Provisional: this
page does not repeat or decide it.

`action` is derived, never stored, as for the other kinds: it is declared
`spec:"-"` and filled by every create, get, list and update. Provisional: it
reads "Works out the Monthly payment", built from `result_name`.

Provisional: how the result is kept for later steps is in
[posting-step.md](posting-step.md), not here.

Example: the step Work out the monthly payment, in the workflow Loan request.

| Field | Value |
|---|---|
| `kind` | `compute` |
| `function` | `fin.pmt@2` |
| `result_name` | `Monthly payment` |
| `role_id` | empty, so the agency runs it |
| `bindings[0]` | `principal`, `answer`, the step "Apply for a loan", question "Amount requested" |
| `bindings[1]` | `monthly_rate`, `fixed`, value `0.01` |
| `bindings[2]` | `months`, `record_field`, the record "Loan terms", question "Months" |

## 2. The four sources and their types

Each source has a type, and a source whose type does not fit the input is refused
when the step is saved. Provisional: the types come from the formula language.

| Source | What it reads | Its type | What it names |
|---|---|---|---|
| `answer` | What the person answered to a question of an earlier step's form. | The answer type of the question, see below. | The step and the question's prompt. |
| `record_field` | What a question of a record's form holds. | The answer type of the question. | The record and the question's prompt. |
| `ledger_balance` | The balance of the member's ledger. | `currency`. | Nothing. |
| `fixed` | A value the builder types in. | The type the input asks for. | The value itself. |

Provisional: an answer type becomes a formula type as follows. It is the same
vocabulary as `StepQuestion`.

| Answer type | Type |
|---|---|
| `number` | number |
| `currency` | currency |
| `date` | date |
| `time` | time |
| `yes_no` | yes_no |
| `free_text`, `single_choice` | text |
| `multi_select`, `photo`, `file` | cannot be a source |

A `fixed` value must parse as the type its input asks for: `0.01` is a number, and
`one percent` is refused. Provisional: the match is exact and nothing is
converted silently, and whether a `currency` source is accepted where a number is
asked is open (section 9).

Example: `principal` asks for a currency. The question "Amount requested" is a
currency, so it is accepted. A source that is the free-text question "Reason for
the loan" is text, so it is refused. `monthly_rate` asks for a number and `0.01`
parses as one. `months` asks for a number, and the question "Months" of the
record "Loan terms" is a `number`, so it is accepted. Provisional: this
paragraph leans on the input types above.

## 3. What create and update refuse

All with `ErrInvalidWorkItem` (400) unless noted:

- any compute field on a step of another kind;
- a compute step with a form, a recipient, a sent step, routes, a run workflow,
  chat fields or a tool;
- no function; a function with no `@version` (`fin.pmt`), or a version that is
  not a whole number (`fin.pmt@2.5`, `fin.pmt@latest`);
- no result name, or a result name another compute step of the workflow already
  has, ignoring case;
- more than one role: `role_id` is one id, as for the other kinds;
- a binding for an input the function does not have (Provisional: the inputs are
  looked up from the library, offered to the manager the way `WithGroupLister`
  offers groups), or two bindings for one input;
- a `source_type` that is none of the four;
- an `answer` binding whose step is in another workflow, is the step itself, is
  not earlier by ordinality, or has no form; or whose question the step's form
  does not ask;
- a `record_field` binding whose question the record's form does not ask;
- a `fixed` binding with no value, or a value that does not fit the input's type;
- a source of the wrong type (section 2).

Not found:

- a role or group that is not offered: `ErrRoleNotFound` (404);
- an answer step that is not in the draft: `ErrWorkItemNotFound` (404);
- a record that is not in the draft: `ErrRecordNotFound` (404);
- Provisional: a function or version that is not in the library:
  `ErrFunctionNotFound` (404), proposed here and not decided.

A step may be saved with fewer bindings than the function has inputs. A
workflow is built step by step, and the inputs are filled in one at a time, as a
route step may be saved with no route. The Issue scan is what stops publishing.

Example: Work out the monthly payment is saved with `function` `fin.pmt` and
nothing else. It is refused: no version. Saved as `fin.pmt@2` with
`result_name` `Monthly payment` and one binding for `principal`, it is accepted
with 1 of 3 inputs bound. A second binding for `principal` is refused. A binding
for `term_days` is refused, because Provisional: the function has no such input.
A second compute step in the workflow named `monthly payment` is refused as a
repeat of the result name. A binding to the step "Apply for a loan" with the
question "Amount asked" is refused, because the form asks "Amount requested".

## 4. What the Issue scan flags

All are `error`, so none can be published:

- an input with no source (the studio shows it as an input with no source);
- an `answer` binding whose step no longer exists, now belongs to another
  workflow, or is no longer earlier than this step;
- an `answer` or `record_field` binding whose question no longer exists or was
  reworded. Questions are known by wording, so rewording is the same as deleting.
  The scan raises it the way it already raises a step whose role is gone;
- a `record_field` binding whose record no longer exists;
- a binding whose source type no longer fits the input, because the question's
  answer type was changed;
- a confirming role that no longer exists. An offered group is not flagged;
- Provisional: a function version that is no longer in the library;
- a compute step with no function or no result name.

None of the deletes cascades into the steps that point at what they remove.

Example: on Lakeside's loan workflow the builder rewords "Amount requested" to
"Amount you want to borrow". The `principal` binding still names the old
wording, so the scan raises one error and the agency cannot be published until
the builder picks the question again. Changing the same question from currency to
number raises one error too, because Provisional: a number does not fit an input
that asks for a currency. Deleting the role Treasurer leaves a confirming role
that does not exist: one error.

## 5. The confirming person

When `role_id` is set the step waits on that role. `agency_list_my_open_steps`
lists it for the role. A compute step with no role is never listed, because
nobody waits on it.

What the confirmer sees: the figure, and the inputs it was worked out from, each
with its name, its source and its value. What they can do is a set of routes the
builder draws (decision 3 in
[compute-decisions.md](../1.%20requirements/compute-decisions.md)): accept it,
send it back, or take a route to an overwrite step. The step has no built-in
overwrite. A figure can be overwritten only where the builder has wired an
overwrite step, so a confirmer cannot change a figure unless the builder gave them
that route. Provisional: what an overwrite step is, as a kind, is not written. In
a bulk run the confirmer also holds back individual rows (section 6).

A time limit on such a step is open. Today `WaitsOnPerson` refuses a limit on any
kind but `fill_form` and `yes_no`, as [step-time-limit.md](step-time-limit.md)
says, and whether a confirming compute step may carry one is not decided.

Example, run by the agency: the step has no role. Wanjiru's case reaches it,
the agency works out 564.88 and the case moves on at once. Nothing waits and
nothing is listed.

Example, with the treasurer confirming: the step names the role Treasurer.
Wanjiru's case waits. The Treasurer sees 564.88 and the three inputs:
`principal` 12,000 (answer "Amount requested"), `monthly_rate` 0.01 (fixed) and
`months` 24 (record "Loan terms", question "Months"). The Treasurer accepts it
and the case moves on.

Provisional: what "moves on" does with the figure is the posting step's
([posting-step.md](posting-step.md)).

## 6. The confirmation list for a bulk run

Decided: a bulk run gives the confirmer one list for the whole run, with the
totals and the unusual figures first. The confirmer confirms the run once, and
may hold back individual members. Nothing is posted until the confirmer confirms.
Provisional: posting is the posting step's ([posting-step.md](posting-step.md)).

Decided (decision 11): a bulk run has two stages. The calculation batch works out
every row and finishes as a whole, and the list above is the finished calculation
batch. Only then does an actualization batch post or release the results, which is
the posting step's work. The agency sets how many rows one batch may hold, and a
batch over the agency's limit is refused whole before any row is worked out
(compute repo decision 16). Nothing real changes in the calculation batch, so it can
be confirmed, held, repeated or cancelled before the actualization batch starts.
Open: whether a run larger than the limit is cut into several calculation batches
by the platform or by the builder.

Per member the list shows the name, the figure and the inputs. The totals are
the number of members, the sum of the figures and the number held back.

Open, and not decided here: how an unusual figure is chosen, and where a
held-back member goes afterwards. The others are not delayed by a held-back
member: they wait only for the confirmation.

Example: a run of four members for Lakeside's Treasurer.

| Member's request | Monthly payment |
|---|---|
| 12,000 | 564.88 |
| 8,000 | 376.59 |
| 50,000 | 2,353.67 |
| 3,000 | 141.22 |

The totals read 4 members and 3,436.36. The treasurer holds back the 50,000
member. The totals then read 3 members to confirm, 1 held back and 1,082.69, and
the Treasurer confirms the run once. The 2,353.67 figure would be shown first
only as an illustration: the rule for choosing the unusual figures is open, so
nothing here says the largest figure goes first.

## 7. Everything that copies a step

- A draft fork remaps `step_id` of the `answer` bindings onto the copied steps,
  `record_id` of the `record_field` bindings onto the copied records, and
  `role_id` as before. Roles, records and steps are all copied before the
  bindings are remapped. Question wordings, values and function strings are
  copied as they are, so the pinned version is kept.
- A publication document carries `function`, `bindings` and `result_name` on
  `DocumentWorkItem`, each left out when empty, and Provisional: the failure
  field. Forking a draft from a publication restores them with the same remap.
- The content hash gains a `WorkItemCompute` record only for a step that has a
  function, bindings or a result name, the way `WorkItemLimit` is added only
  when a limit is set. A publication made before this change hashes as it did.
  The bindings are hashed sorted by input, so reordering them alone is no
  change. A changed function version, binding or result name is a change that can
  be published: `ErrNoChangesDetected` is not returned for it. Where the version
  is recorded and why it is hashed is in
  [library-versioning.md](../../../mwanachama-backend-compute/documentation/2.%20design/library-versioning.md).
- Import of an older agency: a document with no `function`, `bindings` or
  `result_name` reads as a step with none, and nothing is invented. Steps of
  other kinds are unchanged. A document that carries a `compute` step with fewer
  bindings than inputs is accepted, and the scan flags it.

Example, fork: the step "Apply for a loan" is copied as `st-31`, and the record
"Loan terms" as `rc-9`. The copied compute step binds `principal` to `st-31` and
`months` to `rc-9`, both with the wording unchanged, and keeps `fin.pmt@2`.

Example, hash: Lakeside's March publication has no compute step. In July the
builder adds Work out the monthly payment, and the publication hashes
differently. Another builder moves only from `fin.pmt@1` to `fin.pmt@2`: that
also hashes differently and publishes. A third reorders the three bindings and
changes nothing else: the hash is the same and `ErrNoChangesDetected` is
returned.

Example, import: an agency written before this change has 6 steps and none has a
`function`. It imports as 6 steps of their old kinds. A file with one `compute`
step and 1 of 3 bindings imports, and the scan raises one error for each of the
2 inputs with no source.

## 8. Operations and tools the agency adds

| Operation | Tool | Meaning |
|---|---|---|
| none: no new create or update route | `agency_create_work_item`, `agency_update_work_item` | Carry `function`, `bindings` and `result_name`. |
| none: no new read route | `agency_get_work_item`, `agency_list_work_items` | Return the three fields on every step. |
| `GET /input-sources?draft_id=&work_item_id=`, action `agency.input_source.list` | `agency_list_input_sources` | Lists what the builder may pick for that step. |

The four work item tools are declared in `agency.operations.json`, and nothing is
hand-written for them.

`agency_list_input_sources` groups its answer by the four source types, each
with its type:

- `answer`: the answers of the earlier steps of the same workflow, each with the
  step id, the step title, the question and its answer type;
- `record_field`: the fields of the draft's records, each with the record id, the
  record name, the question and its answer type;
- `ledger_balance`: the one balance, a currency;
- `fixed`: a value the builder types in.

An optional `input_type` narrows the list to the sources of that type.

Listing the functions and their inputs is the library's, not the agency's.
Provisional: it is in the compute repo, and this repo only asks it.

wakala-api reads the step as published when a case reaches it. Its side is in
[compute-step-host.md](../3.%20implementation/compute-step-host.md).

Example: the builder opens the input list of Work out the monthly payment, with
`input_type` `currency`. The answer holds the question "Amount requested" of the
step "Apply for a loan" and the ledger balance. It does not hold the number
question "Months" of the record "Loan terms".

## 9. Open questions

1. Decided (decision 3): the step has no built-in overwrite. A confirmer changes a
   figure only through an overwrite step the builder wires. Open: what that step
   is, as a kind.
2. How is an unusual figure chosen for the top of the confirmation list? Not decided.
3. Where does a held-back member go next? Not decided.
4. May a confirming compute step carry a time limit? Not decided.
5. Which row does a `record_field` read when the record has several rows? Rows are
   added by the member app and may be corrected. Not decided.
6. Provisional: which ledger does `ledger_balance` read when a member has several?
   See [posting-step.md](posting-step.md). Not decided.
7. May a later compute step use an earlier step's result as a source? It is not
   one of the owner's four sources, so it is not offered. Not decided.
8. Provisional: is a `currency` source accepted where a number is asked? Not decided.
9. Who cuts a run larger than the agency's batch limit into several calculation
   batches, the platform or the builder? Not decided (decision 11).

## 10. Not here

- The formula language, the library and its versions: the compute repo.
- The failure choice: [compute-step-failure.md](compute-step-failure.md).
- The posting step: [posting-step.md](posting-step.md).
- The studio mockups.
