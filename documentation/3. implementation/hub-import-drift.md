# Hub import drift

This page lists every difference between the hub's in-tree copy of the agency module (`projects/hub-steward/code/wakala-api/modules/agency`, newest commit `c9fbd497` of 2026-11-19) and this repository, measured on 2026-11-19 before the hub imports this module and deletes its copy. It is the input to the merge and the record of what the merge did: each row has a disposition. The method is the one of [hub-copy-diff.md](hub-copy-diff.md), which describes the first round, in which this repository took over the hub's changes; the hub has changed the module again since.

## Method

1. `scripts/hub-copy-diff.sh` copies the `*.go` and `*.json` files of both sides (top level, `mcp`, `models`, `routes`), rewrites the hub's import prefixes (`wakalaserver/modules/shared/` to `github.com/aosanya/mwanachama-backend-shared/`, `wakalaserver/modules/agency` to `github.com/aosanya/mwanachama-backend-agency`), strips comments and reprints every Go file with `gofmt`, then runs `diff -r`.
2. The files under `testdata`, `documentation`, `cmd` and `scripts` are not in that diff; the hub folder holds none of them, so every one is only here.
3. After the merge the script was run again. What it still prints is listed under "Only in this repository" and "Layout only" below.

## Only in the hub

| Item | Disposition |
|---|---|
| `row_allowance.go`: `RowAllowance`, `ExtraDetail`, `WithRowAllowance`. A caller puts extra details and waived questions on the context, and a record row is checked against them. | Moved. `record_impl.go` takes the hub's `validateRowAnswers(ctx, ...)`. |
| `approval_stamp.go`: the prompts `Approved over the limit` and `Limit that applied`, counted as approval stamps. | Moved. |
| A setting on each seat, `approves_above_limit`: `models.Role` and `DocumentRole` fields, kept by `UpdateRole` when the request does not carry it, copied by a fork and by a document import, hashed into a publication, listed in the blueprint. | Moved. |
| `SetRolePermissions`: the `AgencyManager` method, `ErrUnknownPermission` and `ErrLockedPermission` (both 400), the `set_role_permissions` operation (`PUT /roles/{role_id}/permissions`, tool `agency_set_role_permissions`), the sentinels in `mcp` and `routes`, and the action `agency.role.set_permissions` on the locked permission `permission.override`. | Moved. |
| `umbrellaActions` in `control_scan.go`: a control naming `engage.post.create` is known when the catalogue holds any of the three post-creating actions. | Moved. |
| `needed_by_modules` on a module: `models.Module.NeededByModules`, `modulesRequiredByEnabled`, and the descriptions of the module list and read operations. | Moved. |
| Words per module mount: `ModuleMount.Words`, `MaxMountWords`, `MaxMountWordLength`, `validateMountWords`, and the description of the mount argument. | Moved. |
| The `council` module in `modules_catalog.json`, mountable under `council`. | Moved. |
| Two answer types, `time_range` and `address`, and `StepQuestion.AllowFreeText` (only an address question may allow free text). A time-range answer is checked with `answerrules` of the shared module. | Moved. |
| `RecordRow.PlaceRefs` and its blueprint field `place_refs`. | Moved. |
| A role's permissions, staff flag and approval setting in the publication content hash. | Moved. It changes the hash of any published role that holds permissions or is staff, so the first publication after the import of such an agency is a new version. |
| Twenty permissions in the catalogue: `accounting.manage`, `accounting.view`, `application.correct`, `application.decide`, `application.file`, `application.inspect`, `appointment.view_person`, `audit.view`, `consent.record`, `council.manage`, `council.view`, `declaration.read`, `digitaltwin.manage`, `digitaltwin.view`, `location.edit`, `member.add`, `member.edit`, `notification.raise`, `person_raised_items.read`, `risk.record`. | Moved, with a row each in [role-permissions.md](../2.%20design/role-permissions.md). |
| Actions added to existing permissions: `appointment.manage`, `asset.manage`, `backlog.manage`, `board.manage`, `case.vote`, `controls.edit`, `event.post`, `form.results`, `group.manage`, `notice.post`, `register.edit`, `report.publish`, `review.run`, `risk.read`, `service.edit`, `service.read`, `task.edit`, `task.read`, `volunteer_call.post`, and `chat.thread` gains `comm.chat_room.set_question`. `appointment.manage`, `register.edit` and `service.edit` also gain the module `forms`. | Moved. |
| `member.remove` carries `actor.role_assignment.revoke`, `actor.role_assignment.end_on_eviction` and `actor.assignment.unassign` instead of `actor.assignment.delete`, which no module serves. | Moved; the hub's value replaces the one here. |
| 22 test files that this repository lacked: `control_scan_actions_test.go`, `module_needed_by_modules_test.go`, `step_answer_types_test.go`, `mcp/role_permissions_tool_test.go`, `models/module_cover_test.go`, `models/step_cover_test.go`, `models/permission_cover_test.go`, `models/permission_edit_test.go`, `models/permission_actions_declared_test.go` and thirteen `models/permission_catalog_*_test.go` files. | Moved. |
| `models/lifecycle_cover_test.go` | Dropped. Same subject as `models/states_cover_test.go` here, which is the stricter one; the two declare the same test name. |
| Tests added to files both sides have: `TestUpdateRoleToolSetsAndKeepsTheSettingToApproveAboveTheLimit` in `mcp/role_update_tool_test.go`, and the council and words cases of `models/module_catalog_mounts_test.go`. | Moved. |

