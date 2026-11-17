package mwanachamaagency

import (
	"context"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func RegisterComputeStepChecks() {
	registerComputeStep()
}

func PlantStep(mgr AgencyManager, wi models.WorkItem) (models.WorkItem, error) {
	m := mgr.(*agencyManager)
	ctx := context.Background()
	wi.ID = newID()
	wi.CreatedAt = models.NowRFC3339()
	wi.LastUpdated = wi.CreatedAt
	err := m.createWithCode(ctx, roleWorkItem, "workitem", "WI", func(code string) any {
		wi.Code = code
		return wi
	})
	return wi, err
}
