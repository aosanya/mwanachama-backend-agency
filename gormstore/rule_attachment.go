package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RuleAttachmentRow is the GORM row for a [models.RuleAttachment]. See
// [ValueAttachmentRow]'s doc comment (same fixed-composite-index-name
// pitfall applies here).
type RuleAttachmentRow struct {
	ID         string `gorm:"primaryKey"`
	RuleCode   string `gorm:"index"`
	TargetType string `gorm:"index"`
	TargetCode string `gorm:"index"`
	CreatedAt  string
}

func (r *RuleAttachmentRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// RuleAttachmentToRow converts a domain RuleAttachment to its row shape.
func RuleAttachmentToRow(a models.RuleAttachment) RuleAttachmentRow {
	return RuleAttachmentRow{
		ID:         a.ID,
		RuleCode:   a.RuleCode,
		TargetType: string(a.TargetType),
		TargetCode: a.TargetCode,
		CreatedAt:  a.CreatedAt,
	}
}

// RuleAttachmentFromRow converts a row back to the domain RuleAttachment.
func RuleAttachmentFromRow(r RuleAttachmentRow) models.RuleAttachment {
	return models.RuleAttachment{
		ID:         r.ID,
		RuleCode:   r.RuleCode,
		TargetType: models.TargetType(r.TargetType),
		TargetCode: r.TargetCode,
		CreatedAt:  r.CreatedAt,
	}
}
