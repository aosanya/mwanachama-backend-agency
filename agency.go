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
