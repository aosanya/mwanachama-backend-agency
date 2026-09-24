package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) validateRoleAssignment(ctx context.Context, tx *gorm.DB, a models.RoleAssignment, excludeID string) error {
	if !a.Label.Valid() {
		return ErrInvalidRACILabel
	}

	var r models.Role
	q := scopeQuery(tx.WithContext(ctx).Table(m.table(roleRole)), a.DraftID).Where("id = ?", a.RoleID)
	if err := m.take(q, roleRole, &r, ErrRoleNotFound); err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			return ErrRoleNotFound
		}
		return fmt.Errorf("validateRoleAssignment: %w", err)
	}

	var o models.Objective
	q = scopeQuery(tx.WithContext(ctx).Table(m.table(roleObjective)), a.DraftID).Where("id = ?", a.ObjectiveID)
	if err := m.take(q, roleObjective, &o, ErrObjectiveNotFound); err != nil {
		if errors.Is(err, ErrObjectiveNotFound) {
			return ErrObjectiveNotFound
		}
		return fmt.Errorf("validateRoleAssignment: %w", err)
	}

	if a.Label != models.RACILabelAccountable {
		return nil
	}

	held := scopeQuery(tx.WithContext(ctx).Table(m.table(roleRoleAssignment)), a.DraftID).
		Where("objective_id = ? AND label = ?", a.ObjectiveID, string(models.RACILabelAccountable))
	if excludeID != "" {
		held = held.Where("id <> ?", excludeID)
	}
	var count int64
	if err := held.Count(&count).Error; err != nil {
		return fmt.Errorf("validateRoleAssignment: %w", err)
	}
	if count > 0 {
		return ErrDuplicateAccountable
	}
	return nil
}

func (m *agencyManager) CreateRoleAssignment(ctx context.Context, a models.RoleAssignment) (models.RoleAssignment, error) {
	if err := m.validateOpenDraftID(ctx, a.DraftID); err != nil {
		return models.RoleAssignment{}, err
	}
	if err := m.validateRoleAssignment(ctx, m.db, a, ""); err != nil {
		return models.RoleAssignment{}, err
	}
	a.ID = newID()
	a.CreatedAt = models.NowRFC3339()
	a.LastUpdated = a.CreatedAt

	if err := m.insert(ctx, roleRoleAssignment, a); err != nil {
		return models.RoleAssignment{}, fmt.Errorf("CreateRoleAssignment: %w", err)
	}
	return a, nil
}

func (m *agencyManager) GetRoleAssignment(ctx context.Context, id string) (models.RoleAssignment, error) {
	return m.findRoleAssignment(ctx, m.db, id)
}

func (m *agencyManager) ListRoleAssignments(ctx context.Context, draftID string, filter RoleAssignmentFilter) ([]models.RoleAssignment, error) {
	q := eqFilter(scopeQuery(m.q(ctx, roleRoleAssignment), draftID), "label", filter.Label)

	out, err := list[models.RoleAssignment](m, q, roleRoleAssignment)
	if err != nil {
		return nil, fmt.Errorf("ListRoleAssignments: %w", err)
	}
	return out, nil
}

func (m *agencyManager) ListRoleAssignmentsForObjective(ctx context.Context, objectiveID, draftID string, filter RoleAssignmentFilter) ([]models.RoleAssignment, error) {
	q := scopeQuery(m.q(ctx, roleRoleAssignment), draftID).Where("objective_id = ?", objectiveID)
	q = eqFilter(q, "label", filter.Label)

	out, err := list[models.RoleAssignment](m, q, roleRoleAssignment)
	if err != nil {
		return nil, fmt.Errorf("ListRoleAssignmentsForObjective: %w", err)
	}
	return out, nil
}

func (m *agencyManager) UpdateRoleAssignment(ctx context.Context, a models.RoleAssignment) (models.RoleAssignment, error) {
	var out models.RoleAssignment
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := m.findRoleAssignment(ctx, tx, a.ID)
		if err != nil {
			return err
		}
		a.DraftID = existing.DraftID
		if err := m.validateRoleAssignment(ctx, tx, a, a.ID); err != nil {
			return err
		}
		a.CreatedAt = existing.CreatedAt
		a.LastUpdated = models.NowRFC3339()

		if err := m.writeUpdate(ctx, tx, roleRoleAssignment, a.ID, a, ErrRoleAssignmentNotFound, "draft_id"); err != nil {
			return err
		}
		out = a
		return nil
	})
	if err != nil {
		return models.RoleAssignment{}, fmt.Errorf("UpdateRoleAssignment: %w", err)
	}
	return out, nil
}

func (m *agencyManager) DeleteRoleAssignment(ctx context.Context, id string) error {
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := m.findRoleAssignment(ctx, tx, id); err != nil {
			return err
		}
		return m.deleteWhere(ctx, tx, roleRoleAssignment, "id = ?", id)
	})
	if err != nil {
		return fmt.Errorf("DeleteRoleAssignment: %w", err)
	}
	return nil
}

func (m *agencyManager) findRoleAssignment(ctx context.Context, tx *gorm.DB, id string) (models.RoleAssignment, error) {
	var a models.RoleAssignment
	if err := m.find(ctx, tx, roleRoleAssignment, id, &a, ErrRoleAssignmentNotFound); err != nil {
		return models.RoleAssignment{}, err
	}
	return a, nil
}
