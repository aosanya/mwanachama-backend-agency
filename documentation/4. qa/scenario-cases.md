# Scenario cases for calculating and posting steps (ISS-15)

These are cases for tests to be written, not tests that exist. They show what a scenario must do when its workflow has a calculating step or a posting step. The owner reviews this page.

The design pages are [test-scenarios.md](../2.%20design/test-scenarios.md), [compute-step.md](../2.%20design/compute-step.md), [compute-step-failure.md](../2.%20design/compute-step-failure.md) and [posting-step.md](../2.%20design/posting-step.md). The first says how a scenario is played. The other three say what the steps are.

Rows use the six-column table of the compute repo's `example-cases.md`: `| name | formula | inputs | expected | rule | page |`. Provisional: the loader does not read this file yet, and the folder its case files live in is not decided. For a posting row, `formula` is the calculating step's formula that feeds the posting and `expected` is the amount posted. The ledger checks are in the Then lines of the scenario.

## 1. The runner rule

Today a scenario is played by an AI Test agent, the blocks are the steps of the workflow, a Then line is stored on the scenario, and a draft cannot publish until every scenario passed. These rules add to that.

1. A calculating step in a scenario runs the real published function, at the version the step pins, on the test copy, with the scenario's inputs. Provisional: the step kind name `compute` and the pin `function@N` are proposals.
2. The real result, not a guess, is what the next step receives. If the cap cuts 600.00 to 500.00, the next step gets 500.00.
3. A Then line on a calculating step names one exact typed result, and the runner compares it exactly, text and scale included: 50.00 is not 50 and not 50.0. Provisional: the reading of a Then line on a calculating step is proposed here, and the exact-text reading is the compute repo's open question 1.
4. The AI Test agent never computes and never judges a figure. It judges only what a person would say, such as a message or a sentence on a screen. It may be told a figure is plausible, and the runner still decides.
5. A scenario can supply bad inputs on purpose and expects the step's failure choice: a stop with the listed sentences, or the named step. Provisional: `on_failure` and the codes of input rules are proposals.
6. A posting step posts to the test copy's ledger, the `_test` tables, never to the live ledger. Provisional: the post step kind name `post` and the `_test` ledger tables are proposals.
7. A balance is the sum of the account's entries: debits add and credits subtract.
8. The scan reports a calculating or posting step that no scenario passes through, as a warning, so a publish shows what was never exercised.
9. A function's own examples run as part of publishing, separate from the scenarios.

## 2. The invented function and the shared setup

Lakeside Savings Circle has the workflow "Loan payment" with two steps.

| Step | Kind | Fields |
|---|---|---|
| "Work out the payment" | compute | function `loan.cap@3`, inputs `payment` (dec, the answer Wanjiru gives) and `balance` (dec, read from the ledger), formula `min([payment, balance])`, `result_name` `Payment to post` |
| "Post the payment" | post | amount from "Work out the payment", `account` `Member loans`, `counter_account` `Cash`, `direction` `credit`, `reason` `Loan payment` |

Provisional: the name `loan.cap` and its version 3 are invented for this page, and compute-step.md names no such function. Provisional: which ledger `ledger_balance` reads is its open question 6.

Wanjiru Kamau's loan account starts at 500.00 in the test copy: entry E1 debits `Member loans` 500.00 and credits `Cash`. Worked example: a payment of 50 against a balance of 500 gives 50.00, and the balance after is 450.00.

## 3. Scenarios, rule by rule

### a. The real function, and its real result passed on

Scenario "Payment under the balance". Actors: Wanjiru Kamau (member). Fixed inputs: payment 50.00. Steps played: "Work out the payment", "Post the payment".

- Then on "Work out the payment": `Payment to post is 50.00`. The runner compares.
- Then on "Post the payment": `Member loans balance is 450.00`. The runner reads the sum of entries.
- The runner calls `loan.cap@3` with payment 50.00 and balance 500.00, gets `dec:50.00` and hands 50.00 to the posting step. The AI agent judges nothing here.

Scenario "Payment over the balance". Fixed inputs: payment 600.00.

- Then on "Work out the payment": `Payment to post is 500.00`. Then on "Post the payment": `Member loans balance is 0.00`.
- The next step receives 500.00, not 600.00. A runner that passed on 600.00 would leave a balance of -100.00 and fail the Then line.

