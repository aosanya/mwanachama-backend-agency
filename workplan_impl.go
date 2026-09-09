// workplan_impl.go — WorkPlan lifecycle and event-topic matching for
// [agencyManager]. Ported from CodeValdAgency's raci.go (CreateWorkPlan/
// GetWorkPlan/ListWorkPlans/UpdateWorkPlan/MatchWorkPlans) — that file
// implements no RACI logic despite its name; see this repo's
// requirements.md and AG6 for the real RACI matrix, built separately.
package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// CreateWorkPlan stores a new WorkPlan. Returns [ErrInvalidRegex] if
// TriggerTopic or PayloadCondition does not compile as a Go regexp.
func (m *agencyManager) CreateWorkPlan(ctx context.Context, w models.WorkPlan) (models.WorkPlan, error) {
	if err := validateWorkPlanRegex(w.TriggerTopic, w.PayloadCondition); err != nil {
		return models.WorkPlan{}, err
	}
	if w.CreatedAt == "" {
		w.CreatedAt = models.NowRFC3339()
	}
	row := gormstore.WorkPlanToRow(w)
	if err := m.db.WithContext(ctx).Table(m.tables.WorkPlans).Create(&row).Error; err != nil {
		return models.WorkPlan{}, fmt.Errorf("CreateWorkPlan: %w", err)
	}
	return gormstore.WorkPlanFromRow(row), nil
}

// GetWorkPlan returns a WorkPlan by id. Returns [ErrWorkPlanNotFound] if no
// matching WorkPlan exists.
func (m *agencyManager) GetWorkPlan(ctx context.Context, id string) (models.WorkPlan, error) {
	row, err := m.findWorkPlanRow(ctx, m.db, id)
	if err != nil {
		return models.WorkPlan{}, err
	}
	return gormstore.WorkPlanFromRow(row), nil
}

func (m *agencyManager) findWorkPlanRow(ctx context.Context, tx *gorm.DB, id string) (gormstore.WorkPlanRow, error) {
	var row gormstore.WorkPlanRow
	err := tx.WithContext(ctx).Table(m.tables.WorkPlans).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.WorkPlanRow{}, ErrWorkPlanNotFound
	}
	if err != nil {
		return gormstore.WorkPlanRow{}, fmt.Errorf("findWorkPlanRow: %w", err)
	}
	return row, nil
}

// ListWorkPlans returns every non-deleted WorkPlan, id order.
func (m *agencyManager) ListWorkPlans(ctx context.Context) ([]models.WorkPlan, error) {
	var rows []gormstore.WorkPlanRow
	err := m.db.WithContext(ctx).Table(m.tables.WorkPlans).
		Where("deleted = ?", false).Order("id").Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("ListWorkPlans: %w", err)
	}
	out := make([]models.WorkPlan, len(rows))
	for i, r := range rows {
		out[i] = gormstore.WorkPlanFromRow(r)
	}
	return out, nil
}

// UpdateWorkPlan replaces w's mutable fields — see UpdateGoal's doc for why
// this writes an explicit column map rather than GORM's struct-based
// Updates. Returns [ErrInvalidRegex] if TriggerTopic or PayloadCondition
// does not compile as a Go regexp.
func (m *agencyManager) UpdateWorkPlan(ctx context.Context, w models.WorkPlan) (models.WorkPlan, error) {
	if err := validateWorkPlanRegex(w.TriggerTopic, w.PayloadCondition); err != nil {
		return models.WorkPlan{}, err
	}
	existing, err := m.findWorkPlanRow(ctx, m.db, w.ID)
	if err != nil {
		return models.WorkPlan{}, err
	}
	row := gormstore.WorkPlanToRow(w)
	row.CreatedAt = existing.CreatedAt
	cols := map[string]any{
		"code":                 row.Code,
		"name":                 row.Name,
		"description":          row.Description,
		"trigger_topic":        row.TriggerTopic,
		"payload_condition":    row.PayloadCondition,
		"instructions":         row.Instructions,
		"agent_id":             row.AgentID,
		"agent_code":           row.AgentCode,
		"function_code":        row.FunctionCode,
		"function_params":      row.FunctionParams,
		"handler_service":      row.HandlerService,
		"enabled":              row.Enabled,
		"ordinality":           row.Ordinality,
		"success_event":        row.SuccessEvent,
		"failure_event":        row.FailureEvent,
		"on_failure_pipeline":  row.OnFailurePipeline,
		"step_timeout":         row.StepTimeout,
		"review_step_type":     row.ReviewStepType,
		"review_trigger_topic": row.ReviewTriggerTopic,
		"review_success_topic": row.ReviewSuccessTopic,
		"review_failure_topic": row.ReviewFailureTopic,
		"updated_at":           row.UpdatedAt,
		"deleted":              row.Deleted,
	}
	if err := m.db.WithContext(ctx).Table(m.tables.WorkPlans).Where("id = ?", w.ID).Updates(cols).Error; err != nil {
		return models.WorkPlan{}, fmt.Errorf("UpdateWorkPlan: %w", err)
	}
	return gormstore.WorkPlanFromRow(row), nil
}

// MatchWorkPlans evaluates topic and payload against every enabled WorkPlan.
// TriggerTopic is compiled as a Go regex and matched against topic; if
// PayloadCondition is non-empty it is also matched against payload. Returns
// every matching WorkPlan with its ContextSources, ordered by Ordinality
// ascending.
func (m *agencyManager) MatchWorkPlans(ctx context.Context, topic, payload string) ([]models.WorkPlanMatch, error) {
	plans, err := m.ListWorkPlans(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchWorkPlans: %w", err)
	}

	var matches []models.WorkPlanMatch
	for _, plan := range plans {
		if !plan.Enabled {
			continue
		}
		topicRe, err := regexp.Compile(plan.TriggerTopic)
		if err != nil {
			continue // skip malformed regex — should not happen post-validation
		}
		if !topicRe.MatchString(topic) {
			continue
		}
		if plan.PayloadCondition != "" {
			payloadRe, err := regexp.Compile(plan.PayloadCondition)
			if err != nil {
				continue
			}
			if !payloadRe.MatchString(payload) {
				continue
			}
		}

		sources, err := m.ListContextSources(ctx, plan.ID)
		if err != nil {
			return nil, fmt.Errorf("MatchWorkPlans: sources for work plan %s: %w", plan.ID, err)
		}
		matches = append(matches, models.WorkPlanMatch{WorkPlan: plan, ContextSources: sources})
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].WorkPlan.Ordinality < matches[j].WorkPlan.Ordinality
	})
	return matches, nil
}

// validateWorkPlanRegex compiles triggerTopic and (if non-empty)
// payloadCondition as Go regular expressions, returning [ErrInvalidRegex]
// on the first failure.
func validateWorkPlanRegex(triggerTopic, payloadCondition string) error {
	if _, err := regexp.Compile(triggerTopic); err != nil {
		return fmt.Errorf("%w: trigger_topic: %v", ErrInvalidRegex, err)
	}
	if payloadCondition != "" {
		if _, err := regexp.Compile(payloadCondition); err != nil {
			return fmt.Errorf("%w: payload_condition: %v", ErrInvalidRegex, err)
		}
	}
	return nil
}
