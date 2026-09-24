package mwanachamaagency

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) CreateModule(ctx context.Context, mod models.Module) (models.Module, error) {
	if err := m.validateOpenDraftID(ctx, mod.DraftID); err != nil {
		return models.Module{}, err
	}
	if mod.Code == "" {
		return models.Module{}, ErrInvalidModule
	}
	mod.ID = newID()
	mod.CreatedAt = models.NowRFC3339()
	mod.LastUpdated = mod.CreatedAt

	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := scopeQuery(tx.Table(m.table(roleModule)), mod.DraftID).
			Where("code = ?", mod.Code).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrDuplicateModule
		}
		return m.copyRow(ctx, tx, roleModule, mod)
	})
	if err != nil {
		return models.Module{}, fmt.Errorf("CreateModule: %w", err)
	}
	return withDerivedEnabled(mod, false), nil
}

func (m *agencyManager) GetModule(ctx context.Context, id string) (models.Module, error) {
	mod, err := m.findModule(ctx, m.db, id)
	if err != nil {
		return models.Module{}, err
	}
	linked, err := m.linkedModuleIDs(ctx, mod.DraftID)
	if err != nil {
		return models.Module{}, fmt.Errorf("GetModule: %w", err)
	}
	return withDerivedEnabled(mod, linked[mod.ID]), nil
}

func (m *agencyManager) ListModules(ctx context.Context, draftID string, filter ModuleFilter) ([]models.Module, error) {
	q := scopeQuery(m.q(ctx, roleModule), draftID)
	q = likeFilter(q, "code", filter.Code)
	q = likeFilter(q, "name", filter.Name)
	q = likeFilter(q, "description", filter.Description)

	modules, err := list[models.Module](m, q, roleModule)
	if err != nil {
		return nil, fmt.Errorf("ListModules: %w", err)
	}
	linked, err := m.linkedModuleIDs(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("ListModules: %w", err)
	}

	out := make([]models.Module, 0, len(modules))
	for _, mod := range modules {
		mod = withDerivedEnabled(mod, linked[mod.ID])
		if filter.Enabled != nil && mod.Enabled != *filter.Enabled {
			continue
		}
		out = append(out, mod)
	}
	return out, nil
}

func (m *agencyManager) linkedModuleIDs(ctx context.Context, draftID string) (map[string]bool, error) {
	var ids []string
	if err := scopeQuery(m.q(ctx, roleObjectiveModule), draftID).
		Pluck("module_id", &ids).Error; err != nil {
		return nil, err
	}
	linked := make(map[string]bool, len(ids))
	for _, id := range ids {
		linked[id] = true
	}
	return linked, nil
}

func withDerivedEnabled(mod models.Module, linked bool) models.Module {
	if mod.Locked {
		return mod
	}
	mod.Enabled = linked
	return mod
}

func (m *agencyManager) findModule(ctx context.Context, tx *gorm.DB, id string) (models.Module, error) {
	var mod models.Module
	if err := m.find(ctx, tx, roleModule, id, &mod, ErrModuleNotFound); err != nil {
		return models.Module{}, err
	}
	return mod, nil
}
