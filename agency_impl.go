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

// UpdateAgency updates the sole Agency row's name and description
// (single-agency deployment, same lookup as GetAgency — a's own ID is
// ignored). Enabled/DefaultFailurePipelineBudget/InactivityTimeoutSeconds/
// EventFlows/Deleted are untouched; nothing has asked to edit those yet.
// Mission/Vision were removed 2026-09-10 (AGD-001); Description was added
// back 2026-09-11 at the user's explicit request.
func (m *agencyManager) UpdateAgency(ctx context.Context, a models.Agency) (models.Agency, error) {
	existing, err := m.findAgencyRow(ctx, m.db)
	if err != nil {
		return models.Agency{}, err
	}
	a.ID = existing.ID
	a.CreatedAt = existing.CreatedAt
	row := gormstore.AgencyToRow(a)
	cols := map[string]any{
		"name":        row.Name,
		"description": row.Description,
		"updated_at":  row.UpdatedAt,
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Agencies).Where("id = ?", existing.ID).Updates(cols).Error; err != nil {
		return models.Agency{}, fmt.Errorf("UpdateAgency: %w", err)
	}
	row.Enabled = existing.Enabled
	row.DefaultFailurePipelineBudget = existing.DefaultFailurePipelineBudget
	row.InactivityTimeoutSeconds = existing.InactivityTimeoutSeconds
	row.EventFlows = existing.EventFlows
	row.Deleted = existing.Deleted
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
