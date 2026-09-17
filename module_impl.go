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
// this uniqueness. A caller-supplied Enabled is not honoured: a fresh
// Module links no Objective yet, so it comes back disabled unless Locked
// (see [models.Module]).
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
	return withDerivedEnabled(gormstore.ModuleFromRow(row), false), nil
}

func (m *agencyManager) GetModule(ctx context.Context, id string) (models.Module, error) {
	row, err := m.findModuleRow(ctx, m.db, id)
	if err != nil {
		return models.Module{}, err
	}
	mod := gormstore.ModuleFromRow(row)
	linked, err := m.linkedModuleIDs(ctx, mod.DraftID)
	if err != nil {
		return models.Module{}, fmt.Errorf("GetModule: %w", err)
	}
	return withDerivedEnabled(mod, linked[mod.ID]), nil
}

// ListModules returns live Modules when draftID is "", or draftID's
// draft-scoped copies otherwise, narrowed by filter.
func (m *agencyManager) ListModules(ctx context.Context, draftID string, filter ModuleFilter) ([]models.Module, error) {
	var rows []gormstore.ModuleRow
	q := scopeQuery(m.db.WithContext(ctx).Table(m.tables.Modules), draftID)
	q = likeFilter(q, "code", filter.Code)
	q = likeFilter(q, "name", filter.Name)
	q = likeFilter(q, "description", filter.Description)
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListModules: %w", err)
	}
	linked, err := m.linkedModuleIDs(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("ListModules: %w", err)
	}
	// Enabled is derived, not a column, so filter.Enabled is applied here
	// rather than pushed into the query alongside the others.
	out := make([]models.Module, 0, len(rows))
	for _, r := range rows {
		mod := withDerivedEnabled(gormstore.ModuleFromRow(r), linked[r.ID])
		if filter.Enabled != nil && mod.Enabled != *filter.Enabled {
			continue
		}
		out = append(out, mod)
	}
	return out, nil
}

// linkedModuleIDs is the set of Module ids some Objective in draftID's own
// scope links through an ObjectiveModule row — the one input a non-Locked
// Module's Enabled state has (see [models.Module]).
func (m *agencyManager) linkedModuleIDs(ctx context.Context, draftID string) (map[string]bool, error) {
	var ids []string
	if err := scopeQuery(m.db.WithContext(ctx).Table(m.tables.ObjectiveModules), draftID).
		Pluck("module_id", &ids).Error; err != nil {
		return nil, err
	}
	linked := make(map[string]bool, len(ids))
	for _, id := range ids {
		linked[id] = true
	}
	return linked, nil
}

// withDerivedEnabled overwrites mod.Enabled with its derived state: a
// Locked Module keeps the catalog state it was seeded with — it can never
// be disabled — and every other Module is enabled exactly when an
// Objective in its own scope links it.
func withDerivedEnabled(mod models.Module, linked bool) models.Module {
	if mod.Locked {
		return mod
	}
	mod.Enabled = linked
	return mod
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
