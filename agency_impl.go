package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) CreateAgency(ctx context.Context, a models.Agency) (models.Agency, error) {
	if a.CreatedAt == "" {
		a.CreatedAt = models.NowRFC3339()
	}
	row := gormstore.AgencyToRow(a)
	if err := m.db.WithContext(ctx).Table(m.tables.Agencies).Create(&row).Error; err != nil {
		return models.Agency{}, fmt.Errorf("CreateAgency: %w", err)
	}
	return gormstore.AgencyFromRow(row), nil
}

// GetAgency fetches the sole Agency row (single-agency deployment).
func (m *agencyManager) GetAgency(ctx context.Context) (models.Agency, error) {
	row, err := m.findAgencyRow(ctx, m.db)
	if err != nil {
		return models.Agency{}, err
	}
	return gormstore.AgencyFromRow(row), nil
}

func (m *agencyManager) findAgencyRow(ctx context.Context, tx *gorm.DB) (gormstore.AgencyRow, error) {
	var row gormstore.AgencyRow
	err := tx.WithContext(ctx).Table(m.tables.Agencies).Order("created_at").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.AgencyRow{}, ErrAgencyNotFound
	}
	if err != nil {
		return gormstore.AgencyRow{}, fmt.Errorf("findAgencyRow: %w", err)
	}
	return row, nil
}
