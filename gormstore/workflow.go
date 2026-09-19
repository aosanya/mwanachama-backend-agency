package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkflowRow is the GORM row for a [models.Workflow]. DraftID always names
// a real Draft (AGD-006 — there is no live row-scope) — indexed, no GORM
// association.
type WorkflowRow struct {
	ID          string `gorm:"primaryKey"`
	Code        string `gorm:"index"`
	DraftID     string `gorm:"index;not null"`
	Name        string
	Description string
	Ordinality  int
	EventFlows  string
	CreatedAt   string
	UpdatedAt   string
	Deleted     bool
}

func (r *WorkflowRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// WorkflowToRow converts a domain Workflow to its row shape, stamping UpdatedAt.
func WorkflowToRow(w models.Workflow) WorkflowRow {
	return WorkflowRow{
		ID:          w.ID,
		Code:        w.Code,
		DraftID:     w.DraftID,
		Name:        w.Name,
		Description: w.Description,
		Ordinality:  w.Ordinality,
		EventFlows:  w.EventFlows,
		CreatedAt:   w.CreatedAt,
		UpdatedAt:   models.NowRFC3339(),
		Deleted:     w.Deleted,
	}
}

// WorkflowFromRow converts a row back to the domain Workflow.
func WorkflowFromRow(r WorkflowRow) models.Workflow {
	return models.Workflow{
		ID:          r.ID,
		Code:        r.Code,
		DraftID:     r.DraftID,
		Name:        r.Name,
		Description: r.Description,
		Ordinality:  r.Ordinality,
		EventFlows:  r.EventFlows,
		CreatedAt:   r.CreatedAt,
		LastUpdated: r.UpdatedAt,
		Deleted:     r.Deleted,
	}
}
