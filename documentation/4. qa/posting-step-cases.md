# Posting step: cases

Cases for the posting step, written so a later test can load them one by one. The
rules are in [posting-step.md](../2.%20design/posting-step.md); what the host and
the accounting repo must do is in
[posting-step-host.md](../3.%20implementation/posting-step-host.md). What the
accounting repo builds today is in
[accounting-ledger-audit.md](../3.%20implementation/accounting-ledger-audit.md).
These are cases for tests to be written, not tests that exist.

Provisional: the calculating step ([compute-step.md](../2.%20design/compute-step.md),
ISS-19), its failure choice ([compute-step-failure.md](../2.%20design/compute-step-failure.md),
ISS-6), the formula language, the name `post`, the shape of the send key and the
three provenance fields are not written or not built, so every sentence that leans
on one starts with that word. This repo stores and scans the posting fields; the
cases marked Host are run by wakala-api, and the cases marked Accounting are
checked in the accounting repo.

## The shared setup

Every case uses the invented agency Lakeside Savings Circle and its workflow
Loan payment, which has two steps.

| Step | Ordinality | Kind | Fields |
|---|---|---|---|
| Work out the payment | 1 | calculating | Provisional: `function` `loan.cap@3`, `result_name` `Payment to post`, no role. |
| Post the payment | 2 | `post` | `amount_step_id` the step above, `account` `Member loans`, `counter_account` `Cash`, `direction` `credit`, `reason` `Loan payment`. |

The agency has two accounts under the category `Member loans` and `Cash`. A member
has one `Member loans` account, found by category and the member's id as holder.
In the send keys below the posting step is written `post-7`; a test uses the
step id the draft assigns.

The members, each with the balance before the run, which is one opening entry
(the loan paid out) with no send key:

| Member | Opening entry | Opening amount | Payment the calculating step works out |
|---|---|---|---|
| Wanjiru Kamau | E1 | 500 | 50 |
| Otieno Mwangi | E2 | 300 | 30 |
| Achieng Ouma | E3 | 800 | 80 |
| Baraka Njoroge | E4 | 200 | 20 |
| Halima Said | E5 | 400 | 40 |

An opening entry debits `Member loans (member)` and credits `Cash`. A payment
entry debits `Cash` and credits `Member loans (member)`. The labels E1, E2 and so
on belong to the case that uses them; they are not ledger ids, and a test maps
each label to the id the ledger returns. The balance is never stored: it is
debits minus credits over every entry of the member's `Member loans` account, as
the design page section 7 says, so each case writes it as a sum of entries.

Every payment entry, in every case, carries a reason `Loan payment`. Provisional:
it also carries the function `loan.cap`, version `3` and a fingerprint of the
inputs; Wanjiru's carries `c91f0a47`, and a member with other inputs carries
another fingerprint.

| Id | Case | Where it runs |
|---|---|---|
| P1 | One payment | Host, Accounting |
| P2 | A reversal, and a second reversal refused | Host, Accounting |
| P3 | The same step run twice, and a restarted bulk run | Host, Accounting |
| P4 | A bulk run with failed rows | Host |
| P5 | No amount source | save, scan, publish |
| P6 | A direction that is neither debit nor credit | save |
| P7 | An amount step that is not a calculating step | save |
| P8 | An account the agency does not have | save |
| P9 | An amount step that is not a step of the draft | save |
| P10 | The same account on both sides | save |

## P1 One payment

Setup:

- Member Wanjiru Kamau. Her `Member loans` account holds E1.
- Balance before: 500 (E1 +500).
- Provisional: the calculating step Work out the payment returns 50 for her case.
- The posting step is as in the shared setup. Run 12 reaches it for her case.

Steps:

1. The case reaches Post the payment in run 12.
2. The host resolves her to her `Member loans` account and `Cash`.
3. The host posts 50 once, with the reason `Loan payment`.

Expected entries:

| Entry | Debit account | Credit account | Amount | `reverses_entry_id` | Send key |
|---|---|---|---|---|---|
| E1 | Member loans (Wanjiru) | Cash | 500 | none | none |
| E2 | Cash | Member loans (Wanjiru) | 50 | none | `post-7:run-12` |