## Differing beyond the import path

| Item | Here | In the hub | Disposition |
|---|---|---|---|
| `report.read` in the catalogue | Module `forms`, name "Find and read a report published to your groups". | No module, name "Find and read the reports published to my groups". | Kept here. |
| `market.browse`, `market.ask`, `market.ask_answer`, `market.advise` | Also hold the read actions their screens call (`market.notice.read`, `market.ask.read`, `market.quote.read`, `market.draft.read`). | Lack them. | Kept here. |
| `chat.thread` | Holds `comm.chat_thread.read`. | Does not. | Union of both. |
| `kind` values of a step in the blueprint | Nine, with `compute` and `post`. | Seven. | Kept here. |
| Step kinds, route operators, work item fields | Nine kinds, seven route operators (`on_or_before`, `on_or_after` are day operators), the calculating and posting step fields. | Seven kinds, five operators. | Kept here; the merge adds the hub's answer types beside them. |
| `UpdateRole`, `UpdateWorkflow`, `UpdateWorkItem` | Keep the fields the request does not carry (`carriedKeys`). | Role only, with its own `carried` map. | Kept here; the hub's `approves_above_limit` rule is added to it. |
| `CreateDraft` and the document import | Give each copied record a new id and remap the ids a registered step kind holds. | Give each record a new id at insert. | Kept here. |
| `Provision` index check | Checks the indexes of every declared object. | Checks only the workflow objective table. | Kept here. |
| Library part and consequence writes | Take a lock so two writers cannot create or rename a part to the same title at once. | No lock. | Kept here. |
| Multi-select answers and options, the route rule day check | Checked on write. | Not checked. | Kept here. |
| Operation descriptions of the work item tools | Describe the calculating and posting step fields. | Do not. | Kept here. |
| `ErrFunctionNotFound`, `ErrAccountNotFound` (404) | Declared. | Not declared. | Kept here. |
| The risk permissions rule | A test said no unlocked permission carries a risk manager write. | `risk.record` is unlocked and carries `riskmanager.evidence.record`, `riskmanager.action.update` and `riskmanager.action.set_status`. | The hub's permission is kept and the test now names it as the one exception: `TestOnlyRiskRecordCarriesARiskManagerWriteAmongTheUnlockedPermissions`. |

## Only in this repository

| Item | Disposition |
|---|---|
| `compute_step.go`, `compute_step_copy.go`, `compute_failure.go`, `hold_step.go`, `hold_step_copy.go`, `post_step.go`, `post_step_copy.go`, `input_source_impl.go`, `step_kind_registry.go`, `step_shape_checks.go`, `scenario_coverage_impl.go`, `models/carried.go` | Keep. The calculating, posting and holding steps, the input source list and the step kind registry. |
| `ListInputSources` on `AgencyManager` and the operation `list_input_sources` | Keep. Anything in the hub that implements `AgencyManager` with a fake must add the method when it imports this module. |
| `FunctionCatalog` and the `libraryTitles` lock on the manager | Keep. |
| 256 test files (the large majority of the tests of the module), `cmd/import`, `cmd/provision`, `scripts/`, `testdata/`, `documentation/`, `CLAUDE.md`, `README.md`, `Makefile`, `go.mod`, `go.sum` | Keep. The hub copy never carried them. |

## Layout only

