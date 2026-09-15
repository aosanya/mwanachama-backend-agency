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
	if wi.ObjectiveID == "" {
		return models.WorkItem{}, fmt.Errorf("%w: WorkItem.ObjectiveID is required", ErrInvalidWorkItem)
	}
	if err := m.validateOpenDraftID(ctx, wi.DraftID); err != nil {
		return models.WorkItem{}, err
	}
	wi.ID = ""
	wi.CreatedAt = models.NowRFC3339()
	var row gormstore.WorkItemRow
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		code, err := gormstore.NextCode(ctx, tx, m.tables.CodeSequences, "workitem", "WI")
		if err != nil {
			return err
		}
		wi.Code = code
		row = gormstore.WorkItemToRow(wi)
		return tx.Table(m.tables.WorkItems).Create(&row).Error
	})
	if err != nil {
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
// draft-scoped copies otherwise, narrowed by filter.
func (m *agencyManager) ListWorkItems(ctx context.Context, draftID string, filter WorkItemFilter) ([]models.WorkItem, error) {
	var rows []gormstore.WorkItemRow
	q := scopeQuery(m.db.WithContext(ctx).Table(m.tables.WorkItems).Order("ordinality"), draftID)
	q = likeFilter(q, "code", filter.Code)
	q = likeFilter(q, "title", filter.Title)
	q = likeFilter(q, "description", filter.Description)
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
	if wi.ObjectiveID == "" {
		return models.WorkItem{}, fmt.Errorf("%w: WorkItem.ObjectiveID is required", ErrInvalidWorkItem)
	}
	row := gormstore.WorkItemToRow(wi)
	row.CreatedAt = existing.CreatedAt
	row.Code = existing.Code
	cols := map[string]any{
		"workflow_id":  row.WorkflowID,
		"objective_id": row.ObjectiveID,
		"draft_id":     row.DraftID,
		"title":        row.Title,
		"description":  row.Description,
		"ordinality":   row.Ordinality,
		"prompt":       row.Prompt,
		"updated_at":   row.UpdatedAt,
		"deleted":      row.Deleted,
	}
	if err := m.db.WithContext(ctx).Table(m.tables.WorkItems).Where("id = ?", wi.ID).Updates(cols).Error; err != nil {
		return models.WorkItem{}, fmt.Errorf("UpdateWorkItem: %w", err)
	}
	return gormstore.WorkItemFromRow(row), nil
}

// DeleteWorkItem deletes wi — see [AgencyManager]'s doc comment on why
// there is nothing further to cascade into.
func (m *agencyManager) DeleteWorkItem(ctx context.Context, id string) error {
	if _, err := m.findWorkItemRow(ctx, m.db, id); err != nil {
		return err
	}
	if err := m.db.WithContext(ctx).Table(m.tables.WorkItems).Where("id = ?", id).Delete(&gormstore.WorkItemRow{}).Error; err != nil {
		return fmt.Errorf("DeleteWorkItem: %w", err)
	}
	return nil
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
