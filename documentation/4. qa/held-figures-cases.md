# Held figures: cases

Cases for held figures, written so a later test can load them one by one. The rules
are in [held-figures.md](../2.%20design/held-figures.md) (ISS-22); what the host and
the accounting repo must do is in
[held-figures-host.md](../3.%20implementation/held-figures-host.md). These are cases
for tests to be written, not tests that exist.

Provisional: the calculating step ([compute-step.md](../2.%20design/compute-step.md)),
its failure choice ([compute-step-failure.md](../2.%20design/compute-step-failure.md))
and the posting step ([posting-step.md](../2.%20design/posting-step.md)) are not
accepted, and the held figure's own names are proposals, so every sentence that
leans on the kinds `compute` and `post`, `hold`, `held_step_id`, the held figure and
its states, the fingerprint, the send key, `WorkItemHold` or the tools starts with
or contains that word. What the owner has not decided (where a held figure lives,
expiry, who may release) is an open question of the design page, section 11, with
its proposed answer, and is named as such below. The cases marked Host are run by
wakala-api, the cases marked Ledger are checked in the accounting repo, and the
cases marked save, scan, publish, fork or import are run in this repo.

## The shared setup

Every case uses the invented agency Lakeside Savings Circle and its workflow Quick
loan interest, which has three steps.

| Step | Ordinality | Kind | Fields |
|---|---|---|---|
| Work out the interest | 1 | `compute` | Provisional: `function` `loan.daily_interest@1`, `result_name` `Interest due`, `hold` `{"expires_after_days":14}`, no role. Inputs `balance_by_day`, `daily_rate` fixed 0.00125 and `days` fixed 32. |
| Approve the interest | 2 | `yes_no` | Role Loan officer. Provisional: `held_step_id` step 1. |
| Post the interest | 3 | `post` | Provisional: `amount_step_id` step 1, `account` `Member loans`, `counter_account` `Interest income`, `direction` `debit`. |

The agency keeps 2 currency places and rounds half up. Provisional: step 3 waits
while the figure is `held` and posts nothing. The step ids in the send keys are
written `post-9`; a test uses the id the draft assigns.

The members, each with one opening entry (the loan paid out) and no send key:

| Member | Opening entry | Opening amount | Figure the step works out on Monday |
|---|---|---|---|
| Wanjiru Kamau | E1 | 3,500.00 | 140.00 (3,500.00 x 0.00125 x 32) |
| Baraka Otieno | E1 | 2,800.00 | 112.00 (2,800.00 x 0.00125 x 32) |

An opening entry debits `Member loans (member)` and credits `Cash`. A payment entry
debits `Cash` and credits `Member loans (member)`. An interest entry debits
`Member loans (member)` and credits `Interest income`. The labels E1, E2 and E3
belong to the case that uses them; they are not ledger ids. The balance is never
stored: it is debits minus credits over the member's `Member loans` account, so each
case writes it as a sum of entries. Provisional: the held row is `HF-41`, minted
once; the next one is `HF-58`.

A held, rejected or expired figure leaves nothing in the ledger and nothing in the
member's figures. Every case that ends so says "no entry".

| Id | Case | Where it runs |
|---|---|---|
| H1 | Held, then released unchanged | Host, Ledger |
| H2 | Held, reworked at approval, the old figure refused | Host, Ledger |
| H3 | Held, then sent back | Host, Ledger |
| H4 | Held, then expired | Host, Ledger |
| H5 | A bulk release of 3,000 held figures | Host, Ledger |
| H6 | A hold with nothing that can release it | save, scan, publish |
| H7 | A hold or a `held_step_id` on the wrong kind of step | save |
| H8 | An expiry that is not 1 to 90 whole days | save |
| H9 | A releasing step naming the wrong step | save |
| H10 | A caller without the releasing role | Host |
| H11 | Fork, publication, hash and import | fork, publish, import, scan |

## H1 Held, then released unchanged

Setup:

