# Posting step and hold checks

Issue ISS-63. This page records what is refused when a posting step or a hold is saved, what the Issue scan flags, and what stays open. It builds on the step kind registry ([step-kind-registry.md](step-kind-registry.md)) and the stored fields ([step-fields.md](step-fields.md)). Carrying the fields through a fork, a publication, the hash and an import is ISS-66, and the host side is not built here.

Sources: [posting-step.md](../2.%20design/posting-step.md) sections 8 and 9, [held-figures.md](../2.%20design/held-figures.md) sections 7 and 8, [posting-step-cases.md](../4.%20qa/posting-step-cases.md), [held-figures-cases.md](../4.%20qa/held-figures-cases.md), and compute decisions 6, 9 and 10 in [compute-decisions.md](../1.%20requirements/compute-decisions.md).

## Where the code is

| File | What it holds |
|---|---|
| `post_step.go` | the posting step checks, `ErrAccountNotFound`, `AccountDirectory` and `SetAccountDirectory` |
| `hold_step.go` | the hold checks, for the calculating, releasing and every other kind |
| `post_step_test.go`, `hold_step_test.go` | the save refusals and scan findings through the real manager |
| `hold_step_scan_internal_test.go` | the scan findings that only an import can carry in |

The posting step registers its hooks with `RegisterStepKind` for `post`. A rule that must hold on every kind is registered for each kind and for the empty kind, with the same hooks: a posting field on another kind, and a hold or a held step on the wrong kind.

## What is refused when a step is saved

Every refusal is `ErrInvalidWorkItem` (400) unless noted, and the sentence is the one in the test.

