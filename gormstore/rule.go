package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RuleRow is the GORM row for a [models.Rule]. Same shape/conventions as
// ValueRow.
type RuleRow struct {
	ID          string  `gorm:"primaryKey"`
	Code        string  `gorm:"index"`
	ObjectiveID *string `gorm:"index"`
	DraftID     *string `gorm:"index"`
	Text        string
	CreatedAt   string
	UpdatedAt   string
	Deleted     bool
}

func (r *RuleRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// RuleToRow converts a domain Rule to its row shape, stamping UpdatedAt.
func RuleToRow(v models.Rule) RuleRow {
	return RuleRow{
		ID:          v.ID,
		Code:        v.Code,
		ObjectiveID: StringToNullable(v.ObjectiveID),
		DraftID:     StringToNullable(v.DraftID),
		Text:        v.Text,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   models.NowRFC3339(),
		Deleted:     v.Deleted,
	}
}

// RuleFromRow converts a row back to the domain Rule.
func RuleFromRow(r RuleRow) models.Rule {
	return models.Rule{
		ID:          r.ID,
		Code:        r.Code,
		ObjectiveID: nullableToString(r.ObjectiveID),
		DraftID:     nullableToString(r.DraftID),
		Text:        r.Text,
		CreatedAt:   r.CreatedAt,
		LastUpdated: r.UpdatedAt,
		Deleted:     r.Deleted,
	}
}
