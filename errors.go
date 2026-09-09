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
	// Draft (fork-from, edit, promote, archive) is attempted on a Draft
	// whose Status is not "open".
	ErrDraftNotOpen = errors.New("mwanachamaagency: draft is not open")
	// ErrDraftNotPromoted is returned by PublishAgency when given a
	// draftID whose Draft has not been promoted yet.
	ErrDraftNotPromoted = errors.New("mwanachamaagency: draft is not promoted")
	// ErrPublicationNotFound is returned when a publication version has no
	// match.
	ErrPublicationNotFound = errors.New("mwanachamaagency: publication not found")
	// ErrInvalidPublicationStatus is returned by UpdatePublicationStatus
	// for any transition other than active -> archived.
	ErrInvalidPublicationStatus = errors.New("mwanachamaagency: invalid publication status transition")
	// ErrNoChangesDetected is returned by PublishAgency when the current
	// live content hash matches an existing publication's.
	ErrNoChangesDetected = errors.New("mwanachamaagency: no changes detected since last publication")
	// ErrEventFlowStepNotFound is returned when LookupFlowStep finds no
	// matching live EventFlowStep.
	ErrEventFlowStepNotFound = errors.New("mwanachamaagency: event flow step not found")
	// ErrInvalidLookup is returned by LookupFlowStep when no lookup key is
	// set (or TriggerTopic is set without Consumer).
	ErrInvalidLookup = errors.New("mwanachamaagency: invalid event flow step lookup")
	// ErrWorkPlanNotFound is returned when a WorkPlan id has no match.
	ErrWorkPlanNotFound = errors.New("mwanachamaagency: work plan not found")
	// ErrInvalidRegex is returned by CreateWorkPlan/UpdateWorkPlan when
	// TriggerTopic or PayloadCondition does not compile as a Go regexp.
	ErrInvalidRegex = errors.New("mwanachamaagency: invalid regex")
	// ErrContextSourceNotFound is returned when a ContextSource id has no
	// match, or matches a different WorkPlan than the one given.
	ErrContextSourceNotFound = errors.New("mwanachamaagency: context source not found")
	// ErrInvalidDepartment is returned by CreateDepartment/RenameDepartment
	// when Name is empty.
	ErrInvalidDepartment = errors.New("mwanachamaagency: invalid department")
	// ErrDepartmentNotFound is returned when a Department id has no match.
	ErrDepartmentNotFound = errors.New("mwanachamaagency: department not found")
	// ErrRootCannotMove is returned by MoveDepartment when id currently has
	// no parent.
	ErrRootCannotMove = errors.New("mwanachamaagency: a top-level department cannot be moved")
	// ErrParentIsSelf is returned by MoveDepartment when newParentID == id.
	ErrParentIsSelf = errors.New("mwanachamaagency: a department cannot be its own parent")
	// ErrParentInSubtree is returned by MoveDepartment when newParentID is
	// one of id's own descendants.
	ErrParentInSubtree = errors.New("mwanachamaagency: new parent is in the department's own subtree")
	// ErrParentNotFound is returned by MoveDepartment when newParentID
	// names no Department.
	ErrParentNotFound = errors.New("mwanachamaagency: parent department not found")
	// ErrLevelNotFound is returned when a Level id has no match.
	ErrLevelNotFound = errors.New("mwanachamaagency: level not found")
	// ErrLevelWornByDepartments is returned by DeleteLevel when a
	// Department still references the level.
	ErrLevelWornByDepartments = errors.New("mwanachamaagency: level is in use by a department")
)
