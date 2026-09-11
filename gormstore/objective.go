package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ObjectiveRow is the GORM row for a [models.Objective]. GoalID is
// indexed but deliberately not a GORM association/foreign key, matching
// WorkItemRow.WorkflowID/GoalID's precedent.
type ObjectiveRow struct {
	ID          string  `gorm:"primaryKey"`
	Code        string  `gorm:"index"`
	GoalID      string  `gorm:"index"`
	DraftID     *string `gorm:"index"`
	Title       string
	Description string
	Ordinality  int
	CreatedAt   string
	UpdatedAt   string
	Deleted     bool
}

func (r *ObjectiveRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// ObjectiveToRow converts a domain Objective to its row shape, stamping
// UpdatedAt.
func ObjectiveToRow(o models.Objective) ObjectiveRow {
	return ObjectiveRow{
		ID:          o.ID,
		Code:        o.Code,
		GoalID:      o.GoalID,
		DraftID:     StringToNullable(o.DraftID),
		Title:       o.Title,
		Description: o.Description,
		Ordinality:  o.Ordinality,
		CreatedAt:   o.CreatedAt,
		UpdatedAt:   models.NowRFC3339(),
		Deleted:     o.Deleted,
	}
}

// ObjectiveFromRow converts a row back to the domain Objective.
func ObjectiveFromRow(r ObjectiveRow) models.Objective {
	return models.Objective{
		ID:          r.ID,
		Code:        r.Code,
		GoalID:      r.GoalID,
		DraftID:     nullableToString(r.DraftID),
		Title:       r.Title,
		Description: r.Description,
		Ordinality:  r.Ordinality,
		CreatedAt:   r.CreatedAt,
		LastUpdated: r.UpdatedAt,
		Deleted:     r.Deleted,
	}
}
