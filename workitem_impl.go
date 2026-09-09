package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) CreateWorkItem(ctx context.Context, wi models.WorkItem) (models.WorkItem, error) {
	if wi.CreatedAt == "" {
		wi.CreatedAt = models.NowRFC3339()
	}
	row := gormstore.WorkItemToRow(wi)
	if err := m.db.WithContext(ctx).Table(m.tables.WorkItems).Create(&row).Error; err != nil {
		return models.WorkItem{}, fmt.Errorf("CreateWorkItem: %w", err)
	}
	return gormstore.WorkItemFromRow(row), nil
}

func (m *agencyManager) GetWorkItem(ctx context.Context, id string) (models.WorkItem, error) {
	row, err := m.findWorkItemRow(ctx, m.db, id)
	if err != nil {
		return models.WorkItem{}, err
	}
	return gormstore.WorkItemFromRow(row), nil
}

// ListWorkItems returns live WorkItems when draftID is "", or draftID's
// draft-scoped copies otherwise.
func (m *agencyManager) ListWorkItems(ctx context.Context, draftID string) ([]models.WorkItem, error) {
	var rows []gormstore.WorkItemRow
	q := scopeQuery(m.db.WithContext(ctx).Table(m.tables.WorkItems).Order("ordinality"), draftID)
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListWorkItems: %w", err)
	}
	out := make([]models.WorkItem, len(rows))
	for i, r := range rows {
		out[i] = gormstore.WorkItemFromRow(r)
	}
	return out, nil
}

// UpdateWorkItem replaces wi's mutable fields via an explicit column map —
// see UpdateGoal's doc for why (GORM's struct-based Updates skips
// zero-value fields).
func (m *agencyManager) UpdateWorkItem(ctx context.Context, wi models.WorkItem) (models.WorkItem, error) {
	existing, err := m.findWorkItemRow(ctx, m.db, wi.ID)
	if err != nil {
		return models.WorkItem{}, err
	}
	row := gormstore.WorkItemToRow(wi)
	row.CreatedAt = existing.CreatedAt
	cols := map[string]any{
		"workflow_id": row.WorkflowID,
		"goal_id":     row.GoalID,
		"draft_id":    row.DraftID,
		"title":       row.Title,
		"description": row.Description,
		"ordinality":  row.Ordinality,
		"prompt":      row.Prompt,
		"updated_at":  row.UpdatedAt,
		"deleted":     row.Deleted,
	}
	if err := m.db.WithContext(ctx).Table(m.tables.WorkItems).Where("id = ?", wi.ID).Updates(cols).Error; err != nil {
		return models.WorkItem{}, fmt.Errorf("UpdateWorkItem: %w", err)
	}
	return gormstore.WorkItemFromRow(row), nil
}

func (m *agencyManager) findWorkItemRow(ctx context.Context, tx *gorm.DB, id string) (gormstore.WorkItemRow, error) {
	var row gormstore.WorkItemRow
	err := tx.WithContext(ctx).Table(m.tables.WorkItems).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.WorkItemRow{}, ErrWorkItemNotFound
	}
	if err != nil {
		return gormstore.WorkItemRow{}, fmt.Errorf("findWorkItemRow: %w", err)
	}
	return row, nil
}
