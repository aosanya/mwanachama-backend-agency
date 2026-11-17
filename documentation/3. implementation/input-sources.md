# Input sources of a calculating step

Issue ISS-69. A builder setting up a calculating step is offered, for each input, the sources that fit it. The design is [compute-step.md](../2.%20design/compute-step.md), sections 2 and 8; the cases are [compute-step-cases.md](../4.%20qa/compute-step-cases.md).

## The operation

`GET /input-sources?draft_id=&work_item_id=&input_type=`, action `agency.input_source.list`, tool `agency_list_input_sources`. It is declared in `agency.operations.json`; the route and the tool come from that file. The implementation is `ListInputSources` in `input_source_impl.go`.

| Argument | Meaning |
|---|---|
| `draft_id` | The draft whose steps and records are read. |
| `work_item_id` | The calculating step the sources are for. A step that is not in that draft is `ErrWorkItemNotFound`. |
| `input_type` | Optional. `number`, `currency`, `date`, `time`, `yes_no` or `text`. Anything else is `ErrInvalidInputType`. |

## The answer

Four lists, always present, each entry carrying its `type`:

| List | Entries | Fields beside `source` and `type` |
|---|---|---|
| `answer` | Each question of each earlier step of the same workflow, in step order. | `step_id`, `step_title`, `question`, `answer_type` |
| `record_field` | Each question of each record of the draft. | `record_id`, `record_name`, `question`, `answer_type` |
| `ledger_balance` | The one balance, type `currency`. | none |
| `fixed` | One entry, a value the builder types in. | none |

An earlier step is one of the same workflow with a lower position than the step asked about. The type of a question comes from the table in compute-step.md section 2: `number`, `currency`, `date`, `time` and `yes_no` keep their name, `free_text` and `single_choice` are `text`, and `multi_select`, `photo` and `file` are never offered.

With `input_type`, a list holds only the entries of that type. The example of section 8 is the first test: `input_type` `currency` gives the question Amount requested of the step Apply for a loan and the ledger balance, and not the number question Months of the record Loan terms.

## Tests

| File | What it shows |
|---|---|
| `input_source_test.go` | The section 8 example, the unfiltered list grouped by the four types, a number input, the first step with no earlier answers, and an unknown step or type refused. |
| `routes/input_sources_route_test.go` | The route is declared and served. |

## The work item tools

`agency_create_work_item`, `agency_update_work_item`, `agency_get_work_item` and `agency_list_work_items` name the step fields in their descriptions: `function`, `bindings`, `result_name`, `on_failure`, `hold`, `amount_step_id`, `account`, `counter_account`, `direction`, `reason` and `held_step_id`. No route was added; the fields travel in the same body the chat fields do. The fields themselves are in [step-fields.md](step-fields.md).

## What is open

Not decided here, and left as the owner left them in compute-step.md section 9:

- whether a `currency` source is accepted where a `number` is asked (today `input_type` `number` does not offer a currency question);
- which row a `record_field` reads when a record has several rows;
- which ledger `ledger_balance` reads;
- whether the result of an earlier calculating step is offered as a source.

A reading made while building, not the owner's: `fixed` is always listed, typed as the requested `input_type` (empty when none is given), because a typed-in value fits any input. Say so if it should be left out when a type is asked for.

Listing the functions and their inputs belongs to the compute library, not this repo.
