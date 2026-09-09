package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RoleAssignmentRow is the GORM row for a [models.RoleAssignment].
type RoleAssignmentRow struct {
	ID         string  `gorm:"primaryKey"`
	DraftID    *string `gorm:"index"`
	RoleID     string  `gorm:"index"`
	WorkItemID string  `gorm:"index"`
	Label      string
	CreatedAt  string
	UpdatedAt  string
	Deleted    bool
}

func (r *RoleAssignmentRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// RoleAssignmentToRow converts a domain RoleAssignment to its row shape,
// stamping UpdatedAt.
func RoleAssignmentToRow(a models.RoleAssignment) RoleAssignmentRow {
	return RoleAssignmentRow{
		ID:         a.ID,
		DraftID:    StringToNullable(a.DraftID),
		RoleID:     a.RoleID,
		WorkItemID: a.WorkItemID,
		Label:      string(a.Label),
		CreatedAt:  a.CreatedAt,
		UpdatedAt:  models.NowRFC3339(),
		Deleted:    a.Deleted,
	}
}

// RoleAssignmentFromRow converts a row back to the domain RoleAssignment.
func RoleAssignmentFromRow(row RoleAssignmentRow) models.RoleAssignment {
	return models.RoleAssignment{
		ID:          row.ID,
		DraftID:     nullableToString(row.DraftID),
		RoleID:      row.RoleID,
		WorkItemID:  row.WorkItemID,
		Label:       models.RACILabel(row.Label),
		CreatedAt:   row.CreatedAt,
		LastUpdated: row.UpdatedAt,
		Deleted:     row.Deleted,
	}
}
