package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ValueRow is the GORM row for a [models.Value]. Scoping (Agency-wide vs.
// attached to a Role/Goal/Objective/Workflow/WorkItem) lives entirely in
// ValueAttachmentRow now (AGD-003) — this row carries no target reference
// of its own.
type ValueRow struct {
	ID        string  `gorm:"primaryKey"`
	Code      string  `gorm:"index"`
	DraftID   *string `gorm:"index"`
	Text      string
	CreatedAt string
	UpdatedAt string
	Deleted   bool
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
		ID:        v.ID,
		Code:      v.Code,
		DraftID:   StringToNullable(v.DraftID),
		Text:      v.Text,
		CreatedAt: v.CreatedAt,
		UpdatedAt: models.NowRFC3339(),
		Deleted:   v.Deleted,
	}
}

// ValueFromRow converts a row back to the domain Value.
func ValueFromRow(r ValueRow) models.Value {
	return models.Value{
		ID:          r.ID,
		Code:        r.Code,
		DraftID:     nullableToString(r.DraftID),
		Text:        r.Text,
		CreatedAt:   r.CreatedAt,
		LastUpdated: r.UpdatedAt,
		Deleted:     r.Deleted,
	}
}
