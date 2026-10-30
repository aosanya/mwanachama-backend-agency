package mwanachamaagency

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) CreateWorkflow(ctx context.Context, w models.Workflow) (models.Workflow, error) {
	if err := m.validateOpenDraftID(ctx, w.DraftID); err != nil {
		return models.Workflow{}, err
	}
	w.ID = newID()
	w.Deleted = false
	w.CreatedAt = models.NowRFC3339()
	w.LastUpdated = w.CreatedAt

	err := m.createWithCode(ctx, roleWorkflow, "workflow", "W", func(code string) any {
		w.Code = code
		return w
	})
	if err != nil {
		return models.Workflow{}, fmt.Errorf("CreateWorkflow: %w", err)
	}
	return w, nil
}

func (m *agencyManager) GetWorkflow(ctx context.Context, id string) (models.Workflow, error) {
	w, err := m.findWorkflow(ctx, m.db, id)
	if err != nil {
		return models.Workflow{}, err
	}
	out, err := m.withUse(ctx, []models.Workflow{w})
	if err != nil {
		return models.Workflow{}, fmt.Errorf("GetWorkflow: %w", err)
	}
	return out[0], nil
}

func (m *agencyManager) ListWorkflows(ctx context.Context, draftID string, filter WorkflowFilter) ([]models.Workflow, error) {
	q := scopeQuery(m.q(ctx, roleWorkflow).Order("ordinality"), draftID)
	q = likeFilter(q, "code", filter.Code)
	q = likeFilter(q, "name", filter.Name)
	q = likeFilter(q, "description", filter.Description)

	out, err := list[models.Workflow](m, q, roleWorkflow)
	if err != nil {
		return nil, fmt.Errorf("ListWorkflows: %w", err)
	}
	out, err = m.withUse(ctx, out)
	if err != nil {
		return nil, fmt.Errorf("ListWorkflows: %w", err)
	}
	return out, nil
}

func (m *agencyManager) UpdateWorkflow(ctx context.Context, w models.Workflow) (models.Workflow, error) {
	var out models.Workflow
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := m.findWorkflow(ctx, tx, w.ID)
		if err != nil {
			return err
		}
		if err := m.requireOpenDraft(ctx, tx, existing.DraftID); err != nil {
			return err
		}
		w.Code = existing.Code
		w.DraftID = existing.DraftID
		w.CreatedAt = existing.CreatedAt
		w.LastUpdated = models.NowRFC3339()

		if err := m.writeUpdate(ctx, tx, roleWorkflow, w.ID, w, ErrWorkflowNotFound); err != nil {
			return err
		}
		out = w
		return m.sendBack(ctx, tx, existing.DraftID, usageKey("workflow", existing.Code))
	})
	if err != nil {
		return models.Workflow{}, fmt.Errorf("UpdateWorkflow: %w", err)
	}
	return out, nil
}

func (m *agencyManager) DeleteWorkflow(ctx context.Context, id string) error {
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := m.findWorkflow(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := m.requireOpenDraft(ctx, tx, existing.DraftID); err != nil {
			return err
		}
		if err := m.refuseWorkflowMirroredByScenario(ctx, tx, existing); err != nil {
			return err
		}
		if err := m.sendBack(ctx, tx, existing.DraftID, usageKey("workflow", existing.Code)); err != nil {
			return err
		}
		if err := m.deleteInDraft(ctx, tx, roleWorkItem, existing.DraftID, "workflow_id = ?", id); err != nil {
			return err
		}
		if err := m.deleteInDraft(ctx, tx, roleRecordRow, existing.DraftID, "record_id IN (select id from "+m.table(roleRecord)+" where workflow_id = ?)", id); err != nil {
			return err
		}
		if err := m.deleteInDraft(ctx, tx, roleRecord, existing.DraftID, "workflow_id = ?", id); err != nil {
			return err
		}
		return m.deleteWhere(ctx, tx, roleWorkflow, "id = ?", id)
	})
	if err != nil {
		return fmt.Errorf("DeleteWorkflow: %w", err)
	}
	return nil
}

func (m *agencyManager) findWorkflow(ctx context.Context, tx *gorm.DB, id string) (models.Workflow, error) {
	var w models.Workflow
	if err := m.find(ctx, tx, roleWorkflow, id, &w, ErrWorkflowNotFound); err != nil {
		return models.Workflow{}, err
	}
	return w, nil
}
