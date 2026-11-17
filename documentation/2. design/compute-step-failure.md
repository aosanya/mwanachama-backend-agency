# When a calculating step fails

Owner decisions of 2026-11-17: a calculating step can fail in two ways, every
calculating step says what happens when it does, and the answer is one of two
choices. This page adds one field to the step. The step kind itself is in
[compute-step.md](compute-step.md) (the calculating step kind); this extends
[workflow-steps.md](workflow-steps.md) and
[workflow-records.md](workflow-records.md).

Provisional: the calculating step kind, its name on the wire and the formula
language are still being written, so every sentence here that leans on them
starts with that word.

The example throughout is one invented step, Check the loan payment. It takes a
balance, a payment and a term in months, and works out the share of the balance
each month.

## The two failures

- Bad inputs, found before any calculation. The function's own input rules
  catch them: a missing balance, or a payment of -5.00. The whole list of
  problems comes back at once, each naming the input, the value received and the
  rule broken. A member with balance 120.50 who pays -5.00 and leaves the term
  empty gets two problems, not one.
- A failure during the calculation, which is rarer: a division by zero (a term
  of 0 months), a result outside its allowed range, or an aggregate that could
  not be read. It gives one problem.

A failed calculation is a business outcome. It follows the path the builder
chose and does not count against the run's failure budget (see below).

## Where it lives

One field on the step, `on_failure`, stored the way `routes` is: a json column
of the `work_item` object in `agency.blueprint.json`, a field of
`models.WorkItem`, and a field of `models.DocumentWorkItem` with `omitempty`.

| Field | Meaning |
|---|---|
| `on_failure` | What happens when the step fails, either kind. Empty means no choice made yet. |

It is one of two shapes, and there is no third:

- `{"choice":"stop"}`: stop and tell the person what is missing.
- `{"choice":"go_to_step","step_id":"<step id>"}`: send the case to that step,
  which is in the same workflow. Decided (decision 12 in
  [compute-decisions.md](../1.%20requirements/compute-decisions.md)): it may be any
  step of the workflow except the failing step itself, so a `route` step or another
  calculating step is allowed.

One choice covers both kinds of failure; there is no separate choice per kind.
Provisional: `on_failure` exists only on the calculating step kind.

Stop is a real choice. A simple step such as an order total (quantity 4 at 2.50
each) needs no review step: its builder picks stop and nothing else.

## What create and update refuse

With `ErrInvalidWorkItem` (400):

- `on_failure` on any step that is not the calculating kind (Provisional:
  such as a `notify` step carrying `{"choice":"stop"}`);
- a choice that is neither `stop` nor `go_to_step` (`{"choice":"retry"}`);
- `go_to_step` with no `step_id`;
- `stop` with a `step_id` (`{"choice":"stop","step_id":"st-7"}`);
- a `step_id` that is a step of another workflow;
- a `step_id` that is the step itself (Check the loan payment sending the case
  back to Check the loan payment).

With `ErrWorkItemNotFound` (404): a `step_id` that is not a step of the draft.

A calculating step may be saved with no choice. A workflow is built step by
step and the target comes after, so Check the loan payment is saved first and
its review step is made later. The Issue scan is what stops publishing.

## What the person sees on a stop

Every problem is one plain sentence, all in one list, in the order the inputs
were asked.

- Bad inputs: one line per problem, all at once, never one at a time. For the
  member above: "Payment must not be negative; received -5.00." and "Term in
  months is missing."
- During the calculation: one sentence, such as "The monthly share could not be
  worked out because the term is 0 months."

The step stays open, so the person can correct the answers and send it again.
Nothing is saved from the failed step: the answers it was given and anything it
would have worked out are not kept.

Provisional: a problem is the input, the value received, the rule broken, a
plain sentence and a code, and the sentence is the one the person reads.

## What travels to the named step

With `go_to_step` the case continues at that step. It carries:

- the same sentences, readable at the named step as an answer to the failed
  step: the reviewer of Check the loan payment reads "Payment must not be
  negative; received -5.00.";
