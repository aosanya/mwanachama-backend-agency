package mwanachamaagency

import (
	"context"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) ListPermissions(ctx context.Context) ([]models.Permission, error) {
	return models.PermissionCatalog(), nil
}
