package mwanachamaagency

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

// AgencyManager is the behavior surface this package exposes: CRUD for the
// live Agency/Goal/Workflow/WorkItem/EventFlowStep data, the Draft edit
// lifecycle, and publication.
type AgencyManager interface {
	// Agency
	CreateAgency(ctx context.Context, a models.Agency) (models.Agency, error)
	GetAgency(ctx context.Context) (models.Agency, error)
	UpdateAgency(ctx context.Context, a models.Agency) (models.Agency, error)

	// Goal. List's draftID selects scope: "" lists live Goals, a Draft's
	// ID lists that draft's copies.
	CreateGoal(ctx context.Context, g models.Goal) (models.Goal, error)
	GetGoal(ctx context.Context, id string) (models.Goal, error)
	ListGoals(ctx context.Context, draftID string) ([]models.Goal, error)
	UpdateGoal(ctx context.Context, g models.Goal) (models.Goal, error)

	// Workflow
	CreateWorkflow(ctx context.Context, w models.Workflow) (models.Workflow, error)
	GetWorkflow(ctx context.Context, id string) (models.Workflow, error)
	ListWorkflows(ctx context.Context, draftID string) ([]models.Workflow, error)
	UpdateWorkflow(ctx context.Context, w models.Workflow) (models.Workflow, error)

	// WorkItem
	CreateWorkItem(ctx context.Context, wi models.WorkItem) (models.WorkItem, error)
	GetWorkItem(ctx context.Context, id string) (models.WorkItem, error)
	ListWorkItems(ctx context.Context, draftID string) ([]models.WorkItem, error)
	UpdateWorkItem(ctx context.Context, wi models.WorkItem) (models.WorkItem, error)

	// Draft lifecycle. CreateDraft forks the sub-graph scoped to
	// forkedFromID (an Agency.ID when forkedFromType=="live", or another
	// open Draft's ID when forkedFromType=="draft") into new rows scoped
	// to the new Draft. PromoteDraft replaces live data with the Draft's
	// copies; ArchiveDraft discards a Draft without promoting it. Returns
	// [ErrDraftNotOpen] where noted.
	CreateDraft(ctx context.Context, description, forkedFromID, forkedFromType string) (models.Draft, error)
	GetDraft(ctx context.Context, id string) (models.Draft, error)
	ListDrafts(ctx context.Context) ([]models.Draft, error)
	// UpdateDraftDescription returns [ErrDraftNotOpen] if the draft isn't open.
	UpdateDraftDescription(ctx context.Context, id, description string) (models.Draft, error)
	// PromoteDraft returns [ErrDraftNotOpen] if the draft isn't open.
	PromoteDraft(ctx context.Context, id string) (models.Agency, error)
	// ArchiveDraft returns [ErrDraftNotOpen] if the draft isn't open.
	ArchiveDraft(ctx context.Context, id string) (models.Draft, error)

	// Publication. PublishAgency versions and content-hashes whatever is
	// currently live; draftID, if non-"", is recorded as the Draft that
	// prompted the publish and must already be promoted
	// ([ErrDraftNotPromoted] otherwise). Returns [ErrNoChangesDetected] if
	// the content hash matches an existing publication.
	PublishAgency(ctx context.Context, draftID string) (models.AgencyPublication, error)
	GetPublication(ctx context.Context, version int) (models.AgencyPublication, error)
	ListPublications(ctx context.Context) ([]models.AgencyPublication, error)
	// UpdatePublicationStatus only allows the active -> archived
	// transition; anything else returns [ErrInvalidPublicationStatus].
	UpdatePublicationStatus(ctx context.Context, version int, status string) (models.AgencyPublication, error)

	// EventFlowStep
	CreateEventFlowStep(ctx context.Context, s models.EventFlowStep) (models.EventFlowStep, error)
	// LookupFlowStep tries lookup.HandlerCode, then
	// (lookup.TriggerTopic, lookup.Consumer) together, then lookup.Code, in
	// that order. Returns [ErrInvalidLookup] if none are set.
	LookupFlowStep(ctx context.Context, lookup models.EventFlowStepLookup) (models.EventFlowStep, error)
	// ListEventFlowSteps returns live steps ordered by (WorkflowCode,
	// Ordinality, Step), optionally filtered to one workflowCode.
	ListEventFlowSteps(ctx context.Context, workflowCode string) ([]models.EventFlowStep, error)

	// WorkPlan. CreateWorkPlan/UpdateWorkPlan return [ErrInvalidRegex] if
	// TriggerTopic or PayloadCondition does not compile as a Go regexp.
	CreateWorkPlan(ctx context.Context, w models.WorkPlan) (models.WorkPlan, error)
	GetWorkPlan(ctx context.Context, id string) (models.WorkPlan, error)
	ListWorkPlans(ctx context.Context) ([]models.WorkPlan, error)
	UpdateWorkPlan(ctx context.Context, w models.WorkPlan) (models.WorkPlan, error)
	// MatchWorkPlans evaluates topic/payload against every enabled
	// WorkPlan's TriggerTopic/PayloadCondition regex, returning matches
	// (with their ContextSources) ordered by Ordinality ascending.
	MatchWorkPlans(ctx context.Context, topic, payload string) ([]models.WorkPlanMatch, error)

	// ContextSource. AddContextSource/ListContextSources return
	// [ErrWorkPlanNotFound] if workPlanID names no WorkPlan.
	AddContextSource(ctx context.Context, workPlanID string, cs models.ContextSource) (models.ContextSource, error)
	ListContextSources(ctx context.Context, workPlanID string) ([]models.ContextSource, error)
	// RemoveContextSource returns [ErrContextSourceNotFound] if sourceID
	// names no ContextSource, or names one belonging to a different
	// WorkPlan than workPlanID.
	RemoveContextSource(ctx context.Context, workPlanID, sourceID string) error

	// Department. The org chart is a local, self-contained parent/child
	// tree (no dependency on mwanachama-backend-actor's own Group/Hierarchy
	// tables).
	CreateDepartment(ctx context.Context, d models.Department) (models.Department, error)
	GetDepartment(ctx context.Context, id string) (models.Department, error)
	ListDepartments(ctx context.Context) ([]models.Department, error)
	ListDepartmentChildren(ctx context.Context, parentID string) ([]models.Department, error)
	RenameDepartment(ctx context.Context, id, name string) (models.Department, error)
	// MoveDepartment returns [ErrParentIsSelf], [ErrRootCannotMove],
	// [ErrParentNotFound], or [ErrParentInSubtree] where noted on each.
	MoveDepartment(ctx context.Context, id, newParentID string) (models.Department, error)

	// Level. A named rung in the Department tree.
	CreateLevel(ctx context.Context, l models.Level) (models.Level, error)
	GetLevel(ctx context.Context, id string) (models.Level, error)
	ListLevels(ctx context.Context) ([]models.Level, error)
	RenameLevel(ctx context.Context, id, name string) (models.Level, error)
	// DeleteLevel returns [ErrLevelWornByDepartments] if any Department
	// still references id.
	DeleteLevel(ctx context.Context, id string) error

	// Role. A live Role cannot be edited directly — fork a Draft, edit the
	// draft-scoped copy, then promote. List's draftID selects scope, same
	// as Goal/Workflow/WorkItem.
	CreateRole(ctx context.Context, r models.Role) (models.Role, error)
	GetRole(ctx context.Context, id string) (models.Role, error)
	ListRoles(ctx context.Context, draftID string) ([]models.Role, error)
	// UpdateRole/DeleteRole return [ErrCannotEditLiveRACI] if the target
	// row is live.
	UpdateRole(ctx context.Context, r models.Role) (models.Role, error)
	DeleteRole(ctx context.Context, id string) error

	// RoleAssignment. Assigns a Role to a WorkItem with a RACILabel; same
	// live-immutability rule as Role.
	//
	// CreateRoleAssignment/UpdateRoleAssignment return [ErrInvalidRACILabel]
	// for an invalid Label, [ErrRoleNotFound]/[ErrWorkItemNotFound] if the
	// referenced Role/WorkItem doesn't exist in the same scope, or
	// [ErrDuplicateAccountable] if another assignment in that scope already
	// labels the WorkItem Accountable.
	CreateRoleAssignment(ctx context.Context, a models.RoleAssignment) (models.RoleAssignment, error)
	GetRoleAssignment(ctx context.Context, id string) (models.RoleAssignment, error)
	ListRoleAssignments(ctx context.Context, draftID string) ([]models.RoleAssignment, error)
	// ListRoleAssignmentsForWorkItem answers "who is assigned to this
	// WorkItem, and with what RACI label".
	ListRoleAssignmentsForWorkItem(ctx context.Context, workItemID, draftID string) ([]models.RoleAssignment, error)
	// UpdateRoleAssignment/DeleteRoleAssignment return
	// [ErrCannotEditLiveRACI] if the target row is live.
	UpdateRoleAssignment(ctx context.Context, a models.RoleAssignment) (models.RoleAssignment, error)
	DeleteRoleAssignment(ctx context.Context, id string) error
}

type agencyManager struct {
	db     *gorm.DB
	tables TableNames
}

// NewAgencyManager builds an [AgencyManager] backed by db, reading and
// writing the tables named by t.
func NewAgencyManager(db *gorm.DB, t TableNames) (AgencyManager, error) {
	if db == nil {
		return nil, fmt.Errorf("NewAgencyManager: db must not be nil")
	}
	return &agencyManager{db: db, tables: t}, nil
}