The script still prints these because the two sides lay the same lines out differently or because this side holds more: `agency.go` (field alignment of the manager), `consequence_impl.go`, `library_part_impl.go`, `draft_impl.go`, `document.go`, `issue_impl.go`, `provision.go`, `publication_impl.go`, `record_impl.go`, `route_step_impl.go`, `workflow_impl.go`, `workitem_impl.go`, `workitemstep_impl.go`, `models/workflow.go`, `models/workitem.go`, `models/role.go`, `models/workitemstep.go`, the sentinel maps of `mcp` and `routes`, `errors.go`, `agency.operations.json`, `agency.blueprint.json`, `models/permissions_catalog.json` and `models/step_cover_test.go` (the hub's side still counts seven kinds). In each, every hub line is either in this side too or was replaced by the row above that names it. `models/permission_catalog_resident_read_only_test.go` and `routes/review_author_test.go` differ because this side's versions are self-contained and carry one more test.

## Pinned counts the union changes

| Where | Before | After |
|---|---|---|
| `models/states_cover_test.go`, number of answer types | 10 | 12 |
| `models/step_cover_test.go`, number of step kinds | 7 | 9 |
| `models/step_cover_test.go`, number of route operators | 5 | 7 |
| `picker_test.go`, the answer types a picker lists | the ten of the forms module | those ten and `time_range`, `address` |
| `models/permissions_catalog.json`, permissions | 88 | 108 |
| `agency.operations.json`, operations and error mappings | 146 and 86 | 147 and 88 |
| `permission_actions_test.go`, actions of `report.publish`, `notice.post`, `event.post`, `volunteer_call.post` | one to three each | each plus the hub's added actions |
| `control_permissions_test.go`, actions of `controls.edit` | the write actions and three reads | those plus eight more reads |
| `risk_permissions_test.go`, actions of `risk.read` | twelve riskmanager reads | seventeen |
| `models/permission_catalog_hub_test.go`, actions of `board.manage` | four | seven |
| `documentation/2. design/role-permissions.md` | no row for twenty permissions | a row each |

## Fixture refreshed

`testdata/operations/comm.operations.json` is now the file of `mwanachama-backend-comm`. It was older than the catalogue: the action `comm.chat_room.set_question` and `comm.chat_thread.read` were in the catalogue and in no fixture. `TestEveryActionOfTheHubAgentPermissionsIsDeclaredByItsModuleUnderTheToolNameTheAgentsCall` passed again with it.

## Failures present before this work

Five tests of the root package and one of `routes` fail at `84bfc89`, before any hub change: `TestEveryActionOfTheCataloguePermissionGapBelongsToAnUnlockedPermission`, `TestEveryCallTheHubAgentsMakeIsCarriedByAnUnlockedPermission`, `TestTheHubAgentPermissionsCarryExactlyTheirActions`, `TestEveryActionOfTheHubAgentPermissionsIsDeclaredByItsModuleUnderTheToolNameTheAgentsCall`, `TestPostStep_ScanOfAStepWithNothingButAnAccount`, and in `routes` `TestAG74_OpenHole_AttachEditDetachRewritesAnAgencyWideValue`. After the merge the fourth passes (the fixture refresh above) and the other five still fail with the messages they had; none is touched by a hub change.

## Hub-only wiring that is not part of the module

These stay in the hub when it imports this module. They are listed so the import step knows its work.

| Hub place | What it does | Disposition |
|---|---|---|
| `internal/agencies` (23 files) | Builds the agency manager and its options, and wraps role permission edits behind a setter interface. | Stays. Imports change to this module's path. |
| `internal/api/http` (106 files) | Mounts the agency routes with the session, the permission scopes and the module switch; `record_details_allowance.go` builds a `RowAllowance` from the owner's chosen details and puts it on the context with `WithRowAllowance`; the role permission, approval, review and step run routes call the manager. | Stays. Imports change. |
| `internal/reviewruns`, `internal/testruns`, `internal/accounts`, `internal/templates`, `cmd/server` (`reviewruns.go`, `testruns.go`), `modules/accounting/routes` | Run reviews and test runs against the manager and read agency types. | Stay. Imports change. |
| Hub tests with a fake that implements `AgencyManager` | Fakes of the manager. | Each fake must add `ListInputSources`. |
| `internal/specs` and the hub's own specs | The hub's declared domains. | Stay. |

After the import, the hub's own in-tree `wakalaserver/modules/shared` is a different Go package from `mwanachama-backend-shared`. The agency module's `Operations`, `Build` and specs use the shared module's types, so the hub's shared import has to move to the shared module in the same change, or the build breaks at the first call that passes a `spec.Spec` or a `dispatch.Caller`.

## Owner questions

1. The hub's `risk.record` lets an unlocked permission carry three risk manager writes, which an earlier test here forbade. The merge kept the hub's permission and made the test name it as the one exception. Is that the rule to keep?
2. The hub hashes a role's permissions, staff flag and approval setting into the publication content hash; this repository did not. The merge took the hub's rule, so a published role with permissions gets a new content hash on its next publish. Is a new version for such agencies acceptable, or should the first publish after the import reuse the old hash?
3. `testdata/operations/comm.operations.json` was refreshed from the comm repository. Should the other fixtures (`actor`, `forms`, `ai`, `sources`, `riskmanager`) be refreshed from their repositories too before the tag?

## Result

This repository holds the union of both sides. Nothing the hub had is missing here, apart from the one dropped duplicate test file named above, and what only this repository had is untouched.
