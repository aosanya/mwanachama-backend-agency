package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkItemRow is the GORM row for a [models.WorkItem].
//
// WorkflowID/ObjectiveID are indexed but deliberately not GORM
// associations/foreign keys, matching mwanachama-backend-actor's
// GroupRow.ParentID precedent (this package's tables are scoped per
// mounted instance via db.Table(...), which a GORM-managed association
// would not follow).
type WorkItemRow struct {
	ID          string  `gorm:"primaryKey"`
	WorkflowID  string  `gorm:"index"`
	ObjectiveID string  `gorm:"index"`
	DraftID     *string `gorm:"index"`
	Title       string
	Description string
	Ordinality  int
	Prompt      string
	CreatedAt   string
	UpdatedAt   string
	Deleted     bool
}

func (r *WorkItemRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// WorkItemToRow converts a domain WorkItem to its row shape, stamping UpdatedAt.
func WorkItemToRow(w models.WorkItem) WorkItemRow {
	return WorkItemRow{
		ID:          w.ID,
		WorkflowID:  w.WorkflowID,
		ObjectiveID: w.ObjectiveID,
		DraftID:     StringToNullable(w.DraftID),
		Title:       w.Title,
		Description: w.Description,
		Ordinality:  w.Ordinality,
		Prompt:      w.Prompt,
		CreatedAt:   w.CreatedAt,
		UpdatedAt:   models.NowRFC3339(),
		Deleted:     w.Deleted,
	}
}

// WorkItemFromRow converts a row back to the domain WorkItem.
func WorkItemFromRow(r WorkItemRow) models.WorkItem {
	return models.WorkItem{
		ID:          r.ID,
		WorkflowID:  r.WorkflowID,
		ObjectiveID: r.ObjectiveID,
		DraftID:     nullableToString(r.DraftID),
		Title:       r.Title,
		Description: r.Description,
		Ordinality:  r.Ordinality,
		Prompt:      r.Prompt,
		CreatedAt:   r.CreatedAt,
		LastUpdated: r.UpdatedAt,
		Deleted:     r.Deleted,
	}
}

// StringToNullable maps the domain "" (no value) to a nil *string.
func StringToNullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nullableToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
