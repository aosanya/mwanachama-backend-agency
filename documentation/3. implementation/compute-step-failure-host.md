# What wakala-api does when a calculating step fails

This repo stores and scans `on_failure`; the host (wakala-api) runs the
calculation and follows the choice. Rules:
[compute-step-failure.md](../2.%20design/compute-step-failure.md). Cases:
[compute-step-failure-cases.md](../4.%20qa/compute-step-failure-cases.md).

Provisional: the calculating step kind
([compute-step.md](../2.%20design/compute-step.md)), its wire name and the
formula language are not written, so each sentence leaning on them starts with
that word. Example step: Check the loan payment, review step Review the loan
payment.

## What it reads, and what it must not do
- From the published document, `on_failure` on the step: `{"choice":"stop"}` or
  `{"choice":"go_to_step","step_id":"<id>"}`. An empty one never reaches the
  host, because the scan blocks publishing.
- No retry, no third path beside the two, nothing saved from a failed step.

## Telling the two failures apart
Provisional: bad inputs (all problems at once, before any calculation) and a
failure during the calculation (one problem) are told apart by the result the
calculating step kind returns, whose shape is not written. Both follow the same
`on_failure`, so the difference only fills the sentences.

## Stop and go to step
Provisional: on stop the person gets one list of sentences in the order the
inputs were asked, such as "Payment must not be negative; received -5.00."; the
step stays open for corrected answers and nothing from it is saved.
Provisional: on go to step the case moves to the named step with the same
sentences, read there as an answer to the failed step, and one code per sentence.

## Bulk rows
Provisional: each failed row follows the choice. With 200 rows and 3 failed, stop
finishes the 197 and lists the 3 with row number and reasons; a named step
receives only the 3 failed rows.

## Failure budget
A failed calculation spends none of the run's failure budget: 5 before, 5 after.
A timeout is a system fault and spends 1: 5 before, 4 after.

## Open
From the design page's
[open questions](../2.%20design/compute-step-failure.md#open-questions): whether a
target may be a `route` or another calculating step; how many problem lines a
screen shows; whether the named step learns the kind of failure; whether a stop
in a `run_workflow` child ends the parent; how the two failures differ on the wire.
