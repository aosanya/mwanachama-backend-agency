# Step fields of the calculating, posting and holding steps

Issue ISS-48. This page records what is stored on a step so that the later issues can refuse a bad step, scan it, copy it and show it. It stores and reads the fields only. Nothing here refuses a bad value, scans a step or copies a field into a fork, a publication, the hash or an import.

## The two kinds

| Kind | Value | Source |
|---|---|---|
| calculating | `compute` | [compute-step.md](../2.%20design/compute-step.md), section 1 |
| posting | `post` | [posting-step.md](../2.%20design/posting-step.md), section 1; the name `post` is a proposal there |

Both are in `models.StepKinds()` and in the `kind` values of `agency.blueprint.json`. Neither takes a form, so a form on one is refused by the rule that already refuses a form on a route step. Neither is agency-only: whether a role may run them is open on the design pages.

## The fields

Every field is on `models.WorkItem`, on `models.DocumentWorkItem` and in `agency.blueprint.json`. Each is empty on every other kind. On the document each is left out when empty.

| Field | Type | On a step of kind | Source |
|---|---|---|---|
| `function` | string, written `library.function@N` | `compute` | compute-step.md, section 1 |
| `bindings` | json, a list of bindings | `compute` | compute-step.md, section 1 |
| `result_name` | string | `compute` | compute-step.md, section 1 |
| `on_failure` | json, nullable, `{"choice":"stop"}` or `{"choice":"go_to_step","step_id":"..."}` | `compute` | [compute-step-failure.md](../2.%20design/compute-step-failure.md), Where it lives |
| `amount_step_id` | string | `post` | posting-step.md, section 1 |
| `account` | string | `post` | posting-step.md, section 1 |
| `counter_account` | string | `post` | posting-step.md, section 1 |
| `direction` | string, `debit` or `credit` | `post` | posting-step.md, section 1 |
| `reason` | text | `post` | posting-step.md, section 1 |
| `hold` | json, nullable, `{"expires_after_days":14}` | `compute` | [held-figures.md](../2.%20design/held-figures.md), section 3 |
| `held_step_id` | string | `yes_no` or `post` | held-figures.md, section 3 |

A binding is `input`, `source_type` and the fields its source needs:

| `source_type` | Fields beside `input` |
|---|---|
| `answer` | `step_id`, `question` |
| `record_field` | `record_id`, `question` |
| `ledger_balance` | none |
| `fixed` | `value` |

`action` stays derived and is never stored. It is empty for the two new kinds until the issue that derives it is built.

## Where the code is

| What | Where |
|---|---|
| the kinds and the shapes of a binding, a failure choice and a hold | `models/workitemstep.go`, in one block |
| the fields on a step | `models/workitem.go` |
| the fields on the document | `models/agencydocument.go` |
| the columns | `agency.blueprint.json`, after `start_workflow_ids` |
| the round trip tests | `step_fields_test.go` |
| the vocabulary check that the kinds agree with the blueprint | `blueprint_test.go` |

`on_failure` and `hold` are declared nullable in the blueprint. The store refuses a pointer field that the blueprint does not say may be absent, and a step with no choice and no hold must read back as having none.

## Check against the four design tables

Every stored name and meaning was compared with compute-step.md section 1, the failure page's Where it lives, posting-step.md section 1 and held-figures.md section 3. Every field name, every `source_type` value, the two failure choices and the `hold` shape match. Three points are not the pages' words and are listed here as open:

1. `direction` is stored as plain text. The page says `debit` or `credit`. Refusing anything else is the issue that refuses a bad posting step, so the blueprint does not declare an enum here.
2. `reason` is declared as text and the other posting fields as strings. The pages do not say; text was chosen because a reason is a sentence.
3. `function` stores the version as part of the text. A version is required by the page, and refusing a missing one is the issue that refuses a bad calculating step.

## What is open

- The name `post`, and the names `hold` and `held_step_id`, are proposals. A rename is one edit in each of: the kind constant and the `kind` values, and the field's name in `models/workitem.go`, `models/agencydocument.go` and `agency.blueprint.json`, plus `step_fields_test.go`.
- Whether `post` belongs to the accounting module's `step_kinds` (posting-step.md, section 10) is not decided. It is not in a module's catalog yet, so a module is not switched on by a posting step.
- Refusing a bad step, flagging it in the Issue scan, and carrying the fields through a fork, a publication, the hash and an import are separate issues, listed on ISS-48's card.

## Verification

`go vet ./...` is clean. `go test .` passes, including the six step field tests. In `routes/`, the red test `TestAG74_OpenHole_AttachEditDetachRewritesAnAgencyWideValue` fails; it was filed on purpose before this change and is about an agency-wide value.
