# The posting step

Owner decisions of 2026-11-17: the ledger is double-entry, entries are only
appended, a correction is a reversing entry, a balance is worked out by adding up
entries, and each member has one ledger account. A calculating step works out an
amount, for example the payment after a cap such as no more than what is owed.
It does not work out a balance. A separate step, the posting step, writes that
amount to the member's account. This page is the design of that step. It extends
[workflow-steps.md](workflow-steps.md) and [workflow-records.md](workflow-records.md),
and it rests on what the accounting repo really builds, which is listed in
[accounting-ledger-audit.md](../3.%20implementation/accounting-ledger-audit.md).

Provisional: the calculating step's result type, the failure choice and the
formula language are not all written, so every sentence here that leans on
something unwritten, or that could not be confirmed, begins with the word
Provisional. The calculating step is
[compute-step.md](compute-step.md) (ISS-19) and its failure choice is
[compute-step-failure.md](compute-step-failure.md) (ISS-6). This page does not
repeat or decide either.

The example throughout is the invented cooperative "Lakeside Savings Circle" and
its member Wanjiru Kamau. She owes 500 on a loan. Her loan account is where the
500 sits, and her payment of 50 is worked out by a calculating step and posted by
a posting step.

## 1. The step kind

A new step kind, `post`, beside the others. Provisional: the name `post` is a
proposal. It adds five stored fields to the work item, in `agency.blueprint.json`,
carried by `models.WorkItem` and `models.DocumentWorkItem`. All are empty on every
other kind and are left out of the document when empty.

| Field | Meaning |
|---|---|
| `amount_step_id` | The calculating step whose result is posted: an earlier step of the same workflow, of kind `compute`. It is the only source of the amount. |
| `account` | The member's account the entry is written to, named by the account category the agency has, such as `Member loans`. |
| `counter_account` | The other side of the entry, named the same way, such as `Cash`. Every entry debits exactly one account and credits exactly one other, so a posting names both. |
| `direction` | `debit` or `credit`, the side of `account` the amount goes on. The other side is `counter_account`. |
| `reason` | Why the amount is posted, in words, such as "Loan payment for March". |

The amount is the result of the calculating step: nothing else is a source, and
nobody types an amount into the posting step. The calculating step keeps no
stored result for later steps; the posting step reaches it by `amount_step_id`,
and the name it is known by is the calculating step's `result_name`. The calculating step is the one that
worked it out, so a posted figure always has a calculating step behind it.

The member is the person the case is about. The ledger keeps one account per
member, found by the category and the member's id as holder (audit item 5).
Provisional: how the case knows who the member is, and which category the agency
uses for members, are not decided (section 11).

The amount is never signed. Direction is which account sits on which side, as in
the ledger (audit, "direction"). For Wanjiru's loan, whose balance is what she
owes, a payment is a credit on `Member loans` and a debit on `Cash`.

`role_id` is not used: the agency takes the step. Provisional: whether a role may
run a posting step is open (section 11).

`action` is derived, never stored, as for the other kinds: it is declared
`spec:"-"` and filled by every create, get, list and update. Provisional: it reads
"Posts the Payment to post to Member loans", built from the calculating step's
`result_name` and `account`.

Example: the step Post the payment, in the workflow Loan payment.

| Field | Value |
|---|---|
| `kind` | `post` |
| `amount_step_id` | the step "Work out the payment", whose `result_name` is `Payment to post` |
| `account` | `Member loans` |
| `counter_account` | `Cash` |
| `direction` | `credit` |
| `reason` | `Loan payment` |

## 2. Worked example: the entries and the balance

The balance is not stored. It is the sum of the account's entries: debits add and
credits subtract (audit item 4). Wanjiru's account `Member loans`:

| Entry | Posted by | Debit account | Credit account | Amount | `reverses_entry_id` | Effect on Wanjiru's account |
|---|---|---|---|---|---|---|
| E1 | Loan paid out | Member loans (Wanjiru) | Cash | 500 | none | +500 |
| E2 | Post the payment | Cash | Member loans (Wanjiru) | 50 | none | -50 |
| E3 | Correction of E2 | Member loans (Wanjiru) | Cash | 50 | E2 | +50 |

