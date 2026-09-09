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
)
