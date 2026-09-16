package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// CreateGoal mints g a stable "G-<n>" Code (see [gormstore.NextCode]) inside
// the same transaction as the row insert, so the counter increment and the
// row's creation commit together.
func (m *agencyManager) CreateGoal(ctx context.Context, g models.Goal) (models.Goal, error) {
	if err := m.validateOpenDraftID(ctx, g.DraftID); err != nil {
		return models.Goal{}, err
	}
	g.ID = ""
	g.CreatedAt = models.NowRFC3339()
	var row gormstore.GoalRow
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		code, err := gormstore.NextCode(ctx, tx, m.tables.CodeSequences, "goal", "G")
		if err != nil {
			return err
		}
		g.Code = code
		row = gormstore.GoalToRow(g)
		return tx.Table(m.tables.Goals).Create(&row).Error
	})
	if err != nil {
		return models.Goal{}, fmt.Errorf("CreateGoal: %w", err)
	}
	return gormstore.GoalFromRow(row), nil
}

func (m *agencyManager) GetGoal(ctx context.Context, id string) (models.Goal, error) {
	row, err := m.findGoalRow(ctx, m.db, id)
	if err != nil {
		return models.Goal{}, err
	}
	return gormstore.GoalFromRow(row), nil
}

// ListGoals returns live Goals when draftID is "", or draftID's
// draft-scoped copies otherwise, narrowed by filter.
func (m *agencyManager) ListGoals(ctx context.Context, draftID string, filter GoalFilter) ([]models.Goal, error) {
	var rows []gormstore.GoalRow
	q := scopeQuery(m.db.WithContext(ctx).Table(m.tables.Goals).Order("ordinality"), draftID)
	q = likeFilter(q, "code", filter.Code)
	q = likeFilter(q, "title", filter.Title)
	q = likeFilter(q, "description", filter.Description)
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListGoals: %w", err)
	}
	out := make([]models.Goal, len(rows))
	for i, r := range rows {
		out[i] = gormstore.GoalFromRow(r)
	}
	return out, nil
}

// UpdateGoal replaces g's mutable fields. GORM's struct-based Updates skips
// zero-value fields, so this writes an explicit column map instead —
// otherwise clearing Description to "" or Ordinality to 0 would silently
// no-op (the same reason actor's deactivateRole uses a map, not a struct).
func (m *agencyManager) UpdateGoal(ctx context.Context, g models.Goal) (models.Goal, error) {
	existing, err := m.findGoalRow(ctx, m.db, g.ID)
	if err != nil {
		return models.Goal{}, err
	}
	row := gormstore.GoalToRow(g)
	row.CreatedAt = existing.CreatedAt
	row.Code = existing.Code
	cols := map[string]any{
		"draft_id":    row.DraftID,
		"title":       row.Title,
		"description": row.Description,
		"ordinality":  row.Ordinality,
		"updated_at":  row.UpdatedAt,
		"deleted":     row.Deleted,
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Goals).Where("id = ?", g.ID).Updates(cols).Error; err != nil {
		return models.Goal{}, fmt.Errorf("UpdateGoal: %w", err)
	}
	return gormstore.GoalFromRow(row), nil
}

// DeleteGoal deletes g and cascades to every Objective scoped to it, and
// to every RoleAssignment and WorkItem scoped to each of those
// Objectives — see [AgencyManager]'s doc comment for why this bypasses
// RoleAssignment/Objective's own independent-edit rules rather than
// calling DeleteRoleAssignment/DeleteObjective per row. All in one
// transaction, so a failure partway through never leaves a Goal deleted
// with orphaned Objectives/RoleAssignments still behind, or vice versa.
func (m *agencyManager) DeleteGoal(ctx context.Context, id string) error {
	if _, err := m.findGoalRow(ctx, m.db, id); err != nil {
		return err
	}
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var objectives []gormstore.ObjectiveRow
		if err := tx.Table(m.tables.Objectives).Where("goal_id = ?", id).Find(&objectives).Error; err != nil {
			return err
		}
		objectiveIDs := make([]string, len(objectives))
		for i, o := range objectives {
			objectiveIDs[i] = o.ID
		}
		if len(objectiveIDs) > 0 {
			if err := tx.Table(m.tables.RoleAssignments).Where("objective_id IN ?", objectiveIDs).
				Delete(&gormstore.RoleAssignmentRow{}).Error; err != nil {
				return err
			}
			if err := tx.Table(m.tables.WorkItems).Where("objective_id IN ?", objectiveIDs).
				Delete(&gormstore.WorkItemRow{}).Error; err != nil {
				return err
			}
			if err := tx.Table(m.tables.Objectives).Where("goal_id = ?", id).
				Delete(&gormstore.ObjectiveRow{}).Error; err != nil {
				return err
			}
		}
		return tx.Table(m.tables.Goals).Where("id = ?", id).Delete(&gormstore.GoalRow{}).Error
	})
	if err != nil {
		return fmt.Errorf("DeleteGoal: %w", err)
	}
	return nil
}

func (m *agencyManager) findGoalRow(ctx context.Context, tx *gorm.DB, id string) (gormstore.GoalRow, error) {
	var row gormstore.GoalRow
	err := tx.WithContext(ctx).Table(m.tables.Goals).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.GoalRow{}, ErrGoalNotFound
	}
	if err != nil {
		return gormstore.GoalRow{}, fmt.Errorf("findGoalRow: %w", err)
	}
	return row, nil
}
