# One accountable role per step (ISS-179, owner decision D39)

The four-value RACI label (responsible, accountable, consulted, informed) on a
`role_assignment` row is dropped. A step keeps one plain field,
`accountable_role_id`: the one role that answers for the step. The role brief
reads that field, or the step's own `role_id` when the field is empty.

Why: in the one agency file that used the rows, every one of the 36 labels was
accountable, 23 repeated the step's own role, and 13 sat on steps with no role
of their own. The rows added a table, six operations and a label vocabulary to
say one thing per step.

## Where role assignments were read and written before ISS-179

Written:

| What | File, function |
|---|---|
| create, update and delete one row | `roleassignment_impl.go`: `CreateRoleAssignment`, `UpdateRoleAssignment`, `DeleteRoleAssignment`, check in `validateRoleAssignment` |
| import of an agency file | `document.go`: the `doc.RoleAssignments` loop at the end of `applyDocument` |
| fork of a draft | `draft_impl.go`: the `RoleAssignment` copy loop in `CreateDraft` |
| the software-lab seed | `swelab.go`: `RoleAssignments`, `CreateRoleAssignment` in the seed |
| the design-from-spec tool | `mcp/composite.go`: `specRACI`, the `p.RACI` loop in `mcpDesignAgencyFromSpec`, `RoleAssignments` in `mcpDesignFromSpecResult` |
| removal with a step | `workitem_impl.go`: `DeleteWorkItem` |
| removal with a role | `role_impl.go`: `DeleteRole` |
| removal with a workflow | `workflow_impl.go`: `DeleteWorkflow`; `generated_workflows.go`: the removal of a generated workflow's steps |

Read:

| What | File, function |
|---|---|
| one row, a draft's rows, a step's rows | `roleassignment_impl.go`: `GetRoleAssignment`, `ListRoleAssignments`, `ListRoleAssignmentsForWorkItem`; filter `RoleAssignmentFilter` in `filters.go` |
| the publication document | `document.go`: the `RoleAssignments` export in `buildDocument` |
| the publication content hash | `publication_impl.go`: `contentHashFor`, record `RoleAssignment` |
| the Issue scan | `issue_impl.go`: `checkRoleAssignments` (label valid, role exists, step exists) |
| the role brief, the only reader of the labels | `role_brief_impl.go`: `briefFromDocument` (walks `doc.RoleAssignments`), `orderedLabels`, `joinLabels`, `raciOrder`, `renderBrief` |
| the import summary | `cmd/import/main.go`: `stepRaciCount` |

Declared:

| What | File |
|---|---|
| the object | `agency.blueprint.json` and `agency.agency.json` (`role_assignment`, index `role_label_unique`), `store.go` (`roleRoleAssignment`), `tables.go` (`RoleAssignments`), `provision.go` (`draftScopedRoles`) |
| the model | `models/roleassignment.go` (`RoleAssignment`, `RACILabel`), `models/agencydocument.go` (`AgencyDocument.RoleAssignments`, `DocumentRoleAssignment`) |
| the manager interface | `agency.go`: the six `RoleAssignment` methods |
| the errors | `errors.go`: `ErrRoleAssignmentNotFound`, `ErrInvalidRACILabel`, `ErrDuplicateRoleAssignment`; mapped in `mcp/sentinels.go` and `routes/routes.go`, and in `agency.operations.json` |
| the operations and tools | `agency.operations.json`: `create_raci`, `list_raci`, `get_raci`, `update_raci`, `delete_raci`, `list_work_item_raci` (tools `agency_set_raci`, `agency_list_role_assignments`, `agency_get_role_assignment`, `agency_update_role_assignment`, `agency_delete_role_assignment`, `agency_list_role_assignments_for_work_item`) |
| the tool catalog | `mcp/catalog.go` (the `RoleAssignment (RACI)` entry), `mcp/tools.go` (the `role_assignment` name) |

## The model after ISS-179

