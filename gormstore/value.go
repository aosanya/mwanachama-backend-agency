package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ValueRow is the GORM row for a [models.Value]. ObjectiveID is nullable
// (NULL = agency-wide), indexed, deliberately not a GORM
// association/foreign key — matching WorkItemRow.GoalID's precedent.
type ValueRow struct {
	ID          string  `gorm:"primaryKey"`
	ObjectiveID *string `gorm:"index"`
	DraftID     *string `gorm:"index"`
	Text        string
	CreatedAt   string
	UpdatedAt   string
	Deleted     bool
}

func (r *ValueRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// ValueToRow converts a domain Value to its row shape, stamping UpdatedAt.
func ValueToRow(v models.Value) ValueRow {
	return ValueRow{
		ID:          v.ID,
		ObjectiveID: StringToNullable(v.ObjectiveID),
		DraftID:     StringToNullable(v.DraftID),
		Text:        v.Text,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   models.NowRFC3339(),
		Deleted:     v.Deleted,
	}
}

// ValueFromRow converts a row back to the domain Value.
func ValueFromRow(r ValueRow) models.Value {
	return models.Value{
		ID:          r.ID,
		ObjectiveID: nullableToString(r.ObjectiveID),
		DraftID:     nullableToString(r.DraftID),
		Text:        r.Text,
		CreatedAt:   r.CreatedAt,
		LastUpdated: r.UpdatedAt,
		Deleted:     r.Deleted,
	}
}
