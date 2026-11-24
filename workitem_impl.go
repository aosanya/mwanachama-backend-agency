package mwanachamaagency

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) CreateWorkItem(ctx context.Context, wi models.WorkItem) (models.WorkItem, error) {
	if err := m.validateOpenDraftID(ctx, wi.DraftID); err != nil {
		return models.WorkItem{}, err
	}
	wi.RowRecord = m.recordKeyOfRef(ctx, m.db, wi.DraftID, wi.RowRecord)
	if err := m.validateStep(ctx, m.db, wi); err != nil {
		return models.WorkItem{}, fmt.Errorf("CreateWorkItem: %w", err)
	}
	wi.ID = newID()
	wi.Deleted = false
	wi.CreatedAt = models.NowRFC3339()
	wi.LastUpdated = wi.CreatedAt

	err := m.createWithCode(ctx, roleWorkItem, "workitem", "WI", func(code string) any {
		wi.Code = code
		return wi
	})
	if err != nil {
		return models.WorkItem{}, fmt.Errorf("CreateWorkItem: %w", err)
	}
	if err := m.sendBack(ctx, m.db, wi.DraftID, m.stepKeys(ctx, m.db, wi)...); err != nil {
		return models.WorkItem{}, fmt.Errorf("CreateWorkItem: %w", err)
	}
	return m.withAction(ctx, wi)
}

func (m *agencyManager) GetWorkItem(ctx context.Context, id string) (models.WorkItem, error) {
	wi, err := m.findWorkItem(ctx, m.db, id)
	if err != nil {
		return models.WorkItem{}, err
	}
	return m.withAction(ctx, wi)
}

func (m *agencyManager) ListWorkItems(ctx context.Context, draftID string, filter WorkItemFilter) ([]models.WorkItem, error) {
	q := scopeQuery(m.q(ctx, roleWorkItem).Order("ordinality"), draftID)
	q = likeFilter(q, "code", filter.Code)
	q = likeFilter(q, "title", filter.Title)
	q = likeFilter(q, "description", filter.Description)
	if filter.RoleIDs != nil {
		q = q.Where("role_id IN ? OR recipient_role_id IN ?", filter.RoleIDs, filter.RoleIDs)
	}

	found, err := list[models.WorkItem](m, q, roleWorkItem)
	if err != nil {
		return nil, fmt.Errorf("ListWorkItems: %w", err)
	}
	out, err := m.withActions(ctx, found...)
	if err != nil {
		return nil, fmt.Errorf("ListWorkItems: %w", err)
	}
	return out, nil
}

func (m *agencyManager) UpdateWorkItem(ctx context.Context, wi models.WorkItem) (models.WorkItem, error) {
	var out models.WorkItem
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := m.findWorkItem(ctx, tx, wi.ID)
		if err != nil {
			return err
		}
		if err := m.requireOpenDraft(ctx, tx, existing.DraftID); err != nil {
			return err
		}
		if err := m.refuseGeneratedWorkflow(ctx, tx, existing.DraftID, existing.WorkflowID); err != nil {
			return err
		}
		if !wi.Carries("tool") {
			wi.Tool = existing.Tool
		}
		if !wi.Carries("row_record") {
			wi.RowRecord = existing.RowRecord
		}
		if !wi.Carries("row_action") {
			wi.RowAction = existing.RowAction
		}
		if !wi.Carries("send_back_decision") {
			wi.SendBackDecision = existing.SendBackDecision
		}
		if !wi.Carries("send_back_reason") {
			wi.SendBackReason = existing.SendBackReason
		}
		if !wi.Carries("record_code") {
			wi.RecordCode = existing.RecordCode
		}
		wi.Code = existing.Code
		wi.DraftID = existing.DraftID
		wi.CreatedAt = existing.CreatedAt
		wi.LastUpdated = models.NowRFC3339()

		wi.RowRecord = m.recordKeyOfRef(ctx, tx, wi.DraftID, wi.RowRecord)
		if err := m.validateStep(ctx, tx, wi); err != nil {
			return err
		}

		if err := m.writeUpdate(ctx, tx, roleWorkItem, wi.ID, wi, ErrWorkItemNotFound); err != nil {
			return err
		}
		out = wi
		return m.sendBack(ctx, tx, wi.DraftID, m.stepKeys(ctx, tx, wi, existing.WorkflowID)...)
	})
	if err != nil {
		return models.WorkItem{}, fmt.Errorf("UpdateWorkItem: %w", err)
	}
	return m.withAction(ctx, out)
}

func (m *agencyManager) DeleteWorkItem(ctx context.Context, id string) error {
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := m.findWorkItem(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := m.requireOpenDraft(ctx, tx, existing.DraftID); err != nil {
			return err
		}
		if err := m.refuseGeneratedWorkflow(ctx, tx, existing.DraftID, existing.WorkflowID); err != nil {
			return err
		}
		if err := m.sendBack(ctx, tx, existing.DraftID, m.stepKeys(ctx, tx, existing)...); err != nil {
			return err
		}
		if err := m.deleteWhere(ctx, tx, roleScenarioInput, "step_id = ?", id); err != nil {
			return err
		}
		if err := m.deleteWhere(ctx, tx, roleRoleAssignment, "work_item_id = ?", id); err != nil {
			return err
		}
		return m.deleteWhere(ctx, tx, roleWorkItem, "id = ?", id)
	})
	if err != nil {
		return fmt.Errorf("DeleteWorkItem: %w", err)
	}
	return nil
}

func (m *agencyManager) findWorkItem(ctx context.Context, tx *gorm.DB, id string) (models.WorkItem, error) {
	var wi models.WorkItem
	if err := m.find(ctx, tx, roleWorkItem, id, &wi, ErrWorkItemNotFound); err != nil {
		return models.WorkItem{}, err
	}
	return wi, nil
}
