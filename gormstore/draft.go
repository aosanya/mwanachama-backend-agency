package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DraftRow is the GORM row for a [models.Draft]. Status stores
// models.DraftStatus as a plain string column, cast at the
// DraftToRow/DraftFromRow boundary.
type DraftRow struct {
	ID             string `gorm:"primaryKey"`
	Description    string
	ForkedFromID   string
	ForkedFromType string
	Status         string
	CreatedAt      string
	UpdatedAt      string
	Deleted        bool
}

func (r *DraftRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// DraftToRow converts a domain Draft to its row shape, stamping UpdatedAt.
func DraftToRow(d models.Draft) DraftRow {
	return DraftRow{
		ID:             d.ID,
		Description:    d.Description,
		ForkedFromID:   d.ForkedFromID,
		ForkedFromType: d.ForkedFromType,
		Status:         string(d.Status),
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      models.NowRFC3339(),
		Deleted:        d.Deleted,
	}
}

// DraftFromRow converts a row back to the domain Draft.
func DraftFromRow(r DraftRow) models.Draft {
	return models.Draft{
		ID:             r.ID,
		Description:    r.Description,
		ForkedFromID:   r.ForkedFromID,
		ForkedFromType: r.ForkedFromType,
		Status:         models.DraftStatus(r.Status),
		CreatedAt:      r.CreatedAt,
		LastUpdated:    r.UpdatedAt,
		Deleted:        r.Deleted,
	}
}
