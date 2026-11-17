# Held figures (ISS-22)

Owner decisions of 2026-11-17 (decisions 9 and 10 in
[compute-decisions.md](../1.%20requirements/compute-decisions.md)): a workflow can
calculate a figure and hold it, not yet real, until a later step approves or
releases it. Nothing reaches the ledger or the member's figures until then. When a
held figure is opened for approval after its inputs have changed, the agency
works it out again from today's numbers and shows both figures and the reason. The
approver approves the new figure or sends it back. A held figure is never posted
unchanged when its inputs have moved. This extends [compute-step.md](compute-step.md)
and [posting-step.md](posting-step.md).

Provisional: the calculating step, its failure choice and the posting step are not
accepted, so every sentence that leans on a name or rule they propose contains the
word Provisional. The names this page proposes (`hold`, `held_step_id`, the held
figure, its states and errors) are proposals too. What the owner has not decided
(where a held figure lives, expiry, who may release) is an open question with a
proposed answer (section 11), never a decision.

The example is the invented cooperative "Lakeside Savings Circle" and member
Wanjiru Kamau, who has a 3,500.00 quick loan charged 0.125% a day on the balance at
the end of each day, for 32 days to the due date. Provisional: the step Work out
the interest calls `loan.daily_interest@1` with the inputs `balance_by_day`,
`daily_rate` (fixed, 0.00125) and `days` (fixed, 32). The agency keeps 2 currency
places and rounds half up.

## The story in one table

| Day | What happens | Held figure | Interest on the ledger | Wanjiru owes |
|---|---|---|---|---|
| Monday | The agency works out 140.00 and holds it. | `held`, 140.00 | none | 3,500.00 |
| Wednesday | She pays 200.00, late. Another workflow posts that. | `held`, 140.00 | none | 3,300.00 |
| Friday | The Loan officer opens it. It is worked out again: 132.50. She approves 132.50. | `released`, 132.50 | one entry of 132.50 | 3,432.50 |

| Other ending | When | What happens | Interest on the ledger |
|---|---|---|---|
| Rejection | Friday | The Loan officer sends it back: "Interest waived, board minute 14". | none |
| Expiry | Monday, two weeks later | No one decided in 14 days. It becomes `expired`. | none |
| Expiry | The day after | Approving it is refused with `ErrHeldFigureExpired` (409). | none |

Arithmetic: 3,500.00 x 0.00125 x 32 = 140.00. On Friday 2 days at 3,500.00 is 8.75
and 30 days at 3,300.00 is 123.75, which is 132.50, so the difference is -7.50.
Provisional: the entry debits `Member loans` and credits `Interest income`
([posting-step.md](posting-step.md) section 1), and 3,300.00 + 132.50 is 3,432.50.

## 1. What a held figure is and where it lives

A held figure is a calculated result that is not real yet. It is not a ledger entry
(money, real) and not an entry in the record of figures (decision 7: real figures
that are not money). It is one row saying a figure was worked out and waits for a
person.

Provisional: it lives in the agency instance as a new object `held_figure`, kept as
`record_row` is: in the agency, not the catalog, and neither copied by a fork nor
carried by a publication ([workflow-records.md](workflow-records.md)). The case
belongs to the task manager, which the agency cannot reach
([workflow-use.md](workflow-use.md)), so the row names the run and does not own it.
The host writes it ([compute-step-host.md](../3.%20implementation/compute-step-host.md)).
This is not decided: open question 1.

Provisional: its fields. `figure` and `inputs` never change once written.

| Field | Meaning |
|---|---|
| `code` | `HF-41`, minted once and never edited. |
| `draft_id`, `step_id` | The draft and the calculating step. The rework reads the bindings from that draft. |
| `run_id`, `member_id` | The case and the member. Provisional: how the case knows the member is open in posting-step.md section 12. |
| `result_name`, `function` | `Interest due`, `loan.daily_interest@1`. The version held, never a newer one. |
| `figure` | A decimal with the agency's places: `140.00`. |
| `inputs`, `inputs_fingerprint` | Each input's name, source and value, and one code over all, `1825fda0`. |
| `held_at`, `expires_at` | When it was held and when it lapses. |
| `state` | `held`, `released`, `rejected` or `expired`. |
| `decided_by`, `decided_at`, `reason` | Who moved it out of `held`, when and why. Required for `rejected`. |
| `reworked_*` | Figure, inputs, fingerprint and time of a rework. Empty unless reworked. |

## 2. The four states

Provisional: the names are proposals. `held` is the only state that moves.

| From | To | Who | What moves it |
|---|---|---|---|
| (new) | `held` | The agency | The calculating step works out a figure and has a hold. |
| `held` | `released` | The role of the releasing step | Approve (section 4). |
| `held` | `rejected` | The same role | Send back, with a reason. |
| `held` | `expired` | The agency | The expiry passes (section 5). |

