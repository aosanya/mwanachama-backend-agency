# Calculating step failure: cases

Cases for the `on_failure` field of the calculating step, written so a later
test can load them one by one. The rules are in
[compute-step-failure.md](../2.%20design/compute-step-failure.md); what the host
does is in
[compute-step-failure-host.md](../3.%20implementation/compute-step-failure-host.md).
The calculating step kind is
[compute-step.md](../2.%20design/compute-step.md).

Provisional: the calculating step kind, its name on the wire and the formula
language are not written, so every expected result that leans on them starts
with that word. This repo stores and scans the field; the cases marked Host are
run by wakala-api, not here.

Every case uses one invented step, Check the loan payment, in the workflow Loan
check. It takes a balance, a payment and a term in months. The review step is
Review the loan payment, in the same workflow.

| Id | Case | Where it runs |
|---|---|---|
| C1 | No choice | scan, publish |
| C2 | A named step that is missing | scan, save |
| C3 | A self-pointing step | save, scan |
| C4 | A single stop | Host |
| C5 | A single route | Host |
| C6 | A bulk run with failed rows | Host |
| C7 | Target in another workflow | scan, save |
| C8 | Stop carrying a step id | save |
| C9 | Unknown choice | save |
| C10 | `on_failure` on a non-calculating step | save |
| C11 | Bad input, two problems at once | Host |
| C12 | Budget unchanged after a failed calculation | Host |
| C13 | Budget spent by a timeout | Host |
| C14 | `go_to_step` with no step id | save |
| C15 | Saved with no choice | save |
| C16 | A target that is a route step or another calculating step | save |
| C17 | A loop of calculating steps with no way out | scan, publish |

## C1 No choice

- Setup: Check the loan payment is saved with `on_failure` empty; the draft has
  no other issue.
- Expected: the Issue scan reports one `error` for the step, and publishing is
  refused with `ErrPublishValidationFailed`.
- Provisional: the step is the calculating kind, and the owner's rule is that it
  cannot be published without an answer for a failure.
