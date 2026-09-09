package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) CreateDepartment(ctx context.Context, d models.Department) (models.Department, error) {
	if d.Name == "" {
		return models.Department{}, fmt.Errorf("%w: Department.Name is required", ErrInvalidDepartment)
	}
	if d.CreatedAt == "" {
		d.CreatedAt = models.NowRFC3339()
	}
	row := gormstore.DepartmentToRow(d)
	if err := m.db.WithContext(ctx).Table(m.tables.Departments).Create(&row).Error; err != nil {
		return models.Department{}, fmt.Errorf("CreateDepartment: %w", err)
	}
	return gormstore.DepartmentFromRow(row), nil
}

func (m *agencyManager) GetDepartment(ctx context.Context, id string) (models.Department, error) {
	row, err := m.findDepartmentRow(ctx, m.db, id)
	if err != nil {
		return models.Department{}, err
	}
	return gormstore.DepartmentFromRow(row), nil
}

func (m *agencyManager) ListDepartments(ctx context.Context) ([]models.Department, error) {
	var rows []gormstore.DepartmentRow
	if err := m.db.WithContext(ctx).Table(m.tables.Departments).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListDepartments: %w", err)
	}
	out := make([]models.Department, len(rows))
	for i, r := range rows {
		out[i] = gormstore.DepartmentFromRow(r)
	}
	return out, nil
}

// ListDepartmentChildren returns parentID's direct children, sorted by ID.
func (m *agencyManager) ListDepartmentChildren(ctx context.Context, parentID string) ([]models.Department, error) {
	all, err := m.ListDepartments(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListDepartmentChildren: %w", err)
	}
	out := []models.Department{}
	for _, d := range all {
		if d.ParentID == parentID {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (m *agencyManager) RenameDepartment(ctx context.Context, id, name string) (models.Department, error) {
	if name == "" {
		return models.Department{}, fmt.Errorf("%w: name is required", ErrInvalidDepartment)
	}
	row, err := m.findDepartmentRow(ctx, m.db, id)
	if err != nil {
		return models.Department{}, err
	}
	now := models.NowRFC3339()
	if err := m.db.WithContext(ctx).Table(m.tables.Departments).Where("id = ?", id).
		Updates(map[string]any{"name": name, "updated_at": now}).Error; err != nil {
		return models.Department{}, fmt.Errorf("RenameDepartment: %w", err)
	}
	d := gormstore.DepartmentFromRow(row)
	d.Name, d.LastUpdated = name, now
	return d, nil
}

// MoveDepartment re-parents id under newParentID ("" for top-level).
// Refuses: newParentID == id (ErrParentIsSelf), id currently has no parent
// (ErrRootCannotMove), newParentID doesn't exist (ErrParentNotFound), or
// newParentID is id's own descendant (ErrParentInSubtree) — the
// cycle/subtree check is a Go walk over ListDepartments, not a recursive
// CTE, matching mwanachama-backend-actor's MoveGroup.
func (m *agencyManager) MoveDepartment(ctx context.Context, id, newParentID string) (models.Department, error) {
	if newParentID == id {
		return models.Department{}, ErrParentIsSelf
	}
	current, err := m.GetDepartment(ctx, id)
	if err != nil {
		return models.Department{}, err
	}
	if current.ParentID == "" {
		return models.Department{}, ErrRootCannotMove
	}
	if newParentID != "" {
		if _, err := m.GetDepartment(ctx, newParentID); err != nil {
			if errors.Is(err, ErrDepartmentNotFound) {
				return models.Department{}, ErrParentNotFound
			}
			return models.Department{}, err
		}
	}

	all, err := m.ListDepartments(ctx)
	if err != nil {
		return models.Department{}, fmt.Errorf("MoveDepartment: %w", err)
	}
	byID := make(map[string]models.Department, len(all))
	for _, d := range all {
		byID[d.ID] = d
	}
	seen := map[string]bool{}
	for cur, ok := byID[newParentID]; ok && !seen[cur.ID]; cur, ok = byID[cur.ParentID] {
		if cur.ID == id {
			return models.Department{}, ErrParentInSubtree
		}
		seen[cur.ID] = true
	}

	now := models.NowRFC3339()
	if err := m.db.WithContext(ctx).Table(m.tables.Departments).Where("id = ?", id).
		Updates(map[string]any{"parent_id": gormstore.StringToNullable(newParentID), "updated_at": now}).Error; err != nil {
		return models.Department{}, fmt.Errorf("MoveDepartment: %w", err)
	}
	current.ParentID = newParentID
	current.LastUpdated = now
	return current, nil
}

func (m *agencyManager) findDepartmentRow(ctx context.Context, tx *gorm.DB, id string) (gormstore.DepartmentRow, error) {
	var row gormstore.DepartmentRow
	err := tx.WithContext(ctx).Table(m.tables.Departments).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.DepartmentRow{}, ErrDepartmentNotFound
	}
	if err != nil {
		return gormstore.DepartmentRow{}, fmt.Errorf("findDepartmentRow: %w", err)
	}
	return row, nil
}
