# Calculating step checks

Issue ISS-60. This page records what a calculating step is refused for when it is saved, and what the Issue scan flags in one that is already stored. The rules are those of [compute-step.md](../2.%20design/compute-step.md) sections 2 to 4, and the cases are [compute-step-cases.md](../4.%20qa/compute-step-cases.md). The failure choice is [compute-step-failure.md](../2.%20design/compute-step-failure.md) and is not checked here; the posting and holding steps are not checked here either.

## Where the code is

| What | Where |
|---|---|
| the checks, the function catalog and the scan | `compute_step.go` |
| the tests | `compute_step_test.go` |
| a test-only way to store a step without checks, and to register the checks again after a test cleared the registry | `compute_step_export_test.go` |
| the save check that runs for a step of any kind | `step_shape_checks.go`, called first in `validateStepShape` |
| the seam the kind registers on | [step-kind-registry.md](step-kind-registry.md) |

`compute_step.go` registers three things in an `init`:

| Registration | Runs | Does |
|---|---|---|
| `RegisterStepShapeCheck(computeFieldsOnTheRightKind)` | on every save of a step of any kind, before the older shape checks | refuses compute fields on another kind, and refuses a compute step that carries a form, recipient, sent step, routes, workflow to run, chat fields or tool |
| `RegisterStepKind(compute, Shape, References)` | on a save of a compute step, after the older checks | the shape of the function, the result name and the bindings; then the references, which read the draft |
| `RegisterIssueCheck(computeIssueCheck)` | once per Issue scan | one pass over every compute step of the scope; it reads the steps and the records once |

The registry had no hook that runs for a step of another kind, because a kind's `Shape` runs only for that kind. `RegisterStepShapeCheck` is the one addition to the seam, and any kind may use it.

## The function catalog

The library is in another repo, so this repo asks it through one small interface, filled by a fake in the tests the way `WithGroupLister` is filled.

| Name | Meaning |
|---|---|
| `FunctionInput` | an input of a function: its `Name` and its `Type` (`number`, `currency`, `date`, `time`, `yes_no`, `text`) |
| `FunctionCatalog` | `InputsOf(ctx, name, version)` returns the inputs and whether the library has that function and version |
| `WithFunctionCatalog(catalog)` | the manager option |
| `ErrFunctionNotFound` | the sentinel for a function or version the library does not have; proposed and not decided, see Open |