Scenario "Payment worked out by a library function" runs `fin.pmt@2` for principal 12000.00, monthly_rate 0.01, months 24, the figures of compute-step.md. Then: `Monthly payment is 564.88`. Provisional: that `fin.pmt@2` returns its figure at 2 places is not written in the library.

| name | formula | inputs | expected | rule | page |
|---|---|---|---|---|---|
| loan-cap-partial-payment | min([payment, balance]) | payment=dec:50.00, balance=dec:500.00 | result: dec:50.00 | A payment under the balance is passed on as it is. | documentation/2. design/compute-step.md |
| loan-cap-over-balance | min([payment, balance]) | payment=dec:600.00, balance=dec:500.00 | result: dec:500.00 | A payment over the balance is cut to the balance, and the next step gets the cut figure. | documentation/2. design/compute-step.md |
| loan-cap-equal-to-balance | min([payment, balance]) | payment=dec:500.00, balance=dec:500.00 | result: dec:500.00 | A payment equal to the balance is left as it is. | documentation/2. design/compute-step.md |
| pmt-twenty-four-months | fin.pmt@2(principal, monthly_rate, months) | principal=dec:12000.00, monthly_rate=dec:0.01, months=int:24 | result: dec:564.88 | The runner uses the real figure of a library function, never a guess. | documentation/2. design/compute-step.md |

### b. The comparison is exact

Scenario "The payment is exactly 50.00". Then on "Work out the payment": `Payment to post is 50.00`.

- Wanjiru answers 50.00: the real result is `dec:50.00`, and the line passes.
- Wanjiru answers 50.0: the real result is `dec:50.0`. The line fails with the proposed text "Payment to post is 50.0, expected 50.00." Provisional: that text is proposed.
- Wanjiru answers 50: the real result is `dec:50`, and the line fails the same way.
- A formula that rounds, `round(balance / months, 2)` on 500.00 over 10 months, gives `dec:50.00` and passes. The same formula rounded to 0 places gives `dec:50` and fails.
- The AI agent is told the figure is plausible, and the line still fails. The runner decides, not the agent.
- The scale is part of the answer. Provisional: this is the reading proposed in the compute repo's open question 1.

| name | formula | inputs | expected | rule | page |
|---|---|---|---|---|---|
| exact-scale-two-places | min([payment, balance]) | payment=dec:50.00, balance=dec:500.00 | result: dec:50.00 | A result written with 2 places matches a Then line of 50.00. | documentation/2. design/test-scenarios.md |
| exact-scale-one-place | min([payment, balance]) | payment=dec:50.0, balance=dec:500.00 | result: dec:50.0 | A result written with 1 place does not match a Then line of 50.00. | documentation/2. design/test-scenarios.md |
| exact-scale-no-places | min([payment, balance]) | payment=dec:50, balance=dec:500 | result: dec:50 | A result written with no places does not match a Then line of 50.00. | documentation/2. design/test-scenarios.md |
| exact-scale-from-round-two | round(balance / months, 2) | balance=dec:500.00, months=int:10 | result: dec:50.00 | A result rounded to 2 places has 2 places and matches 50.00. | documentation/2. design/test-scenarios.md |
| exact-scale-from-round-zero | round(balance / months, 0) | balance=dec:500.00, months=int:10 | result: dec:50 | A result rounded to 0 places has none and does not match 50.00. | documentation/2. design/test-scenarios.md |

### c. Bad inputs on purpose

Step "Check the loan payment" takes `balance`, `payment` and `months` and works out the monthly share, `round(balance / months, 2)`. Wanjiru's scenario supplies balance 120.50, payment -5.00 and months empty. Both bad inputs come back at once, in the order the inputs are asked: the payment, then the term. Provisional: the codes `payment-below-minimum` and `months-missing` are invented, because the input rules of the formula language are not written. A row leaves an empty input out of its `inputs`.

Scenario "Bad payment, the step stops". Choice `stop`.

- Then on "Check the loan payment" (person words, the only lines the AI agent judges): the person sees exactly two sentences, "Payment must not be negative; received -5.00." and "Term in months is missing." The step stays open and nothing from it is saved.
- The runner compares the two codes exactly, in order, and checks that the run did not go on to another step.

Scenario "Bad payment, the case goes to review". Choice `go_to_step` "Review the loan payment".

- Then on "Review the loan payment" (person words): the reviewer reads the same two sentences. The runner checks that the step received exactly the codes `payment-below-minimum` and `months-missing`, one per sentence.

