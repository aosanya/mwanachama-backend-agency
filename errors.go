package mwanachamaagency

import "errors"

var (
	// ErrAgencyNotFound is returned when no Agency row exists.
	ErrAgencyNotFound = errors.New("mwanachamaagency: agency not found")
	// ErrGoalNotFound is returned when a Goal id has no match.
	ErrGoalNotFound = errors.New("mwanachamaagency: goal not found")
	// ErrWorkflowNotFound is returned when a Workflow id has no match.
	ErrWorkflowNotFound = errors.New("mwanachamaagency: workflow not found")
	// ErrWorkItemNotFound is returned when a WorkItem id has no match.
	ErrWorkItemNotFound = errors.New("mwanachamaagency: work item not found")
	// ErrDraftNotFound is returned when a Draft id has no match.
	ErrDraftNotFound = errors.New("mwanachamaagency: draft not found")
	// ErrDraftNotOpen is returned when an operation that requires an open
	// Draft (fork-from, edit, archive) is attempted on a Draft whose Status
	// is not "open".
	ErrDraftNotOpen = errors.New("mwanachamaagency: draft is not open")
	// ErrDraftRequired is returned by every Create<Type> when DraftID is
	// empty — AGD-006: every row belongs to a real Draft, there is no live
	// row-scope left to default into.
	ErrDraftRequired = errors.New("mwanachamaagency: a draft id is required")
	// ErrInvalidForkSource is returned by CreateDraft when forkedFromType
	// is none of "" (start empty), "draft", or "publication" (AGD-006
	// dropped "live" — there is no working-set row-scope left to fork
	// from).
	ErrInvalidForkSource = errors.New("mwanachamaagency: forkedFromType must be \"\", \"draft\", or \"publication\"")
	// ErrPublicationNotFound is returned when a publication version has no
	// match.
	ErrPublicationNotFound = errors.New("mwanachamaagency: publication not found")
	// ErrNoChangesDetected is returned by PublishAgency when the current
	// live content hash matches an existing publication's.
	ErrNoChangesDetected = errors.New("mwanachamaagency: no changes detected since last publication")
	// ErrEventFlowStepNotFound is returned when LookupFlowStep finds no
	// matching live EventFlowStep.
	ErrEventFlowStepNotFound = errors.New("mwanachamaagency: event flow step not found")
	// ErrInvalidLookup is returned by LookupFlowStep when no lookup key is
	// set (or TriggerTopic is set without Consumer).
	ErrInvalidLookup = errors.New("mwanachamaagency: invalid event flow step lookup")
	// ErrInvalidRole is returned by CreateRole/UpdateRole when Name is empty.
	ErrInvalidRole = errors.New("mwanachamaagency: invalid role")
	// ErrRoleNotFound is returned when a Role id has no match in the
	// expected scope.
	ErrRoleNotFound = errors.New("mwanachamaagency: role not found")
	// ErrRoleAssignmentNotFound is returned when a RoleAssignment id has no
	// match.
	ErrRoleAssignmentNotFound = errors.New("mwanachamaagency: role assignment not found")
	// ErrInvalidRACILabel is returned by CreateRoleAssignment/
	// UpdateRoleAssignment when Label is not one of the four RACI values.
	ErrInvalidRACILabel = errors.New("mwanachamaagency: invalid RACI label")
	// ErrDuplicateAccountable is returned by CreateRoleAssignment/
	// UpdateRoleAssignment when another assignment in the same scope
	// already labels the WorkItem Accountable.
	ErrDuplicateAccountable = errors.New("mwanachamaagency: work item already has an accountable role")
	// ErrObjectiveNotFound is returned when an Objective id has no match
	// in the expected scope.
	ErrObjectiveNotFound = errors.New("mwanachamaagency: objective not found")
	// ErrInvalidWorkItem is returned by CreateWorkItem/UpdateWorkItem when
	// ObjectiveID is empty — every WorkItem belongs to exactly one
	// Objective (AGD-001).
	ErrInvalidWorkItem = errors.New("mwanachamaagency: invalid work item")
	// ErrInvalidValue is returned by CreateValue/UpdateValue when Text is
	// empty.
	ErrInvalidValue = errors.New("mwanachamaagency: invalid value")
	// ErrValueNotFound is returned when a Value id has no match.
	ErrValueNotFound = errors.New("mwanachamaagency: value not found")
	// ErrInvalidRule is returned by CreateRule/UpdateRule when Text is
	// empty.
	ErrInvalidRule = errors.New("mwanachamaagency: invalid rule")
	// ErrRuleNotFound is returned when a Rule id has no match.
	ErrRuleNotFound = errors.New("mwanachamaagency: rule not found")
	// ErrCannotEditLive is returned by UpdateValue/DeleteValue/
	// UpdateRule/DeleteRule when the target row carries no attachment
	// (see [hasAnyAttachment]) — an Agency-wide Value/Rule is never forked
	// into another Draft, so this is permanent, not just "until archived".
	ErrCannotEditLive = errors.New("mwanachamaagency: agency-wide data cannot be edited directly")
	// ErrInvalidTargetType is returned by CreateValueAttachment/
	// CreateRuleAttachment when TargetType is not one of the five known
	// values (see [models.TargetType.Valid]).
	ErrInvalidTargetType = errors.New("mwanachamaagency: invalid target type")
	// ErrDuplicateAttachment is returned by CreateValueAttachment/
	// CreateRuleAttachment when the same (Value/Rule, TargetType,
	// TargetID) attachment already exists.
	ErrDuplicateAttachment = errors.New("mwanachamaagency: attachment already exists")
	// ErrValueAttachmentNotFound is returned when a ValueAttachment id has
	// no match.
	ErrValueAttachmentNotFound = errors.New("mwanachamaagency: value attachment not found")
	// ErrRuleAttachmentNotFound is returned when a RuleAttachment id has
	// no match.
	ErrRuleAttachmentNotFound = errors.New("mwanachamaagency: rule attachment not found")
	// ErrPublishValidationFailed is returned by PublishAgency when the
	// scan of the Draft being published found at least one error-severity
	// Issue. Its message names them; ListIssues(ctx, draftID,
	// IssueFilter{Severity: "error"}) reads the same findings back as
	// structured rows.
	ErrPublishValidationFailed = errors.New("mwanachamaagency: agency has validation errors and cannot be published")
	// ErrDuplicateTag is returned by PublishAgency when another
	// publication already carries the requested tag.
	ErrDuplicateTag = errors.New("mwanachamaagency: publication tag already exists")
	// ErrInvalidModule is returned by CreateModule when Code is empty.
	ErrInvalidModule = errors.New("mwanachamaagency: invalid module")
	// ErrDuplicateModule is returned by CreateModule when a Module with
	// the same Code already exists in the same scope.
	ErrDuplicateModule = errors.New("mwanachamaagency: module code already exists")
	// ErrModuleNotFound is returned when a Module id has no match in the
	// expected scope.
	ErrModuleNotFound = errors.New("mwanachamaagency: module not found")

	ErrInvalidModuleSettings = errors.New("mwanachamaagency: module settings must be a JSON object")
	// ErrDuplicateObjectiveModule is returned by CreateObjectiveModule when
	// the same (Objective, Module) pair is already linked in that scope.
	ErrDuplicateObjectiveModule = errors.New("mwanachamaagency: objective already linked to this module")
	// ErrRationaleRequired is returned by CreateObjectiveModule when the link
	// carries no Rationale.
	ErrRationaleRequired = errors.New("mwanachamaagency: objective module needs a rationale")
	// ErrObjectiveModuleNotFound is returned when an ObjectiveModule id has
	// no match.
	ErrObjectiveModuleNotFound = errors.New("mwanachamaagency: objective module not found")

	ErrInvalidRecord    = errors.New("mwanachamaagency: invalid record")
	ErrRecordNotFound   = errors.New("mwanachamaagency: record not found")
	ErrInvalidRecordRow = errors.New("mwanachamaagency: invalid record row")
	ErrRecordRowRefused = errors.New("mwanachamaagency: this role does not add rows to the record")
)
