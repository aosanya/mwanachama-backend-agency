# Copy rules of the calculating, posting and holding steps

Issue ISS-66. This page records what each field of the three steps does on a draft fork, a publication, the content hash and an import. The fields are stored by [step-fields.md](step-fields.md). The rules are registered with the seam of [step-kind-registry.md](step-kind-registry.md) from an `init` in three files, so none of the six shared step files has a rule of its own for them.

The rules come from the last section of each design page: [compute-step.md](../2.%20design/compute-step.md) section 7, [compute-step-failure.md](../2.%20design/compute-step-failure.md) Everything that copies a step, [posting-step.md](../2.%20design/posting-step.md) section 10 and [held-figures.md](../2.%20design/held-figures.md) section 10.

## Where the code is

| What | Where |
|---|---|
| `function`, `bindings`, `result_name`, `on_failure`, registered on `compute` | `compute_step_copy.go` |
| `amount_step_id`, `account`, `counter_account`, `direction`, `reason`, registered on `post` | `post_step_copy.go` |
| `hold` and `held_step_id`, registered on `compute`, `post` and `yes_no` | `hold_step_copy.go` |
| the tests | `step_copy_test.go` (through the manager) and `step_copy_fixed_hash_test.go` (a fixed document) |
| the test-build entry that registers the rules again after a test clears the registry | `step_copy_export_test.go` |

A test that clears the registry registers the rules again with `RegisterStepCopyRules`. A kind that registers more than one set of hooks has them chained in the order of registration.

## What each field does

| Field | Draft fork | Publication document | Content hash | Import |
|---|---|---|---|---|
| `function` | copied as it is, so the pinned version is kept | carried, left out when empty | in `WorkItemCompute` | read back as it is |
| `bindings` | `step_id` of an `answer` binding is remapped onto the copied step, `record_id` of a `record_field` binding onto the copied record; wording, values and inputs are copied as they are | carried, left out when empty | in `WorkItemCompute`, sorted by input, so reordering alone is no change | the same remap onto the imported steps and records |
| `result_name` | copied as it is | carried | in `WorkItemCompute` | read back |
| `on_failure` | `step_id` of a `go_to_step` choice is remapped; a `stop` has no id and is copied as it is | carried | in `WorkItemCompute` only when set | the same remap |
| `amount_step_id` | remapped onto the copied step | carried | in `WorkItemPost` | remapped |
| `account`, `counter_account` | copied as they are; accounts are the agency's, not a draft's | carried | in `WorkItemPost` | read back as they are |
| `direction`, `reason` | copied as they are | carried | in `WorkItemPost` | read back |
| `hold` | copied as it is; it has no id | carried | in `WorkItemHold` | read back |
| `held_step_id` | remapped onto the copied step | carried | in `WorkItemHold` | remapped |

Held figures belong to a run and are never copied.

## How the remap reaches records

A fork and an import copy the steps before the records. A `record_field` binding names a record, so the record ids are minted before the steps are copied, in `draft_impl.go` (`forkScope`) and in `document.go` (`applyDocumentToDraft`), and the registry's `StepIDMaps` carries them as `Records` beside `Workflows`, `Roles` and `Steps`. This is the only change to a shared file. The record rows are then written with the ids already minted.

## The hash

| Record | Added for a step that has | Fields |
|---|---|---|
| `WorkItemCompute` | a function, a binding, a result name or a failure choice | `id`, `function`, `result_name`, `bindings` sorted by input then source, and `on_failure` when set |
| `WorkItemPost` | any of the five posting fields | `id` and the five fields |
| `WorkItemHold` | a `hold` or a `held_step_id` | `id`, `held_step_id` and `hold` when set |

A step with none of these fields adds nothing, so a publication made before the fields existed hashes as it did. A failure choice alone also adds the compute record, so a change to it can never be missed. The record names `WorkItemPost` and `WorkItemHold` are proposals on the design pages.

A changed function version, binding, result name, failure choice, amount step, account, counter account, direction, reason, hold setting or releasing step changes the hash, so the publication can be made. Reordering the bindings alone gives `ErrNoChangesDetected`.

## What the tests show

| Case | Test |
|---|---|
| a fork gives the copied step its own bindings pointing at the copied step and the copied record, the failure choice at the copied target, a stop unchanged, the releasing and the posting step at the copied calculating step, accounts as they are | `TestStepCopy_ADraftForkRemapsBindingsRecordsFailureAndAmountOntoTheCopies` |
| the document carries the fields, leaves them out of a step of an older kind, and a fork from the tag restores them with the same remap | `TestStepCopy_APublicationCarriesTheFieldsAndAForkFromItRestoresThemWithTheSameRemap` |
| an agency of older kinds, with an empty calculating step and an empty posting step, hashes as before | `TestStepCopy_AnAgencyOfOlderKindsHashesAsItDidBeforeAnyRuleWasRegistered` and, from a fixed document with fixed ids, `TestStepCopyFixedHash_AFixedAgencyOfOlderKindsHashesTheSameWithAndWithoutTheRules` |
| a new function version, a binding, a result name, a failure choice and a hold each change the hash; a reorder does not | `TestStepCopy_ACalculatingStepChangesTheHashWhenItsFieldsChangeAndNotWhenOnlyTheBindingsAreReordered` and `TestStepCopyFixedHash_ACalculatingStepHashesByItsFieldsAndNotByTheOrderOfItsBindings` |
| a reason, an account, a counter account, a direction, an amount step, a held step and a releasing target each change the hash | `TestStepCopy_APostingStepAndAReleasingStepChangeTheHashWhenAnyOfTheirFieldsChange` |
| an agency written before the fields imports as 6 steps of its old kinds with nothing invented | `TestStepCopy_AnAgencyWrittenBeforeTheFieldsImportsAsSixStepsOfItsOldKinds` |
| a calculating step with fewer bindings than inputs, a posting step with no amount source and a hold with no releasing step are accepted, and every id is remapped | `TestStepCopy_AnImportKeepsFewBindingsNoAmountSourceAndAHoldWithNoReleasingStepAndRemapsEveryId`, `TestStepCopy_AHoldWithNoReleasingStepIsAcceptedOnImport` |

The hash proof uses two ways because ids are minted at random by the manager. The first publishes the same rows with and without the rules and reads `ErrNoChangesDetected`. The second writes rows with fixed ids and compares the content hash itself.

## Open

Copied from the design pages, none decided here:

- Whether import checks an `account` against the importing agency's accounts or accepts it and lets the scan flag it (posting-step.md section 10). Import accepts it as it is.
- The names `WorkItemPost` and `WorkItemHold`, and the field names `hold` and `held_step_id`, are proposals.
- Whether `post` belongs to the accounting module's `step_kinds` (posting-step.md section 10).

Left open by this issue:

- `held_step_id` is registered on `yes_no` and `post`, as held-figures.md section 3 puts it. If a releasing kind is added, its registration is one more entry in `registerHoldStepCopyRules`.
- Whether a failure choice with no function, binding or result name should add the compute record is this page's choice (it does), because a published change to it must not read as no change.
- The scan findings for a bad binding, a missing amount source or a hold with no releasing step are the other issues of the step kind registry, so import accepts these and nothing here flags them.
