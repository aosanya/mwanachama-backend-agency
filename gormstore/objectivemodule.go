package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ObjectiveModuleRow is the GORM row for a [models.ObjectiveModule].
type ObjectiveModuleRow struct {
	ID          string `gorm:"primaryKey"`
	DraftID     string `gorm:"index;not null"`
	ObjectiveID string `gorm:"index"`
	ModuleID    string `gorm:"index"`
	Rationale   string
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
		DraftID:     om.DraftID,
		ObjectiveID: om.ObjectiveID,
		ModuleID:    om.ModuleID,
		Rationale:   om.Rationale,
		CreatedAt:   om.CreatedAt,
		UpdatedAt:   models.NowRFC3339(),
		Deleted:     om.Deleted,
	}
}

// ObjectiveModuleFromRow converts a row back to the domain ObjectiveModule.
func ObjectiveModuleFromRow(row ObjectiveModuleRow) models.ObjectiveModule {
	return models.ObjectiveModule{
		ID:          row.ID,
		DraftID:     row.DraftID,
		ObjectiveID: row.ObjectiveID,
		ModuleID:    row.ModuleID,
		Rationale:   row.Rationale,
		CreatedAt:   row.CreatedAt,
		LastUpdated: row.UpdatedAt,
		Deleted:     row.Deleted,
	}
}
