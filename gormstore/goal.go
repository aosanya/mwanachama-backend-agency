package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GoalRow is the GORM row for a [models.Goal].
type GoalRow struct {
	ID          string `gorm:"primaryKey"`
	Title       string
	Description string
	Ordinality  int
	CreatedAt   string
	UpdatedAt   string
	Deleted     bool
}

func (r *GoalRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// GoalToRow converts a domain Goal to its row shape, stamping UpdatedAt.
func GoalToRow(g models.Goal) GoalRow {
	return GoalRow{
		ID:          g.ID,
		Title:       g.Title,
		Description: g.Description,
		Ordinality:  g.Ordinality,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   models.NowRFC3339(),
		Deleted:     g.Deleted,
	}
}

// GoalFromRow converts a row back to the domain Goal.
func GoalFromRow(r GoalRow) models.Goal {
	return models.Goal{
		ID:          r.ID,
		Title:       r.Title,
		Description: r.Description,
		Ordinality:  r.Ordinality,
		CreatedAt:   r.CreatedAt,
		LastUpdated: r.UpdatedAt,
		Deleted:     r.Deleted,
	}
}
