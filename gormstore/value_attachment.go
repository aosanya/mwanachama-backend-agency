package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ValueAttachmentRow is the GORM row for a [models.ValueAttachment]. No
// Code (a join row, matching RoleAssignmentRow's precedent) and no
// DraftID — attachments are live-editable, never Draft-scoped (AGD-003).
type ValueAttachmentRow struct {
	ID         string `gorm:"primaryKey"`
	ValueCode  string `gorm:"index:idx_value_attachment_value"`
	TargetType string `gorm:"index:idx_value_attachment_target"`
	TargetCode string `gorm:"index:idx_value_attachment_target"`
	CreatedAt  string
}

func (r *ValueAttachmentRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// ValueAttachmentToRow converts a domain ValueAttachment to its row shape.
func ValueAttachmentToRow(a models.ValueAttachment) ValueAttachmentRow {
	return ValueAttachmentRow{
		ID:         a.ID,
		ValueCode:  a.ValueCode,
		TargetType: string(a.TargetType),
		TargetCode: a.TargetCode,
		CreatedAt:  a.CreatedAt,
	}
}

// ValueAttachmentFromRow converts a row back to the domain ValueAttachment.
func ValueAttachmentFromRow(r ValueAttachmentRow) models.ValueAttachment {
	return models.ValueAttachment{
		ID:         r.ID,
		ValueCode:  r.ValueCode,
		TargetType: models.TargetType(r.TargetType),
		TargetCode: r.TargetCode,
		CreatedAt:  r.CreatedAt,
	}
}