After E1 the balance is 500. After E2 it is 500 - 50 = 450. After E3 it is
500 - 50 + 50 = 500, so the correction returns it to 500. E2 is still there and
still shows 50: the original stays visible, and the correction is a second line
that cancels it.

The figures here are whole units. Provisional: the scale the ledger keeps for
cents is not decided (section 11).

## 3. Provenance on every entry

Every entry a posting step writes says where its figure came from, so each figure
can be traced back to the function that worked it out. Three facts travel with it:

| Fact | Example |
|---|---|
| The function's code. | `loan.cap` |
| The published version of that function, the version the publication pins (as `fin.pmt@2` pins 2). | `3` |
| A fingerprint of the inputs, the same inputs giving the same fingerprint. | `c91f0a47` |

Wanjiru's E2 carries `loan.cap`, version `3`, fingerprint `c91f0a47`. A second
case with the same inputs would carry the same fingerprint, and a case with a
different balance would carry another.

What the audit found: the accounting repo's `Entry` has none of the three. The
nearest fields are `document_kind`, `document_id`, `actor_id` and `narration`,
which say which document and who caused the posting, not which function, version
or input (audit, "provenance": missing; gap 3). So this design needs the entry
to gain the three fields. That is the accounting repo's change, and it is not
made here. Provisional: until the entry has them, a posting step cannot record
its provenance.

Provisional: how the fingerprint is worked out, and whether the agency's
own publication tag is carried as well, are open (section 11). The function and
its version are on the calculating step (`function`, as `library.function@N`), so
the posting step does not store them.

## 4. The send key: running a step twice posts once

A step that runs twice, or a bulk run that is restarted, must not write the same
amount twice. A posting carries a send key, and a second posting with a key that
is already used writes nothing and gives back the first entry.

Record rows already work this way. A record row has a `send_key`: "the step and
run a workflow step sent this row for", written by the step's own send and by
nothing else, with the unique index `send_key_unique` refusing a second send of
one step in one run. Rows with no key are left out of the index. See
[provision-index-check.md](../3.%20implementation/provision-index-check.md). The
posting step uses the same idea: the key is the step, the run and, in a bulk run,
the row, such as `post-7:run-12:wanjiru`. Provisional: the exact shape of the key
is not decided.

It matters most in a bulk run that is restarted. A run over 200 members stops
after 120 postings. When it is restarted, the 120 keys are already used and
write nothing, and the other 80 post. Without the key the restart would post the
first 120 again.

What the audit found: the accounting repo has no key field and no unique index.
A repeat is recognised by a read inside the transaction when the document kind,
document id, both accounts, the amount, `reverses_entry_id` and `settles_entry_id`
all match, and the first entry is returned (audit, "idempotency": partly built;
gap 4). That match is not enough for this step: two real payments of 50 on the
same document and the same two accounts collapse into one. Provisional: the key
and its unique index are the accounting repo's change, not made here, and the
posting step is not safe to restart until they exist.

## 5. A bulk run posts each row on its own

A bulk run over many members posts each row separately: one calculating result,
one posting, one entry. A row never waits on another row to post, and no row posts
half.

A row whose calculating step failed follows the step's failure choice, which is in
[compute-step-failure.md](compute-step-failure.md) and not decided here. Under
either choice the failed row posts nothing: no entry is written for it, and the
other rows are not held back.

Where a confirming role is named on the calculating step, nothing is posted until
the confirmer confirms the run, as [compute-step.md](compute-step.md) section 6
says, and a member the confirmer holds back posts nothing.

Example: 200 rows at Lakeside, 3 fail the calculation (a payment of -5.00, no
balance, a term of 0 months). 197 entries are written and 3 are not. Provisional:
which of the failure choices Lakeside picked decides only where the 3 failed rows
go, not whether they post.

Provisional: what a result of 0 does, since the ledger posts only positive
amounts (audit item 2), is open (section 11).

## 6. Reversal: the correction

A mistake is not edited out. It is corrected by a reversing entry that names the
entry it reverses in `reverses_entry_id` (section 2: E3 reverses E2). The
reversing entry swaps the two accounts and repeats the amount. The original stays
visible, and so does the reversal, and the balance is the sum of both.

