package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// validateObjectiveModule checks that om's Objective and Module both exist
// within om's own scope (om.DraftID) — a Draft's Objective can only link a
// Module already forked into that same Draft; a live Objective can only
// link a live Module (AGD-002 decision #6, the same same-scope rule AG6
// established for RoleAssignment).
func (m *agencyManager) validateObjectiveModule(ctx context.Context, tx *gorm.DB, om models.ObjectiveModule) error {
	var objectiveRow gormstore.ObjectiveRow
	err := scopeQuery(tx.WithContext(ctx).Table(m.tables.Objectives), om.DraftID).
		Where("id = ?", om.ObjectiveID).First(&objectiveRow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrObjectiveNotFound
	}
	if err != nil {
		return fmt.Errorf("validateObjectiveModule: %w", err)
	}

	var moduleRow gormstore.ModuleRow
	err = scopeQuery(tx.WithContext(ctx).Table(m.tables.Modules), om.DraftID).
		Where("id = ?", om.ModuleID).First(&moduleRow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrModuleNotFound
	}
	if err != nil {
		return fmt.Errorf("validateObjectiveModule: %w", err)
	}
	return nil
}

// CreateObjectiveModule returns [ErrRationaleRequired] when om.Rationale is
// blank, and [ErrDuplicateObjectiveModule] if the
// (ObjectiveID, ModuleID) pair is already linked in this scope.
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
	om.ID = ""
	om.CreatedAt = models.NowRFC3339()
	var row gormstore.ObjectiveModuleRow
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := scopeQuery(tx.Table(m.tables.ObjectiveModules), om.DraftID).
			Where("objective_id = ? AND module_id = ?", om.ObjectiveID, om.ModuleID).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrDuplicateObjectiveModule
		}
		row = gormstore.ObjectiveModuleToRow(om)
		return tx.Table(m.tables.ObjectiveModules).Create(&row).Error
	})
	if err != nil {
		return models.ObjectiveModule{}, fmt.Errorf("CreateObjectiveModule: %w", err)
	}
	return gormstore.ObjectiveModuleFromRow(row), nil
}

// ListObjectiveModules returns live links when draftID is "", or draftID's
// draft-scoped copies otherwise, narrowed by filter.
func (m *agencyManager) ListObjectiveModules(ctx context.Context, draftID string, filter ObjectiveModuleFilter) ([]models.ObjectiveModule, error) {
	q := scopeQuery(m.db.WithContext(ctx).Table(m.tables.ObjectiveModules), draftID)
	q = eqFilter(q, "objective_id", filter.ObjectiveID)
	q = eqFilter(q, "module_id", filter.ModuleID)
	var rows []gormstore.ObjectiveModuleRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListObjectiveModules: %w", err)
	}
	out := make([]models.ObjectiveModule, len(rows))
	for i, r := range rows {
		out[i] = gormstore.ObjectiveModuleFromRow(r)
	}
	return out, nil
}

// ListObjectiveModulesForObjective answers "which Modules does this
// Objective link, in this scope" — mirrors
// ListRoleAssignmentsForObjective's shape.
func (m *agencyManager) ListObjectiveModulesForObjective(ctx context.Context, objectiveID, draftID string, filter ObjectiveModuleFilter) ([]models.ObjectiveModule, error) {
	q := scopeQuery(m.db.WithContext(ctx).Table(m.tables.ObjectiveModules), draftID).
		Where("objective_id = ?", objectiveID)
	q = eqFilter(q, "module_id", filter.ModuleID)
	var rows []gormstore.ObjectiveModuleRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListObjectiveModulesForObjective: %w", err)
	}
	out := make([]models.ObjectiveModule, len(rows))
	for i, r := range rows {
		out[i] = gormstore.ObjectiveModuleFromRow(r)
	}
	return out, nil
}

func (m *agencyManager) UpdateObjectiveModule(ctx context.Context, id, rationale string) (models.ObjectiveModule, error) {
	rationale = strings.TrimSpace(rationale)
	if rationale == "" {
		return models.ObjectiveModule{}, ErrRationaleRequired
	}
	row, err := m.findObjectiveModuleRow(ctx, m.db, id)
	if err != nil {
		return models.ObjectiveModule{}, err
	}
	row.Rationale = rationale
	row.UpdatedAt = models.NowRFC3339()
	if err := m.db.WithContext(ctx).Table(m.tables.ObjectiveModules).Where("id = ?", id).
		Updates(map[string]any{"rationale": row.Rationale, "updated_at": row.UpdatedAt}).Error; err != nil {
		return models.ObjectiveModule{}, fmt.Errorf("UpdateObjectiveModule: %w", err)
	}
	return gormstore.ObjectiveModuleFromRow(row), nil
}

func (m *agencyManager) DeleteObjectiveModule(ctx context.Context, id string) error {
	if _, err := m.findObjectiveModuleRow(ctx, m.db, id); err != nil {
		return err
	}
	if err := m.db.WithContext(ctx).Table(m.tables.ObjectiveModules).Where("id = ?", id).
		Delete(&gormstore.ObjectiveModuleRow{}).Error; err != nil {
		return fmt.Errorf("DeleteObjectiveModule: %w", err)
	}
	return nil
}

func (m *agencyManager) findObjectiveModuleRow(ctx context.Context, tx *gorm.DB, id string) (gormstore.ObjectiveModuleRow, error) {
	var row gormstore.ObjectiveModuleRow
	err := tx.WithContext(ctx).Table(m.tables.ObjectiveModules).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.ObjectiveModuleRow{}, ErrObjectiveModuleNotFound
	}
	if err != nil {
		return gormstore.ObjectiveModuleRow{}, fmt.Errorf("findObjectiveModuleRow: %w", err)
	}
	return row, nil
}