Scenario "Term of 0 months". Balance 120.50, payment 5.00, months 0. A failure during the calculation gives one problem, `divide-by-zero`, and one sentence, "The monthly share could not be worked out because the term is 0 months." (person words).

- A failed calculation spends none of the run's failure budget. A run that starts with a budget of 5 ends with 5, and the runner compares it. A step that times out would spend 1.
- With the term set to 12 the step works out `dec:10.04` and nothing fails.

| name | formula | inputs | expected | rule | page |
|---|---|---|---|---|---|
| check-payment-two-bad-inputs | round(balance / months, 2) | balance=dec:120.50, payment=dec:-5.00 | problems: payment-below-minimum, months-missing | Provisional: all bad inputs come back at once, in the order asked. | documentation/2. design/compute-step-failure.md |
| check-payment-term-zero | round(balance / months, 2) | balance=dec:120.50, payment=dec:5.00, months=int:0 | problems: divide-by-zero | A term of 0 months is one problem found during the calculation. | documentation/2. design/compute-step-failure.md |
| check-payment-good-inputs | round(balance / months, 2) | balance=dec:120.50, payment=dec:5.00, months=int:12 | result: dec:10.04 | Good inputs give the monthly share rounded to 2 places. | documentation/2. design/compute-step-failure.md |

### d. Posting to the test copy's ledger

The ledger in each scenario below is the test copy's. Provisional: how a scenario starts the test copy with E1 is not written.

Scenario "One payment is posted". Payment 50.00.

- Then on "Post the payment": `Member loans balance is 450.00`, the sum of E1 (+500.00) and E2 (-50.00). The live ledger holds no new entry.

Scenario "A payment is reversed". After E2 the runner asks for a reversal E3 of E2. Provisional: how a person asks for a correction is open in posting-step.md.

- Then: `Member loans balance is 500.00`, and `E2 still shows 50.00`. E2 is not edited out and E3 repeats 50.00 with the two accounts swapped.

Scenario "The same step runs twice". The runner runs "Post the payment" twice with the same send key. Provisional: the send key shape (`post-7:run-12:wanjiru`) is a proposal.

- Then: `Member loans balance is 450.00` after both runs. The second run returns the first entry, E2, and writes no E3.

Scenario "Two real payments". Two payments of 50.00 with different send keys. The second works out against the balance 450.00.

- Then: `Member loans balance is 400.00`, from 500.00 to 450.00 to 400.00. Two entries were posted.

| name | formula | inputs | expected | rule | page |
|---|---|---|---|---|---|
| ledger-one-payment-posts | min([payment, balance]) | payment=dec:50.00, balance=dec:500.00 | result: dec:50.00 | A posted 50.00 leaves 450.00, the sum of the entries. | documentation/2. design/posting-step.md |
| ledger-reversal-keeps-original | min([payment, balance]) | payment=dec:50.00, balance=dec:500.00 | result: dec:50.00 | A reversal of E2 returns 500.00, and E2 still shows 50.00. | documentation/2. design/posting-step.md |
| ledger-same-key-posts-once | min([payment, balance]) | payment=dec:50.00, balance=dec:500.00 | result: dec:50.00 | Provisional: a second run with the same send key posts nothing. | documentation/2. design/posting-step.md |
| ledger-two-payments-first | min([payment, balance]) | payment=dec:50.00, balance=dec:500.00 | result: dec:50.00 | The first of two real payments takes 500.00 to 450.00. | documentation/2. design/posting-step.md |
| ledger-two-payments-second | min([payment, balance]) | payment=dec:50.00, balance=dec:450.00 | result: dec:50.00 | The second of two real payments takes 450.00 to 400.00. | documentation/2. design/posting-step.md |

### e. The left-over cent of a split

Decided by the owner: the left-over cent of a split is not paid out. It stays in the account the payout came from. posting-step.md does not carry this yet.

Lakeside's workflow "Fund payout" has "Work out each share" (`fund.split@1`, `round(amount / people, 2)`) and "Post each share" (account `Member savings`, counter_account `Fund`, direction `debit`). A bulk run posts each row on its own. Provisional: `fund.split@1`, the account names and the workflow are invented for this page.

Scenario "100.00 shared among 3". The Fund starts at 100.00. Actors: Wanjiru Kamau, Baraka Otieno and Zawadi Njeri.

- Then on "Work out each share": `Each share is 33.33`. The three shares total 99.99.
- Then on "Post each share": each `Member savings` account is up 33.33, the Fund is down 99.99 and `the Fund keeps 0.01`. The runner checks that no entry of 0.01 exists and no share was rounded up.
- The runner checks the entries sum to zero across accounts: +33.33 three times and -99.99 once.

