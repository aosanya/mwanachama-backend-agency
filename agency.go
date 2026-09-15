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
	// Agency. UpdateAgency only ever touches Name/Description —
	// Mission/Vision were removed 2026-09-10 (AGD-001); Description was
	// added back 2026-09-11 at the user's explicit request. Nothing else
	// has asked to edit Enabled/DefaultFailurePipelineBudget/
	// InactivityTimeoutSeconds/EventFlows yet.
	CreateAgency(ctx context.Context, a models.Agency) (models.Agency, error)
	GetAgency(ctx context.Context) (models.Agency, error)
	UpdateAgency(ctx context.Context, a models.Agency) (models.Agency, error)

	// Goal. List's draftID selects scope: "" lists live Goals, a Draft's
	// ID lists that draft's copies.
	CreateGoal(ctx context.Context, g models.Goal) (models.Goal, error)
	GetGoal(ctx context.Context, id string) (models.Goal, error)
	ListGoals(ctx context.Context, draftID string, filter GoalFilter) ([]models.Goal, error)
	UpdateGoal(ctx context.Context, g models.Goal) (models.Goal, error)
	// DeleteGoal cascades: every Objective scoped to it, and every
	// RoleAssignment scoped to each of those Objectives, is deleted too —
	// bypassing RoleAssignment/Objective's own independent-edit rules
	// (Role's live-immutability check exists to stop a live RACI row
	// being tampered with piecemeal, not to block a cascade from the
	// Goal it already hangs off, which itself carries no such
	// restriction).
	DeleteGoal(ctx context.Context, id string) error

	// Objective. A Goal may have several — each one a granular unit a
	// single Role is RACI-assigned to via RoleAssignment. Same
	// Draft-scoping/list convention as Goal.
	CreateObjective(ctx context.Context, o models.Objective) (models.Objective, error)
	GetObjective(ctx context.Context, id string) (models.Objective, error)
	ListObjectives(ctx context.Context, draftID string, filter ObjectiveFilter) ([]models.Objective, error)
	UpdateObjective(ctx context.Context, o models.Objective) (models.Objective, error)
	// DeleteObjective cascades: every RoleAssignment scoped to it is
	// deleted too — same bypass of RoleAssignment's own independent-edit
	// rule as DeleteGoal's cascade (see its doc comment above).
	DeleteObjective(ctx context.Context, id string) error

	// Workflow
	CreateWorkflow(ctx context.Context, w models.Workflow) (models.Workflow, error)
	GetWorkflow(ctx context.Context, id string) (models.Workflow, error)
	ListWorkflows(ctx context.Context, draftID string, filter WorkflowFilter) ([]models.Workflow, error)
	UpdateWorkflow(ctx context.Context, w models.Workflow) (models.Workflow, error)
	// DeleteWorkflow cascades: every WorkItem owned by it is deleted too —
	// same reasoning as DeleteGoal's cascade above (see its doc comment).
	// A WorkItem carries no RACI of its own (RoleAssignment is
	// Objective-scoped, AG12), so there is nothing further to cascade to.
	DeleteWorkflow(ctx context.Context, id string) error

	// WorkItem. Belongs to exactly one Objective — CreateWorkItem/
	// UpdateWorkItem return [ErrInvalidWorkItem] if ObjectiveID is empty.
	CreateWorkItem(ctx context.Context, wi models.WorkItem) (models.WorkItem, error)
	GetWorkItem(ctx context.Context, id string) (models.WorkItem, error)
	ListWorkItems(ctx context.Context, draftID string, filter WorkItemFilter) ([]models.WorkItem, error)
	UpdateWorkItem(ctx context.Context, wi models.WorkItem) (models.WorkItem, error)

	// Draft lifecycle. CreateDraft forks the sub-graph scoped to
	// forkedFromID (an Agency.ID when forkedFromType=="live", or another
	// open Draft's ID when forkedFromType=="draft") into new rows scoped
	// to the new Draft. PromoteDraft replaces live data with the Draft's
	// copies; ArchiveDraft discards a Draft without promoting it. Returns
	// [ErrDraftNotOpen] where noted.
	CreateDraft(ctx context.Context, description, forkedFromID, forkedFromType string) (models.Draft, error)
	GetDraft(ctx context.Context, id string) (models.Draft, error)
	ListDrafts(ctx context.Context, filter DraftFilter) ([]models.Draft, error)
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
	ListPublications(ctx context.Context, filter PublicationFilter) ([]models.AgencyPublication, error)
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
	ListEventFlowSteps(ctx context.Context, workflowCode string, filter EventFlowStepFilter) ([]models.EventFlowStep, error)

	// WorkPlan. CreateWorkPlan/UpdateWorkPlan return [ErrInvalidRegex] if
	// TriggerTopic or PayloadCondition does not compile as a Go regexp.
	CreateWorkPlan(ctx context.Context, w models.WorkPlan) (models.WorkPlan, error)
	GetWorkPlan(ctx context.Context, id string) (models.WorkPlan, error)
	ListWorkPlans(ctx context.Context, filter WorkPlanFilter) ([]models.WorkPlan, error)
	UpdateWorkPlan(ctx context.Context, w models.WorkPlan) (models.WorkPlan, error)
	// MatchWorkPlans evaluates topic/payload against every enabled
	// WorkPlan's TriggerTopic/PayloadCondition regex, returning matches
	// (with their ContextSources) ordered by Ordinality ascending.
	MatchWorkPlans(ctx context.Context, topic, payload string) ([]models.WorkPlanMatch, error)

	// ContextSource. AddContextSource/ListContextSources return
	// [ErrWorkPlanNotFound] if workPlanID names no WorkPlan.
	AddContextSource(ctx context.Context, workPlanID string, cs models.ContextSource) (models.ContextSource, error)
	ListContextSources(ctx context.Context, workPlanID string, filter ContextSourceFilter) ([]models.ContextSource, error)
	// RemoveContextSource returns [ErrContextSourceNotFound] if sourceID
	// names no ContextSource, or names one belonging to a different
	// WorkPlan than workPlanID.
	RemoveContextSource(ctx context.Context, workPlanID, sourceID string) error

	// Department. The org chart is a local, self-contained parent/child
	// tree (no dependency on mwanachama-backend-actor's own Group/Hierarchy
	// tables).
	CreateDepartment(ctx context.Context, d models.Department) (models.Department, error)
	GetDepartment(ctx context.Context, id string) (models.Department, error)
	ListDepartments(ctx context.Context, filter DepartmentFilter) ([]models.Department, error)
	ListDepartmentChildren(ctx context.Context, parentID string, filter DepartmentFilter) ([]models.Department, error)
	RenameDepartment(ctx context.Context, id, name string) (models.Department, error)
	// MoveDepartment returns [ErrParentIsSelf], [ErrRootCannotMove],
	// [ErrParentNotFound], or [ErrParentInSubtree] where noted on each.
	MoveDepartment(ctx context.Context, id, newParentID string) (models.Department, error)

	// Level. A named rung in the Department tree.
	CreateLevel(ctx context.Context, l models.Level) (models.Level, error)
	GetLevel(ctx context.Context, id string) (models.Level, error)
	ListLevels(ctx context.Context, filter LevelFilter) ([]models.Level, error)
	RenameLevel(ctx context.Context, id, name string) (models.Level, error)
	// DeleteLevel returns [ErrLevelWornByDepartments] if any Department
	// still references id.
	DeleteLevel(ctx context.Context, id string) error

	// Role. A live Role cannot be edited directly — fork a Draft, edit the
	// draft-scoped copy, then promote. List's draftID selects scope, same
	// as Goal/Workflow/WorkItem.
	CreateRole(ctx context.Context, r models.Role) (models.Role, error)
	GetRole(ctx context.Context, id string) (models.Role, error)
	ListRoles(ctx context.Context, draftID string, filter RoleFilter) ([]models.Role, error)
	// UpdateRole/DeleteRole return [ErrCannotEditLiveRACI] if the target
	// row is live.
	UpdateRole(ctx context.Context, r models.Role) (models.Role, error)
	DeleteRole(ctx context.Context, id string) error

	// RoleAssignment. Assigns a Role to an Objective with a RACILabel
	// (moved from WorkItem-scoped 2026-09-10, AGD-001); same
	// live-immutability rule as Role.
	//
	// CreateRoleAssignment/UpdateRoleAssignment return [ErrInvalidRACILabel]
	// for an invalid Label, [ErrRoleNotFound]/[ErrObjectiveNotFound] if the
	// referenced Role/Objective doesn't exist in the same scope, or
	// [ErrDuplicateAccountable] if another assignment in that scope already
	// labels the Objective Accountable.
	CreateRoleAssignment(ctx context.Context, a models.RoleAssignment) (models.RoleAssignment, error)
	GetRoleAssignment(ctx context.Context, id string) (models.RoleAssignment, error)
	ListRoleAssignments(ctx context.Context, draftID string, filter RoleAssignmentFilter) ([]models.RoleAssignment, error)
	// ListRoleAssignmentsForObjective answers "who is assigned to this
	// Objective, and with what RACI label" — a caller that needs a
	// WorkItem's RACI resolves via WorkItem.ObjectiveID first.
	ListRoleAssignmentsForObjective(ctx context.Context, objectiveID, draftID string, filter RoleAssignmentFilter) ([]models.RoleAssignment, error)
	// UpdateRoleAssignment/DeleteRoleAssignment return
	// [ErrCannotEditLiveRACI] if the target row is live.
	UpdateRoleAssignment(ctx context.Context, a models.RoleAssignment) (models.RoleAssignment, error)
	DeleteRoleAssignment(ctx context.Context, id string) error

	// Value/Rule. Plain-text statements an AI agent acting as a Role
	// needs made explicit (see models.Value/models.Rule's doc comments).
	// ObjectiveID == "" means Agency-wide: those are meant to be set once
	// (at Agency creation/seed time) and never edited — nothing here
	// stops a caller from trying, but [CreateDraft] never forks them, so
	// UpdateValue/DeleteValue's [ErrCannotEditLive] check makes it
	// permanent in practice. Objective-scoped ones are Draft-scoped, same
	// live-immutability rule as Role. The write-authorization rule from
	// AGD-001 (only the Objective's Accountable may write) is NOT
	// enforced here — flagged as new machinery needing its own design.
	CreateValue(ctx context.Context, v models.Value) (models.Value, error)
	GetValue(ctx context.Context, id string) (models.Value, error)
	ListValues(ctx context.Context, draftID string, filter ValueFilter) ([]models.Value, error)
	// UpdateValue/DeleteValue return [ErrCannotEditLive] if the target row
	// is live.
	UpdateValue(ctx context.Context, v models.Value) (models.Value, error)
	DeleteValue(ctx context.Context, id string) error

	CreateRule(ctx context.Context, r models.Rule) (models.Rule, error)
	GetRule(ctx context.Context, id string) (models.Rule, error)
	ListRules(ctx context.Context, draftID string, filter RuleFilter) ([]models.Rule, error)
	// UpdateRule/DeleteRule return [ErrCannotEditLive] if the target row
	// is live.
	UpdateRule(ctx context.Context, r models.Rule) (models.Rule, error)
	DeleteRule(ctx context.Context, id string) error

	// Issue. A Draft-scoped, system-detected diagnostics list (info/
	// warning/error) — e.g. an Objective with no Accountable, a Goal with
	// no Objectives. ListIssues re-scans on every call: it deletes
	// draftID's existing Issue rows, re-runs every check against that
	// Draft's current data, re-inserts, then returns the (possibly
	// filtered) result — so there is no Create/Update/Delete surface;
	// nothing here is ever authored directly. Returns [ErrDraftIDRequired]
	// if draftID is "" — unlike every other List method above, an Issue
	// never exists against the live Agency.
	ListIssues(ctx context.Context, draftID string, filter IssueFilter) ([]models.Issue, error)
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
