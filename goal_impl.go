package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) CreateGoal(ctx context.Context, g models.Goal) (models.Goal, error) {
	if g.CreatedAt == "" {
		g.CreatedAt = models.NowRFC3339()
	}
	row := gormstore.GoalToRow(g)
	if err := m.db.WithContext(ctx).Table(m.tables.Goals).Create(&row).Error; err != nil {
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
// draft-scoped copies otherwise.
func (m *agencyManager) ListGoals(ctx context.Context, draftID string) ([]models.Goal, error) {
	var rows []gormstore.GoalRow
	q := m.db.WithContext(ctx).Table(m.tables.Goals).Order("ordinality")
	if draftID == "" {
		q = q.Where("draft_id IS NULL")
	} else {
		q = q.Where("draft_id = ?", draftID)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListGoals: %w", err)
	}
	out := make([]models.Goal, len(rows))
	for i, r := range rows {
		out[i] = gormstore.GoalFromRow(r)
	}
	return out, nil
}

func (m *agencyManager) UpdateGoal(ctx context.Context, g models.Goal) (models.Goal, error) {
	if _, err := m.findGoalRow(ctx, m.db, g.ID); err != nil {
		return models.Goal{}, err
	}
	row := gormstore.GoalToRow(g)
	if err := m.db.WithContext(ctx).Table(m.tables.Goals).Where("id = ?", g.ID).Updates(&row).Error; err != nil {
		return models.Goal{}, fmt.Errorf("UpdateGoal: %w", err)
	}
	return gormstore.GoalFromRow(row), nil
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
