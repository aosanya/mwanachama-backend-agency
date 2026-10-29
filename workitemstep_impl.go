package mwanachamaagency

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func invalidStep(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidWorkItem, fmt.Sprintf(format, args...))
}

func validateStepShape(wi models.WorkItem) error {
	if strings.ContainsAny(wi.RoleID, ", \t\r\n;") {
		return invalidStep("a step has one role, got %q", wi.RoleID)
	}
	if wi.Kind == "" {
		if wi.RoleID != "" || wi.RunWorkflowID != "" || wi.FormName != "" || len(wi.Questions) > 0 {
			return invalidStep("a step with a role, a form or a workflow to run needs a kind")
		}
		return nil
	}
	if !wi.Kind.Valid() {
		return invalidStep("kind %q is not one of %v", wi.Kind, models.StepKinds())
	}
	if wi.Kind.AgencyOnly() && wi.RoleID != "" {
		return invalidStep("a %s step is always the agency's, so it names no role", wi.Kind)
	}
	if wi.Kind == models.StepKindRunWorkflow && wi.RunWorkflowID == "" {
		return invalidStep("a run_workflow step names the workflow it runs")
	}
	if wi.Kind != models.StepKindRunWorkflow && wi.RunWorkflowID != "" {
		return invalidStep("only a run_workflow step names a workflow to run")
	}
	return validateStepForm(wi)
}

func validateStepForm(wi models.WorkItem) error {
	if !wi.Kind.TakesForm() {
		if wi.FormName != "" || len(wi.Questions) > 0 {
			return invalidStep("a %s step takes no form", wi.Kind)
		}
		return nil
	}
	if wi.FormName == "" {
		return invalidStep("a %s step names its form", wi.Kind)
	}
	if len(wi.Questions) == 0 {
		return invalidStep("the form %q of a %s step asks at least one question", wi.FormName, wi.Kind)
	}
	for i, q := range wi.Questions {
		if err := validateStepQuestion(i+1, q); err != nil {
			return err
		}
	}
	return nil
}

func validateStepQuestion(number int, q models.StepQuestion) error {
	if strings.TrimSpace(q.Prompt) == "" {
		return invalidStep("question %d has no prompt", number)
	}
	if !q.AnswerType.Valid() {
		return invalidStep("question %d: answer type %q is not one of yes_no, single_choice, free_text, date, number, currency", number, q.AnswerType)
	}
	pickOne := q.AnswerType == models.AnswerSingleChoice
	if pickOne && len(q.Options) == 0 && q.OptionsFrom == "" {
		return invalidStep("question %d is pick-one and names neither its options nor the list they come from", number)
	}
	if !pickOne && (len(q.Options) > 0 || q.OptionsFrom != "") {
		return invalidStep("question %d has options but is not pick-one", number)
	}
	return nil
}

func (m *agencyManager) validateStepReferences(ctx context.Context, tx *gorm.DB, wi models.WorkItem) error {
	if wi.RoleID != "" {
		var r models.Role
		q := scopeQuery(tx.WithContext(ctx).Table(m.table(roleRole)), wi.DraftID).Where("id = ?", wi.RoleID)
		if err := m.take(q, roleRole, &r, ErrRoleNotFound); err != nil {
			return err
		}
	}
	if wi.RunWorkflowID != "" {
		var w models.Workflow
		q := scopeQuery(tx.WithContext(ctx).Table(m.table(roleWorkflow)), wi.DraftID).Where("id = ?", wi.RunWorkflowID)
		if err := m.take(q, roleWorkflow, &w, ErrWorkflowNotFound); err != nil {
			return err
		}
		if wi.RunWorkflowID == wi.WorkflowID {
			return invalidStep("a step cannot run the workflow that owns it")
		}
	}
	return nil
}

func (m *agencyManager) validateStep(ctx context.Context, tx *gorm.DB, wi models.WorkItem) error {
	if err := validateStepShape(wi); err != nil {
		return err
	}
	return m.validateStepReferences(ctx, tx, wi)
}

func (m *agencyManager) withActions(ctx context.Context, items ...models.WorkItem) ([]models.WorkItem, error) {
	names := map[string]string{}
	var runIDs []string
	for _, wi := range items {
		if wi.RunWorkflowID != "" {
			runIDs = append(runIDs, wi.RunWorkflowID)
		}
	}
	if len(runIDs) > 0 {
		workflows, err := list[models.Workflow](m, m.q(ctx, roleWorkflow).Where("id IN ?", runIDs), roleWorkflow)
		if err != nil {
			return nil, err
		}
		for _, w := range workflows {
			names[w.ID] = w.Name
		}
	}
	out := make([]models.WorkItem, len(items))
	for i, wi := range items {
		wi.Action = models.ActionOf(wi.Kind, wi.FormName, names[wi.RunWorkflowID])
		out[i] = wi
	}
	return out, nil
}

func (m *agencyManager) withAction(ctx context.Context, wi models.WorkItem) (models.WorkItem, error) {
	out, err := m.withActions(ctx, wi)
	if err != nil {
		return models.WorkItem{}, err
	}
	return out[0], nil
}
