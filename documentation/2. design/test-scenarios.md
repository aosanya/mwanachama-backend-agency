# Test scenarios, their actors and inputs, and test runs (HS260)

Hub Steward decisions 82, 84, 85, 87, 89, 95 and 96 (2026-10-29): a scenario is
one path through one workflow, played out by a Test agent, and a draft cannot
be published until every scenario on it has passed. This page is the backend
half; the executor that really plays a run, with its `_test` tables, is HS280.

## The four objects

Declared in `agency.blueprint.json`, named in `agency.agency.json`, carried by
`models/scenario.go`:

| Object | Holds |
|---|---|
| `scenario` | `name`, `summary` (what it tries), `workflow_id` (the workflow it mirrors, in the same draft), `result`, `last_run_id`, `thens` (the Then lines, see below), the draft. |
| `scenario_actor` | `name` (unique in the scenario, ignoring case and outer spaces), `role_id` (an agency role of the same draft), `kind`, `ordinality`. |
| `scenario_input` | `step_id`, `field` (the prompt of a question on that step's form) and `value`. No row means the field is Free. |
| `test_run` | `scenario_id`, `draft_version`, `result`, `stop_reason`, `tokens`, `steps`, `started_at`, `finished_at`. |

A scenario has no code (`SC-n`): nothing quotes one by handle, so it is
identified by its id and its name.

`result` is `not_run` (Not run yet), `running`, `passed`, `failed` or
`stopped`. A run's `result` is `running`, `passed`, `failed` or `stopped`.

There is no block object. A scenario's blocks are the steps of its workflow,
one actor each, and are derived on every read (`scenario.blocks`): the step,
its kind, its action, the actor, and for each question of its form the field
with `free` or the fixed `value`, and the block's `then`.

### Then lines

A block's Then line is what the Test agent judges against the test copy once
that step has been played. It is stored on the scenario as `thens`, a list of
`step_id` and `then`, and read back on every block as `then` (and as `thens`
beside the blocks, in step order, only for steps that have a line). Create and
update take `thens`: an entry sets the line of that step, an empty `then`
clears it, a step with no entry keeps what it has, and omitting `thens` changes
nothing. An entry naming a step outside the scenario's workflow is refused with
`ErrInvalidScenario` and nothing is written. A line whose step has since been
deleted is not returned, and a fork or a publication carries the lines onto the
new draft's steps.

### Who played a run step

Each step of a run's `steps` has `played_by`, filled by the executor with who
played it, such as `Test agent · Fill a form`; `actor` stays the character in
the script. The agency stores and returns it as given.

### Actors

Only actors that hold a role are stored. The agents and the System are always
present and read only, so they are added on every read and never stored:

- the System, id `system`, plays every step nobody takes (the agency's own
  steps: `send_form`, `route`, and any step with no role);
- an agent is any role of the draft that holds the power `ai.use`, which is the
  power a hub's agents are given (`role-powers.md`). Its id is the role's id.

A name may not repeat a stored actor, the System or an agent. A step is played
by the first stored actor (in the order they were added) that holds the step's
role; failing that by the agent holding it; failing that the role's name.
Removing, renaming or adding built-in actors answers
`ErrScenarioActorNotFound`.

### Inputs

`SetScenarioInput` fixes a field of a step of the scenario's workflow. It
refuses a step of another workflow, a step that does not exist and a field the
step's form does not ask, all with `ErrInvalidScenario`. Setting an empty value
frees the field and deletes the row. Deleting a step deletes the inputs fixed
on it.

### Drafting a scenario in one call (HS320)

The Test agent drafts a scenario for an untested rule and the designer approves
it, so the scenario, its people, its fixed answers and its Then lines must be
writable in one tool call before any scenario exists. `agency_create_scenario`
takes `actors`, `inputs` and `thens` with entries that carry only what the
designer would write: an actor is `name` and `role_id`, an input is `step_id`,
`field` and `value`, a Then line is `step_id` and `then`. The `scenario_id` and
`draft_id` of an entry are the store's, set from the scenario being written,
and the tool schemas of create and update do not ask for them (they used to,
which refused every draft that carried an input). The questions to fix an input
on are read beforehand from `agency_list_work_items`, which returns each step's
form questions.

## What each write does

- `CreateScenario` needs an open draft, a name and a workflow of that draft
  (`ErrWorkflowNotFound` otherwise); it may carry `actors` and `inputs`, and
  all of it lands or none of it does. It starts `not_run`.
- `UpdateScenario` changes the name, the summary and the Then lines, and may
  carry `actors` and `inputs` (see below). The result in the body is ignored
  and the workflow cannot change (a scenario mirrors one workflow; write a new
  one). The scenario goes back to `not_run`; its old runs and `last_run_id`
  stay.
- Adding, renaming and removing an actor, and setting an input, each send the
  scenario back to `not_run`.
- `DeleteScenario` removes its actors, inputs and runs with it.
- Every write to a scenario needs its draft to be open (`ErrDraftNotOpen`).

### Updating the whole set in one call (HS410)

One Save in the console is one proposal, so `UpdateScenario` takes the same
`actors` and `inputs` that create takes and applies the whole set or none of it.

- Omitted (or null), `actors` and `inputs` leave what the scenario has. Sent
  as a list, each is the complete new list, so `[]` clears it.
- An actor entry carries `name` and `role_id`, and optionally the `id` of a
  stored actor of this scenario. With an `id` the entry is that actor: a new
  `name` renames it and a new `role_id` changes its role, keeping the row, its
  id and its place in the blocks. Without an `id` the entry is the stored
  actor that has that name now (ignoring case and outer spaces), else a new
  actor. An entry with no `role_id` keeps the role the actor has; a new actor
  needs one. An `id` is claimed before names are matched, so two actors can
  swap names in one call. Stored actors the list leaves out are removed, and
  the position in the list becomes `ordinality`.
- Entries read back from the scenario with `built_in` true (the agents and the
  System) are skipped, so a read-back scenario can be sent unchanged. A
  built-in actor still counts for the unique-name check.
- An input entry is `step_id`, `field` and `value`, checked as
  `SetScenarioInput` checks it. A field of the scenario that the list leaves
  out is freed, an empty `value` frees that field, an unchanged value is not
  rewritten, and the same field twice is refused.
- Everything is checked before the first write: a repeated name
  (`ErrDuplicateScenarioActor`), a nameless actor, a new actor with no role,
  an actor `id` that is not this scenario's (`ErrScenarioActorNotFound`), a
  role outside the draft (`ErrRoleNotFound`), and a step or question that does
  not fit (`ErrInvalidScenario`). A refusal leaves the scenario, its result
  and its runs as they were.
- `AddScenarioActor`, `RenameScenarioActor`, `RemoveScenarioActor` and
  `SetScenarioInput` stay as they are for single changes.

## Sending scenarios back

A change to a rule, a workflow, a step or a role sends back to `not_run` only
the scenarios that use it, whatever result they held. A scenario uses:

- the workflow it mirrors, and every workflow its steps start (a
  `run_workflow` step), followed transitively;
- every step of those workflows, and the role of each step;
- the role of each of its actors;
- every rule attached to any of those.

Written as keys of `kind:code` (`workflow:W-1`, `work_item:WI-4`, `role:R-2`,
`rule:RU-1`), computed by `scenarioUses`. The writes that send back:
`UpdateWorkflow`, `DeleteWorkflow`; `CreateWorkItem`, `UpdateWorkItem` and
`DeleteWorkItem` (the step and the workflow it sits in, and the workflow it
left); `UpdateRole`, `DeleteRole`; `UpdateRule`, `DeleteRule`;
`CreateRuleAttachment` and `DeleteRuleAttachment` (the thing it attaches to,
in every draft, since an attachment names a code and no draft). A rename of a
role sends its scenarios back too: the row asks for a change to a role, not a
change in behaviour, to send them back.

## Deleting a role or a workflow a scenario uses

`DeleteRole` refuses while a stored actor of any scenario holds the role, and
`DeleteWorkflow` refuses while any scenario mirrors the workflow. Both answer
`ErrUsedByScenario` (409), and the message names every scenario that holds it,
by name and in alphabetical order, so the designer knows which to remove or
change first. Nothing is deleted by a refused call: the role keeps its
assignments and the workflow keeps its steps. Once the scenarios are deleted
(or the actor is removed from them) the delete goes through.

The built-in agent actors are not stored, so a role that only an agent actor
stands for, or that only a step names, is not refused; deleting it still sends
back the scenarios whose steps name it. A workflow that a scenario only reaches
through a `run_workflow` step is likewise not refused and sends that scenario
back.

A rename is never refused. A scenario holds the role's and the workflow's ids,
so reading it afterwards shows the new names: `Workflow` on the scenario and
`RoleName` on each actor.

## Running

`RunScenario(id)` writes a `running` test run and marks the scenario
`running`, calls the runner, then writes the outcome to the run and the
scenario. `RunScenarios(draft, ids)` runs the named scenarios, or every one of
the draft when none are named, one after the other, and refuses the whole batch
before running any if a name is not a scenario of that draft.

The runner is an interface, handed the run id, the scenario as the designer
sees it (blocks, actors, inputs), the workflow, the rules it shows and the
draft version (a hash of the draft's content), and answering a `RunOutcome`:

```go
type ScenarioRunner interface {
	Run(ctx context.Context, req RunRequest) (RunOutcome, error)
}
```

`WithScenarioRunner(r)` installs it. With none installed, the default stops
every run with the reason "No test executor is installed" so nothing can be
published by accident; HS280 replaces it. An error from the runner, or an
outcome that is not `passed`, `failed` or `stopped`, becomes a stopped run
with a reason, so a scenario is never left `running`.

A scenario already `running` refuses a second run (`ErrScenarioRunning`). The
run is synchronous here; keeping a run going when the page closes, and how
many run at once, belong to the executor.

The scenario takes the run's result only if it is still `running` on this run.
Editing a scenario, or changing what it uses, while it runs sends it to
`not_run`, and the finished run is kept as it ended but does not pass the
scenario: the pass would be for something that no longer exists.

Runs are listed newest first (`ListTestRuns`) and read with `GetTestRun`.
There is no operation to remove a run yet.

## The publish gate

`scanIssues` raises one error-severity issue per scenario of the draft that is
not `passed`, with entity type `scenario` and the scenario's id. The message
names the scenario and its result ("Scenario "Buy now: the shop says yes" has
not passed on this draft (Not run yet)"), or says its workflow no longer
exists. `PublishAgency` already refuses on any error issue, so it answers
`ErrPublishValidationFailed` naming them. A draft with no scenarios publishes
as before. Scenarios are not part of the content hash.

## Travelling

- A draft fork copies scenarios, actors and inputs onto the new draft's
  workflows, roles and steps. The copy is `not_run`, has no `last_run_id` and
  no runs.
- A publication document carries `scenarios`: name, summary, workflow, actors
  (name, role) and inputs (step, field, value). It never carries a result, a
  run, tokens or a draft version. Forking a draft from a publication and
  importing a document write them back, `not_run`. A scenario naming a workflow
  the document lacks refuses the import with `ErrInvalidScenario`.

## Operations

All under `agency.scenario.*` and `agency.test_run.*` in
`agency.operations.json`; the tools are `agency_create_scenario`,
`agency_list_scenarios`, `agency_get_scenario`, `agency_update_scenario`,
`agency_delete_scenario`, `agency_run_scenario`, `agency_run_scenarios`,
`agency_add_scenario_actor`, `agency_rename_scenario_actor`,
`agency_remove_scenario_actor`, `agency_set_scenario_input`,
`agency_get_test_run` and `agency_list_test_runs`.

## Not here

The SQL mirror in the gateway's active migrations (another repo's), the
executor and the `_test` tables (HS280, decision 85), the token budget (HS281),
refusing the removal of a role or workflow a scenario uses (HS289), and
removing a single run.