- Member Baraka Otieno. Balance before: 2,800.00 (E1 +2,800.00).
- Provisional: Monday the step works out 112.00 and holds it as `HF-42`, with the
  fingerprint `ef771254`, `held`, expiring Monday two weeks later.
- No payment and nothing else moves his balance.

Steps:

1. Monday: the case reaches Work out the interest. The figure is held.
2. Wednesday: step 3 is still waiting. Nothing is posted.
3. Friday: the Loan officer opens `HF-42`. The host finds the fingerprint of the
   inputs now is `ef771254`, the same. The figure is shown as it is, 112.00.
4. She approves, naming `ef771254`.

Expected entries:

| Entry | Debit account | Credit account | Amount | Date | Note |
|---|---|---|---|---|---|
| E1 | Member loans (Baraka) | Cash | 2,800.00 | opening | none |
| E2 | Member loans (Baraka) | Interest income | 112.00 | Friday | none |

No entry is written on Monday or Wednesday. Balance after: 2,800.00 + 112.00 =
2,912.00 (E1 +2,800.00, E2 +112.00). Provisional: E2 carries `loan.daily_interest`,
version 1 and `ef771254`, and no "Reworked at approval" note. The held row is
`released` with `reworked_*` empty and `figure` 112.00.

Rule: [Rework at approval](../2.%20design/held-figures.md#4-rework-at-approval),
[Provenance](../2.%20design/held-figures.md#10-provenance-and-everything-that-copies-a-step)

## H2 Held, reworked at approval, the old figure refused

Setup:

- Member Wanjiru Kamau. Balance before: 3,500.00 (E1 +3,500.00).
- Provisional: Monday the step works out 140.00 and holds it as `HF-41`, with the
  fingerprint `1825fda0`.
- Wednesday she pays 200.00, late, in another workflow (E2).

Steps:

1. Monday: `HF-41` is held, 140.00. No entry.
2. Wednesday: E2 is posted. Her balance is 3,500.00 - 200.00 = 3,300.00. `HF-41`
   is still `held`, 140.00. No interest entry.
3. Friday: the Loan officer opens `HF-41`. The fingerprint of the inputs now is
   `1e73c5cb`, not `1825fda0`. The agency works it out again, same function and
   version, and shows 140.00 held on Monday, 132.50 worked out today, a difference
   of -7.50, and the reason: "The input Balance by day changed: it was 3,500.00 on
   every day when the figure was held, and is 3,300.00 from Wednesday."
4. She approves the old figure, naming `1825fda0`. It is refused with
   `ErrHeldFigureMoved` (409). The answer carries 132.50 and the reason.
5. She approves the new figure, naming `1e73c5cb`.
6. On Saturday she sends `HF-41` back. It is refused with `ErrHeldFigureState` (409).

Expected entries after step 4: none new. `HF-41` is still `held`, and nothing is
written.

Expected entries after step 5:

| Entry | Debit account | Credit account | Amount | Send key | Note |
|---|---|---|---|---|---|
| E1 | Member loans (Wanjiru) | Cash | 3,500.00 | none | none |
| E2 | Cash | Member loans (Wanjiru) | 200.00 | none | none |
| E3 | Member loans (Wanjiru) | Interest income | 132.50 | `post-9:run-31` | Reworked at approval; held 140.00 on Monday |

Balance after: 3,500.00 - 200.00 + 132.50 = 3,432.50 (E1 +3,500.00, E2 -200.00, E3
+132.50). Exactly one entry of 132.50 and none of 140.00. Provisional: E3 carries
`loan.daily_interest`, version 1 and the fingerprint `1e73c5cb`; the old
fingerprint `1825fda0` stays on the held row. The row is `released`, `figure` stays
140.00 and `reworked_*` hold 132.50 and `1e73c5cb`. After step 6 nothing changes.

Rule: [Rework at approval](../2.%20design/held-figures.md#4-rework-at-approval),
[The four states](../2.%20design/held-figures.md#2-the-four-states)

## H3 Held, then sent back

Setup: as H2 up to Friday. Balance 3,500.00 - 200.00 = 3,300.00 (E1 +3,500.00, E2
-200.00). `HF-41` is `held`, 140.00.

Steps:

1. Friday: the Loan officer opens `HF-41` and sees both figures and the reason of H2.
2. She sends it back with a blank reason. It is refused with `ErrInvalidHeldFigure`
   (400). `HF-41` stays `held`.
3. She sends it back with the reason "Interest waived, board minute 14".
4. She then approves it, naming `1e73c5cb`. It is refused with `ErrHeldFigureState`
   (409).

Expected entries: none beyond E1 and E2. After step 3 the row is `rejected` with the
reason, `decided_by` the Loan officer and `decided_at` set. The row stays in the
record of held figures. Balance: 3,500.00 - 200.00 = 3,300.00 (E1 +3,500.00, E2
-200.00). Nothing is in the member's figures. After step 4 nothing is written and
the row is still `rejected`. Provisional: where the case goes after send back is a
route the builder draws (decision 3) and is not checked here.

Rule: [The four states](../2.%20design/held-figures.md#2-the-four-states),
[Rework at approval](../2.%20design/held-figures.md#4-rework-at-approval)

## H4 Held, then expired

Setup: as H2 up to Wednesday. Balance 3,500.00 - 200.00 = 3,300.00. `HF-41` is held
on Monday and nobody decides. Open question 2: the owner has not decided expiry;
the proposal is 14 days, applied on read, with no timer.

Steps:

1. The Sunday before the second Monday: `HF-41` reads `held`.
2. Monday, two weeks after it was held: `HF-41` reads `expired`, with the reason
   "Expired after 14 days without a decision", written by the agency.
3. The day after: the Loan officer approves it, naming `1e73c5cb`. It is refused with
   `ErrHeldFigureExpired` (409).
4. The same day she sends it back. It is refused with `ErrHeldFigureExpired` (409).
5. A release batch naming `HF-41` returns `refused` for it, with
   `ErrHeldFigureExpired`.
6. The same day the step runs again for Wanjiru.

Expected entries: none beyond E1 and E2, at every step. Balance: 3,500.00 - 200.00 =
3,300.00 (E1 +3,500.00, E2 -200.00). `HF-41` stays as an `expired` row. Step 6 makes
a new held figure, `HF-58`, from that day's inputs, `held` with its own expiry, and
still no entry. `HF-41` is not changed by it.

Rule: [Expiry](../2.%20design/held-figures.md#5-expiry)

## H5 A bulk release of 3,000 held figures

Setup:

- 3,000 members of Lakeside, each with one held figure from one run of Work out the
  interest. Wanjiru Kamau is one of them. Each has an opening entry.
- 2,958 have inputs that have not moved. 40 have inputs that moved, in the last
  1,998 figures of the release. 2 members are held back by the Loan officer, and are
  among the unmoved.
- Provisional: the limits are the design page's proposals (open question 9): a list
  page of 100, at most 200, and a release batch of 500.

Steps:

1. List the held figures of the step in the run: 30 pages of 100, totals first
   (3,000 members, the sum of the figures, 2 held back).
2. The Loan officer leaves the 2 held-back members out of the call. She releases the
   other 2,998 in 6 calls: 5 of 500 and 1 of 498.
3. For each of the 40 `moved` figures, she approves the new figure in a second call
   naming its new fingerprint: 39 of them, Wanjiru's among them. She sends the 40th
   back with a reason.

Expected outcomes, one for each figure:

| Outcome | Count | Held row after | Ledger |
|---|---|---|---|
| `released` | 2,958 | `released`, unchanged | one entry each |
| `moved` | 40 | `held` until the second call | no entry |
| left out of the call | 2 | `held` | no entry |

After step 3: 2,958 + 39 = 2,997 `released`, 1 `rejected` and 2 `held`, which is
3,000. Entries: 2,958 after step 2, 2,997 after step 3, one for each released
figure and none for the rejected or held ones. Wanjiru's is E3 of 132.50 as in H2.
One member's moved inputs change nobody else's figure.

Restart: step 2 is cut off after 1,000 figures (calls 1 and 2). 1,000 entries exist.
Sending all 2,998 again in 6 calls returns `already_released` for those 1,000 and
writes nothing for them. The other 1,998 are handled as before: 1,958 `released` and
40 `moved`. The ledger holds 2,958 entries, not 3,958. Provisional: the design page
writes "1,998 release"; this case counts the 40 moved among them as `moved`, since
a moved figure stays `held`. Provisional: releasing a `released` figure returns the
first result, and the posting step's send key keeps the ledger to one entry.

Limits, each refused with `ErrInvalidHeldFigure` (400) and nothing written:

| Request | Expected |
|---|---|
| A list page of 200 | Accepted. |
| A list page of 201 | Refused. |
| A release batch of 500 | Accepted. |
| A release batch of 501 | Refused. |
| An empty batch | Refused. |
| An unknown state in the list filter | Refused. |

Rule: [Bulk](../2.%20design/held-figures.md#6-bulk-3000-members)

## H6 A hold with nothing that can release it

Setup: Provisional: Work out the interest is saved with `hold` `{"expires_after_days":14}`. There
is no step with `held_step_id` naming it, so Approve the interest does not exist.
The draft has no other issue.

Steps:

1. Save the step.
2. Run the Issue scan.
3. Publish.
4. Add Approve the interest with the role Loan officer and `held_step_id` step 1.
   Scan and publish again.

Expected:

1. The save is accepted, as a route step is saved with no route.
2. The scan raises one `error`: nothing can release the held figure.
3. Publishing is refused with `ErrPublishValidationFailed` (409).
4. The scan reports nothing for the step and the publish succeeds.

Entries: none, and no held figure is written, because no case runs a step that
cannot be published. Wanjiru's balance stays 3,500.00 (E1 +3,500.00).

Rule: [What the Issue scan flags](../2.%20design/held-figures.md#8-what-the-issue-scan-flags)

## Refusals

Each row is one save, on a draft with Lakeside's workflow. Every refusal is
`ErrInvalidWorkItem` (400) unless the row says otherwise. Nothing is stored when a
save is refused, no held figure is written and no entry is written.

### H7 A hold or a releasing field on the wrong kind

Provisional: `hold` and `held_step_id` are proposed names, so each row leans on them.

| Id | Setup | Expected |
|---|---|---|
| H7a | Create a `fill_form` step with `hold` `{"expires_after_days":14}`. | Refused: `hold` only on a `compute` step. |
| H7b | Create a `notify` step with `held_step_id` step 1. | Refused: `held_step_id` only on `yes_no` or `post`. |
| H7c | Create a `compute` step with a hold that also names a role. | Refused: open question 8, proposed answer no. |
| H7d | Create a `compute` step with `hold` `{"expires_after_days":14,"notify":true}`. | Refused: any other key. |

### H8 An expiry that is not whole days from 1 to 90

Provisional: the field `expires_after_days` and its range are proposals (open question 2).

| Id | `expires_after_days` | Expected |
|---|---|---|
| H8a | `0` | `ErrInvalidWorkItem` (400) |
| H8b | `-3` | `ErrInvalidWorkItem` (400) |
| H8c | `2.5` | `ErrInvalidWorkItem` (400) |
| H8d | `"fourteen"` | `ErrInvalidWorkItem` (400) |
| H8e | `91` | `ErrInvalidWorkItem` (400) |
| H8f | `1`, `14`, `90`, or left out | Accepted. Left out reads as 14. |

Open question 2: the range is proposed, not decided.

### H9 A releasing step naming the wrong step

Provisional: `held_step_id` and `amount_step_id` are proposed names, so each row leans on them.

| Id | Setup | Expected |
|---|---|---|
| H9a | Approve the interest names Apply for a loan, a `fill_form` step. | Refused: not a `compute` step. |
| H9b | Approve the interest names Work out the interest, which has no hold. | Refused. |
| H9c | Approve the interest is moved before Work out the interest, so the step it names is not earlier. | Refused. |
| H9d | `held_step_id` names the step itself. | Refused. |
| H9e | `held_step_id` names a step of another workflow. | Refused. |
| H9f | A second step names Work out the interest, which Approve the interest already names. | Refused. |
| H9g | Post the interest has `held_step_id` step 1 and `amount_step_id` another compute step. | Refused: the two must name the same step. |
| H9h | `held_step_id` is `st-404`, not a step of the draft. | `ErrWorkItemNotFound` (404). |

Rule: [What create and update refuse](../2.%20design/held-figures.md#7-what-create-and-update-refuse)

## H10 A caller without the releasing role

Setup: `HF-41` is `held`. The caller is the Treasurer of Lakeside, who does not hold
the role Loan officer and is not in a group offered to Approve the interest. Open
question 3: who may release is not decided; the proposal is only the releasing
step's role or offered group, "the agency owner included".

Steps:

1. The Treasurer approves `HF-41`, naming its fingerprint.
2. The Treasurer sends it back with a reason.

Expected: both are refused with `ErrHeldFigureRefused` (403). `HF-41` stays `held`,
no entry is written, and Wanjiru's balance stays 3,500.00 - 200.00 = 3,300.00 (E1
+3,500.00, E2 -200.00). The Loan officer can then act as in H2 or H3.

Rule: [The four states](../2.%20design/held-figures.md#2-the-four-states)

## H11 Fork, publication, hash and import

Setup for fork and hash, Provisional: `hold` and `held_step_id` are proposed names. Lakeside's March publication has no hold. Work out the
interest is copied by a fork as `st-52`.

| Id | Setup | Expected |
|---|---|---|
| H11a | Fork a draft where Work out the interest is copied as `st-52`. | The copied Approve the interest has `held_step_id` `st-52`. The `hold` is copied as it is, 14 days. No held figure is copied. |
| H11b | Publish the agency of H6 step 4. Fork a draft from the publication. | The document carries `hold` and `held_step_id` on those two steps and on no other. The fork restores both, with `held_step_id` remapped. |
| H11c | Publish an agency with a `fill_form` step. | The document step carries neither `hold` nor `held_step_id`. |
| H11d | Hash March's draft again after this change. | The hash is the same as before, because Provisional: a `WorkItemHold` record is added only for a step with a hold or a `held_step_id`. |
| H11e | In July add a hold and a releasing step, and publish. | The hash differs and the publication is made. Provisional: the hash gains a `WorkItemHold` record for those steps. |
| H11f | Change only `14` to `30` and publish. | The hash differs and the publication is made. `ErrNoChangesDetected` is not returned. |
| H11g | Change only the approver's role and publish. | The hash differs and the publication is made. |
| H11h | Change nothing and publish. | The hash is the same and `ErrNoChangesDetected` is returned. |
| H11i | Import an agency written before this change: 6 steps, none with `hold` or `held_step_id`. | It imports as 6 steps of their old kinds, nothing is invented, and the scan raises nothing about holds. |
| H11j | Import a document with a `hold` and no releasing step. | It imports, and the scan raises one error. Publishing is refused with `ErrPublishValidationFailed` (409). |
| H11k | Import a document with `hold` `{"expires_after_days":91}`. | It imports, and the scan raises one error. |

Entries: none in any row. Nothing here runs a case.

Rule: [Provenance, and everything that copies a step](../2.%20design/held-figures.md#10-provenance-and-everything-that-copies-a-step)

## Waiting on the owner

No case until the owner decides the open questions of the design page, section 11:
where a held figure lives (1), whether and when it expires (2), who may release (3),
whether the figure stays `held` when the rework fails (4), how a new figure equal to
the old one is shown (5), a balance for each day (6), the date of a record of figures
entry (7), a step that both confirms and holds (8), the limits (9) and the version
the rework uses (10). H2, H4, H7c, H8 and H10 lean on a proposal and change with it.
