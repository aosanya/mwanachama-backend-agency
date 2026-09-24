package mwanachamaagency

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) CreateWorkItem(ctx context.Context, wi models.WorkItem) (models.WorkItem, error) {
	if wi.ObjectiveID == "" {
		return models.WorkItem{}, fmt.Errorf("%w: WorkItem.ObjectiveID is required", ErrInvalidWorkItem)
	}
	if err := m.validateOpenDraftID(ctx, wi.DraftID); err != nil {
		return models.WorkItem{}, err
	}
	wi.ID = newID()
	wi.CreatedAt = models.NowRFC3339()
	wi.LastUpdated = wi.CreatedAt

	err := m.createWithCode(ctx, roleWorkItem, "workitem", "WI", func(code string) any {
		wi.Code = code
		return wi
	})
	if err != nil {
		return models.WorkItem{}, fmt.Errorf("CreateWorkItem: %w", err)
	}
	return wi, nil
}

func (m *agencyManager) GetWorkItem(ctx context.Context, id string) (models.WorkItem, error) {
	return m.findWorkItem(ctx, m.db, id)
}

func (m *agencyManager) ListWorkItems(ctx context.Context, draftID string, filter WorkItemFilter) ([]models.WorkItem, error) {
	q := scopeQuery(m.q(ctx, roleWorkItem).Order("ordinality"), draftID)
	q = likeFilter(q, "code", filter.Code)
	q = likeFilter(q, "title", filter.Title)
	q = likeFilter(q, "description", filter.Description)

	out, err := list[models.WorkItem](m, q, roleWorkItem)
	if err != nil {
		return nil, fmt.Errorf("ListWorkItems: %w", err)
	}
	return out, nil
}

func (m *agencyManager) UpdateWorkItem(ctx context.Context, wi models.WorkItem) (models.WorkItem, error) {
	if wi.ObjectiveID == "" {
		return models.WorkItem{}, fmt.Errorf("%w: WorkItem.ObjectiveID is required", ErrInvalidWorkItem)
	}
	var out models.WorkItem
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := m.findWorkItem(ctx, tx, wi.ID)
		if err != nil {
			return err
		}
		wi.Code = existing.Code
		wi.CreatedAt = existing.CreatedAt
		wi.LastUpdated = models.NowRFC3339()

		if err := m.writeUpdate(ctx, tx, roleWorkItem, wi.ID, wi, ErrWorkItemNotFound); err != nil {
			return err
		}
		out = wi
		return nil
	})
	if err != nil {
		return models.WorkItem{}, fmt.Errorf("UpdateWorkItem: %w", err)
	}
	return out, nil
}

func (m *agencyManager) DeleteWorkItem(ctx context.Context, id string) error {
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := m.findWorkItem(ctx, tx, id); err != nil {
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