Every other move is refused with `ErrHeldFigureState` (409): `rejected` to
`released`, or `held` to `held`. `released`, `rejected` and `expired` are final. A
later change is a new held figure from a new run of the step, never an edit.

Provisional: who may release is the role named on the releasing step, or an offered
group as for any step. A caller without it is refused with `ErrHeldFigureRefused`
(403). Whether anyone else may is open question 3.

Example: Friday the Loan officer approves `HF-41`: `held` to `released`. Rejecting
it on Saturday is refused with `ErrHeldFigureState` (409).

## 3. Where it sits in the workflow

Provisional: two new fields name the pair, calculating step and releasing step.

| Field | On a step of kind | Meaning |
|---|---|---|
| `hold` | `compute` | Present means the result is held, not kept: `{"expires_after_days":14}`. Empty means no hold, as today. |
| `held_step_id` | `yes_no` or `post` | The calculating step whose held figure this step decides: an earlier step of the same workflow. |

Both are in `agency.blueprint.json`, carried by `models.WorkItem` and
`models.DocumentWorkItem`, empty on every other kind and left out of the document
when empty. `hold` is a json column, as `routes` is.

Provisional: two kinds release. A `yes_no` step with a role is the approval: yes
releases, no sends back and asks for the reason. A `post` step releases only when it
names a role (open in posting-step.md section 12), and then approving and posting
are one act. `notify`, `fill_form` and `route` may sit between the two steps and
cannot release. What send back leads to is a route the builder draws (decision 3);
this page adds no overwrite.

Provisional: a `post` step that reads a held figure through `amount_step_id` reads
nothing while it is `held`. It waits and never posts it. A `post` naming
`held_step_id` must name the same step as `amount_step_id`.

The approver sees the held figure, its inputs, when it was held, the days left and,
when the inputs moved, section 4. Provisional: it is listed in the role's
`agency_list_my_open_steps`, as a confirming step is in compute-step.md section 5.

Example: Quick loan interest has 1 Work out the interest (`compute`, hold 14 days),
2 Approve the interest (`yes_no`, role Loan officer, `held_step_id` step 1) and 3
Post the interest (`post`, `amount_step_id` step 1, `account` `Member loans`,
`counter_account` `Interest income`, `direction` `debit`). Step 3 waits for step 2.

## 4. Rework at approval

When the approver opens a held figure the host compares the fingerprint of the
inputs now with `inputs_fingerprint`. Provisional: the fingerprint is posting-step.md
section 3's, and how it is worked out is open there.

- Same fingerprint: the inputs have not moved. The figure is shown as it is and
  approving releases it unchanged.
- Different fingerprint: the agency works it out again, same function and version,
  from today's inputs. It shows both figures and the reason, and the approver
  approves the new figure or sends it back.

The reason is one plain sentence for each input that moved, naming the input, its
held value and its value now. Example, Friday: the fingerprint is `1e73c5cb`, not
`1825fda0`. The Loan officer sees 140.00 held on Monday, 132.50 worked out today,
a difference of -7.50, and: "The input Balance by day changed: it was 3,500.00 on
every day when the figure was held, and is 3,300.00 from Wednesday." She approves
132.50.

Approving names the figure approved by its fingerprint. Approving the old figure
when the inputs have moved is refused with `ErrHeldFigureMoved` (409), and the
answer carries the new figure and the reason. That also covers inputs that move
between opening the page and pressing approve.

| Outcome | Held figure | Ledger | Member's figures |
|---|---|---|---|
| Approve the new figure | `released`, `reworked_*` filled | One entry, 132.50 | The figure, if not money |
| Approve, inputs not moved | `released`, `reworked_*` empty | One entry, 140.00 | The same |
| Send back | `rejected`, with the reason | Nothing | Nothing |

`figure` stays 140.00 on the row and the rework sits beside it. A rejection leaves
the row in the record of held figures as `rejected` with its reason, and nothing in
the ledger or the member's figures.

Provisional: when the rework itself fails, the calculating step's failure choice in
[compute-step-failure.md](compute-step-failure.md) applies, and this page does not
decide it. The old figure is not approvable. Whether the row stays `held` is open
question 4.

## 5. Expiry

Not decided. Provisional: the proposal is that a held figure lapses
`expires_after_days` after it was held: 14 if left out, any whole number from 1 to
90. Held Monday, it expires Monday two weeks later. 0, -3, 2.5, `fourteen` and 91
are refused with `ErrInvalidWorkItem` (400).

An expired figure is final. The row stays as `expired` with the reason "Expired
after 14 days without a decision", written by the agency, and nothing is left in the
ledger or the member's figures. Approving it or sending it back is refused with
`ErrHeldFigureExpired` (409). Provisional: the case follows the send back route, and
expiry is applied on every read and move, so no timer is needed (open question 2).

