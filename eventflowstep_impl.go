package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) CreateEventFlowStep(ctx context.Context, s models.EventFlowStep) (models.EventFlowStep, error) {
	if err := m.validateOpenDraftID(ctx, s.DraftID); err != nil {
		return models.EventFlowStep{}, err
	}
	s.ID = ""
	s.CreatedAt = models.NowRFC3339()
	row := gormstore.EventFlowStepToRow(s)
	if err := m.db.WithContext(ctx).Table(m.tables.EventFlowSteps).Create(&row).Error; err != nil {
		return models.EventFlowStep{}, fmt.Errorf("CreateEventFlowStep: %w", err)
	}
	return gormstore.EventFlowStepFromRow(row), nil
}

// LookupFlowStep tries lookup.HandlerCode, then (lookup.TriggerTopic,
// lookup.Consumer) together, then lookup.Code, in that precedence order,
// scoped to live steps.
func (m *agencyManager) LookupFlowStep(ctx context.Context, lookup models.EventFlowStepLookup) (models.EventFlowStep, error) {
	q := m.db.WithContext(ctx).Table(m.tables.EventFlowSteps).Where("draft_id IS NULL")
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

	var row gormstore.EventFlowStepRow
	err := q.First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.EventFlowStep{}, ErrEventFlowStepNotFound
	}
	if err != nil {
		return models.EventFlowStep{}, fmt.Errorf("LookupFlowStep: %w", err)
	}
	return gormstore.EventFlowStepFromRow(row), nil
}

// ListEventFlowSteps returns live steps ordered by (WorkflowCode,
// Ordinality, Step), optionally filtered to one workflowCode and/or
// filter. Step sorts lexically, same as source — "1.10" sorts before
// "1.2".
func (m *agencyManager) ListEventFlowSteps(ctx context.Context, workflowCode string, filter EventFlowStepFilter) ([]models.EventFlowStep, error) {
	q := m.db.WithContext(ctx).Table(m.tables.EventFlowSteps).
		Where("draft_id IS NULL").
		Order("workflow_code, ordinality, step")
	if workflowCode != "" {
		q = q.Where("workflow_code = ?", workflowCode)
	}
	q = likeFilter(q, "code", filter.Code)
	q = likeFilter(q, "name", filter.Name)
	q = likeFilter(q, "description", filter.Description)
	var rows []gormstore.EventFlowStepRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListEventFlowSteps: %w", err)
	}
	out := make([]models.EventFlowStep, len(rows))
	for i, r := range rows {
		out[i] = gormstore.EventFlowStepFromRow(r)
	}
	return out, nil
}