A reversal carries a reason in words. Provisional: the reason on a reversal is
not required by the ledger today (section 11).

What the audit found: the accounting repo allows a reversal once only
(`ErrAlreadyReversed`, backed by the unique index `reverses_once`), requires the
reversed entry to exist, and copies the original's dimensions. It does not check
that the reversal swaps the two accounts or repeats the amount, so a "reversal"
of 500 for 50 in the same direction is accepted (audit item 3, partly built; gap
1). Provisional: until that check exists, the posting step must write the mirror
itself and cannot rely on the ledger to refuse a wrong one.

Provisional: how a person asks for a correction (a step, a button or a tool) is
not decided (section 11).

## 7. How the screen and the reports read the balance

The member's screen and the month-end reports both read the balance the same way:
they add up the account's entries on every read. No page stores a balance, the
accounting repo has no balance field on any object (audit, "stored balance": none
found), and a posting step never writes one.

Wanjiru's screen after E1, E2 and E3 reads 500, and between E2 and E3 it read 450.
A month-end report for Lakeside is the same sum, over every member's account of
the category.

The calculating step's `ledger_balance` source reads the same sum
([compute-step.md](compute-step.md) section 2), so a figure worked out from the
balance and the figure the member sees cannot differ. Provisional: which account
it reads when a member has accounts in more than one category is compute-step.md's
open question 6, and is not decided here (section 12).

What the audit found: the sum loads every entry of the account into Go and adds
there; it does not add in the database (gap 6). That is a cost for a long
account and not a wrong answer. Provisional: a balance as at a date, which
month-end needs, is not listed in the audit, so whether the ledger reads one is
open (section 11).

## 8. What create and update refuse

All with `ErrInvalidWorkItem` (400) unless noted:

- any posting field on a step of another kind;
- a posting step with a form, a recipient, a sent step, routes, a run workflow,
  chat fields, a tool or calculating fields;
- a `direction` that is neither `debit` nor `credit`, such as `credit side`;
- a blank `reason`;
- an `account` and a `counter_account` that are the same account;
- an `amount_step_id` that is the step itself, is a step of another workflow, is
  not earlier by ordinality, or is not a calculating step: a `notify` step,
  or a `fill_form` step whose answer would be typed in, is not a source of the
  amount.

Not found:

- an `amount_step_id` that is not a step of the draft: `ErrWorkItemNotFound` (404);
- an `account` or `counter_account` the agency does not have: `ErrAccountNotFound`
  (404). Provisional: this name is proposed here and not decided, and where the
  agency's accounts are listed is open (section 11).

A posting step may be saved with no amount step, no account or no direction. A
workflow is built step by step, so the calculating step is made first and picked
afterwards, as a route step is saved with no route. The Issue scan is what stops
publishing.

Example: Post the payment is saved as a `post` step with `account` `Member loans`
and nothing else. It is accepted, with no amount source yet. Saved with
`direction` `credit side` it is refused. Saved with `amount_step_id` naming "Apply
for a loan", a form step, it is refused: that is not a calculating step. Saved with
`account` `Member loans` and `counter_account` `Member loans` it is refused.

## 9. What the Issue scan flags

All are `error`, so none can be published:

- a posting step with no amount source: it names no calculating step. This is the
  owner's rule;
- a posting step with no `account`, no `counter_account`, no `direction` or no
  `reason`;
- an amount step that no longer exists, now belongs to another workflow, is no
  longer earlier than this step, or is no longer a calculating step;
- an `account` or `counter_account` the agency no longer has: the scan raises it
  the way it already raises a step whose role is gone, with the sentence naming
  the step and the account.

None of the deletes cascades into the steps that point at what they remove.

Example: Lakeside deletes the calculating step "Work out the payment". The posting
step still names it, so the scan raises one error and the agency cannot be
published until the builder picks a calculating step again. A posting step created
and never given an amount step raises one error.

## 10. Everything that copies a step

- A draft fork remaps `amount_step_id` onto the copied steps, as it does
  `sent_step_id`. `account`, `counter_account`, `direction` and `reason` are copied
  as they are: accounts are the agency's, not a draft's, so they are not remapped,
  the way an offered group is not.