Example: `HF-41` is held on Monday and nobody decides. Two weeks later it reads
`expired`. The next run of the step makes `HF-58` from that day's inputs.

## 6. Bulk: 3,000 members

This joins the confirmation list in compute-step.md section 6 and changes none of
what it decides. Provisional: the list is the held figures of one step in one run,
with the totals first (members, sum of the figures, how many held back). Provisional:
the limits below are proposals. Decision 11 in
[compute-decisions.md](../1.%20requirements/compute-decisions.md) makes a bulk
release the actualization batch, run only after the calculation batch has finished
and sized by the agency. Provisional: the release batch of 500 is the proposed
default for that size, and it is not decided.

| Limit | Value | Over it |
|---|---|---|
| List page size | 100, at most 200 | `ErrInvalidHeldFigure` (400) |
| Release batch | 500 figures a call | `ErrInvalidHeldFigure` (400) |

Release works one member at a time. Each member's inputs are compared and, if moved,
reworked on their own, so one member's moved inputs change nobody else's figure. A
batch returns one outcome per figure:

- `released`: inputs not moved, released unchanged;
- `moved`: inputs moved. Not released. The new figure and reason come back, and it
  stays `held` until a second call names the new fingerprint;
- `already_released`: nothing is written;
- `refused`: its own refusal, such as `ErrHeldFigureExpired`. The rest are not
  affected.

Holding one member back means leaving the figure out of the call. It stays `held`,
nothing else changes, and the others release. Where a held-back member goes next is
open in compute-step.md.

Example: 3,000 held figures are 30 pages of 100. The Loan officer holds back 2 and
releases 2,998 in 6 calls (5 of 500 and 1 of 498). 2,958 come back `released` and 40,
Wanjiru among them, come back `moved`. She approves the new figure for 39 and sends
1 back. At the end 2,997 are `released`, 1 is `rejected` and 2 are `held`.

Releasing twice releases once. Provisional: releasing a `released` figure returns
the first result and writes nothing, and the posting step's send key keeps the ledger
to one entry ([posting-step.md](posting-step.md) section 4). A batch cut off after
1,000 can be sent again whole: 1,000 come back `already_released`, 1,958 release and 40 come back `moved`.

## 7. What create and update refuse

All with `ErrInvalidWorkItem` (400) unless noted:

- `hold` on a step that is not `compute`, or `held_step_id` on one that is not
  `yes_no` or `post`;
- a `hold` on a compute step that also names a role, because the confirming role and
  the hold are two ways to wait for a person (open question 8);
- a `hold` with `expires_after_days` of 0, negative, not whole or over 90, or with
  any other key;
- a `held_step_id` that is the step itself, is in another workflow, is not earlier by
  ordinality, is not a `compute` step, or is a compute step with no `hold`;
- a second step naming a calculating step another step already names;
- a `post` step whose `held_step_id` differs from its `amount_step_id`.

Not found: a `held_step_id` that is not a step of the draft: `ErrWorkItemNotFound`
(404).

A calculating step may be saved with a hold and no releasing step, as a route step is
saved with no route. The Issue scan stops publishing. Example: Approve the interest
naming "Apply for a loan", a form step, is refused. Naming "Work out the interest"
with no hold is refused. A hold of 0 days is refused.

Provisional: refusals of the held figure's own operations (section 9):

| Refusal | Error | Status |
|---|---|---|
| No such held figure | `ErrHeldFigureNotFound` | 404 |
| A move section 2 does not allow | `ErrHeldFigureState` | 409 |
| Approve or send back an expired figure | `ErrHeldFigureExpired` | 409 |
| Approve a figure whose inputs have moved | `ErrHeldFigureMoved` | 409 |
| The caller does not hold the releasing role | `ErrHeldFigureRefused` | 403 |
| Blank reason, empty batch, batch over 500, page over 200, unknown state | `ErrInvalidHeldFigure` | 400 |

## 8. What the Issue scan flags

All are `error`, so none can be published:

- a calculating step with a hold that no step names as `held_step_id`: nothing can
  release it;
- a step with `held_step_id` whose calculating step no longer exists, is in another
  workflow, is no longer earlier, has no hold, or is no longer a `compute` step;
- a releasing step with no role: the agency cannot approve its own figure;
- a releasing step whose role no longer exists, raised as a step whose role is gone
  is. An offered group is not flagged;
- `hold` on a step of another kind, or `held_step_id` on one that is not `yes_no` or
  `post`, as an import can carry in;
- a `hold` with `expires_after_days` outside 1 to 90;
- a `post` step reading a held figure that comes before the step that releases it.

None of the deletes cascades into the steps that point at what they remove.

