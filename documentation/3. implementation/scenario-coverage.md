# The scenario coverage warning (ISS-70)

The Issue scan tells a builder when a calculating step or a posting step has never been exercised by a test scenario. It is a warning. It never blocks a publish and never makes a scenario fail. The rule comes from [scenario-compute-host.md](scenario-compute-host.md), section The warning for a step no scenario passes through, and from [test-scenarios.md](../2.%20design/test-scenarios.md), where the publish gate keys on severity `error`. The cases are in [scenario-cases.md](../4.%20qa/scenario-cases.md), section f.

Example throughout: the workflow "Loan payment" with the steps "Work out the payment" (kind `compute`) and "Post the payment" (kind `post`).

## What the check does

`checkScenarioCoverage` in `scenario_coverage_impl.go` runs in `scanIssues`, in the list right after `checkScenarios`. It reads the draft being scanned and does this:

1. Collects the steps of the draft whose kind is `compute` or `post` and that are not deleted. With none, it stops.
2. Reads every scenario of the draft and, for each, the run named by its `last_run_id`.
3. Takes the draft's current content hash and keeps only the runs whose `draft_version` equals it. A run of an older version of the draft covers nothing.
4. Marks every step id found in the kept runs' `steps` as played.
5. Adds one finding for each collected step that is not marked: severity `warning`, entity type `work_item`, entity id the step's id.

The sentence is:

`Work item WI-3 "Work out the payment" is not passed through by any scenario`

The same sentence is used for a posting step, with its own code and title. Provisional: the sentence is the one proposed on the design pages.

## What counts as played

| Setup | Warnings |
|---|---|
| "Loan payment" has no scenario | 2: "Work out the payment" and "Post the payment" |
| One scenario whose last run played only "Work out the payment" | 1: "Post the payment" |
| One scenario or two that together played both | 0 |
| A scenario at `not_run` | 0 steps played, so 2 |
| A scenario run twice, the second run playing less | only the second run counts |
| A step renamed after the run | the run is of an older version, so both steps warn again |

A step is played when its id is among the steps of the run, whether the run passed, failed or was stopped. Provisional: this is the page's proposal for what "passes through" means (scenario-cases.md open question 4), and is the only reading built.

Steps of other kinds never warn, and neither do the steps of workflows that hold no `compute` or `post` step.

## Where it is registered

It is one more entry in the list in `scanIssues` (`issue_impl.go`, one added line), next to `checkScenarios`, not a check registered through `RegisterIssueCheck` of the step kind registry ([step-kind-registry.md](step-kind-registry.md)). Two reasons:

- A registered `IssueCheck` receives only the draft's steps and a report. It cannot read scenarios or test runs. The check needs both.
- `ClearStepKindRegistry`, used by the registry's own tests, empties the registered checks for the rest of the test run, so a check registered at start-up would disappear from any test that ran after it.

If a later change gives `IssueCheck` access to the scan, this check can move into the registry without a change of behaviour.

## What it does not do

- It does not block a publish: the gate keeps only findings of severity `error`.
- It does not change a scenario's result.
- It does not run a scenario or read a formula.
- It does not add a `warnings` list to the publish result. Today a refusal names only errors.

## Checked against the scenario cases

Case f of [scenario-cases.md](../4.%20qa/scenario-cases.md) has three rows. Each is a test in `scenario_coverage_test.go`:

| Row | Test |
|---|---|
| No scenario: 2 warnings | `TestScenarioCoverage_NoScenarioWarnsOnTheCalculatingAndThePostingStep` |
| One scenario plays only "Work out the payment": 1 warning | `TestScenarioCoverage_AScenarioPlayingOnlyTheCalculatingStepLeavesTheWarningOnThePostingStep` |
| Both played: 0 warnings | `TestScenarioCoverage_ScenariosThatPlayBothStepsLeaveNoWarning` and `TestScenarioCoverage_TwoScenariosTogetherCoverBothSteps` |

Severity `warning` and entity type `work_item` are in the filter of each test. The tests also cover a scenario at `not_run`, a second run replacing the first, a run of an older version, and other workflows in the same draft.

One column of the case table is not shown by a real publish: Publish "Allowed". The test world's steps carry no fields, and since ISS-60, ISS-61 and ISS-63 landed their checks, a bare `compute` or `post` step is itself an error (no failure choice, no account). A full publish of that world is therefore refused for those reasons. The publish test (`TestScenarioCoverage_TheWarningNeverBlocksAPublishAndNeverFailsAScenario`) proves the part this check owns: no coverage finding has severity `error`, a refusal never names the coverage sentence, and the scenario stays `passed`. It does not prove a publish succeeds. Filling the steps in fully so the publish goes through belongs with the end-to-end cases of wakala-api.

## What is open

All are the owner's, and are copied from scenario-cases.md section 4.

1. What "passes through" means for the scan: question 4. Built as proposed.
2. Whether the publish result lists the warnings: question 8. Not built.
3. Whether a failing function example blocks a publish: question 5. Not touched.

Also open, not asked yet: whether a step that the last run reached but that failed should count as passed through. It counts today.