With no catalog the inputs of a function are unknown. A save then checks everything that does not need them (the shape of the function, the result name, the bindings' own fields, the answer and record references and that the source can be a source) and the scan does not flag an input with no source. The host that builds the manager is expected to give it the catalog.

## What a save refuses

All are `ErrInvalidWorkItem` (400) unless the row says otherwise. Each sentence is checked word for word in `compute_step_test.go`. Case ids are those of compute-step-cases.md.

| Case | Refusal | Sentence (the part after `invalid work item:`) |
|---|---|---|
| CS-1 to CS-3 | a compute field on a step of another kind, or on a step with no kind | `a fill_form step takes no function, bindings or result name` |
| CS-4 to CS-10 | a compute step with a form, recipient, sent step, routes, workflow to run, chat fields or tool | `a compute step takes no form, recipient, sent step, routes, workflow to run, chat fields or tool` |
| CS-11 | no function | `a compute step names the function it calls, written library.function@version` |
| CS-12 | no `@version` | `the function "fin.pmt" has no version, write it as library.function@version` |
| CS-13, CS-14 | a version that is not a whole number | `the version of the function "fin.pmt@2.5" is not a whole number` |
| CS-15 | no result name | `a compute step names the result it keeps, in result_name` |
| CS-16 | a result name another compute step of the workflow has, ignoring case | `the result name "monthly payment" is already used by another compute step of this workflow` |
| CS-17 | more than one role | the older sentence `a step has one role, got ...` |
| CS-18 | a binding for an input the function does not have | `the function fin.pmt@2 has no input "term_days"` |
| CS-19 | two bindings for one input | `input "monthly_rate" has two bindings, and only one source is allowed` |
| CS-20 | a `source_type` that is none of the four | `the binding for input "months" has source type "formula", which is none of answer, record_field, ledger_balance and fixed` |
| CS-21 | an answer of a step of another workflow | `the answer for input "principal" comes from the step "Open an account", which belongs to another workflow` |
| CS-22 | an answer of the step itself | `the answer for input "principal" comes from the step itself` |
| CS-23 | an answer of a step that is not earlier by ordinality | `... comes from the step "Confirm the loan", which does not come before this step` |
| CS-24 | an answer of a step with no form | `... comes from the step "Greet the member", which has no form to read an answer from` |
| CS-25 | an answer to a question the step's form does not ask | `... comes from the step "Apply for a loan", which does not ask "Amount asked"` |
| CS-26 | a record question the record does not ask | `the record field for input "months" comes from the record "Loan terms", which does not ask "Weeks"` |
| CS-27 | a fixed binding with no value | `the fixed value for input "monthly_rate" is empty` |
| CS-28 | a fixed value that does not fit the input's type | `the fixed value "one percent" for input "monthly_rate" is not a number` |
| CS-29 | a source of another type than the input asks for | `the question "Reason for the loan" is text, but input "principal" asks for currency` |
| CS-30 | a `multi_select`, `photo` or `file` question | `the question "Photo of the id" is a photo question, which cannot be a source` |
| none | an answer or record field that names no step, record or question; a binding with no input | `the answer for input "principal" names no step`, `... names no question`, `the record field for input "months" names no record`, `binding 1 names no input` |
| none | a ledger balance for an input that is not a currency | `the ledger balance is currency, but input "months" asks for number` |

Not found:

| Case | Refusal |
|---|---|
| CS-32 | a role or group that is not offered: `ErrRoleNotFound` (404), from the check every step already has |
| CS-33 | an answer step that is not in the draft: `ErrWorkItemNotFound` (404) |
| CS-34 | a record that is not in the draft: `ErrRecordNotFound` (404) |
| CS-35 | a function or version the library does not have: `ErrFunctionNotFound`, `the library has no function fin.pmt@9` |

A step with fewer bindings than the function has inputs is saved (CS-36). An update is refused as a create is and the stored step is unchanged (CS-31). The fit of a source is exact: `currency` fits `currency`, `number` fits `number`, and nothing is converted.

## What the Issue scan flags

All are `error`, against the work item. Each names the step by code and title, `Work item WI-6 "Work out the monthly payment" ...`. A step with one broken binding gets one finding, never two for the same binding.

| Case | Finding (the part after the step) |
|---|---|
| CS-41, CS-42 | `has no source for input "months"`, one for each input with no binding |
| CS-43 | `reads input "principal", which asks for currency, from the question "Amount requested", which is number` |
| CS-44, CS-45 | `reads input "principal" from the question "Amount requested", which the step "Apply for a loan" does not ask any more`; a deleted question and a reworded one read the same, because a question is known by its wording |
| CS-46 | the same for a record: `... which the record "Loan terms" does not ask any more` |
| CS-47 | `reads input "principal" from a step that does not exist`; the delete does not cascade and the binding stays stored |
| CS-48 | `reads input "principal" from the step "Apply for a loan", which belongs to another workflow` |
| CS-49 | `reads input "principal" from the step "Apply for a loan", which no longer comes before it` |
| CS-50 | `reads input "months" from a record that does not exist` |
| CS-51 | the older finding that every step with a role already has: `is acted on by Role ..., which does not exist`. An offered group is not flagged (CS-52). |
| CS-53, CS-54 | `is a compute step that names no function` and `is a compute step that names no result name` |
| none | a binding whose stored source type is none of the four, or a question that became a photo, file or multi-select |

CS-56 (two findings are both listed and publishing is refused once) and CS-58 to CS-60 (no compute step, no role, a clean step) have tests. The publish refusal itself is the one the scan already feeds.

## Check against the design

Every refusal of compute-step.md section 3 and every finding of section 4 has a row above, except the two the issue leaves open (see below). The sentences are this issue's wording: the design page does not word them.

## Cases not covered here

| Cases | Why |
|---|---|
| CS-37, CS-38, CS-39 | covered by the accepts: the example step reads back unchanged, a role and an offered group both confirm, and a `fill_form` step reads back with no compute field |
| CS-40 | the derived `action` text is not in this issue's list; `models.ActionOf` still gives nothing for the two new kinds |
| CS-55 | a function version no longer in the library: open on the design page, not built |
| CS-61 to CS-84 | fork, publication, hash and import are [step-copy rules](step-copy-rules.md), not checks |
| CS-85, CS-86 | the input source list, which ISS-69 built |

## Open

Copied from compute-step.md and the issue; none is decided here.

1. The error name `ErrFunctionNotFound` is a proposal. It is not yet mapped to a status in `agency.operations.json`, `routes/routes.go` or `mcp/sentinels.go`, so over HTTP it is not yet a 404. The mapping is one line in each when the owner accepts the name.
2. A function version that is no longer in the library: the save refuses it (CS-35) but the scan does not flag an already stored step whose version has gone, and a function the catalog does not know skips the input checks of that step in the scan.
3. Whether a `currency` source is accepted where a number is asked. The fit is exact today.
4. Which row a `record_field` reads, which ledger `ledger_balance` reads, whether a later compute step's result may be a source, and whether a confirming step may carry a time limit. None of these is checked.
5. What a fixed value must look like for each type is this issue's proposal: a number or currency is an optional minus sign and digits with an optional decimal part, a date is `YYYY-MM-DD`, a time is `HH:MM`, a yes or no is `yes`, `no`, `true` or `false`, and a text is anything. The formula language's own rules may differ.
6. The result name is unique among the compute steps of one workflow, ignoring case, and is compared after trimming blanks. Whether it must also not clash with the name of a record or a question is not said.

## Tests

`go vet ./...` is clean. The tests of this issue are `TestComputeStep_*` and `TestComputeScan_*` in `compute_step_test.go`, 61 cases and subtests, and they reset the step kind registry and register this issue's checks themselves, as the registry page says a test does. Two tests written for ISS-48 and ISS-69 stored a compute step with no function, and one answered from a step with no form; they now build valid steps, because these rules refuse the old ones.

The root package, `models` and `mcp` pass; the root package has one test that flickers on a clean copy too, `TestLibraryPartTitles_SimultaneousCreatesOfOneTitleStoreOnePart`. In `routes`, `TestAG74_OpenHole_AttachEditDetachRewritesAnAgencyWideValue` fails as before.
