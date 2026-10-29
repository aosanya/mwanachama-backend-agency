# A waiting step has a time limit (HS304)

Hub Steward decision 99 (2026-10-29), built on decision 87 (a cancel is a
workflow that a step of another workflow runs). A step that waits on a person
can carry a time limit set in the workflow editor. When it passes with no
answer the run closes and the step's cancel workflow runs; that workflow's
notify step tells the person who was waiting. This repo holds what the
designer sets. The clock is in the task manager's
`step-time-limit.md`, which closes the run and opens the cancel workflow.

## Where it lives

Two more stored fields on the work item, in `agency.blueprint.json`, carried
by `models.WorkItem` and `models.DocumentWorkItem`:

| Field | Meaning |
|---|---|
| `time_limit_minutes` | Minutes the person has to answer. Zero is no limit: the step waits as long as it takes. There is no default. |
| `cancel_workflow_id` | The workflow, in the same draft, that runs when the limit passes. Required with a limit, empty without one. |

## What create and update refuse

All with `ErrInvalidWorkItem` (400) unless noted:

- a negative limit;
- a limit with no cancel workflow, or a cancel workflow with no limit;
- a limit on a step that does not wait on a person. Only a `fill_form` or
  `yes_no` step that a role acts on does (`WorkItem.WaitsOnPerson`): a step
  the agency takes, a `send_form`, `notify`, `route` or `run_workflow` step,
  and a step with no kind are refused;
- a step that cancels with the workflow that owns it;
- a cancel workflow that is not in the step's draft: `ErrWorkflowNotFound`
  (404).

## Everything that copies a step

- A draft fork remaps `cancel_workflow_id` onto the workflow it minted.
- A publication document carries both fields, and forking a draft from a
  publication restores them, remapping the workflow.
- The content hash gains a `WorkItemLimit` record only for a step that has a
  limit, so a publication made before this change hashes as it did, and a
  limit that is added, changed or removed is a change to publish
  (`ErrNoChangesDetected` is not returned for it).
- The Issue scan raises an `error` for a step whose cancel workflow no longer
  exists in the draft. Deleting the workflow does not cascade into the step.
- A test scenario that mirrors a workflow is sent back when a step it reaches
  changes, and the cancel workflow of a step counts as reached, the same way
  a workflow a step runs does.

## Reads

`agency_list_work_items` and `agency_get_work_item` return both fields on every
step, so the workflow editor shows the limit and the screens that show a limit
and a Closed state read it from the published step. The Closed state itself is
the run's status in the task manager.

## Not here

- Nothing here counts time or opens a run. The service that mounts both
  modules reads the published step's limit when a run reaches the step and
  hands it to the task manager's `BeginWorkflowRunStep`.
- The console field in the workflow editor is a console change, not this repo's.
- A test run does not wait: decision 101 has the Test agent answer for the
  person at once, and a limit is tested by a scenario that says the person did
  not answer (HS306).