Scenario "200.00 shared among 7". The Fund starts at 200.00.

- Then: `Each share is 28.57`. The seven shares total 199.99. The Fund is down 199.99 and keeps 0.01.

Scenario "5 cents shared among 2" shows the hazard in open question 6 and is not a decided case. The row gives 0.03, so two shares total 0.06 of 0.05.

| name | formula | inputs | expected | rule | page |
|---|---|---|---|---|---|
| split-hundred-among-three | round(amount / people, 2) | amount=dec:100.00, people=int:3 | result: dec:33.33 | Three shares of 33.33 total 99.99, and 0.01 stays in the Fund. | documentation/2. design/posting-step.md |
| split-two-hundred-among-seven | round(amount / people, 2) | amount=dec:200.00, people=int:7 | result: dec:28.57 | Seven shares of 28.57 total 199.99, and 0.01 stays in the Fund. | documentation/2. design/posting-step.md |
| split-five-cents-among-two | round(amount / people, 2) | amount=dec:0.05, people=int:2 | result: dec:0.03 | Provisional: rounding half away from zero overpays here, see open question 6. | documentation/2. design/posting-step.md |

### f. The scan warns about a step no scenario passes through

Warnings exist today (the bowtie scan raises them, with sentences such as `Bowtie <code> "<title>" has no owner`). The scan blocks a publish only on `error`, and a warning never does. This scan reads no formula, so it has no row in the case table.

| Setup | Warnings | Publish |
|---|---|---|
| "Loan payment" has no scenario through either step | 2: one for "Work out the payment", one for "Post the payment" | Allowed |
| One scenario plays only "Work out the payment" | 1: for "Post the payment" | Allowed |
| Scenarios play both steps | 0 | Allowed |

Provisional: the proposed sentences are `Work item WI-3 "Work out the payment" is not passed through by any scenario` and `Work item WI-4 "Post the payment" is not passed through by any scenario`, with severity `warning` and entity type `work_item`. Provisional: that the publish result lists the warnings is proposed, because today the publish returns only the errors.

### g. A function's own examples run at publish

`loan.cap@3` carries two examples of its own: 50.00 against 500.00 gives 50.00, and 600.00 against 500.00 gives 500.00. At publish they run on the library side, not through any scenario.

- A scenario that passes does not rescue a failing example. A failing example does not fail a scenario.
- Provisional: a failing example blocks the publish as an error naming the function and the example, see open question 5.

| name | formula | inputs | expected | rule | page |
|---|---|---|---|---|---|
| loan-cap-example-partial | min([payment, balance]) | payment=dec:50.00, balance=dec:500.00 | result: dec:50.00 | Provisional: a function's own example runs at publish, outside any scenario. | documentation/2. design/compute-step.md |
| loan-cap-example-over | min([payment, balance]) | payment=dec:600.00, balance=dec:500.00 | result: dec:500.00 | Provisional: the second example of the function runs at publish too. | documentation/2. design/compute-step.md |

## 4. Open questions

Each is the owner's and not decided. The answer after it is a proposal.

1. Does a Then line compare a decimal by exact text or by value? Proposed: exact text, so 50 and 50.0 do not match 50.00, because for money the scale is part of the answer.
2. May a Then line name a result by tolerance, such as 50.00 within 0.01? Proposed: never, exact only.
3. What does the scenario show when the function version the step pins is not in the library? Proposed: the run stops with the reason "Function loan.cap@3 is not in the library."
4. What does "passes through" mean for the scan? Proposed: the step was played in the last run of a scenario, at the draft's current version.
5. Does a failing function example block the publish? Proposed: yes, as an error naming the function and the example.
6. How is each share of a split rounded? `round` is half away from zero, so a share can round up and overpay: 0.05 among 2 gives 0.03 each, which is 0.06. Proposed: round each share down to the cent, so the left-over cent stays. The language has no round-down built-in today. The figures in e are the same under both.
7. How does a scenario start the test copy ledger with E1, and how does it ask for a reversal? Proposed: the scenario lists its starting entries, and the runner makes the reversal by a ledger call.
8. Does the publish result list the warnings? Proposed: yes, as a `warnings` list beside the version.
9. Does a Then line on a stop name the sentences or the codes? Proposed: the runner compares the codes exactly, and the AI agent judges the sentences.
