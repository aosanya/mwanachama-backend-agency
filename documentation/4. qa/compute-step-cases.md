# Calculating step cases (ISS-19)

These are cases for tests to be written from [compute-step.md](../2.%20design/compute-step.md), not tests that exist. Each row is one setup and one expected result.

Provisional: the input names `principal`, `monthly_rate` and `months`, their types and the library lookup come from the compute repo, so a row that leans on them is marked Provisional. The failure choice is [compute-step-failure.md](../2.%20design/compute-step-failure.md) and the posting step is [posting-step.md](../2.%20design/posting-step.md); no row here decides either.

"The example step" is Work out the monthly payment in the workflow Loan request of Lakeside Savings Circle: kind `compute`, `function` `fin.pmt@2`, `result_name` `Monthly payment`, no role, and three bindings. `principal` is the answer "Amount requested" of the earlier step "Apply for a loan" (a currency). `monthly_rate` is fixed `0.01`. `months` is the question "Months" of the record "Loan terms" (a number). Every refusal is `ErrInvalidWorkItem` (400) unless the row says otherwise.

## Save refusals

| Id | Setup | Expected |
|---|---|---|
| CS-1 | Create a `fill_form` step with `function` `fin.pmt@2`. | Refused: a compute field on a step of another kind. |
| CS-2 | Create a `notify` step with `result_name` `Monthly payment`. | Refused, same reason. |
| CS-3 | Create a `yes_no` step with one binding. | Refused, same reason. |
| CS-4 | Create the example step with a form. | Refused. |
| CS-5 | Create the example step with a recipient. | Refused. |
| CS-6 | Create the example step with a sent step. | Refused. |
| CS-7 | Create the example step with routes. | Refused. |
| CS-8 | Create the example step with a run workflow. | Refused. |
| CS-9 | Create the example step with chat fields. | Refused. |
| CS-10 | Create the example step with a tool. | Refused. |
| CS-11 | Create a `compute` step with no function. | Refused. |
| CS-12 | Create a `compute` step with `function` `fin.pmt` (no version). | Refused. |
| CS-13 | Create a `compute` step with `function` `fin.pmt@2.5`. | Refused: the version is not a whole number. |
| CS-14 | Create a `compute` step with `function` `fin.pmt@latest`. | Refused: the version is not a whole number. |
| CS-15 | Create a `compute` step with `function` `fin.pmt@2` and no `result_name`. | Refused. |
| CS-16 | The workflow already has a compute step with `result_name` `Monthly payment`; create a second one with `monthly payment`. | Refused as a repeat, ignoring case. |
| CS-17 | Create the example step with two roles in `role_id`, Treasurer and Chair. | Refused: `role_id` is one id. |
| CS-18 | Provisional: save the example step with a binding for `term_days`. | Refused: the function has no such input. |
| CS-19 | Save the example step with two bindings for `principal`. | Refused: two bindings for one input. |
| CS-20 | Save a binding with `source_type` `formula`. | Refused: not one of the four. |
| CS-21 | Bind `principal` to an answer of a step in another workflow. | Refused. |
| CS-22 | Bind `principal` to an answer of the compute step itself. | Refused. |
| CS-23 | Bind `principal` to an answer of a step that comes after the compute step by ordinality. | Refused: not earlier. |
| CS-24 | Bind `principal` to an answer of an earlier step that has no form. | Refused. |
| CS-25 | Bind `principal` to the step "Apply for a loan" with the question "Amount asked", which its form does not ask. | Refused. |
| CS-26 | Bind `months` to the record "Loan terms" with a question its form does not ask. | Refused. |
| CS-27 | Save a `fixed` binding with no value. | Refused. |
| CS-28 | Provisional: save `monthly_rate` as `fixed` with the value `one percent`. | Refused: the value does not fit a number. |
| CS-29 | Provisional: bind `principal`, which asks for a currency, to the free-text question "Reason for the loan". | Refused: the source is text, not a currency. |
| CS-30 | Provisional: bind `principal` to a `photo` question of an earlier step. | Refused: a photo cannot be a source. |
| CS-31 | Update the example step so `function` is `fin.pmt`. | Refused as on create (CS-12); the stored step is unchanged. |
| CS-32 | Save the example step with `role_id` naming a role that is not offered. | `ErrRoleNotFound` (404). |
| CS-33 | Bind `principal` to an answer of a step id that is not in the draft. | `ErrWorkItemNotFound` (404). |
| CS-34 | Bind `months` to a record id that is not in the draft. | `ErrRecordNotFound` (404). |
| CS-35 | Provisional: save the example step with a function or version the library does not have. | `ErrFunctionNotFound` (404), proposed and not decided. |

