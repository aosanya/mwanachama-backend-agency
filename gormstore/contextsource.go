package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ContextSourceRow is the GORM row for a [models.ContextSource]. All three
// configs' fields are flattened onto one row, prefixed by kind
// (Git*/Comm*/Work*) — GORM has no polymorphic-entity concept the way
// CodeValdAgency's entitygraph did, so only the columns matching SourceType
// are ever populated; the rest stay zero.
type ContextSourceRow struct {
	ID         string `gorm:"primaryKey"`
	WorkPlanID string `gorm:"index"`
	SourceType string

	GitSignals    string
	GitMaxResults int
	GitMatchMode  string
	GitCascade    bool
	GitFileTypes  string

	CommLookbackDays int
	CommMaxResults   int

	WorkIncludeDescription bool
	WorkIncludeHistory     bool

	CreatedAt string
}

func (r *ContextSourceRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// ContextSourceToRow converts a domain ContextSource to its row shape.
func ContextSourceToRow(cs models.ContextSource) ContextSourceRow {
	row := ContextSourceRow{
		ID:         cs.ID,
		WorkPlanID: cs.WorkPlanID,
		SourceType: string(cs.SourceType),
		CreatedAt:  cs.CreatedAt,
	}
	if cs.Git != nil {
		row.GitSignals = cs.Git.Signals
		row.GitMaxResults = cs.Git.MaxResults
		row.GitMatchMode = cs.Git.MatchMode
		row.GitCascade = cs.Git.Cascade
		row.GitFileTypes = cs.Git.FileTypes
	}
	if cs.Comm != nil {
		row.CommLookbackDays = cs.Comm.LookbackDays
		row.CommMaxResults = cs.Comm.MaxResults
	}
	if cs.Work != nil {
		row.WorkIncludeDescription = cs.Work.IncludeDescription
		row.WorkIncludeHistory = cs.Work.IncludeHistory
	}
	return row
}

// ContextSourceFromRow converts a row back to the domain ContextSource,
// populating only the config matching r.SourceType.
func ContextSourceFromRow(r ContextSourceRow) models.ContextSource {
	cs := models.ContextSource{
		ID:         r.ID,
		WorkPlanID: r.WorkPlanID,
		SourceType: models.ContextSourceType(r.SourceType),
		CreatedAt:  r.CreatedAt,
	}
	switch cs.SourceType {
	case models.ContextSourceGit:
		cs.Git = &models.GitContextSourceConfig{
			Signals:    r.GitSignals,
			MaxResults: r.GitMaxResults,
			MatchMode:  r.GitMatchMode,
			Cascade:    r.GitCascade,
			FileTypes:  r.GitFileTypes,
		}
	case models.ContextSourceComm:
		cs.Comm = &models.CommContextSourceConfig{
			LookbackDays: r.CommLookbackDays,
			MaxResults:   r.CommMaxResults,
		}
	case models.ContextSourceWork:
		cs.Work = &models.WorkContextSourceConfig{
			IncludeDescription: r.WorkIncludeDescription,
			IncludeHistory:     r.WorkIncludeHistory,
		}
	}
	return cs
}