- `WorkItem.AccountableRoleID` (`accountable_role_id`) is one role of the same
  draft. It is allowed on a step of any kind, including a step with no kind,
  and a step with no role of its own may carry it. It is refused when it names
  no role of the draft (`ErrRoleNotFound`) or more than one id.
- Deleting a role empties `accountable_role_id` on every step of the draft
  that named it, in the same transaction, as deleting a role used to remove its
  assignments. The step's own `role_id` is left as it always was.
- Fork, the publication document (`DocumentWorkItem.AccountableRoleID`), the
  publication content hash and the Issue scan carry the field. The scan raises
  an `error` for a step whose accountable role no longer exists.
- There is no `role_assignment` object, no `RACILabel`, no
  `ErrInvalidRACILabel`, `ErrDuplicateRoleAssignment` or
  `ErrRoleAssignmentNotFound`, and none of the six operations or their tools.
  `agency_design_from_spec` has no `raci` section; a work item in the spec takes
  `accountable_role_key` or `accountable_role_id`.
- The role brief of a seat lists a step when the step's effective role is the
  seat. The effective role is `accountable_role_id`, or `role_id` when
  `accountable_role_id` is empty. Objectives are those of the workflows that
  have such a step. `BriefWorkItem` and `BriefObjective` no longer carry labels;
  a work item says whether the seat is accountable for it (`accountable`).

## Old files and old publications

An agency file or a stored publication written before ISS-179 carries
`role_assignments`. `foldLegacyRoleAssignments` (`legacy_role_assignments.go`)
turns it into the field, and import and the role brief both call it:

- an entry is read only when it names a step and a role of the file and its label
  is `accountable`; every other label and every entry that names an
  objective and no step is dropped;
- an entry whose role is the step's own `role_id` is a duplicate and is dropped;
- the first remaining entry of a step, in file order, sets
  `accountable_role_id`; a step that already carries the field keeps it and the
  entries for it are dropped.

The document written out by a new export has no `role_assignments` key.

A database written before ISS-179 keeps its `role_assignments` table and rows.
Nothing reads them; nothing converts them. Rows are carried to the field only
through a re-import of the file, or by setting the field on the step.

## What the hub must change when this sibling is tagged

The hub imports this module (`code/wakala-api/go.mod`). It is not edited by
ISS-179. When a tag carries this change the hub has to:

- `internal/api/http/instance_import.go`: drop `importRoleAssignment` and the
  `in.RoleAssignments` loop that calls `CreateRoleAssignment`; add
  `accountable_role_id` to the imported work item and resolve it with the same
  role lookup `role_id` uses; read an old file's `role_assignments` by the rules
  above (the loop is the hub's own, it does not call `ImportDocument`). Count
  `accountable_role_id` where it counted `role_assignments`.
- `internal/api/http/instance_import_limits.go`: remove `len(in.RoleAssignments)`
  from the entry count.
- `internal/api/http/approvals_apply_raci.go`: remove `applySetRaci`,
  `applyDeleteRoleAssignment` and the appliers `agency_set_raci` and
  `agency_delete_role_assignment`; a change of a step's accountable role is an
  `agency_update_work_item` approval and needs its own applier if the hub
  gates it.
- `internal/templates/build.go`: stop building `doc.RoleAssignments` from
  `spec.RACI` and drop `raciLabels`; a spec's work item names its accountable
  role, written to `DocumentWorkItem.AccountableRoleID`.
- The console data modules that call `agency_set_raci` or read
  `role-assignments` (`console/src/data/pages/stepRoles.ts`,
  `console/src/data/pages/goals.ts`, `console/src/data/chatPages/*`,
  `console/src/pages/objectives/*`) read and write the step's
  `accountable_role_id` instead.
- The hub's database: the `role_assignments` table the hand-kept migration
  mirror creates for the agency module is no longer written; the mirror needs
  the new `accountable_role_id` column on `work_items`, and rows already in the
  table are converted by a one-off statement or a re-import.
- The IAS agency file: its 36 entries become 13 step fields (the 23 that repeat a
  step's own role are dropped). That conversion belongs to the hub issue that
  owns the file, not to this module.