## Save accepts

| Id | Setup | Expected |
|---|---|---|
| CS-36 | Save the example step with `fin.pmt@2`, `result_name` `Monthly payment` and one binding, for `principal`. | Accepted with 1 of 3 inputs bound. |
| CS-37 | Save the example step as in the design, with all three bindings and `role_id` empty. | Accepted. Reading it back returns `function`, the three bindings and `result_name` unchanged. |
| CS-38 | Save the example step with `role_id` set to Treasurer, then to an offered group. | Both accepted. |
| CS-39 | Read a `fill_form` step. | `function`, `bindings` and `result_name` are empty. |
| CS-40 | Provisional: save the example step and read it back. | `action` is derived, not stored, and reads "Works out the Monthly payment". |

## Scan

Every row below is an `error`, so each expects the publish of that agency to be refused with `ErrPublishValidationFailed` (409), and the same publish to succeed once the one error is fixed. CS-60 is the clean case.

| Id | Setup | Expected |
|---|---|---|
| CS-41 | The example step has 2 of 3 bindings, `months` missing. | One error: an input with no source. The agency cannot be published. |
| CS-42 | The example step has no binding at all. | Three errors, one per input with no source. |
| CS-43 | Provisional: the question "Amount requested" has its answer type changed from currency to number. | One error: the source no longer fits `principal`, which asks for a currency. |
| CS-44 | The question "Amount requested" is deleted from the form of "Apply for a loan". | One error: the question no longer exists. |
| CS-45 | The question "Amount requested" is reworded "Amount you want to borrow". | One error: the binding still names the old wording. Picking the question again clears it. |
| CS-46 | The question "Months" is reworded in the form of the record "Loan terms". | One error, as CS-45, for the `record_field` binding. |
| CS-47 | The step "Apply for a loan" is deleted. | One error. The delete does not cascade, so the binding is still stored. |
| CS-48 | The step "Apply for a loan" is moved to another workflow. | One error: the step now belongs to another workflow. |
| CS-49 | The step "Apply for a loan" is moved after the compute step. | One error: no longer earlier. |
| CS-50 | The record "Loan terms" is deleted. | One error. The delete does not cascade, so the binding is still stored. |
| CS-51 | The example step names the role Treasurer as confirming role, and Treasurer is deleted. | One error: the confirming role no longer exists. |
| CS-52 | The example step names an offered group as confirming role. | No error. |
| CS-53 | A `compute` step has no function and no result name. | Two errors, one for each. |
| CS-54 | A `compute` step has a function and no result name. | One error. |
| CS-55 | Provisional: the example step keeps `fin.pmt@2` and the library no longer has that version. | One error: the function version is not in the library. |
| CS-56 | Two errors are present (CS-41 and CS-45) on one agency. | Both are listed; publishing is refused once, with `ErrPublishValidationFailed` (409). |
| CS-57 | The agency has an error from a compute step and the step is edited to fix it. | The scan lists no error from that step, and the publish succeeds. |
| CS-58 | The agency has no compute step. | The scan raises nothing new; publishing is as before. |
| CS-59 | The example step has `role_id` empty. | No error and no issue about a confirming role. |
| CS-60 | Provisional: the example step has 3 of 3 inputs bound, every source exists, is earlier and fits, `function` and `result_name` are set. | No issue. The agency is published. |

## Fork

