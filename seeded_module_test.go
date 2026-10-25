package mwanachamaagency_test

import (
	"context"
	"fmt"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func seededModule(mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID, code string) (models.Module, error) {
	mods, err := mgr.ListModules(ctx, draftID, mwanachamaagency.ModuleFilter{Code: code})
	if err != nil {
		return models.Module{}, err
	}
	for _, m := range mods {
		if m.Code == code {
			return m, nil
		}
	}
	return models.Module{}, fmt.Errorf("no seeded module %q in draft %q", code, draftID)
}
