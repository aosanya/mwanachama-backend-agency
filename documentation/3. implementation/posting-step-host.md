# What the host and the accounting repo must do for the posting step

This repo stores and scans the posting step; it does not post. The host
(wakala-api, the service that runs agency workflows) runs the step and calls the
ledger, and the accounting repo keeps the ledger. Design:
[posting-step.md](../2.%20design/posting-step.md). Cases:
[posting-step-cases.md](../4.%20qa/posting-step-cases.md). What the accounting
repo builds today: [accounting-ledger-audit.md](accounting-ledger-audit.md).

Provisional: the calculating step ([compute-step.md](../2.%20design/compute-step.md),
ISS-19), its failure choice ([compute-step-failure.md](../2.%20design/compute-step-failure.md),
ISS-6), the formula language and the name `post` are not all written, so each
sentence leaning on one starts with that word. This page lists the work; it files
no issue. The lead files them after the owner has seen the list.

Example throughout: Lakeside Savings Circle, the workflow Loan payment, the step
Post the payment, and the member Wanjiru Kamau, who owes 500 and pays 50.

## What wakala-api does

What it does today is partly unknown. In `mwanachama-wakala-api`,
`internal/api/http/accounting_routes.go` mounts the accounting repo's routes per
instance under `/instances/{instanceID}/accounting`, so a caller can reach the
ledger. A search of the repo's Go files found no step kind `post` or `compute`, no
code that posts an entry for a workflow step, and no send key. Provisional: what
the host does when a case reaches a step of an unknown kind is not known from the
repos, and this page does not say.

1. Read the posting step from the document published to the production channel:
   `amount_step_id`, `account`, `counter_account`, `direction`, `reason`. Design
   section 1. Provisional: the five fields are not stored in this repo yet.
2. Resolve the member to the ledger account: the person the case is about, under
   the category `Member loans`, opened on first use with the member's id as
   holder, so there is one account per member. Case P1. Provisional: how the case
   knows who the member is, and which category the agency uses, are open (design
   section 12, question 3).
3. Read the calculating step's result by `amount_step_id`, under its
   `result_name` (`Payment to post`). Cases P1, P4. Provisional: where the result
   is kept for later steps, and its type, are not written (ISS-19).
4. Build the send key from the step, the run and, in a bulk run, the row, such as
   `post-7:run-12:wanjiru`, and send it with the entry. Cases P3a, P3b.
   Provisional: the key's exact shape is open (design section 12, question 6).
5. Build the provenance: the function code (`loan.cap`), the published version
   the publication pins (`3`) and a fingerprint of the inputs (`c91f0a47` for
   Wanjiru). Case P1. Provisional: how the fingerprint is worked out is open, and
   the calculating step's function and version come from its `function` field.
6. Post once per row: one calculating result, one entry, with the amount unsigned,
   the side set by `direction` and the other side `counter_account`, and the
   `reason`. A row never waits on another row and never posts half. Cases P1, P4.
   Design section 5.
7. When the ledger answers that the key is used, treat it as success and carry on
   with the entry it returned, writing nothing. A bulk run that is restarted
   skips the rows already posted and posts the rest. Cases P3a, P3b.
8. Follow the calculating step's failure choice for a row with no amount, and post
   nothing for it, under either choice. Case P4. Provisional: what the choice does
   is [compute-step-failure.md](../2.%20design/compute-step-failure.md), and the
   host does not decide it.
9. Where the calculating step names a confirming role, post nothing until the
   confirmer confirms the run, and post nothing for a member held back. Design
   section 5, [compute-step.md](../2.%20design/compute-step.md) section 6.
10. Post a correction as a reversing entry that names the entry it reverses, swaps
    the two accounts and repeats the amount, so the host writes the mirror itself
    until the ledger checks it. Case P2. Provisional: how a person asks for a
    correction is open (design section 12, question 7).
11. Read the balance as a fold for the screen and for reports: add up the
    account's entries on each read (debits minus credits), and never store a
    balance or write one from a posting step. Cases P1 to P4, design section 7.
    Provisional: a balance as at a date, which month-end needs, is not listed in
    the audit.
12. Keep the posting fields when the reviewer renames steps
    (`reviewruns/workflows.go`), as for the calculating fields, or a review's
    rename blanks them. Provisional: whether that file keeps unknown fields today
    is not checked here.

## What the accounting repo must do

These are the audit's gaps 1 to 6, stated as the accounting repo's work. None is
made here and none is filed here.

1. Refuse a reversal that does not mirror its original: the two accounts not
   swapped, or the amount not equal. Today a "reversal" of 500 for 50 in the same
   direction is accepted. Audit gap 1, case P2.
2. Fix the category for member accounts and test one account per member. Today
   one account per member holds only if every caller opens members under one
   category, and no test uses a member. Audit gap 2, cases P1 and P4.
3. Carry the function code, the published version and the input fingerprint on an
   entry. Today the entry has none of the three. Audit gap 3, case P1.
4. Add a send key to an entry, with a unique index, and return the first entry
   when the key is used. Today the replay is a lookup on document kind, document
   id, both accounts, the amount, `reverses_entry_id` and `settles_entry_id`, so
   two real payments of 50 on the same document and accounts collapse into one.
   Audit gap 4, cases P3a and P3b.
5. Require a reason on a reversal. Today the reason is optional, even on a
   reversal. Audit gap 5, case P2.
6. Fold the account balance in the database instead of loading every entry into
   Go. This is a cost, not a wrong answer. Audit gap 6, design section 7.

Provisional: a balance as at a date, how the ledger keeps cents (an entry's
amount is a whole number today), whether the ledger accepts a result of 0, and
which `document_kind` and `document_id` a posting step writes are open (design
section 12, questions 4, 5, 8 and 9) and are not accounting work yet.

## Not here

- The calculating step's work: [compute-step-host.md](compute-step-host.md).
- What happens when a calculating step fails:
  [compute-step-failure-host.md](compute-step-failure-host.md).
- What this repo stores and scans for the step: the design page, sections 8 to 10.
