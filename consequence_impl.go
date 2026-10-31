package mwanachamaagency

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) CreateConsequence(ctx context.Context, p models.Consequence) (models.Consequence, error) {
	out, err := m.createPart(ctx, consequenceKind, models.Hazard(p))
	if err != nil {
		return models.Consequence{}, fmt.Errorf("CreateConsequence: %w", err)
	}
	return models.Consequence(out), nil
}

func (m *agencyManager) GetConsequence(ctx context.Context, id string) (models.Consequence, error) {
	out, err := m.findPart(ctx, m.db, consequenceKind, id)
	if err != nil {
		return models.Consequence{}, err
	}
	return models.Consequence(out), nil
}

func (m *agencyManager) ListConsequences(ctx context.Context, draftID string, filter LibraryPartFilter) ([]models.Consequence, error) {
	rows, err := m.listParts(ctx, consequenceKind, draftID, filter)
	if err != nil {
		return nil, fmt.Errorf("ListConsequences: %w", err)
	}
	out := make([]models.Consequence, len(rows))
	for i, r := range rows {
		out[i] = models.Consequence(r)
	}
	return out, nil
}

func (m *agencyManager) UpdateConsequence(ctx context.Context, p models.Consequence) (models.Consequence, error) {
	out, err := m.updatePart(ctx, consequenceKind, models.Hazard(p))
	if err != nil {
		return models.Consequence{}, fmt.Errorf("UpdateConsequence: %w", err)
	}
	return models.Consequence(out), nil
}

func (m *agencyManager) DeleteConsequence(ctx context.Context, id string) error {
	if err := m.deletePart(ctx, consequenceKind, id); err != nil {
		return fmt.Errorf("DeleteConsequence: %w", err)
	}
	return nil
}