| Id | Setup | Expected |
|---|---|---|
| CS-61 | Fork a draft where "Apply for a loan" is copied as `st-31`. | The copied `principal` binding has `step_id` `st-31`. |
| CS-62 | Fork a draft where the record "Loan terms" is copied as `rc-9`. | The copied `months` binding has `record_id` `rc-9`. |
| CS-63 | Fork a draft where the role Treasurer is copied. | The copied step's `role_id` is the copied role. |
| CS-64 | Fork a draft with the example step. | The question wordings, the fixed value `0.01` and `fin.pmt@2` are copied unchanged. |
| CS-65 | Fork a draft from a publication that holds the example step. | The step is restored with `function`, `bindings` and `result_name`, and `step_id`, `record_id` and `role_id` remapped as CS-61 to CS-63. |
| CS-66 | Fork a draft from a publication that holds no compute step. | No compute field appears on any step. |
| CS-67 | Fork a clean draft (CS-60). | The copy has no issue. |
| CS-68 | Fork a draft whose source has the error of CS-45. | The copy has the same one error. |
| CS-69 | Fork a draft whose source has the two errors of CS-56. | The copy has the same two errors. |
| CS-70 | Provisional: fork a draft whose compute step has the failure field of [compute-step-failure.md](../2.%20design/compute-step-failure.md) set. | The field is copied; what it holds is not decided here. |

## Publication and hash

| Id | Setup | Expected |
|---|---|---|
| CS-71 | Publish an agency with the example step. | The document carries `function`, `bindings` and `result_name` on that step. |
| CS-72 | Publish an agency with a `fill_form` step. | The document step carries none of `function`, `bindings` or `result_name`. |
| CS-73 | Lakeside's March publication has no compute step; hash its draft again after this change. | The hash is the same as before. |
| CS-74 | In July add Work out the monthly payment and publish. | The hash differs and the publication is made. |
| CS-75 | Change only `fin.pmt@1` to `fin.pmt@2` and publish. | The hash differs and the publication is made. |
| CS-76 | Change only the fixed `monthly_rate` from `0.01` to `0.015` and publish. | The hash differs and the publication is made. |
| CS-77 | Change only `result_name` to `Loan payment` and publish. | The hash differs and the publication is made. |
| CS-78 | Reorder the three bindings and change nothing else. | The hash is the same and `ErrNoChangesDetected` is returned. |
| CS-79 | Reorder the bindings and also change `fin.pmt@1` to `fin.pmt@2`. | The hash differs and the publication is made. |
| CS-80 | Provisional: publish a compute step with the failure field set. | The document carries the failure field; its content is not decided here. |

## Import

| Id | Setup | Expected |
|---|---|---|
| CS-81 | Import an agency written before this change: 6 steps, none with a `function`. | It imports as 6 steps of their old kinds, nothing is invented, and the scan raises nothing about compute. |
| CS-82 | Import a document with one `compute` step and 1 of 3 bindings. | It imports, and the scan raises 2 errors, one for each input with no source. |
| CS-83 | Import a document with one fully bound `compute` step. | It imports and the scan is clean. |
| CS-84 | Publish the imported agency of CS-82 without fixing it. | `ErrPublishValidationFailed` (409). |

## Input sources list

| Id | Setup | Expected |
|---|---|---|
| CS-85 | Call `agency_list_input_sources` for the example step with `input_type` `currency`. | The answer holds the question "Amount requested" of "Apply for a loan" and the ledger balance. It does not hold the number question "Months" of "Loan terms". |
| CS-86 | Call it with no `input_type`. | The list is grouped by `answer`, `record_field`, `ledger_balance` and `fixed`, each with its type. Answers come only from earlier steps of the same workflow. |

## Waiting on the owner

No case until the owner decides:

- What an overwrite step is, as a kind. A confirmer changes a figure only through an overwrite step the builder wires (decision 3).
- How unusual figures are chosen for the top of a bulk confirmation list.
- Where a held-back member goes next.
- Whether a confirming compute step may carry a time limit.
- Which row a `record_field` reads when the record has several rows.
- Which ledger `ledger_balance` reads when a member has several (Provisional: see [posting-step.md](../2.%20design/posting-step.md)).
- Whether a later compute step may use an earlier step's result as a source.
- Whether a `currency` source is accepted where a number is asked (Provisional: this follows the formula language).