- A publication document carries the five fields on `DocumentWorkItem`, each left
  out when empty. Forking a draft from a publication restores them with the same
  remap.
- The content hash gains a `WorkItemPost` record only for a step that has any of
  the five fields set, the way `WorkItemLimit` is added only when a limit is set. A
  publication made before this change hashes as it did. A changed amount step,
  account, direction or reason is a change that can be published:
  `ErrNoChangesDetected` is not returned for it. Provisional: the record's name
  `WorkItemPost` is a proposal.
- Import of an older agency: a document with none of the five fields reads as a
  step with none, and nothing is invented. A document that carries a `post` step
  with no amount source is accepted, and the scan flags it, as for a calculating
  step with fewer bindings than inputs.
- Provisional: whether import should check an `account` against the importing
  agency's accounts, or accept it and let the scan flag it, is not decided. The
  import of a group id is the precedent: it keeps the id as it is.
- Provisional: whether `post` is one of the `accounting` module's `step_kinds` in
  `models/modules_catalog.json`, as `chat` is the `comm` module's, so that a
  posting step keeps accounting running while its switch is off, is not decided.

Example, fork: the step "Work out the payment" is copied as `st-44`. The copied
posting step has `amount_step_id` `st-44` and keeps `Member loans`, `Cash`,
`credit` and `Loan payment`.

Example, hash: Lakeside's March publication has no posting step. In July the
builder adds Post the payment and the publication hashes differently. Another
builder changes only `reason` from `Loan payment` to `Loan payment for the month`:
that hashes differently too and publishes.

## 11. Open question for the owner: values that are not money

A ratio or a score, for example Wanjiru's repayment ratio of 0.82, is worked out
by a calculating step but is not an amount. The ledger posts only money, so such a
value has no home yet. Which of these three, or another?

1. A second kind of account whose entries are not money. Pros: the same
   append-only entries, reversal and sum, and the same provenance. Cons: the
   ledger's entry is double-entry on a positive whole amount, and a score on that
   shape needs the account type and the entry's rules changed.
2. A separate figures record beside the ledger. Pros: the ledger stays money only,
   and each figure can keep its own provenance. Cons: a second place to read, and
   it must be append-only too or it loses the trail.
3. A column on the record row. Pros: no new object, and the figure sits next to the
   answers it came from. Cons: record rows are corrected by a replacing row and
   are per record, not per member over time, and nothing sums them.

This is not decided here. Until the owner picks, the posting step posts money to a
ledger account and nothing else.

## 12. Open questions

1. Values that are not money: section 11, the owner's to pick.
2. Which category does the agency use for member accounts, and where is the
   agency's list of accounts kept, so that "an account the agency does not have"
   can be checked? The accounting module owns the accounts today and this repo
   keeps no list of them.
3. How does the case know which member is meant, may a role run a posting
   step, and which account does `ledger_balance` read when a member has accounts
   in more than one category?
4. What does the posting step do with a result of 0 or a negative result, since the
   ledger posts only positive amounts?
5. What scale does the ledger keep? An entry's amount is a whole number, and a
   result such as 564.88 needs a stated number of decimals.
6. What is the exact shape of the send key, and how is the fingerprint of the
   inputs worked out? Does an entry also carry the agency's publication tag?
7. How does a person ask for a correction, and must a reversal carry a reason?
8. Does the ledger read a balance as at a date, for month-end?
9. What `document_kind` and `document_id` does a posting step write, as the
   ledger's replay match and its reports read them?
10. Is `post` an `accounting` module step kind, and does import check the account
    names?

## 13. Not here

- The calculating step: [compute-step.md](compute-step.md).
- What happens when a calculating step fails:
  [compute-step-failure.md](compute-step-failure.md).
- The formula language and the library: the compute repo.
- The accounting repo's changes (provenance, send key, mirrored reversal): that
  repo's, listed as gaps in
  [accounting-ledger-audit.md](../3.%20implementation/accounting-ledger-audit.md).
- How wakala-api runs a posting step and calls the ledger.
