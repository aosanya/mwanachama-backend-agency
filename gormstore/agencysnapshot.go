package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AgencySnapshotRow is the GORM row for a [models.AgencySnapshot].
type AgencySnapshotRow struct {
	ID         string `gorm:"primaryKey"`
	DraftID    string `gorm:"index"`
	SnapshotAt string
}

func (r *AgencySnapshotRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// AgencySnapshotToRow converts a domain AgencySnapshot to its row shape.
func AgencySnapshotToRow(s models.AgencySnapshot) AgencySnapshotRow {
	return AgencySnapshotRow{
		ID:         s.ID,
		DraftID:    s.DraftID,
		SnapshotAt: s.SnapshotAt,
	}
}

// AgencySnapshotFromRow converts a row back to the domain AgencySnapshot.
func AgencySnapshotFromRow(r AgencySnapshotRow) models.AgencySnapshot {
	return models.AgencySnapshot{
		ID:         r.ID,
		DraftID:    r.DraftID,
		SnapshotAt: r.SnapshotAt,
	}
}
