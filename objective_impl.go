package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// CreateObjective creates an Objective for a Goal. A Goal may have more
// than one Objective — each one a granular unit a single Role can be
// RACI-assigned to (see [AgencyManager]'s doc comment).
func (m *agencyManager) CreateObjective(ctx context.Context, o models.Objective) (models.Objective, error) {
	if o.CreatedAt == "" {
		o.CreatedAt = models.NowRFC3339()
	}
	row := gormstore.ObjectiveToRow(o)
	if err := m.db.WithContext(ctx).Table(m.tables.Objectives).Create(&row).Error; err != nil {
		return models.Objective{}, fmt.Errorf("CreateObjective: %w", err)
	}
	return gormstore.ObjectiveFromRow(row), nil
}

func (m *agencyManager) GetObjective(ctx context.Context, id string) (models.Objective, error) {
	row, err := m.findObjectiveRow(ctx, m.db, id)
	if err != nil {
		return models.Objective{}, err
	}
	return gormstore.ObjectiveFromRow(row), nil
}

// ListObjectives returns live Objectives when draftID is "", or draftID's
// draft-scoped copies otherwise.
func (m *agencyManager) ListObjectives(ctx context.Context, draftID string) ([]models.Objective, error) {
	var rows []gormstore.ObjectiveRow
	q := scopeQuery(m.db.WithContext(ctx).Table(m.tables.Objectives).Order("ordinality"), draftID)
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListObjectives: %w", err)
	}
	out := make([]models.Objective, len(rows))
	for i, r := range rows {
		out[i] = gormstore.ObjectiveFromRow(r)
	}
	return out, nil
}

// UpdateObjective replaces o's mutable fields via an explicit column map
// (see UpdateGoal's doc for why). No live-immutability check — Objective
// follows Goal/Workflow/WorkItem's laxer convention, not Role's stricter
// one.
func (m *agencyManager) UpdateObjective(ctx context.Context, o models.Objective) (models.Objective, error) {
	existing, err := m.findObjectiveRow(ctx, m.db, o.ID)
	if err != nil {
		return models.Objective{}, err
	}
	row := gormstore.ObjectiveToRow(o)
	row.CreatedAt = existing.CreatedAt
	cols := map[string]any{
		"draft_id":    row.DraftID,
		"goal_id":     row.GoalID,
		"title":       row.Title,
		"description": row.Description,
		"ordinality":  row.Ordinality,
		"updated_at":  row.UpdatedAt,
		"deleted":     row.Deleted,
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Objectives).Where("id = ?", o.ID).Updates(cols).Error; err != nil {
		return models.Objective{}, fmt.Errorf("UpdateObjective: %w", err)
	}
	return gormstore.ObjectiveFromRow(row), nil
}

func (m *agencyManager) findObjectiveRow(ctx context.Context, tx *gorm.DB, id string) (gormstore.ObjectiveRow, error) {
	var row gormstore.ObjectiveRow
	err := tx.WithContext(ctx).Table(m.tables.Objectives).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.ObjectiveRow{}, ErrObjectiveNotFound
	}
	if err != nil {
		return gormstore.ObjectiveRow{}, fmt.Errorf("findObjectiveRow: %w", err)
	}
	return row, nil
}