- the codes a program can read, one per problem, beside each sentence.

Provisional: a code such as `payment_below_minimum` is the problem's code and
one is carried for every problem.

## What the Issue scan flags

All are `error`, so none can be published:

- a calculating step with no choice;
- a named step that no longer exists, or now belongs to another workflow;
- a named step that is the calculating step itself;
- a loop of calculating steps that fail onto each other with no way out (decision
  12). Provisional: "no way out" is read as a cycle of calculating steps, each
  going to the next on failure, from which no step leaves by stopping or by going to
  a step outside the cycle. How the scan walks the choices is not written.

Provisional: the first of these is the owner's rule that a calculating step
with no answer for a failure cannot be published.

Example: Check the loan payment goes on failure to Check the term, and Check the
term goes on failure back to Check the loan payment. Neither can stop, so the scan
raises one error for the loop. If Check the term chose stop, there is a way out and
nothing is raised.

The scan checks the target the way `checkStepTarget` checks a route target, so
the sentences read like the existing ones: `Work item WI-12 "Check the loan
payment" goes on failure to Step st-9, which does not exist`.

## Everything that copies a step

- A draft fork remaps `step_id` onto the forked steps, as it does `routes` and
  `otherwise_step_id`. A stop has no id and is copied as it is.
- The publication document carries `on_failure` on `DocumentWorkItem`, and
  forking a draft from a publication restores it and remaps `step_id` onto the
  restored steps.
- The content hash adds `on_failure` to the step's record only when it is set,
  so a publication made before the field existed hashes the same and does not
  read as changed. A publication whose only change is a calculating step going
  from `stop` to a named step is a change that can be published.
- Import keeps the choice and remaps the step id onto the imported steps.
- Deleting the target step does not cascade. The choice keeps the old id and the
  scan finds it, as for a route target.

## Bulk runs

Each failed row follows the step's own choice.

Example: Check the loan payment runs over 200 rows and 3 fail (one with a
payment of -5.00, one with no balance, one with a term of 0 months).

- Stop: the 197 other rows finish. The run then lists the 3 failed rows, each
  with its row number and its reasons: "Row 41: Payment must not be negative;
  received -5.00."
- Go to step Review the loan payment: that step receives only the 3 failed rows,
  each with its sentences and codes. The 197 others carry on as usual and never
  reach it.

A bad-input row gives all its problems, so a row with two bad inputs shows two
lines under its row number.

A failed row does not hold up the calculation batch: the calculation batch works
out every row and finishes as a whole, and a failed row is one of its outcomes
(decision 11 in [compute-decisions.md](../1.%20requirements/compute-decisions.md)).
Provisional: the failed rows follow the choice when the calculation batch has
finished. Nothing is posted for any row until the actualization batch.

## Failure budget

A run has a failure budget, kept for system faults such as a timeout. A failed
calculation spends none of it: the 3 failed rows above leave a budget of 5 at
5. A step that times out spends 1 of it.

## What wakala-api does

wakala-api runs the calculation and follows the choice: it lists the problems on
a stop, sends the case and the problems to the named step, and leaves the budget
alone. This repo only stores and checks the field. The host's side is in
[compute-step-failure-host.md](../3.%20implementation/compute-step-failure-host.md).

## Open questions

1. Decided (decision 12): a `go_to_step` target may be any step except the failing
   step itself, a `route` step or another calculating step included. A loop of
   calculating steps with no way out is a blocking scan error. Open: how the scan
   finds such a loop.
2. What is the most problem lines one screen shows? A bulk row with many bad
   inputs, or a stop listing 3 rows of 4 problems each, can run long.
3. Provisional: does the named step learn which kind of failure it was (bad
   inputs or a failure during the calculation), or only the sentences and codes?
4. Provisional: when a calculating step sits in a workflow started by a
   `run_workflow` step, does a stop end only that workflow or also the one that
   started it?
5. Provisional: how is a bad-input failure told from a failure during the
   calculation on the wire, since the step kind's result shape is not written?