Balance after: 500 - 50 = 450 (E1 +500, E2 -50). The amount on E2 is 50, never
-50: direction is which account sits on which side.

Today's accounting code:

- Provisional: it cannot yet satisfy the send key on E2: the entry has no key
  field (audit gap 4).
- Provisional: it cannot yet satisfy the provenance on E2: the entry has no
  function code, version or fingerprint (audit gap 3).
- Provisional: it can keep one account per member only if the host always opens
  members under the category `Member loans` (audit gap 2).

Rule: [The step kind](../2.%20design/posting-step.md#1-the-step-kind),
[Worked example](../2.%20design/posting-step.md#2-worked-example-the-entries-and-the-balance)

## P2 A reversal, and a second reversal refused

Setup: P1 has run. Wanjiru's balance is 450 (E1 +500, E2 -50). Her payment of 50
was a mistake and is to be corrected.

Steps:

1. A correction of E2 is posted with `reverses_entry_id` E2.
2. A second correction of E2 is posted.

Expected entries after step 1:

| Entry | Debit account | Credit account | Amount | `reverses_entry_id` | Send key |
|---|---|---|---|---|---|
| E1 | Member loans (Wanjiru) | Cash | 500 | none | none |
| E2 | Cash | Member loans (Wanjiru) | 50 | none | `post-7:run-12` |
| E3 | Member loans (Wanjiru) | Cash | 50 | E2 | none |

Balance after step 1: 500 - 50 + 50 = 500 (E1 +500, E2 -50, E3 +50). Between E2
and E3 the balance read 450. E2 is still in the list and still shows 50: the
original stays visible and E3 is a second line that cancels it. E3 swaps the two
accounts of E2 and repeats its amount.

Step 2: the ledger refuses with `ErrAlreadyReversed`. No fourth entry is written
(E4 does not exist), and the balance stays 500 - 50 + 50 = 500.

Today's accounting code:

- It already satisfies step 2: `ErrAlreadyReversed`, backed by the unique index
  `reverses_once`, and a reversal naming an entry that does not exist is refused.
- Provisional: it cannot yet satisfy the mirror: nothing checks that E3 swaps the
  accounts or repeats 50, so a "reversal" of 500 against E2 in the same direction
  as E2 is accepted today and must be refused (audit gap 1). Until it is, the
  test for that refusal fails and the host writes the mirror itself.
- Provisional: it cannot yet satisfy a required reason on E3, since the reason is
  optional even on a reversal (audit gap 5).
- Provisional: how a person asks for the correction (a step, a button or a tool)
  is open (design page section 12, question 7), so the first step of this case is
  run by a test calling the ledger, not by a person.

Rule: [Reversal](../2.%20design/posting-step.md#6-reversal-the-correction)

## P3 The same step run twice, and a restarted bulk run

### P3a The same step run twice

Setup: P1 has run. Wanjiru's balance is 450 (E1 +500, E2 -50) and the key
`post-7:run-12` is used.

Steps: run 12 reaches Post the payment for Wanjiru's case a second time, with the
same step and the same run.

Expected: the second run posts nothing and returns E2, the first entry.

| Entry | Debit account | Credit account | Amount | `reverses_entry_id` | Send key |
|---|---|---|---|---|---|
| E1 | Member loans (Wanjiru) | Cash | 500 | none | none |
| E2 | Cash | Member loans (Wanjiru) | 50 | none | `post-7:run-12` |

Balance: 500 - 50 = 450 (E1 +500, E2 -50). The ledger holds 2 entries for her
account, not 3.

A second payment that is really a second payment is not a repeat. Run 13 reaches
the step for her with the key `post-7:run-13`, the calculating step returns 50
again, and the entry is written:

| Entry | Debit account | Credit account | Amount | `reverses_entry_id` | Send key |
|---|---|---|---|---|---|
| E3 | Cash | Member loans (Wanjiru) | 50 | none | `post-7:run-13` |

Balance after run 13: 500 - 50 - 50 = 400 (E1 +500, E2 -50, E3 -50).

### P3b A bulk run restarted after 3 of 5 rows

Setup: all 5 members of the shared setup, opening entries E1 to E5, balances 500,
300, 800, 200, 400. Run 12 posts each row on its own, with the key
`post-7:run-12:<member>`. The run stops after 3 rows: Wanjiru, Otieno and Achieng
are posted, Baraka and Halima are not.

Steps:

1. Run 12 stops after row 3: payments E6, E7 and E8 exist.
2. Run 12 is restarted over the same 5 rows.

Expected entries after the restart (the opening entries E1 to E5 are not repeated
here):

| Entry | Debit account | Credit account | Amount | `reverses_entry_id` | Send key | Written by |
|---|---|---|---|---|---|---|
| E6 | Cash | Member loans (Wanjiru) | 50 | none | `post-7:run-12:wanjiru` | first run |
| E7 | Cash | Member loans (Otieno) | 30 | none | `post-7:run-12:otieno` | first run |
| E8 | Cash | Member loans (Achieng) | 80 | none | `post-7:run-12:achieng` | first run |
| E9 | Cash | Member loans (Baraka) | 20 | none | `post-7:run-12:baraka` | restart |
| E10 | Cash | Member loans (Halima) | 40 | none | `post-7:run-12:halima` | restart |

The restart finds 3 keys already used (Wanjiru, Otieno, Achieng): those 3 rows
write nothing and return E6, E7 and E8. The other 2 rows post. There are 5 payment
entries in all, not 8.

| Member | Balance after |
|---|---|
| Wanjiru Kamau | 500 - 50 = 450 |
| Otieno Mwangi | 300 - 30 = 270 |
| Achieng Ouma | 800 - 80 = 720 |
| Baraka Njoroge | 200 - 20 = 180 |
| Halima Said | 400 - 40 = 360 |

Today's accounting code:

- Provisional: it cannot yet satisfy P3a or P3b: there is no send key and no
  unique index, only a lookup inside the transaction that matches on document
  kind, document id, both accounts, the amount, `reverses_entry_id` and
  `settles_entry_id` (audit gap 4).
- Provisional: that match is also wrong for the second payment of P3a, if both
  payments carry the same document: E3 would collapse into E2 and the balance
  would stay 450 instead of 400. The key must decide, not the match.
- Provisional: what `document_kind` and `document_id` a posting step writes is
  open (design page section 12, question 9), so which of the two failures a test
  sees depends on that.
- Provisional: the exact shape of the key is open (section 12, question 6); the
  keys above are the design page's example shape.

Rule: [The send key](../2.%20design/posting-step.md#4-the-send-key-running-a-step-twice-posts-once)

## P4 A bulk run with failed rows

Setup: all 5 members of the shared setup, run 12, balances before 500, 300, 800,
200, 400 (opening entries E1 to E5). Provisional: the calculating step works out 50,
30 and 80 for Wanjiru, Otieno and Achieng. It fails for Baraka (the payment input
is -5.00) and for Halima (her balance is missing), so it returns no amount for
them.

Steps:

1. Run 12 reaches Post the payment for 5 rows.
2. Run the case twice: once with the calculating step's failure choice `stop` and
   once with `go_to_step` naming Review the loan payment.

Expected entries, the same under both choices:

| Entry | Debit account | Credit account | Amount | `reverses_entry_id` | Send key |
|---|---|---|---|---|---|
| E6 | Cash | Member loans (Wanjiru) | 50 | none | `post-7:run-12:wanjiru` |
| E7 | Cash | Member loans (Otieno) | 30 | none | `post-7:run-12:otieno` |
| E8 | Cash | Member loans (Achieng) | 80 | none | `post-7:run-12:achieng` |
| none | none | none | none | none | none for Baraka |
| none | none | none | none | none | none for Halima |

3 entries are written and 2 are not. Neither failed row is posted half, and no
row waits on another.

| Member | Balance after |
|---|---|
| Wanjiru Kamau | 500 - 50 = 450 |
| Otieno Mwangi | 300 - 30 = 270 |
| Achieng Ouma | 800 - 80 = 720 |
| Baraka Njoroge | 200 = 200 (E4 only) |
| Halima Said | 400 = 400 (E5 only) |

Where the 2 failed rows go (listed on the run, or sent to the named step) follows
the step's failure choice, which is
[compute-step-failure.md](../2.%20design/compute-step-failure.md) and
[compute-step-failure-cases.md](compute-step-failure-cases.md) C6; this case does
not decide it and checks only that no entry is written for them.

Today's accounting code:

- Provisional: it cannot yet satisfy the keys on E6 to E8 (audit gap 4) or their
  provenance (audit gap 3).
- It does satisfy the balances: debits minus credits, read on each call, with no
  stored balance (audit item 4).
- Provisional: the calculating step's result for a failed row, and so what the
  host sees when there is no amount, is not written (ISS-19, ISS-6).
- Provisional: a failed row uses no key, so a later run could post it once its
  inputs are corrected; whether that run is allowed is not decided.
- Where a confirming role is named on the calculating step, none of E6 to E8 is
  written until the confirmer confirms the run, and a member held back posts
  nothing. Provisional: that is [compute-step.md](../2.%20design/compute-step.md)
  section 6, and this case runs without a confirming role.

Rule: [A bulk run posts each row on its own](../2.%20design/posting-step.md#5-a-bulk-run-posts-each-row-on-its-own)

## P5 No amount source

Setup: Post the payment is saved with `account` `Member loans` and nothing else:
no `amount_step_id`, no `counter_account`, no `direction`, no `reason`. The draft
has no other issue.

Steps:

1. Save the step.
2. Run the Issue scan.
3. Publish.
4. Pick the calculating step Work out the payment as `amount_step_id`, and fill
   the other three fields as in the shared setup. Scan and publish again.

Expected:

1. The save is accepted. A workflow is built step by step, so a posting step may
   be saved before its calculating step is picked.
2. The scan reports `error` entries for the step: no amount source (the owner's
   rule), and no `counter_account`, no `direction` and no `reason`, 4 in all.
3. Publishing is refused with `ErrPublishValidationFailed` (409).
4. The scan reports nothing for the step and the publish succeeds.

Entries: none are written in any step, because no case runs a step that cannot be
published. Wanjiru's balance stays 500 (E1 +500).

Today's accounting code: nothing in the accounting repo is involved.
Provisional: this repo does not store the posting fields yet, since
`amount_step_id` is not written anywhere but the design page, so the test cannot
run until the field, the scan and the save checks exist.

Rule: [What create and update refuse](../2.%20design/posting-step.md#8-what-create-and-update-refuse),
[What the Issue scan flags](../2.%20design/posting-step.md#9-what-the-issue-scan-flags)

## Refusals

Each row is one save of Post the payment, on a draft with Lakeside's accounts,
and nothing is stored when it is refused. Wanjiru's balance is 500 (E1 +500) and
no entry is written in any of them.

| Id | Setup | Expected | Rule |
|---|---|---|---|
| P6 | `direction` is `credit side`. | `ErrInvalidWorkItem` (400). | section 8 |
| P7 | `amount_step_id` names Apply for a loan, a `fill_form` step. | `ErrInvalidWorkItem` (400): not a calculating step. | section 8 |
| P8 | `account` is `Member savings`, which Lakeside does not have. | Provisional: `ErrAccountNotFound` (404); the name is proposed, not decided. | section 8 |
| P9 | `amount_step_id` is `st-404`, which is not a step of the draft. | `ErrWorkItemNotFound` (404). | section 8 |
| P10 | `account` and `counter_account` are both `Member loans`. | `ErrInvalidWorkItem` (400). | section 8 |

Today's accounting code: nothing in the accounting repo is involved.
Provisional: P8 also needs the agency's list of accounts, which this repo does not
keep and the design page leaves open (section 12, question 2), so it cannot run
until that is decided. Provisional: P6, P7, P9 and P10 cannot run until this repo
stores the posting fields.
