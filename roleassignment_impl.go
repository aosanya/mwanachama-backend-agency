package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// validateRoleAssignment checks a's Label, that its Role and WorkItem
// exist in the same scope (a.DraftID), and — for excludeID != "" on
// update, "" on create — that no other assignment in that scope already
// labels the WorkItem Accountable.
func (m *agencyManager) validateRoleAssignment(ctx context.Context, tx *gorm.DB, a models.RoleAssignment, excludeID string) error {
	if !a.Label.Valid() {
		return ErrInvalidRACILabel
	}

	var roleRow gormstore.RoleRow
	err := scopeQuery(tx.WithContext(ctx).Table(m.tables.Roles), a.DraftID).
		Where("id = ?", a.RoleID).First(&roleRow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrRoleNotFound
	}
	if err != nil {
		return fmt.Errorf("validateRoleAssignment: %w", err)
	}

	var workItemRow gormstore.WorkItemRow
	err = scopeQuery(tx.WithContext(ctx).Table(m.tables.WorkItems), a.DraftID).
		Where("id = ?", a.WorkItemID).First(&workItemRow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrWorkItemNotFound
	}
	if err != nil {
		return fmt.Errorf("validateRoleAssignment: %w", err)
	}

	if a.Label == models.RACILabelAccountable {
		q := scopeQuery(tx.WithContext(ctx).Table(m.tables.RoleAssignments), a.DraftID).
			Where("work_item_id = ? AND label = ?", a.WorkItemID, string(models.RACILabelAccountable))
		if excludeID != "" {
			q = q.Where("id <> ?", excludeID)
		}
		var count int64
		if err := q.Count(&count).Error; err != nil {
			return fmt.Errorf("validateRoleAssignment: %w", err)
		}
		if count > 0 {
			return ErrDuplicateAccountable
		}
	}
	return nil
}

func (m *agencyManager) CreateRoleAssignment(ctx context.Context, a models.RoleAssignment) (models.RoleAssignment, error) {
	if err := m.validateRoleAssignment(ctx, m.db, a, ""); err != nil {
		return models.RoleAssignment{}, err
	}
	if a.CreatedAt == "" {
		a.CreatedAt = models.NowRFC3339()
	}
	row := gormstore.RoleAssignmentToRow(a)
	if err := m.db.WithContext(ctx).Table(m.tables.RoleAssignments).Create(&row).Error; err != nil {
		return models.RoleAssignment{}, fmt.Errorf("CreateRoleAssignment: %w", err)
	}
	return gormstore.RoleAssignmentFromRow(row), nil
}

func (m *agencyManager) GetRoleAssignment(ctx context.Context, id string) (models.RoleAssignment, error) {
	row, err := m.findRoleAssignmentRow(ctx, m.db, id)
	if err != nil {
		return models.RoleAssignment{}, err
	}
	return gormstore.RoleAssignmentFromRow(row), nil
}

// ListRoleAssignments returns live assignments when draftID is "", or
// draftID's draft-scoped copies otherwise.
func (m *agencyManager) ListRoleAssignments(ctx context.Context, draftID string) ([]models.RoleAssignment, error) {
	var rows []gormstore.RoleAssignmentRow
	if err := scopeQuery(m.db.WithContext(ctx).Table(m.tables.RoleAssignments), draftID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListRoleAssignments: %w", err)
	}
	out := make([]models.RoleAssignment, len(rows))
	for i, row := range rows {
		out[i] = gormstore.RoleAssignmentFromRow(row)
	}
	return out, nil
}

// ListRoleAssignmentsForWorkItem answers "who is assigned to this
// WorkItem, and with what RACI label" — the lookup a caller (e.g.
// mwanachama-backend-api-kazi's K6) uses to find who to notify.
func (m *agencyManager) ListRoleAssignmentsForWorkItem(ctx context.Context, workItemID, draftID string) ([]models.RoleAssignment, error) {
	var rows []gormstore.RoleAssignmentRow
	err := scopeQuery(m.db.WithContext(ctx).Table(m.tables.RoleAssignments), draftID).
		Where("work_item_id = ?", workItemID).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("ListRoleAssignmentsForWorkItem: %w", err)
	}
	out := make([]models.RoleAssignment, len(rows))
	for i, row := range rows {
		out[i] = gormstore.RoleAssignmentFromRow(row)
	}
	return out, nil
}

// UpdateRoleAssignment replaces a's mutable fields via an explicit column
// map. Returns [ErrCannotEditLiveRACI] if the existing row is live.
func (m *agencyManager) UpdateRoleAssignment(ctx context.Context, a models.RoleAssignment) (models.RoleAssignment, error) {
	existing, err := m.findRoleAssignmentRow(ctx, m.db, a.ID)
	if err != nil {
		return models.RoleAssignment{}, err
	}
	if existing.DraftID == nil {
		return models.RoleAssignment{}, ErrCannotEditLiveRACI
	}
	a.DraftID = gormstore.RoleAssignmentFromRow(existing).DraftID
	if err := m.validateRoleAssignment(ctx, m.db, a, a.ID); err != nil {
		return models.RoleAssignment{}, err
	}
	row := gormstore.RoleAssignmentToRow(a)
	row.CreatedAt = existing.CreatedAt
	cols := map[string]any{
		"role_id": row.RoleID, "work_item_id": row.WorkItemID, "label": row.Label,
		"updated_at": row.UpdatedAt, "deleted": row.Deleted,
	}
	if err := m.db.WithContext(ctx).Table(m.tables.RoleAssignments).Where("id = ?", a.ID).Updates(cols).Error; err != nil {
		return models.RoleAssignment{}, fmt.Errorf("UpdateRoleAssignment: %w", err)
	}
	return gormstore.RoleAssignmentFromRow(row), nil
}

// DeleteRoleAssignment returns [ErrCannotEditLiveRACI] if the target row
// is live.
func (m *agencyManager) DeleteRoleAssignment(ctx context.Context, id string) error {
	existing, err := m.findRoleAssignmentRow(ctx, m.db, id)
	if err != nil {
		return err
	}
	if existing.DraftID == nil {
		return ErrCannotEditLiveRACI
	}
	if err := m.db.WithContext(ctx).Table(m.tables.RoleAssignments).Where("id = ?", id).
		Delete(&gormstore.RoleAssignmentRow{}).Error; err != nil {
		return fmt.Errorf("DeleteRoleAssignment: %w", err)
	}
	return nil
}

func (m *agencyManager) findRoleAssignmentRow(ctx context.Context, tx *gorm.DB, id string) (gormstore.RoleAssignmentRow, error) {
	var row gormstore.RoleAssignmentRow
	err := tx.WithContext(ctx).Table(m.tables.RoleAssignments).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.RoleAssignmentRow{}, ErrRoleAssignmentNotFound
	}
	if err != nil {
		return gormstore.RoleAssignmentRow{}, fmt.Errorf("findRoleAssignmentRow: %w", err)
	}
	return row, nil
}
