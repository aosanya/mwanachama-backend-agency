# Failure choice checks of a calculating step

Issue ISS-61. This page records the checks of the field `on_failure` of a calculating step: what a save refuses, what the Issue scan flags, and how the loop of steps with no way out is found. The rules are in [compute-step-failure.md](../2.%20design/compute-step-failure.md), the cases in [compute-step-failure-cases.md](../4.%20qa/compute-step-failure-cases.md), and the decisions are 5 and 12 in [compute-decisions.md](../1.%20requirements/compute-decisions.md). The field itself is stored by ISS-48, see [step-fields.md](step-fields.md). What the host does when a step fails is wakala-api's, in [compute-step-failure-host.md](compute-step-failure-host.md), and is not built here.

## Where the code is

| What | Where |
|---|---|
| the checks and their registration | `compute_failure.go` |
| the tests, one for each refusal and finding | `compute_failure_test.go` |
| test-build access to the registration | `compute_failure_export_test.go` |

Nothing in the six shared step files was edited. The checks hang on the step kind registry ([step-kind-registry.md](step-kind-registry.md)): an `init` in `compute_failure.go` registers them.

| Check | Registry hook | Registered for |
|---|---|---|
| the choice itself is well formed | `Shape` | the compute kind |
| a choice on a step that is not calculating is refused | `Shape` | every other kind, and a step with no kind |
| the step to go to is a step of this draft, in this workflow | `References` | the compute kind |
| every finding of the Issue scan and the loop | `RegisterIssueCheck`, once per scan | none, it reads the whole list of steps |

Registrations of one kind are chained by the registry, so these hooks run beside the ones another issue registers for the compute kind. The refusal on other kinds is registered by looping over `models.StepKinds()`, so a kind added to that list is covered with no change here.

## What a save refuses

With `ErrInvalidWorkItem` (400). The sentence reaches the caller as written here.

| Case | Cases page | Sentence |
|---|---|---|
| a choice on a step that is not calculating | C10 | `a notify step has no failure choice; only a compute step can say what happens when it cannot work out a result` (`a step with no kind` for a step with no kind) |
| a choice that is neither `stop` nor `go_to_step` | C9 | `the failure choice "retry" is neither stop nor go_to_step` |
| `go_to_step` with no `step_id` | C14 | `go_to_step names the step to go to` |
| `stop` with a `step_id` | C8 | `stop names no step, got st-7` |
| a `step_id` of another workflow | C7 | `the step to go to on failure, st-9, belongs to another workflow` |
| a `step_id` that is the step itself | C3 | `a step cannot go to itself on failure` |

With `ErrWorkItemNotFound` (404): a `step_id` that is not a step of the draft, `the step to go to on failure, st-404, is not a step of this draft` (C2, setup B).

A calculating step with no choice is saved (C15). Any other step of the workflow is a valid target, a route step or another calculating step included (C16, decision 12).

A step cannot name itself on create, because it has no id yet; the self check runs on update.

## What the Issue scan flags

All are `error`, so none can be published, and the publish gate refuses with `ErrPublishValidationFailed` carrying the sentence (C1, C17).

| Finding | Sentence |
|---|---|
| a calculating step with no choice | `Work item WI-12 "Check the loan payment" has no answer for a failure; choose stop or a step to go to` |
| a named step that no longer exists | `Work item WI-12 "Check the loan payment" goes on failure to Step st-9, which does not exist` |
| a named step now in another workflow | `Work item WI-12 "Check the loan payment" goes on failure to Step st-9, which belongs to another workflow` |
| a calculating step that names itself | `Work item WI-12 "Check the loan payment" goes on failure to itself` |
| a loop with no way out | `Work items WI-12 "Check the loan payment", WI-13 "Check the term" fail onto each other with no way out; one of them must stop or go to a step outside the loop` |

The sentences of the first four read like `checkStepTarget`'s.

## The loop

Proposed reading, marked proposed because the design page says how the scan walks the choices is not written, and the owner has not settled it:

- Only calculating steps are walked, and only their `go_to_step` choices.
- A step has one choice, so each calculating step has at most one step it goes to. An edge is drawn only when the target exists, is in the same workflow, is another step and is itself a calculating step.
- A loop is a cycle in those edges. Every member of a cycle goes to another member, so none of them stops or leaves for a step outside it: the cycle has no way out by construction. A step that stops, or goes to a step that is not calculating, has no edge, so it can never be part of a cycle.
- One error is raised for each cycle, however many steps it holds, and it is filed against the member with the lowest id. The members are named in the order they go on to one another, starting at that member.
- A step that only feeds a loop is not in it and is not named.
- A step that names itself is its own finding, and is not also counted as a loop of one.

The cases on the page: C17 (two steps), a loop of three, a loop where one step stops (nothing raised), a loop where one step goes to a step that is not calculating (nothing raised), and a step feeding a loop.

## Tests

`compute_failure_test.go` has 20 tests: every refusal with its sentence, the choices that must be allowed, every finding with its sentence word for word, the planted cases (a target moved to another workflow, a choice that names itself, written by an `update` on the row so the save check is bypassed) and the publish refusals for a missing choice and for a loop. Each test clears the registry and registers these checks again, so it does not depend on what another test left behind.

## Open

Copied from the design page, none decided here:

1. How the scan walks the choices to find a loop. The reading above is a proposal.
2. The most problem lines one screen shows.
3. Whether the named step learns the kind of failure.
4. What a stop does in a workflow started by a `run_workflow` step.
5. How a bad-input failure is told from a failure during the calculation on the wire.

Found while building, not decided:

- The save-time checks on another kind and on a step with no kind run as `Shape` hooks registered for each kind in the list and for the empty kind, because the registry has no hook that runs for every step whatever its kind. A registry hook of that sort would replace the loop; it was not added because the registry belongs to another issue.
- A choice that is neither `stop` nor `go_to_step` and is written past the save check (for example by an import) is not reported by the scan. The design page lists no finding for it.
- Deleting a target step does not cascade, as the page says; the scan finds it.
