# What wakala-api does at a calculating step (ISS-19)

The agency stores the step; wakala-api runs it. The step is defined in [compute-step.md](../2.%20design/compute-step.md), and the cases for it are in [compute-step-cases.md](../4.%20qa/compute-step-cases.md). Provisional: the library call, the failure choice, the posting step and where a result is kept are not written, so every sentence below that leans on one contains the word Provisional.

## When a case reaches a step whose kind is `compute`

1. Read the step from the draft behind the version published to the production channel, as for other live reads. Take `function`, `bindings`, `result_name` and `role_id` from it.
2. Resolve each binding, one per input:
   - `answer`: the case's answer to the question of that earlier step, found by the prompt.
   - `record_field`: the record's row. Provisional: which row is open (compute-step.md section 9, question 5).
   - `ledger_balance`: the member's ledger balance. Provisional: which ledger is open (section 9, question 6).
   - `fixed`: the stored value.
3. A binding that cannot be resolved (no answer, a blank answer) goes to the failure choice. Provisional: what that does is in [compute-step-failure.md](../2.%20design/compute-step-failure.md), and this page does not repeat or decide it.
4. Call the library function exactly as pinned, `fin.pmt@2`, never a newer version, even when the library has one. Provisional: the call interface is the compute repo's.
5. `role_id` empty: keep the result under `result_name` on the case and move the case on at once. Provisional: where it is kept is [posting-step.md](../2.%20design/posting-step.md)'s.
6. `role_id` set: put the step in the role's open steps (`agency_list_my_open_steps`) with the figure and the inputs it was worked out from, and wait. The confirmer then does one of three things:
   - accept: the case moves on;
   - hold back: the figure does not go on;
   - send back: it goes to the failure choice (Provisional: see compute-step-failure.md).

## A bulk run

1. Collect the run's figures into one confirmation list per compute step, with the totals first and the unusual figures first. How an unusual figure is chosen is open.
2. Nothing is posted until the confirmer confirms the run once. Provisional: posting is [posting-step.md](../2.%20design/posting-step.md)'s.
3. A held-back member is not posted. Where they go next is open.
4. The others wait only for the confirmation, not for a held-back member.

## What the host must not do

- Add a route for the agency to serve: the step adds no create, update or read route, only `GET /input-sources` for the builder.
- Recompute when the library has a newer version of the function.
- Let anyone change the figure. Whether a confirmer may is open.
- Convert a value silently to make a source fit an input.

## The reviewer

The reviewer that renames steps (`reviewruns/workflows.go` in wakala-api) rewrites the fields it knows. It must keep `function`, `bindings` and `result_name` on a compute step, as [workflow-steps.md](../2.%20design/workflow-steps.md) says for the chat fields, or a review's rename blanks them. Provisional: it must also keep the failure field named in compute-step-failure.md.

## One worked line

Wanjiru Kamau asks for 12,000 on "Apply for a loan". The step Work out the monthly payment reads that answer as `principal`, `0.01` as `monthly_rate` and 24 from the record "Loan terms" as `months`, calls `fin.pmt@2`, and gets 564.88. With no role the case moves on at once with 564.88 kept under `Monthly payment`. With the role Treasurer, the case waits and the Treasurer sees 564.88 and the three inputs.
