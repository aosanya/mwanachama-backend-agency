package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AgencyPublicationRow is the GORM row for a [models.AgencyPublication].
type AgencyPublicationRow struct {
	ID          string `gorm:"primaryKey"`
	DraftID     string `gorm:"index"`
	ContentHash string `gorm:"index"`
	Version     int    `gorm:"index"`
	Tag         string
	Status      string `gorm:"index"`
	PublishedAt string
	UpdatedAt   string
}

func (r *AgencyPublicationRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// AgencyPublicationToRow converts a domain AgencyPublication to its row
// shape, stamping UpdatedAt.
func AgencyPublicationToRow(p models.AgencyPublication) AgencyPublicationRow {
	return AgencyPublicationRow{
		ID:          p.ID,
		DraftID:     p.DraftID,
		ContentHash: p.ContentHash,
		Version:     p.Version,
		Tag:         p.Tag,
		Status:      p.Status,
		PublishedAt: p.PublishedAt,
		UpdatedAt:   models.NowRFC3339(),
	}
}

// AgencyPublicationFromRow converts a row back to the domain AgencyPublication.
func AgencyPublicationFromRow(r AgencyPublicationRow) models.AgencyPublication {
	return models.AgencyPublication{
		ID:          r.ID,
		DraftID:     r.DraftID,
		ContentHash: r.ContentHash,
		Version:     r.Version,
		Tag:         r.Tag,
		Status:      r.Status,
		PublishedAt: r.PublishedAt,
		LastUpdated: r.UpdatedAt,
	}
}
