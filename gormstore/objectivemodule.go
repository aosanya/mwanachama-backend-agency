package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ObjectiveModuleRow is the GORM row for a [models.ObjectiveModule].
type ObjectiveModuleRow struct {
	ID          string  `gorm:"primaryKey"`
	DraftID     *string `gorm:"index"`
	ObjectiveID string  `gorm:"index"`
	ModuleID    string  `gorm:"index"`
	CreatedAt   string
	UpdatedAt   string
	Deleted     bool
}

func (r *ObjectiveModuleRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// ObjectiveModuleToRow converts a domain ObjectiveModule to its row shape,
// stamping UpdatedAt.
func ObjectiveModuleToRow(om models.ObjectiveModule) ObjectiveModuleRow {
	return ObjectiveModuleRow{
		ID:          om.ID,
		DraftID:     StringToNullable(om.DraftID),
		ObjectiveID: om.ObjectiveID,
		ModuleID:    om.ModuleID,
		CreatedAt:   om.CreatedAt,
		UpdatedAt:   models.NowRFC3339(),
		Deleted:     om.Deleted,
	}
}

// ObjectiveModuleFromRow converts a row back to the domain ObjectiveModule.
func ObjectiveModuleFromRow(row ObjectiveModuleRow) models.ObjectiveModule {
	return models.ObjectiveModule{
		ID:          row.ID,
		DraftID:     nullableToString(row.DraftID),
		ObjectiveID: row.ObjectiveID,
		ModuleID:    row.ModuleID,
		CreatedAt:   row.CreatedAt,
		LastUpdated: row.UpdatedAt,
		Deleted:     row.Deleted,
	}
}
