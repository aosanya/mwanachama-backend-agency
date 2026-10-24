package mwanachamaagency

import (
	"context"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) ListPowers(ctx context.Context) ([]models.Power, error) {
	return models.PowerCatalog(), nil
}
