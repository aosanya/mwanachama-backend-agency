package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LevelRow is the GORM row for a [models.Level].
type LevelRow struct {
	ID        string `gorm:"primaryKey"`
	Code      string `gorm:"uniqueIndex"`
	Name      string
	Depth     int
	CreatedAt string
	UpdatedAt string
	Deleted   bool
}

func (r *LevelRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// LevelToRow converts a domain Level to its row shape, stamping UpdatedAt.
func LevelToRow(l models.Level) LevelRow {
	return LevelRow{
		ID:        l.ID,
		Code:      l.Code,
		Name:      l.Name,
		Depth:     l.Depth,
		CreatedAt: l.CreatedAt,
		UpdatedAt: models.NowRFC3339(),
		Deleted:   l.Deleted,
	}
}

// LevelFromRow converts a row back to the domain Level.
func LevelFromRow(r LevelRow) models.Level {
	return models.Level{
		ID:          r.ID,
		Code:        r.Code,
		Name:        r.Name,
		Depth:       r.Depth,
		CreatedAt:   r.CreatedAt,
		LastUpdated: r.UpdatedAt,
		Deleted:     r.Deleted,
	}
}
