package mwanachamaagency

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) CreateObjective(ctx context.Context, o models.Objective) (models.Objective, error) {
	if err := m.validateOpenDraftID(ctx, o.DraftID); err != nil {
		return models.Objective{}, err
	}
	o.ID = newID()
	o.CreatedAt = models.NowRFC3339()
	o.LastUpdated = o.CreatedAt

	err := m.createWithCode(ctx, roleObjective, "objective", "O", func(code string) any {
		o.Code = code
		return o
	})
	if err != nil {
		return models.Objective{}, fmt.Errorf("CreateObjective: %w", err)
	}
	return o, nil
}

func (m *agencyManager) GetObjective(ctx context.Context, id string) (models.Objective, error) {
	return m.findObjective(ctx, m.db, id)
}

func (m *agencyManager) ListObjectives(ctx context.Context, draftID string, filter ObjectiveFilter) ([]models.Objective, error) {
	q := scopeQuery(m.q(ctx, roleObjective).Order("ordinality"), draftID)
	q = likeFilter(q, "code", filter.Code)
	q = likeFilter(q, "title", filter.Title)
	q = likeFilter(q, "description", filter.Description)

	out, err := list[models.Objective](m, q, roleObjective)
	if err != nil {
		return nil, fmt.Errorf("ListObjectives: %w", err)
	}
	return out, nil
}

func (m *agencyManager) UpdateObjective(ctx context.Context, o models.Objective) (models.Objective, error) {
	var out models.Objective
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := m.findObjective(ctx, tx, o.ID)
		if err != nil {
			return err
		}
		o.Code = existing.Code
		o.CreatedAt = existing.CreatedAt
		o.LastUpdated = models.NowRFC3339()

		if err := m.writeUpdate(ctx, tx, roleObjective, o.ID, o, ErrObjectiveNotFound); err != nil {
			return err
		}
		out = o
		return nil
	})
	if err != nil {
		return models.Objective{}, fmt.Errorf("UpdateObjective: %w", err)
	}
	return out, nil
}

func (m *agencyManager) DeleteObjective(ctx context.Context, id string) error {
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := m.findObjective(ctx, tx, id); err != nil {
			return err
		}
		for _, role := range []string{roleRoleAssignment, roleWorkItem, roleObjectiveModule} {
			if err := m.deleteWhere(ctx, tx, role, "objective_id = ?", id); err != nil {
				return err
			}
		}
		return m.deleteWhere(ctx, tx, roleObjective, "id = ?", id)
	})
	if err != nil {
		return fmt.Errorf("DeleteObjective: %w", err)
	}
	return nil
}

func (m *agencyManager) findObjective(ctx context.Context, tx *gorm.DB, id string) (models.Objective, error) {
	var o models.Objective
	if err := m.find(ctx, tx, roleObjective, id, &o, ErrObjectiveNotFound); err != nil {
		return models.Objective{}, err
	}
	return o, nil
}
