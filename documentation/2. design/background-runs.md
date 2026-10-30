# Background test runs (HS330)

`RunScenario(id)` no longer waits for the runner. It answers at once with the
run, and the run goes on in the manager's own goroutine on a context the caller
cannot cancel (the caller's values are kept, its cancellation is not).

## States

A run is `waiting`, `running`, `passed`, `failed` or `stopped`. The scenario
takes the same word while its last run is open: `waiting` or `running`, and the
run's end when it finishes. `waiting` is new on both and is declared in
`agency.blueprint.json` and `models/`.

A scenario that is `waiting` or `running` refuses a second press
(`ErrScenarioRunning`).

## Steps as they finish

`RunRequest.Done` holds the steps already stored for the run. A runner that only
implements `ScenarioRunner` is called with `Run` and its steps appear when it
returns. A runner that also implements `StepReporter` is called with
`RunReporting` instead and hands each finished step to `finished`, which writes
it into the run (steps and tokens) and answers whether to carry on. `GetTestRun`
therefore shows the partial steps while the run is `running`.

`RunOutcome.Tokens` counts what this execution used. The run keeps the tokens of
the steps it already held plus those. The final steps are the stored ones, then
the steps the runner reported; a runner that returns its whole list again from
the first step, or ignores `Done`, does not make a stored step repeat.

## Scenarios at once

`WithScenariosAtOnce(func(ctx) int)` gives the manager the hub setting. `0`, a
negative number or no option means no limit. The limit is read at every press
and every time a run ends.

A press past the limit is stored `waiting` with `queued_at`, a time that is
strictly later than the press before it in the same process. A run that ends
starts the earliest `waiting` one. A press also queues behind any run already
waiting, so a late press never overtakes. A waiting run is started on the draft
as it is then: it takes the draft version of that moment.

`started_at` is empty while a run waits.

## Stopping

`StopRun(ctx, runID)` and the operation `agency_stop_test_run` (`test_run_id`):

- a `waiting` run ends at once;
- a `running` run is told to stop and ends after its current step. The answer is
  the run as it stands, still `running`; read it again to see it `stopped`. A
  runner that is not a `StepReporter` has no steps to stop between, so it runs to
  its end and the run is still recorded as stopped;
- the run ends `stopped` with the reason `Stopped by the designer`, keeps its
  steps and tokens, and the scenario goes back to `not_run` (Not run yet), not to
  `stopped`;
- a run that has finished is refused with `ErrTestRunFinished` (409).

## After a restart

`ResumeRuns(ctx)` is called once at start-up, before the first press. It finds
every run left `running` or `waiting`, in `queued_at` order, and starts as many
as the limit allows, each handed its stored steps in `Done` so the runner goes on
at its next step. The rest wait in the same order; a run that was `running` and
no longer fits is put back to `waiting`. A run whose scenario or workflow has
gone is ended `stopped` with the reason.

A stop asked of a `running` run is held in memory only. A restart before the run
reached its next step forgets it.

## Waiting for a run

`AwaitRun(ctx, runID)` returns once the run has ended in this process, and is for
tests and for callers that want the answer. It is not an operation.

## What the console run page reads (HS440)

`agency_get_test_run` (`test_run_id`) and `agency_list_test_runs` carry, beside
the steps, tokens, `draft_version` and times:

- `stored`: the records the run left in the test copy, `[{record, holds, rows}]`.
  The runner hands them back in `RunOutcome.Stored`; the run keeps them when it
  ends, and a run stopped before the runner answered keeps what it held. A run
  that left nothing reads `[]`.
- `token_budget`: the tokens-per-test limit the run was started under, `0` for
  none. `WithTokensPerTest(func(ctx) int)` gives the manager the hub setting, read
  when the run starts (when it leaves `waiting`, for a queued one) and not again,
  so a later change of the setting does not rewrite old runs. The runner is told
  in `RunRequest.TokenBudget`.
- `draft_label`: a name for the draft the run used. Not stored; read each time.
  `Latest edit` while the draft is open, else the tag of the publication whose
  content hash is the run's `draft_version`, else the `draft_version`.
- on each step `waited` (always present): the runner sets it when it acted out a
  wait on that step.
- `running_step`: the id of the step in progress, empty between steps and once the
  run has ended. The runner calls `RunRequest.Starting(stepID)` as it begins a
  step; the manager writes the marker into the run and clears it when the step is
  reported. A runner that never calls `Starting` leaves it empty.

`RunRequest.TokenBudget`, `RunRequest.Starting` and `RunOutcome.Stored` are
additions; a runner that ignores them still compiles and still runs.