| Check | Sentence | Case |
|---|---|---|
| a posting field (amount step, account, counter account, direction, reason) on any other kind, or on a step with no kind | an amount step, an account, a counter account, a direction and a reason belong to a post step, not to a notify step | posting-step.md section 8 |
| a function, bindings or result name on a post step | a post step takes no function, bindings or result name (from ISS-60's check, which runs first) | section 8 |
| a failure choice, a hold or a tool on a post step | the failure choice, the hold and the tool each have their own sentence | section 8 |
| a form, a recipient, routes or an otherwise, a chat field, a workflow to run on a post step | the sentences that already refuse them on every kind that does not take them | section 8 |
| a direction that is neither debit nor credit | the direction of a post step is debit or credit, not "credit side" | P6 |
| a reason that is only spaces | the reason of a post step is written in words, not left blank | section 8 |
| the same account on both sides | a post step debits one account and credits another, so "Member loans" cannot be both sides | P10 |
| an amount step that is the step itself | a post step cannot take its amount from itself | section 8 |
| an amount step of another workflow | the amount step "Work out the review" belongs to another workflow | section 8 |
| an amount step that is not earlier by ordinality | the amount step "Work out the payment" is not earlier than this step | section 8 |
| an amount step that is not a calculating step | the amount step "Apply for a loan" is not a calculating step: only a calculating step works out an amount | P7 |
| hold on a step that is not a calculating step | only a calculating step can hold its result, not a fill_form step | H7a |
| held step on a step that is not yes_no or post | only a yes_no or post step names the held figure it decides, not a notify step | H7b |
| a hold on a calculating step that also names a role | a calculating step waits for a person either through the role that confirms it or through a hold, not both | H7c |
| a hold of 0, a negative number or more than 90 days | a hold expires after a whole number of days from 1 to 90, not 91 | H8a, H8b, H8e |
| a held step that is the step itself | a step cannot release its own figure | H9d |
| a held step of another workflow | the held step "Work out the review" belongs to another workflow | H9e |
| a held step that is not earlier | the held step "Work out the interest" is not earlier than this step | H9c |
| a held step that is not a calculating step | the held step "Apply for a loan" is not a calculating step | H9a |
| a held step with no hold | the calculating step "Work out the interest" has no hold, so it has no held figure to release | H9b |
| a second step naming a held step another step already names | the held figure of that calculating step is already released by step WI-2 "Approve the interest" | H9f |
| a post step whose held step differs from its amount step | a post step that releases a held figure posts that same figure: its held step and its amount step are one step | H9g |

Not found:

| Check | Error | Case |
|---|---|---|
| an amount step that is not a step of the draft | `ErrWorkItemNotFound` (404): the amount step st-404 is not a step of this draft | P9 |
| a held step that is not a step of the draft | `ErrWorkItemNotFound` (404): the held step st-404 is not a step of this draft | H9h |
| an account or counter account the agency does not have | `ErrAccountNotFound` (404): the agency has no account "Member savings" | P8 |

A posting step may be saved with no amount step, account, counter account, direction or reason, and a calculating step may be saved with a hold and no releasing step. Only the scan stops publishing.

## What the Issue scan flags

Every finding is an `error`. Each sentence begins "Work item WI-n "Title"".

| Finding | Ends with | Case |
|---|---|---|
| a post step with no amount step | is a post step with no amount source: it names no calculating step | P5 |
| a post step with no account, counter account, direction or reason | is a post step with no account (or counter account, direction, reason), one finding for each | P5 |
| an amount step that is gone | posts the result of step st-9, which does not exist | section 9 |
| an amount step of another workflow, not earlier, or not a calculating step | posts the result of step WI-1 "Work out the payment", which belongs to another workflow, is not earlier than it, or is not a calculating step | section 9 |
| an account the agency no longer has | posts to account "Cash", which the agency does not have | section 9 |
| a hold that no step releases | holds its result, and no step releases it | H6 |
| a hold outside 1 to 90 days | holds its result for 91 days, and a hold lasts 1 to 90 days | section 8 |
| a hold on another kind | is a notify step with a hold, and only a calculating step can hold its result | section 8 |
| a held step on another kind | is a notify step that names a held figure, and only a yes_no or post step can release one | section 8 |
| a held step that is gone, of another workflow, not earlier, no longer a calculating step, or with no hold | releases the figure of step WI-1 "Work out the interest", which has no hold (and the other four) | section 8 |
| a releasing step with no role | releases a held figure and names no role: the agency cannot approve its own figure | section 8 |
| a releasing step whose role was deleted | the sentence that already flags any step whose role is gone: acted on by Role r1, which does not exist. An offered group is not flagged, as for every step | section 8 |
| a post step that reads a held figure before the step that releases it | posts the held figure of step WI-1 "Work out the interest" before the step that releases it | section 8 |

The checks against the amount step and the held step are run on the whole draft, so a delete or a move that breaks a link is caught on the next scan. None of the deletes cascades.

## Changes outside the two new files

| Change | Why |
|---|---|
| `RegisterStepKind` chains a second registration of the same kind onto the first | it used to replace the first. The calculating step, the failure choice, the copy rules and these checks all register for `compute`, and a kind registered by several files lost all but the last. The chain runs the hooks in registration order, stops at the first refusal and joins the hash records. Test: `step_kind_chain_test.go` |
| a step with no kind runs the shape hooks registered for the empty kind | a hold or a posting field on a step with no kind slipped past every per-kind hook. `validateStepShape` returned before the hooks ran |

## Proposed names

| Name | Where |
|---|---|
| `ErrAccountNotFound` (404, mapped in the operations file, `routes/` and `mcp/`) | `post_step.go`, one declaration |
| the post kind's name `post`, and the names `hold` and `held_step_id` | as on step-fields.md |
| the sentences in the tables | tests check each word for word |

## Cases this repo can check

| Case | Result |
|---|---|
| P5, P6, P7, P8, P9, P10 | checked |
| H6, H7a, H7b, H7c, H8a, H8b, H8e, H9a to H9h | checked |
| H8c, H8d (a hold of 2.5 or "fourteen") | not reachable: `expires_after_days` is a whole number in the stored type, so a request carrying anything else fails to read before a check runs |
| H7d (any other key in a hold) | not reachable: the stored type has no other key, so another key is dropped when the request is read |
| H8f (left out reads as 14) | not checked here: a missing `expires_after_days` and a 0 are the same value in the stored type, so 0 is refused (H8a). Reading "left out" as 14 needs the host or the stored type to tell them apart |
| H10 | host (wakala-api) |
| H11 | ISS-66 |
| P1 to P4 | host and accounting |

## What stays open

- The name `post`, the error name `ErrAccountNotFound`, whether a role may run a posting step, whether `post` is an accounting module step kind, how the case knows the member, where held figures are kept and who else may release one: as the design pages leave them. Nothing here decides one.
- Where the agency's accounts are listed. The check goes through `AccountDirectory`; until `SetAccountDirectory` is called with one, no account is checked, on save or in the scan. The scan treats an error from the directory as "unknown" and flags nothing.
- A post step that names a role and a held step releases the figure it posts (held-figures.md section 3). Without a role the scan flags it like a yes_no step. Whether a post step may carry a role at all is the open question above, so the check is built behind the proposal.
- A post step saved with a held step and no amount step is accepted (the amount may be picked afterwards); once both are set they must name the same step.
- A hold saved with a role named on the step is refused, which follows the proposed answer to open question 8 of held-figures.md.
- The reading of "left out reads as 14", above.
- Copying `amount_step_id`, `held_step_id` and `hold` through a fork, a publication, the hash and an import is ISS-66, and its remap uses the same chained registration.

## Check

`go vet ./...` is clean. The new tests pass with `go test . -run 'TestPostStep|TestHold|TestHoldScan|StepKindRegistry'`. In the whole module, `TestAG74_OpenHole_AttachEditDetachRewritesAnAgencyWideValue` in `routes` fails and was filed on purpose, and `TestLibraryPartTitles_SimultaneousCreatesOfOneTitleStoreOnePart` flickers; neither is touched by this change.
