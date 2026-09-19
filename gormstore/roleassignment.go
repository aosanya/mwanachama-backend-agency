package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RoleAssignmentRow is the GORM row for a [models.RoleAssignment].
type RoleAssignmentRow struct {
	ID          string `gorm:"primaryKey"`
	DraftID     string `gorm:"index;not null"`
	RoleID      string `gorm:"index"`
	ObjectiveID string `gorm:"index"`
	Label       string
	CreatedAt   string
	UpdatedAt   string
	Deleted     bool
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
		ID:          a.ID,
		DraftID:     a.DraftID,
		RoleID:      a.RoleID,
		ObjectiveID: a.ObjectiveID,
		Label:       string(a.Label),
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   models.NowRFC3339(),
		Deleted:     a.Deleted,
	}
}

// RoleAssignmentFromRow converts a row back to the domain RoleAssignment.
func RoleAssignmentFromRow(row RoleAssignmentRow) models.RoleAssignment {
	return models.RoleAssignment{
		ID:          row.ID,
		DraftID:     row.DraftID,
		RoleID:      row.RoleID,
		ObjectiveID: row.ObjectiveID,
		Label:       models.RACILabel(row.Label),
		CreatedAt:   row.CreatedAt,
		LastUpdated: row.UpdatedAt,
		Deleted:     row.Deleted,
	}
}