Example: Lakeside deletes the role Loan officer, so Approve the interest names a role
that does not exist: one error. A builder adds a hold to Work out the interest and
makes no releasing step: one error.

## 9. Operations and tools

Provisional: if the agency keeps the rows (open question 1) it adds these, all
proposed. They are declared in `agency.operations.json`; nothing is hand-written.

| Operation | Tool | Meaning |
|---|---|---|
| `POST /held-figures` | `agency_hold_figure` | The host writes a row in state `held`. |
| `GET /held-figures?draft_id=&step_id=&run_id=&state=&page=&page_size=` | `agency_list_held_figures` | 100 to a page, with the totals on the first. |
| `GET /held-figures/{id}` | `agency_get_held_figure` | One row, with `reworked_*` when set. |
| `POST /held-figures/release` | `agency_release_held_figures` | Up to 500 `{id, fingerprint}` pairs, one outcome each. |
| `POST /held-figures/{id}/reject` | `agency_reject_held_figure` | `reason` required. |
| `POST /held-figures/{id}/rework` | `agency_record_rework` | The host saves the new figure, inputs and reason. |

The step fields travel through the existing work item tools. Working the figure out
again is the compute repo's, called by the host.

## 10. Provenance, and everything that copies a step

Provisional: a released figure carries the function code, version and fingerprint of
the inputs it was worked out from ([posting-step.md](posting-step.md) section 3):
those of the figure approved, so the reworked ones when reworked, with the note
"Reworked at approval; held 140.00 on Monday". A posted entry, or an entry in the
record of figures for a figure that is not money, carries them on. The entry gaining
those fields is the accounting repo's change
([accounting-ledger-audit.md](../3.%20implementation/accounting-ledger-audit.md),
gap 3). The old fingerprint stays on the held row.

Example: Friday's entry is 132.50 with `loan.daily_interest`, version 1, fingerprint
`1e73c5cb` and the note. Baraka Otieno, whose inputs did not move, is released at
112.00 with `ef771254` and no note.

- A draft fork remaps `held_step_id` onto the copied steps. `hold` has no id and is
  copied as it is. Held figures belong to a run and are not copied.
- A publication document carries `hold` and `held_step_id` on `DocumentWorkItem`, each
  left out when empty. Forking a draft from a publication restores them with the same
  remap.
- The content hash gains a `WorkItemHold` record only for a step with a `hold` or a
  `held_step_id`, the way `WorkItemLimit` is added only when a limit is set, so a
  publication made before this change hashes as it did. A changed hold setting (14
  days to 30, a hold added, another approver) can be published:
  `ErrNoChangesDetected` is not returned for it. Provisional: the record's name is a
  proposal.
- Import of an older agency: a document with no `hold` or `held_step_id` reads as a
  step with none and nothing is invented. A hold with no releasing step is accepted
  and the scan flags it.

Example, hash: Lakeside's March publication has no hold. In July a builder adds one
and it hashes differently. Another changes only 14 days to 30: it publishes. A third
changes nothing: the hash is the same and `ErrNoChangesDetected` is returned.
Example, fork: "Work out the interest" is copied as `st-52`, and the copied approval
step has `held_step_id` `st-52`.

## 11. Open questions

Each carries a proposed answer. None is decided. All are the owner's.

1. Where does a held figure live? Proposed: the agency instance, as `held_figure`
   (section 1). The other place is the task manager with the run.
2. Does it expire? Proposed: yes, 14 days, 1 to 90, `expired` is final, the case
   follows the send back route, and expiry is applied on read.
3. Who may release? Proposed: only the releasing step's role or offered group, the
   agency owner included.
4. When the rework fails, does the figure stay `held`? Proposed: yes, and only a send
   back moves it, until it expires. The failure choice stays compute-step-failure.md's.
5. When the inputs moved but the new figure is the same? Proposed: still shown as two
   figures with the reason, and the new fingerprint is approved.
6. Can `ledger_balance` give a balance for each day, as `balance_by_day` needs?
   Proposed: yes, through the ledger's balance as at a date (posting-step.md section
   12, question 8). Compute-step.md's source gives one balance.
7. For a figure that is not money, which date does its record-of-figures entry carry?
   Proposed: the day it was released, with the held date in the provenance.
8. May a step both confirm (a role on it) and hold? Proposed: no, refused.
9. Are 100 a page and 500 a release batch right? Proposed: yes, for 3,000 members.
10. Does the rework use the version held or the newest? Proposed: the version held, as
    compute-step-host.md says never to recompute with a newer one.

## 12. Not here

- The formula language, the library and `loan.daily_interest`: the compute repo.
- The accounting repo's changes (provenance fields, send key):
  [accounting-ledger-audit.md](../3.%20implementation/accounting-ledger-audit.md).
- How wakala-api runs the hold, the rework and the release.
- The record of figures itself (decision 7).
- The studio mockups.
