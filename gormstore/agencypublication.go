package gormstore

import (
	"encoding/json"

	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AgencyPublicationRow is the GORM row for a [models.AgencyPublication].
// Document holds the published Agency as JSON text rather than a typed
// column: it is written once and read whole, never queried into, so a
// portable text column beats a driver-specific JSON type here.
type AgencyPublicationRow struct {
	ID          string `gorm:"primaryKey"`
	DraftID     string `gorm:"index"`
	ContentHash string `gorm:"index"`
	Version     int    `gorm:"index"`
	Tag         string `gorm:"index"`
	Channel     string `gorm:"index"`
	Document    string
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
// shape, stamping UpdatedAt and marshalling the document.
func AgencyPublicationToRow(p models.AgencyPublication) AgencyPublicationRow {
	document, err := json.Marshal(p.Document)
	if err != nil {
		// AgencyDocument is plain structs and strings throughout — there
		// is no value of it json.Marshal can reject.
		document = []byte("{}")
	}
	return AgencyPublicationRow{
		ID:          p.ID,
		DraftID:     p.DraftID,
		ContentHash: p.ContentHash,
		Version:     p.Version,
		Tag:         p.Tag,
		Channel:     p.Channel,
		Document:    string(document),
		PublishedAt: p.PublishedAt,
		UpdatedAt:   models.NowRFC3339(),
	}
}

// AgencyPublicationFromRow converts a row back to the domain
// AgencyPublication. A row written before AGD-005 carries no document at
// all; its zero value is returned rather than an error, matching
// [models.AgencyPublication.Document]'s own note.
func AgencyPublicationFromRow(r AgencyPublicationRow) models.AgencyPublication {
	var document models.AgencyDocument
	if r.Document != "" {
		_ = json.Unmarshal([]byte(r.Document), &document)
	}
	return models.AgencyPublication{
		ID:          r.ID,
		DraftID:     r.DraftID,
		ContentHash: r.ContentHash,
		Version:     r.Version,
		Tag:         r.Tag,
		Channel:     r.Channel,
		Document:    document,
		PublishedAt: r.PublishedAt,
		LastUpdated: r.UpdatedAt,
	}
}
