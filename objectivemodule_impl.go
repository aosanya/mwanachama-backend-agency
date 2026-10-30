package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) validateObjectiveModule(ctx context.Context, tx *gorm.DB, om models.ObjectiveModule) error {
	var o models.Objective
	q := scopeQuery(tx.WithContext(ctx).Table(m.table(roleObjective)), om.DraftID).Where("id = ?", om.ObjectiveID)
	if err := m.take(q, roleObjective, &o, ErrObjectiveNotFound); err != nil {
		if errors.Is(err, ErrObjectiveNotFound) {
			return ErrObjectiveNotFound
		}
		return fmt.Errorf("validateObjectiveModule: %w", err)
	}

	var mod models.Module
	q = scopeQuery(tx.WithContext(ctx).Table(m.table(roleModule)), om.DraftID).Where("id = ?", om.ModuleID)
	if err := m.take(q, roleModule, &mod, ErrModuleNotFound); err != nil {
		if errors.Is(err, ErrModuleNotFound) {
			return ErrModuleNotFound
		}
		return fmt.Errorf("validateObjectiveModule: %w", err)
	}
	return nil
}

func (m *agencyManager) CreateObjectiveModule(ctx context.Context, om models.ObjectiveModule) (models.ObjectiveModule, error) {
	if err := m.validateOpenDraftID(ctx, om.DraftID); err != nil {
		return models.ObjectiveModule{}, err
	}
	if err := m.validateObjectiveModule(ctx, m.db, om); err != nil {
		return models.ObjectiveModule{}, err
	}
	om.Rationale = strings.TrimSpace(om.Rationale)
	if om.Rationale == "" {
		return models.ObjectiveModule{}, ErrRationaleRequired
	}
	om.ID = newID()
	om.Deleted = false
	om.CreatedAt = models.NowRFC3339()
	om.LastUpdated = om.CreatedAt

	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := scopeQuery(tx.Table(m.table(roleObjectiveModule)), om.DraftID).
			Where("objective_id = ? AND module_id = ?", om.ObjectiveID, om.ModuleID).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrDuplicateObjectiveModule
		}
		return m.copyRow(ctx, tx, roleObjectiveModule, om)
	})
	if err != nil {
		return models.ObjectiveModule{}, fmt.Errorf("CreateObjectiveModule: %w", err)
	}
	return om, nil
}

func (m *agencyManager) ListObjectiveModules(ctx context.Context, draftID string, filter ObjectiveModuleFilter) ([]models.ObjectiveModule, error) {
	q := scopeQuery(m.q(ctx, roleObjectiveModule), draftID)
	q = eqFilter(q, "objective_id", filter.ObjectiveID)
	q = eqFilter(q, "module_id", filter.ModuleID)

	out, err := list[models.ObjectiveModule](m, q, roleObjectiveModule)
	if err != nil {
		return nil, fmt.Errorf("ListObjectiveModules: %w", err)
	}
	return out, nil
}

func (m *agencyManager) ListObjectiveModulesForObjective(ctx context.Context, objectiveID, draftID string, filter ObjectiveModuleFilter) ([]models.ObjectiveModule, error) {
	q := scopeQuery(m.q(ctx, roleObjectiveModule), draftID).Where("objective_id = ?", objectiveID)
	q = eqFilter(q, "module_id", filter.ModuleID)

	out, err := list[models.ObjectiveModule](m, q, roleObjectiveModule)
	if err != nil {
		return nil, fmt.Errorf("ListObjectiveModulesForObjective: %w", err)
	}
	return out, nil
}

func (m *agencyManager) UpdateObjectiveModule(ctx context.Context, id, rationale string) (models.ObjectiveModule, error) {
	rationale = strings.TrimSpace(rationale)
	if rationale == "" {
		return models.ObjectiveModule{}, ErrRationaleRequired
	}

	var out models.ObjectiveModule
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		om, err := m.findObjectiveModule(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := m.requireOpenDraft(ctx, tx, om.DraftID); err != nil {
			return err
		}
		om.Rationale = rationale
		om.LastUpdated = models.NowRFC3339()

		if err := m.writeUpdate(ctx, tx, roleObjectiveModule, id, om, ErrObjectiveModuleNotFound, "draft_id"); err != nil {
			return err
		}
		out = om
		return nil
	})
	if err != nil {
		return models.ObjectiveModule{}, fmt.Errorf("UpdateObjectiveModule: %w", err)
	}
	return out, nil
}

func (m *agencyManager) DeleteObjectiveModule(ctx context.Context, id string) error {
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := m.findObjectiveModule(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := m.requireOpenDraft(ctx, tx, existing.DraftID); err != nil {
			return err
		}
		return m.deleteWhere(ctx, tx, roleObjectiveModule, "id = ?", id)
	})
	if err != nil {
		return fmt.Errorf("DeleteObjectiveModule: %w", err)
	}
	return nil
}

func (m *agencyManager) findObjectiveModule(ctx context.Context, tx *gorm.DB, id string) (models.ObjectiveModule, error) {
	var om models.ObjectiveModule
	if err := m.find(ctx, tx, roleObjectiveModule, id, &om, ErrObjectiveModuleNotFound); err != nil {
		return models.ObjectiveModule{}, err
	}
	return om, nil
}
