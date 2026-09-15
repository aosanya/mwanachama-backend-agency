package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// CreateModule requires a non-empty, same-scope-unique Code — unlike Goal's
// Code, Module's is caller-supplied rather than minted via
// [gormstore.NextCode] (AGD-002), so nothing else in the schema enforces
// this uniqueness.
func (m *agencyManager) CreateModule(ctx context.Context, mod models.Module) (models.Module, error) {
	if err := m.validateOpenDraftID(ctx, mod.DraftID); err != nil {
		return models.Module{}, err
	}
	if mod.Code == "" {
		return models.Module{}, ErrInvalidModule
	}
	mod.ID = ""
	mod.CreatedAt = models.NowRFC3339()
	var row gormstore.ModuleRow
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := scopeQuery(tx.Table(m.tables.Modules), mod.DraftID).
			Where("code = ?", mod.Code).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrDuplicateModule
		}
		row = gormstore.ModuleToRow(mod)
		return tx.Table(m.tables.Modules).Create(&row).Error
	})
	if err != nil {
		return models.Module{}, fmt.Errorf("CreateModule: %w", err)
	}
	return gormstore.ModuleFromRow(row), nil
}

func (m *agencyManager) GetModule(ctx context.Context, id string) (models.Module, error) {
	row, err := m.findModuleRow(ctx, m.db, id)
	if err != nil {
		return models.Module{}, err
	}
	return gormstore.ModuleFromRow(row), nil
}

// ListModules returns live Modules when draftID is "", or draftID's
// draft-scoped copies otherwise, narrowed by filter.
func (m *agencyManager) ListModules(ctx context.Context, draftID string, filter ModuleFilter) ([]models.Module, error) {
	var rows []gormstore.ModuleRow
	q := scopeQuery(m.db.WithContext(ctx).Table(m.tables.Modules), draftID)
	q = likeFilter(q, "code", filter.Code)
	q = likeFilter(q, "name", filter.Name)
	q = likeFilter(q, "description", filter.Description)
	q = eqBoolFilter(q, "enabled", filter.Enabled)
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListModules: %w", err)
	}
	out := make([]models.Module, len(rows))
	for i, r := range rows {
		out[i] = gormstore.ModuleFromRow(r)
	}
	return out, nil
}

func (m *agencyManager) findModuleRow(ctx context.Context, tx *gorm.DB, id string) (gormstore.ModuleRow, error) {
	var row gormstore.ModuleRow
	err := tx.WithContext(ctx).Table(m.tables.Modules).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.ModuleRow{}, ErrModuleNotFound
	}
	if err != nil {
		return gormstore.ModuleRow{}, fmt.Errorf("findModuleRow: %w", err)
	}
	return row, nil
}