- Rule: [What the Issue scan flags](../2.%20design/compute-step-failure.md#what-the-issue-scan-flags)

## C2 A named step that is missing

- Setup A (scan): `on_failure` is `{"choice":"go_to_step","step_id":"st-9"}` and
  `st-9` was deleted after the choice was saved.
- Expected A: the choice keeps `st-9` (a delete does not cascade); the scan
  reports one `error` of the form `Work item WI-12 "Check the loan payment"
  goes on failure to Step st-9, which does not exist`; publishing is refused.
- Setup B (save): create or update the step with `step_id` `st-404`, which is
  not a step of the draft.
- Expected B: `ErrWorkItemNotFound` (404); nothing is stored.
- Provisional: the step is the calculating kind.
- Rule: [What create and update refuse](../2.%20design/compute-step-failure.md#what-create-and-update-refuse),
  [What the Issue scan flags](../2.%20design/compute-step-failure.md#what-the-issue-scan-flags)

## C3 A self-pointing step

- Setup: `on_failure` is `{"choice":"go_to_step","step_id":"<id of Check the
  loan payment>"}`.
- Expected (save): `ErrInvalidWorkItem` (400); nothing is stored.
- Expected (scan, planted by writing the row without the save check): one
  `error`, and publishing is refused.
- Provisional: the step is the calculating kind.
- Rule: [What create and update refuse](../2.%20design/compute-step-failure.md#what-create-and-update-refuse),
  [What the Issue scan flags](../2.%20design/compute-step-failure.md#what-the-issue-scan-flags)

## C4 A single stop

- Setup: `on_failure` is `{"choice":"stop"}`. One case answers balance 120.50,
  payment -5.00, term empty.
- Expected: Provisional: the person gets one list of two sentences, "Payment
  must not be negative; received -5.00." and "Term in months is missing."; the
  step stays open; nothing from the step (answers or worked-out figures) is
  saved; no later step runs.
- Rule: [What the person sees on a stop](../2.%20design/compute-step-failure.md#what-the-person-sees-on-a-stop)

## C5 A single route

- Setup: `on_failure` is `{"choice":"go_to_step","step_id":"<id of Review the
  loan payment>"}`. One case answers balance 120.50, payment 45.25, term 0
  months.
- Expected: Provisional: the case continues at Review the loan payment, which
  reads "The monthly share could not be worked out because the term is 0
  months." as an answer to the failed step, with its code beside it; the failed
  step saves nothing; the run's failure budget is unchanged.
- Rule: [What travels to the named step](../2.%20design/compute-step-failure.md#what-travels-to-the-named-step)

## C6 A bulk run with failed rows

- Setup: Check the loan payment runs over 200 rows. Row 41 has payment -5.00,
  row 97 has no balance, row 163 has a term of 0 months. The other 197 are
  valid. Run it twice, once with `{"choice":"stop"}` and once with `go_to_step`
  naming Review the loan payment.
- Expected with stop: Provisional: 197 rows finish; the run lists 3 failed rows,
  each with its row number and reasons, for example "Row 41: Payment must not be
  negative; received -5.00."
- Expected with a named step: Provisional: Review the loan payment receives
  exactly 3 rows, each with its sentences and codes; the 197 others carry on and
  never reach it.
- Rule: [Bulk runs](../2.%20design/compute-step-failure.md#bulk-runs)

## C7 Target in another workflow

- Setup: `step_id` is a step of the workflow Savings check, not of Loan check.
- Expected (save): `ErrInvalidWorkItem` (400). Expected (scan, if planted): one
  `error`, publishing refused. This also holds when the target was moved to
  another workflow after the choice was saved.
- Provisional: the step is the calculating kind.
- Rule: [What create and update refuse](../2.%20design/compute-step-failure.md#what-create-and-update-refuse),
  [What the Issue scan flags](../2.%20design/compute-step-failure.md#what-the-issue-scan-flags)

## C8 Stop carrying a step id

- Setup: `on_failure` is `{"choice":"stop","step_id":"st-7"}`.
- Expected: `ErrInvalidWorkItem` (400); nothing is stored.
- Provisional: the step is the calculating kind.
- Rule: [What create and update refuse](../2.%20design/compute-step-failure.md#what-create-and-update-refuse)

## C9 Unknown choice

- Setup: `on_failure` is `{"choice":"retry"}`.
- Expected: `ErrInvalidWorkItem` (400); nothing is stored.
- Provisional: the step is the calculating kind.
- Rule: [What create and update refuse](../2.%20design/compute-step-failure.md#what-create-and-update-refuse)

## C10 `on_failure` on a non-calculating step

- Setup: a step that is not the calculating kind, such as a `notify` step named
  Tell the member, is created with `on_failure` `{"choice":"stop"}`.
- Expected: `ErrInvalidWorkItem` (400); nothing is stored.
- Provisional: the check separates the calculating kind from every other kind.
- Rule: [What create and update refuse](../2.%20design/compute-step-failure.md#what-create-and-update-refuse)

## C11 Bad input, two problems at once

- Setup: Check the loan payment with `{"choice":"stop"}`; balance 120.50,
  payment -5.00, term empty.
- Expected: Provisional: the answer holds both problems in one list, in the
  order the inputs were asked; each problem names the input, the value received
  and the rule broken, with a plain sentence and a code; the calculation is not
  attempted.
- Rule: [The two failures](../2.%20design/compute-step-failure.md#the-two-failures)

## C12 Budget unchanged after a failed calculation

- Setup: a run with a failure budget of 5; Check the loan payment fails once
  with balance 120.50, payment 45.25, term 0 months.
- Expected: Provisional: the budget reads 5 before and 5 after; the case follows
  the choice.
- Rule: [Failure budget](../2.%20design/compute-step-failure.md#failure-budget)

## C13 Budget spent by a timeout

- Setup: the same run, budget 5; Check the loan payment times out.
- Expected: the budget reads 5 before and 4 after, and `on_failure` is not
  followed, because a timeout is a system fault, not a failed calculation.
- Provisional: the step is the calculating kind.
- Rule: [Failure budget](../2.%20design/compute-step-failure.md#failure-budget)

## C14 `go_to_step` with no step id

- Setup: `on_failure` is `{"choice":"go_to_step"}`.
- Expected: `ErrInvalidWorkItem` (400); nothing is stored.
- Provisional: the step is the calculating kind.
- Rule: [What create and update refuse](../2.%20design/compute-step-failure.md#what-create-and-update-refuse)

## C15 Saved with no choice

- Setup: Check the loan payment is created with `on_failure` empty, before
  Review the loan payment exists.
- Expected: the save succeeds; only the scan (C1) stops publishing.
- Provisional: the step is the calculating kind.
- Rule: [What create and update refuse](../2.%20design/compute-step-failure.md#what-create-and-update-refuse)

## C16 A target that is a route step or another calculating step

- Setup: `on_failure` of Check the loan payment is `{"choice":"go_to_step",
  "step_id":"<id of a route step in Loan check>"}`. A second save names another
  calculating step of the workflow, Check the term.
- Expected: both saves succeed; any step of the workflow except the step itself is
  allowed (decision 12).
- Provisional: the step is the calculating kind.
- Rule: [Where it lives](../2.%20design/compute-step-failure.md#where-it-lives)

## C17 A loop of calculating steps with no way out

- Setup: Check the loan payment goes on failure to Check the term, and Check the
  term goes on failure to Check the loan payment. Both are saved.
- Expected (save): both saves succeed, since each names a step other than itself.
- Expected (scan): Provisional: one `error` for the loop, and publishing is
  refused with `ErrPublishValidationFailed` (409). Changing Check the term to
  `{"choice":"stop"}` leaves a way out, and the scan raises nothing for the loop.
- Rule: [What the Issue scan flags](../2.%20design/compute-step-failure.md#what-the-issue-scan-flags)
