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
	// RoleAssignment and WorkItem scoped to each of those Objectives, is
	// deleted too — bypassing RoleAssignment/Objective's own
	// independent-edit rules (Role's live-immutability check exists to
	// stop a live RACI row being tampered with piecemeal, not to block a
	// cascade from the Goal it already hangs off, which itself carries no
	// such restriction).
	DeleteGoal(ctx context.Context, id string) error

	// Objective. A Goal may have several — each one a granular unit a
	// single Role is RACI-assigned to via RoleAssignment. Same
	// Draft-scoping/list convention as Goal.
	CreateObjective(ctx context.Context, o models.Objective) (models.Objective, error)
	GetObjective(ctx context.Context, id string) (models.Objective, error)
	ListObjectives(ctx context.Context, draftID string, filter ObjectiveFilter) ([]models.Objective, error)
	UpdateObjective(ctx context.Context, o models.Objective) (models.Objective, error)
	// DeleteObjective cascades: every RoleAssignment and WorkItem scoped
	// to it is deleted too — same bypass of RoleAssignment's own
	// independent-edit rule as DeleteGoal's cascade (see its doc comment
	// above). WorkItem.ObjectiveID is required, so a surviving WorkItem
	// would be unreachable rather than merely untagged.
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
	// DeleteWorkItem carries no RACI of its own (RoleAssignment is
	// Objective-scoped, AG12), so there is nothing to cascade into.
	DeleteWorkItem(ctx context.Context, id string) error

	// Draft lifecycle. CreateDraft forks into new rows scoped to the new
	// Draft. forkedFromType selects the source: "live" forks the working
	// set (forkedFromID is the Agency's ID), "draft" forks another open
	// Draft (its ID), and "publication" forks a published tag —
	// forkedFromID is then the Tag, and the publication's document is
	// materialized into the Draft rather than rows being copied
	// ([ErrPublicationNotFound] if no publication carries that tag). That
	// last one is what "go back to what release-1 was and change one
	// thing" needs. PromoteDraft replaces the working set with the
	// Draft's copies; ArchiveDraft discards a Draft without promoting it.
	// Returns [ErrDraftNotOpen] where noted.
	CreateDraft(ctx context.Context, description, forkedFromID, forkedFromType string) (models.Draft, error)
	GetDraft(ctx context.Context, id string) (models.Draft, error)
	ListDrafts(ctx context.Context, filter DraftFilter) ([]models.Draft, error)
	// UpdateDraftDescription returns [ErrDraftNotOpen] if the draft isn't open.
	UpdateDraftDescription(ctx context.Context, id, description string) (models.Draft, error)
	// PromoteDraft returns [ErrDraftNotOpen] if the draft isn't open.
	PromoteDraft(ctx context.Context, id string) (models.Agency, error)
	// ArchiveDraft returns [ErrDraftNotOpen] if the draft isn't open.
	ArchiveDraft(ctx context.Context, id string) (models.Draft, error)

	// Publication. A publication carries the whole Agency as a
	// [models.AgencyDocument], and production operates from that rather
	// than from the working rows (AGD-005).
	//
	// PublishAgency freezes the working set into a new publication.
	// draftID, if non-"", is recorded as the Draft that prompted it and
	// must already be promoted ([ErrDraftNotPromoted] otherwise). tag
	// names it and must be unused ([ErrDuplicateTag]); "" falls back to
	// "v<version>". channel is the live channel it serves; "" means
	// [models.DefaultChannel], and whichever publication held that
	// channel is archived — publications on other channels keep serving
	// them. The scan runs first: any error-severity Issue refuses the
	// publish with [ErrPublishValidationFailed]. Returns
	// [ErrNoChangesDetected] if the content hash matches an existing
	// publication.
	PublishAgency(ctx context.Context, draftID, tag, channel string) (models.AgencyPublication, error)
	GetPublication(ctx context.Context, version int) (models.AgencyPublication, error)
	// GetPublicationForChannel is the production read: whichever
	// publication currently serves channel ("" means
	// [models.DefaultChannel]). [ErrPublicationNotFound] if none does.
	GetPublicationForChannel(ctx context.Context, channel string) (models.AgencyPublication, error)
	ListPublications(ctx context.Context, filter PublicationFilter) ([]models.AgencyPublication, error)
	// SetPublicationChannel moves a channel onto version, taking it off
	// whichever publication held it; [models.ChannelArchived] archives
	// version instead. Re-channelling an archived publication is the
	// rollback path, so both directions are allowed.
	SetPublicationChannel(ctx context.Context, version int, channel string) (models.AgencyPublication, error)

	// EventFlowStep
	CreateEventFlowStep(ctx context.Context, s models.EventFlowStep) (models.EventFlowStep, error)
	// LookupFlowStep tries lookup.HandlerCode, then
	// (lookup.TriggerTopic, lookup.Consumer) together, then lookup.Code, in
	// that order. Returns [ErrInvalidLookup] if none are set.
	LookupFlowStep(ctx context.Context, lookup models.EventFlowStepLookup) (models.EventFlowStep, error)
	// ListEventFlowSteps returns live steps ordered by (WorkflowCode,
	// Ordinality, Step), optionally filtered to one workflowCode.
	ListEventFlowSteps(ctx context.Context, workflowCode string, filter EventFlowStepFilter) ([]models.EventFlowStep, error)

	// Level. A named rung an org's nodes sit at ("National", "County",
	// "Faculty") — the ladder's vocabulary, which the Agency designs. The
	// nodes themselves are operational and live in
	// mwanachama-backend-actor's Group tables, not here: an Agency records
	// that it has a "Ward" rung, never which wards exist (AGD-017, which
	// removed the Department type this package used to carry).
	CreateLevel(ctx context.Context, l models.Level) (models.Level, error)
	GetLevel(ctx context.Context, id string) (models.Level, error)
	ListLevels(ctx context.Context, filter LevelFilter) ([]models.Level, error)
	RenameLevel(ctx context.Context, id, name string) (models.Level, error)
	DeleteLevel(ctx context.Context, id string) error

	// Role. A live Role cannot be edited directly — fork a Draft, edit the
	// draft-scoped copy, then promote. List's draftID selects scope, same
	// as Goal/Workflow/WorkItem.
	CreateRole(ctx context.Context, r models.Role) (models.Role, error)
	GetRole(ctx context.Context, id string) (models.Role, error)
	ListRoles(ctx context.Context, draftID string, filter RoleFilter) ([]models.Role, error)
	// UpdateRole/DeleteRole return [ErrCannotEditLiveRACI] if the target
	// row is live. DeleteRole cascades into every RoleAssignment naming
	// the Role, the mirror of DeleteObjective's cascade from the other
	// side of the same join table.
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

	// Module. A catalog entry for a capability domain (e.g. "git",
	// "taskmanager") this Agency may enable — a real gate on a mounting
	// gateway's route registration for that domain, not a passive record
	// (AGD-002). Fully read-only from the outside: no Update/Delete, and
	// no route/MCP tool calls CreateModule either — every Module code,
	// its Name/Description, and its default Enabled/Locked state come
	// from [models.ModuleCatalog], applied by [gormstore.BackfillModules]
	// at Migrate time. CreateModule stays on this interface only as
	// SeedSWELab's own fallback for a code Migrate somehow didn't
	// backfill; it returns [ErrInvalidModule] if Code is empty,
	// [ErrDuplicateModule] if Code already exists in that scope.
	CreateModule(ctx context.Context, mod models.Module) (models.Module, error)
	GetModule(ctx context.Context, id string) (models.Module, error)
	ListModules(ctx context.Context, draftID string, filter ModuleFilter) ([]models.Module, error)

	// ObjectiveModule. Links an Objective to a Module — many-to-many,
	// Draft-scoped, enforcing the same same-scope rule as RoleAssignment: a
	// Draft's Objective can only link a Module already forked into that
	// same Draft. CreateObjectiveModule returns [ErrObjectiveNotFound]/
	// [ErrModuleNotFound] if the referenced Objective/Module doesn't exist
	// in that scope, or [ErrDuplicateObjectiveModule] if the pair is
	// already linked.
	CreateObjectiveModule(ctx context.Context, om models.ObjectiveModule) (models.ObjectiveModule, error)
	ListObjectiveModules(ctx context.Context, draftID string, filter ObjectiveModuleFilter) ([]models.ObjectiveModule, error)
	// ListObjectiveModulesForObjective answers "which Modules does this
	// Objective link, in this scope."
	ListObjectiveModulesForObjective(ctx context.Context, objectiveID, draftID string, filter ObjectiveModuleFilter) ([]models.ObjectiveModule, error)
	DeleteObjectiveModule(ctx context.Context, id string) error

	// Value/Rule. Plain-text statements an AI agent acting as a Role
	// needs made explicit (see models.Value/models.Rule's doc comments).
	// A Value/Rule with no [ValueAttachment]/[RuleAttachment] is
	// Agency-wide: meant to be set once (at Agency creation/seed time)
	// and never edited — nothing here stops a caller from trying, but
	// [CreateDraft] never forks one with zero attachments, so
	// UpdateValue/DeleteValue's [ErrCannotEditLive] check makes it
	// permanent in practice. One with at least one attachment is
	// Draft-scoped for Text edits, same live-immutability rule as Role —
	// but attaching/detaching a target (AGD-003, 2026-09-15) is
	// live-editable and never goes through the Draft cycle at all, even
	// on a live row. The write-authorization rule from AGD-001 (only a
	// target's Accountable may write) is NOT enforced here — flagged as
	// new machinery needing its own design.
	CreateValue(ctx context.Context, v models.Value) (models.Value, error)
	GetValue(ctx context.Context, id string) (models.Value, error)
	// ListValues' filter can narrow to "attached to this
	// Role/Goal/Objective/Workflow/WorkItem" via ValueFilter's target-id
	// fields, alongside its existing Code/Text substring match.
	ListValues(ctx context.Context, draftID string, filter ValueFilter) ([]models.Value, error)
	// UpdateValue/DeleteValue return [ErrCannotEditLive] if the target row
	// is live.
	UpdateValue(ctx context.Context, v models.Value) (models.Value, error)
	DeleteValue(ctx context.Context, id string) error
	// CreateValueAttachment attaches valueID to one target. Returns
	// [ErrInvalidTargetType] for an unrecognized targetType, that target
	// type's own NotFound sentinel (e.g. [ErrRoleNotFound]) if targetID
	// names no row, or [ErrDuplicateAttachment] if already attached.
	CreateValueAttachment(ctx context.Context, valueID string, targetType models.TargetType, targetID string) (models.ValueAttachment, error)
	// ListValueAttachments lists every target valueID is attached to.
	ListValueAttachments(ctx context.Context, valueID string) ([]models.ValueAttachment, error)
	// DeleteValueAttachment returns [ErrValueAttachmentNotFound] if id
	// names no attachment.
	DeleteValueAttachment(ctx context.Context, id string) error

	CreateRule(ctx context.Context, r models.Rule) (models.Rule, error)
	GetRule(ctx context.Context, id string) (models.Rule, error)
	// ListRules' filter can narrow the same way ListValues' does.
	ListRules(ctx context.Context, draftID string, filter RuleFilter) ([]models.Rule, error)
	// UpdateRule/DeleteRule return [ErrCannotEditLive] if the target row
	// is live.
	UpdateRule(ctx context.Context, r models.Rule) (models.Rule, error)
	DeleteRule(ctx context.Context, id string) error
	// CreateRuleAttachment/ListRuleAttachments/DeleteRuleAttachment work
	// exactly like their Value counterparts above.
	CreateRuleAttachment(ctx context.Context, ruleID string, targetType models.TargetType, targetID string) (models.RuleAttachment, error)
	ListRuleAttachments(ctx context.Context, ruleID string) ([]models.RuleAttachment, error)
	DeleteRuleAttachment(ctx context.Context, id string) error

	// Issue. A system-detected diagnostics list (info/warning/error) — a
	// dangling reference, an Objective with no Accountable, a Goal with
	// no Objectives. draftID selects scope like every other scoped read:
	// "" scans the live Agency, a Draft's id scans that Draft.
	// ListIssues re-scans on every call: it deletes that scope's existing
	// Issue rows, re-runs every check against its current data,
	// re-inserts, then returns the (possibly filtered) result — so there
	// is no Create/Update/Delete surface; nothing here is ever authored
	// directly. An error-severity Issue is what PublishAgency refuses on.
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
