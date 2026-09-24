package mwanachamaagency

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) CreateEventFlowStep(ctx context.Context, s models.EventFlowStep) (models.EventFlowStep, error) {
	if err := m.validateOpenDraftID(ctx, s.DraftID); err != nil {
		return models.EventFlowStep{}, err
	}
	s.ID = newID()
	s.CreatedAt = models.NowRFC3339()
	s.LastUpdated = s.CreatedAt

	if err := m.insert(ctx, roleEventFlowStep, s); err != nil {
		return models.EventFlowStep{}, fmt.Errorf("CreateEventFlowStep: %w", err)
	}
	return s, nil
}

func (m *agencyManager) LookupFlowStep(ctx context.Context, draftID string, lookup models.EventFlowStepLookup) (models.EventFlowStep, error) {
	q := scopeQuery(m.q(ctx, roleEventFlowStep), draftID)
	switch {
	case lookup.HandlerCode != "":
		q = q.Where("handler_code = ?", lookup.HandlerCode)
	case lookup.TriggerTopic != "" && lookup.Consumer != "":
		q = q.Where("trigger_topic = ? AND consumer = ?", lookup.TriggerTopic, lookup.Consumer)
	case lookup.Code != "":
		q = q.Where("code = ?", lookup.Code)
	default:
		return models.EventFlowStep{}, ErrInvalidLookup
	}

	var step models.EventFlowStep
	if err := m.take(q, roleEventFlowStep, &step, ErrEventFlowStepNotFound); err != nil {
		return models.EventFlowStep{}, err
	}
	return step, nil
}

func (m *agencyManager) ListEventFlowSteps(ctx context.Context, draftID, workflowCode string, filter EventFlowStepFilter) ([]models.EventFlowStep, error) {
	q := scopeQuery(m.q(ctx, roleEventFlowStep), draftID).Order("workflow_code, ordinality, step")
	if workflowCode != "" {
		q = q.Where("workflow_code = ?", workflowCode)
	}
	q = likeFilter(q, "code", filter.Code)
	q = likeFilter(q, "name", filter.Name)
	q = likeFilter(q, "description", filter.Description)

	out, err := list[models.EventFlowStep](m, q, roleEventFlowStep)
	if err != nil {
		return nil, fmt.Errorf("ListEventFlowSteps: %w", err)
	}
	return out, nil
}
