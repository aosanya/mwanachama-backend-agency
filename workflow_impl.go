package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) CreateWorkflow(ctx context.Context, w models.Workflow) (models.Workflow, error) {
	if w.CreatedAt == "" {
		w.CreatedAt = models.NowRFC3339()
	}
	row := gormstore.WorkflowToRow(w)
	if err := m.db.WithContext(ctx).Table(m.tables.Workflows).Create(&row).Error; err != nil {
		return models.Workflow{}, fmt.Errorf("CreateWorkflow: %w", err)
	}
	return gormstore.WorkflowFromRow(row), nil
}

func (m *agencyManager) GetWorkflow(ctx context.Context, id string) (models.Workflow, error) {
	row, err := m.findWorkflowRow(ctx, m.db, id)
	if err != nil {
		return models.Workflow{}, err
	}
	return gormstore.WorkflowFromRow(row), nil
}

// ListWorkflows returns live Workflows when draftID is "", or draftID's
// draft-scoped copies otherwise.
func (m *agencyManager) ListWorkflows(ctx context.Context, draftID string) ([]models.Workflow, error) {
	var rows []gormstore.WorkflowRow
	q := scopeQuery(m.db.WithContext(ctx).Table(m.tables.Workflows).Order("ordinality"), draftID)
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListWorkflows: %w", err)
	}
	out := make([]models.Workflow, len(rows))
	for i, r := range rows {
		out[i] = gormstore.WorkflowFromRow(r)
	}
	return out, nil
}

// UpdateWorkflow replaces w's mutable fields via an explicit column map —
// see UpdateGoal's doc for why (GORM's struct-based Updates skips
// zero-value fields).
func (m *agencyManager) UpdateWorkflow(ctx context.Context, w models.Workflow) (models.Workflow, error) {
	existing, err := m.findWorkflowRow(ctx, m.db, w.ID)
	if err != nil {
		return models.Workflow{}, err
	}
	row := gormstore.WorkflowToRow(w)
	row.CreatedAt = existing.CreatedAt
	cols := map[string]any{
		"draft_id":    row.DraftID,
		"name":        row.Name,
		"description": row.Description,
		"ordinality":  row.Ordinality,
		"event_flows": row.EventFlows,
		"updated_at":  row.UpdatedAt,
		"deleted":     row.Deleted,
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Workflows).Where("id = ?", w.ID).Updates(cols).Error; err != nil {
		return models.Workflow{}, fmt.Errorf("UpdateWorkflow: %w", err)
	}
	return gormstore.WorkflowFromRow(row), nil
}

func (m *agencyManager) findWorkflowRow(ctx context.Context, tx *gorm.DB, id string) (gormstore.WorkflowRow, error) {
	var row gormstore.WorkflowRow
	err := tx.WithContext(ctx).Table(m.tables.Workflows).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.WorkflowRow{}, ErrWorkflowNotFound
	}
	if err != nil {
		return gormstore.WorkflowRow{}, fmt.Errorf("findWorkflowRow: %w", err)
	}
	return row, nil
}
