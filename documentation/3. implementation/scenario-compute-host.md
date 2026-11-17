# What wakala-api and the test runner do for a scenario that computes or posts (ISS-15)

The agency stores scenarios and runs the scan that gates a publish. wakala-api's executor plays the run. This page lists what each must do when the workflow has a calculating step or a posting step. The cases are in [scenario-cases.md](../4.%20qa/scenario-cases.md). The steps are in [compute-step.md](../2.%20design/compute-step.md), [compute-step-failure.md](../2.%20design/compute-step-failure.md) and [posting-step.md](../2.%20design/posting-step.md). Today the Test agent plays a scenario through `ScenarioRunner` and `WithScenarioRunner`, as [test-scenarios.md](../2.%20design/test-scenarios.md) says. The executor is in wakala-api (`internal/testruns`, `internal/agencies`).

Provisional: the kind names `compute` and `post`, the pin `loan.cap@3`, `on_failure`, the `_test` ledger and the warning sentences are proposals from the design pages. Every sentence below that leans on one starts with that word.

Example throughout: Lakeside Savings Circle, the workflow "Loan payment", the member Wanjiru Kamau, who owes 500.00 and pays 50.00.

## When a scenario reaches a calculating step

1. Read the step as the run's draft has it: `function` pinned `loan.cap@3`, the bindings, `result_name` `Payment to post` and the failure choice. Provisional: the step fields and `on_failure` are not stored yet.
2. Resolve every binding from the test copy:
   - `answer`: the scenario's fixed input, or what the Test agent's form fill gave.
   - `record_field`: the test copy's record rows, the `_test` tables with `run_id`.
   - `ledger_balance`: the sum of the test copy's ledger entries for the account.
   - `fixed`: the stored value.
3. Call the library function at exactly the pinned version. It is the same code the production host calls. Never a stub, never a copy of the formula, never the AI.
4. Hand the real typed result, scale kept, to the next step as its input. Write it into the run's step: `sent`, `answer` and `trail`, as test-scenarios.md says. Payment 50.00 against balance 500.00 gives `dec:50.00`. Payment 600.00 gives `dec:500.00`, and the next step gets 500.00.
5. Compare each Then line on the step to the real result exactly: typed value, text and scale, no tolerance. 50.00 is not 50 and not 50.0. A mismatch fails the scenario with the line, the expected value and the real value. Provisional: the exact-text reading is scenario-cases.md open question 1.
6. Show the Test agent the real result, read-only. It never computes and never judges a figure. It judges only Then lines written in a person's words, such as a message or a sentence on a screen.
7. Bad inputs: collect the whole list of problems at once, then follow the step's failure choice.
   - `stop`: the person gets the listed sentences, the step stays open and nothing is saved.
   - the named step: it receives the sentences and the codes.
   - A scenario that expects this passes when the run does exactly that.
   - A failed calculation spends none of the failure budget: 5 before, 5 after. A timeout spends 1.
   - Provisional: the codes and the sentences are the compute repo's input rules, which are not written.
8. A confirming role on the step: the runner plays the role's actor, and the actor confirms. Provisional: whether a Then line may name a held-back figure is not decided.

## When a scenario reaches a posting step

1. Write to the test copy's ledger only, never the live one.
2. Read the amount from the calculating step's real result. Never from the Test agent, never typed.
3. Build the send key from the step, the run and the row. Provisional: the shape is `post-7:run-12:wanjiru`. A repeat posts once and returns the first entry.
4. Write the entry double-entry, with the amount never signed. For a reversal the posting writes the mirror itself: the two accounts swapped, the same amount.
5. Write the provenance (function, version, input fingerprint) on the test entry too. Provisional: the accounting entry does not carry them yet (audit gap 3).
6. A balance a Then line reads is the sum of the account's entries in the test copy, read on the spot: debits add, credits subtract. 500.00 less 50.00 reads 450.00.
7. At the run's end the test copy is dropped or marked as the other `_test` tables are. This page does not decide how. See test-scenarios.md and the hub's test-runner.md.

## A split and its left-over cent

Decided by the owner: the left-over cent of a split is not paid out. It stays in the account the payout came from.

- The host posts each share and posts nothing for the left-over.
- The runner checks that the entries sum to zero.
- 100.00 among 3: 33.33 each, 99.99 posted, 0.01 stays in the Fund.
- How a share is rounded is not decided: scenario-cases.md open question 6. 0.05 among 2 gives 0.03 each under round half away from zero, which is 0.06.

## The warning for a step no scenario passes through

This is the agency's part. The rule goes in `scanIssues` (`issue_impl.go`), as one more check in the list, next to `checkScenarios` (`scenario_sendback_impl.go`) and `checkBowties`. It adds a finding the way the bowtie scan does, with `s.add(models.IssueSeverityWarning, "work_item", step.ID, ...)`. It raises one warning per `compute` or `post` step of the draft that is not passed through.

Provisional: "passes through" means the step is among the played steps of the last run of some scenario of the draft at its current version. A scenario at `not_run` has exercised nothing (scenario-cases.md open question 4).

| Setup | Warnings |
|---|---|
| "Loan payment" has no scenario | 2: "Work out the payment" and "Post the payment" |
| One scenario plays only "Work out the payment" | 1: "Post the payment" |
| Scenarios play both | 0 |

Provisional: the sentences are `Work item WI-3 "Work out the payment" is not passed through by any scenario` and `Work item WI-4 "Post the payment" is not passed through by any scenario`.

- It is a warning and never blocks a publish. The gate keys on severity `error` (test-scenarios.md, "The publish gate"; `validationErrors` keeps only errors).
- Warnings already exist: the bowtie scan raises them, and `ListIssues` returns them with the errors.
- The warning does not make a scenario fail.
- The publish shows the list. Provisional: the publish result carries a `warnings` list (scenario-cases.md open question 8). Today `PublishAgency` returns only the publication, and a refusal names only errors.

## A function's own examples at publish

They run as part of publishing, on the library side, apart from scenarios. A passing scenario does not stand in for a failing example, and a passing example does not stand in for a scenario. `loan.cap@3`'s two examples are 50.00 against 500.00 gives 50.00, and 600.00 against 500.00 gives 500.00. Provisional: whether a failing example blocks the publish is scenario-cases.md open question 5.

## What the host must not do

- Use a stub or a fake result for a calculating step.
- Compare a value by tolerance.
- Recompute at a newer function version than the step pins.
- Post to the live ledger from a run.
- Let the AI work out a figure.
- Pass a scenario whose calculating step was skipped.
- Make the warning an error.

## What is open

All are the owner's. Numbers are those of scenario-cases.md, section 4.

1. Exact text or value for a Then line on a decimal: question 1. Tolerance: question 2.
2. A pinned function version missing from the library: question 3.
3. What "passes through" means for the scan: question 4.
4. Whether a failing function example blocks a publish: question 5.
5. How a share of a split is rounded: question 6.
6. How a scenario starts the test copy ledger with E1, and how it asks for a reversal: question 7.
7. Whether the publish result lists the warnings: question 8.
8. Whether a Then line on a stop names the sentences or the codes: question 9.
