package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AgencyRow is the GORM row for a [models.Agency].
type AgencyRow struct {
	ID                           string `gorm:"primaryKey"`
	Name                         string
	Mission                      string
	Vision                       string
	Enabled                      bool
	DefaultFailurePipelineBudget int
	InactivityTimeoutSeconds     int
	EventFlows                   string
	CreatedAt                    string
	UpdatedAt                    string
	Deleted                      bool
}

func (r *AgencyRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// AgencyToRow converts a domain Agency to its row shape, stamping UpdatedAt.
func AgencyToRow(a models.Agency) AgencyRow {
	return AgencyRow{
		ID:                           a.ID,
		Name:                         a.Name,
		Mission:                      a.Mission,
		Vision:                       a.Vision,
		Enabled:                      a.Enabled,
		DefaultFailurePipelineBudget: a.DefaultFailurePipelineBudget,
		InactivityTimeoutSeconds:     a.InactivityTimeoutSeconds,
		EventFlows:                   a.EventFlows,
		CreatedAt:                    a.CreatedAt,
		UpdatedAt:                    models.NowRFC3339(),
		Deleted:                      a.Deleted,
	}
}

// AgencyFromRow converts a row back to the domain Agency.
func AgencyFromRow(r AgencyRow) models.Agency {
	return models.Agency{
		ID:                           r.ID,
		Name:                         r.Name,
		Mission:                      r.Mission,
		Vision:                       r.Vision,
		Enabled:                      r.Enabled,
		DefaultFailurePipelineBudget: r.DefaultFailurePipelineBudget,
		InactivityTimeoutSeconds:     r.InactivityTimeoutSeconds,
		EventFlows:                   r.EventFlows,
		CreatedAt:                    r.CreatedAt,
		LastUpdated:                  r.UpdatedAt,
		Deleted:                      r.Deleted,
	}
}
