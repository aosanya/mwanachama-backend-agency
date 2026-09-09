package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RoleRow is the GORM row for a [models.Role].
type RoleRow struct {
	ID          string  `gorm:"primaryKey"`
	DraftID     *string `gorm:"index"`
	Name        string
	Description string
	CreatedAt   string
	UpdatedAt   string
	Deleted     bool
}

func (r *RoleRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// RoleToRow converts a domain Role to its row shape, stamping UpdatedAt.
func RoleToRow(r models.Role) RoleRow {
	return RoleRow{
		ID:          r.ID,
		DraftID:     StringToNullable(r.DraftID),
		Name:        r.Name,
		Description: r.Description,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   models.NowRFC3339(),
		Deleted:     r.Deleted,
	}
}

// RoleFromRow converts a row back to the domain Role.
func RoleFromRow(row RoleRow) models.Role {
	return models.Role{
		ID:          row.ID,
		DraftID:     nullableToString(row.DraftID),
		Name:        row.Name,
		Description: row.Description,
		CreatedAt:   row.CreatedAt,
		LastUpdated: row.UpdatedAt,
		Deleted:     row.Deleted,
	}
}
