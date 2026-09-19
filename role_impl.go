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

func (m *agencyManager) CreateRole(ctx context.Context, r models.Role) (models.Role, error) {
	if r.Name == "" {
		return models.Role{}, fmt.Errorf("%w: Role.Name is required", ErrInvalidRole)
	}
	if err := m.validateOpenDraftID(ctx, r.DraftID); err != nil {
		return models.Role{}, err
	}
	r.ID = ""
	r.CreatedAt = models.NowRFC3339()
	var row gormstore.RoleRow
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		code, err := gormstore.NextCode(ctx, tx, m.tables.CodeSequences, "role", "R")
		if err != nil {
			return err
		}
		r.Code = code
		row = gormstore.RoleToRow(r)
		return tx.Table(m.tables.Roles).Create(&row).Error
	})
	if err != nil {
		return models.Role{}, fmt.Errorf("CreateRole: %w", err)
	}
	return gormstore.RoleFromRow(row), nil
}

func (m *agencyManager) GetRole(ctx context.Context, id string) (models.Role, error) {
	row, err := m.findRoleRow(ctx, m.db, id)
	if err != nil {
		return models.Role{}, err
	}
	return gormstore.RoleFromRow(row), nil
}

// ListRoles returns draftID's Roles, narrowed by filter.
func (m *agencyManager) ListRoles(ctx context.Context, draftID string, filter RoleFilter) ([]models.Role, error) {
	q := scopeQuery(m.db.WithContext(ctx).Table(m.tables.Roles), draftID)
	q = likeFilter(q, "code", filter.Code)
	q = likeFilter(q, "name", filter.Name)
	q = likeFilter(q, "description", filter.Description)
	var rows []gormstore.RoleRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListRoles: %w", err)
	}
	out := make([]models.Role, len(rows))
	for i, row := range rows {
		out[i] = gormstore.RoleFromRow(row)
	}
	return out, nil
}

// UpdateRole replaces r's mutable fields via an explicit column map (see
// UpdateGoal's doc for why). Every Role belongs to some open Draft
// (AGD-006), so there is no live row to refuse editing.
func (m *agencyManager) UpdateRole(ctx context.Context, r models.Role) (models.Role, error) {
	existing, err := m.findRoleRow(ctx, m.db, r.ID)
	if err != nil {
		return models.Role{}, err
	}
	if r.Name == "" {
		return models.Role{}, fmt.Errorf("%w: Role.Name is required", ErrInvalidRole)
	}
	row := gormstore.RoleToRow(r)
	row.CreatedAt = existing.CreatedAt
	row.Code = existing.Code
	cols := map[string]any{
		"name": row.Name, "description": row.Description,
		"updated_at": row.UpdatedAt, "deleted": row.Deleted,
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Roles).Where("id = ?", r.ID).Updates(cols).Error; err != nil {
		return models.Role{}, fmt.Errorf("UpdateRole: %w", err)
	}
	return gormstore.RoleFromRow(row), nil
}

// DeleteRole deletes the Role and cascades to every RoleAssignment naming
// it, the mirror of DeleteObjective's cascade from the other side of the
// same join table.
func (m *agencyManager) DeleteRole(ctx context.Context, id string) error {
	if _, err := m.findRoleRow(ctx, m.db, id); err != nil {
		return err
	}
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table(m.tables.RoleAssignments).Where("role_id = ?", id).
			Delete(&gormstore.RoleAssignmentRow{}).Error; err != nil {
			return err
		}
		return tx.Table(m.tables.Roles).Where("id = ?", id).Delete(&gormstore.RoleRow{}).Error
	})
	if err != nil {
		return fmt.Errorf("DeleteRole: %w", err)
	}
	return nil
}

func (m *agencyManager) findRoleRow(ctx context.Context, tx *gorm.DB, id string) (gormstore.RoleRow, error) {
	var row gormstore.RoleRow
	err := tx.WithContext(ctx).Table(m.tables.Roles).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.RoleRow{}, m.roleNotFoundError(ctx, tx, id)
	}
	if err != nil {
		return gormstore.RoleRow{}, fmt.Errorf("findRoleRow: %w", err)
	}
	return row, nil
}

// roleNotFoundError enriches ErrRoleNotFound with any roles whose Code or
// Name contains id (case-insensitive) — an AI agent that mistakes a Role's
// id for its Code/Name (or guesses one from the other, e.g. slugifying
// "Test" into "test") gets pointed at the real id right in this error,
// instead of a bare "not found" that costs it another blind round trip.
func (m *agencyManager) roleNotFoundError(ctx context.Context, tx *gorm.DB, id string) error {
	pattern := "%" + strings.ToLower(id) + "%"
	var matches []gormstore.RoleRow
	tx.WithContext(ctx).Table(m.tables.Roles).
		Where("LOWER(code) LIKE ? OR LOWER(name) LIKE ?", pattern, pattern).
		Find(&matches)
	if len(matches) == 0 {
		return ErrRoleNotFound
	}
	hints := make([]string, len(matches))
	for i, r := range matches {
		hints[i] = fmt.Sprintf("%s %q (id: %s)", r.Code, r.Name, r.ID)
	}
	return fmt.Errorf("%w: no role with id %q — did you mean one of: %s", ErrRoleNotFound, id, strings.Join(hints, "; "))
}
