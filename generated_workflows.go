package mwanachamaagency

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

type generatedPlan struct {
	record   models.Record
	action   string
	workflow models.Workflow
	step     models.WorkItem
}

func generatedCode(recordCode, action string) string {
	return recordCode + "." + action
}

func itemNameOf(r models.Record) string {
	for _, name := range []string{r.ItemName, r.FormName, r.Name} {
		if strings.TrimSpace(name) != "" {
			return strings.TrimSpace(name)
		}
	}
	return r.Code
}

func withArticle(item string) string {
	lower := strings.ToLower(item)
	if strings.ContainsAny(lower[:1], "aeiou") {
		return "an " + lower
	}
	return "a " + lower
}

var generatedVerbs = map[string]string{
	models.RowActionAdd:    "Add",
	models.RowActionEdit:   "Edit",
	models.RowActionDelete: "Delete",
}

func generatedPlansOf(r models.Record) []generatedPlan {
	item := itemNameOf(r)
	replacedBy := map[string]string{
		models.RowActionAdd: r.AddReplacedBy, models.RowActionEdit: r.EditReplacedBy, models.RowActionDelete: r.DeleteReplacedBy,
	}
	var plans []generatedPlan
	for _, action := range models.RowActions() {
		code := generatedCode(r.Identity(), action)
		name := generatedVerbs[action] + " " + withArticle(item)
		formName, questions := r.FormName, r.Questions
		if action == models.RowActionDelete {
			formName, questions = "Why delete this "+strings.ToLower(item), []models.StepQuestion{models.DeleteReasonQuestion()}
		}
		plans = append(plans, generatedPlan{
			record: r, action: action,
			workflow: models.Workflow{
				DraftID: r.DraftID, Code: code, Name: name, Generated: true, GeneratedFor: r.Identity(), ReplacedBy: replacedBy[action],
				Description: fmt.Sprintf("Made from the record %s: %s a row of it.", r.Name, strings.ToLower(generatedVerbs[action])),
			},
			step: models.WorkItem{
				DraftID: r.DraftID, Code: code + ".1", Title: name, Ordinality: 1, Kind: models.StepKindFillForm,
				RowRecord: r.Identity(), RowAction: action, FormName: formName, Questions: questions,
			},
		})
	}
	return plans
}

func sameQuestions(a, b []models.StepQuestion) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return string(left) == string(right) || (len(a) == 0 && len(b) == 0)
}

func (p generatedPlan) workflowDifference(w models.Workflow) string {
	switch {
	case !w.Generated || w.GeneratedFor != p.workflow.GeneratedFor:
		return "it is not marked as made from the record"
	case w.Name != p.workflow.Name:
		return fmt.Sprintf("it is named %q, not %q", w.Name, p.workflow.Name)
	case w.ReplacedBy != p.workflow.ReplacedBy:
		return fmt.Sprintf("it is replaced by %q, not %q", w.ReplacedBy, p.workflow.ReplacedBy)
	}
	return ""
}

func (p generatedPlan) stepDifference(s models.WorkItem) string {
	switch {
	case s.Kind != models.StepKindFillForm:
		return fmt.Sprintf("its step is %q, not a fill_form step", s.Kind)
	case s.RowRecord != p.step.RowRecord || s.RowAction != p.step.RowAction:
		return fmt.Sprintf("its step writes %q %q, not %q %q", s.RowRecord, s.RowAction, p.step.RowRecord, p.step.RowAction)
	case s.RoleID != p.step.RoleID:
		return fmt.Sprintf("its step is held by role %q, not by %q", s.RoleID, p.step.RoleID)
	case s.FormName != p.step.FormName:
		return fmt.Sprintf("its form is %q, not %q", s.FormName, p.step.FormName)
	case !sameQuestions(s.Questions, p.step.Questions):
		return "its form asks other questions than the record's now"
	case s.Title != p.step.Title:
		return fmt.Sprintf("its step is titled %q, not %q", s.Title, p.step.Title)
	}
	return ""
}

type generatedState struct {
	records   []models.Record
	workflows map[string][]models.Workflow
	steps     map[string][]models.WorkItem
}

func (m *agencyManager) generatedStateOf(ctx context.Context, tx *gorm.DB, draftID string) (generatedState, error) {
	scoped := func(role string) *gorm.DB {
		return scopeQuery(tx.WithContext(ctx).Table(m.table(role)), draftID)
	}
	records, err := list[models.Record](m, scoped(roleRecord).Order("created_at"), roleRecord)
	if err != nil {
		return generatedState{}, err
	}
	state := generatedState{workflows: map[string][]models.Workflow{}, steps: map[string][]models.WorkItem{}}
	for _, r := range records {
		if !r.Deleted {
			state.records = append(state.records, r)
		}
	}
	workflows, err := list[models.Workflow](m, scoped(roleWorkflow).Where("generated = ?", true).Order("created_at, id"), roleWorkflow)
	if err != nil {
		return generatedState{}, err
	}
	var ids []string
	for _, w := range workflows {
		state.workflows[w.Code] = append(state.workflows[w.Code], w)
		ids = append(ids, w.ID)
	}
	if len(ids) == 0 {
		return state, nil
	}
	steps, err := list[models.WorkItem](m, scoped(roleWorkItem).Where("workflow_id IN ?", ids).Order("created_at, id"), roleWorkItem)
	if err != nil {
		return generatedState{}, err
	}
	for _, s := range steps {
		state.steps[s.WorkflowID] = append(state.steps[s.WorkflowID], s)
	}
	return state, nil
}

func (m *agencyManager) insertGeneratedRow(ctx context.Context, tx *gorm.DB, role string, v any) error {
	row, err := m.encode(role, v)
	if err != nil {
		return err
	}
	return tx.WithContext(ctx).Table(m.table(role)).Create(row).Error
}

func (m *agencyManager) removeGeneratedWorkflow(ctx context.Context, tx *gorm.DB, draftID, workflowID string) error {
	stepsOfWorkflow := "select id from " + m.table(roleWorkItem) + " where workflow_id = ?"
	if err := m.deleteInDraft(ctx, tx, roleScenarioInput, draftID, "step_id IN ("+stepsOfWorkflow+")", workflowID); err != nil {
		return err
	}
	if err := m.deleteInDraft(ctx, tx, roleWorkItem, draftID, "workflow_id = ?", workflowID); err != nil {
		return err
	}
	return m.deleteInDraft(ctx, tx, roleWorkflow, draftID, "id = ?", workflowID)
}

func (m *agencyManager) ensureGenerated(ctx context.Context, tx *gorm.DB, draftID string) error {
	state, err := m.generatedStateOf(ctx, tx, draftID)
	if err != nil {
		return err
	}
	kept := map[string]bool{}
	now := models.NowRFC3339()
	for _, record := range state.records {
		for _, plan := range generatedPlansOf(record) {
			kept[plan.workflow.Code] = true
			if err := m.settleGenerated(ctx, tx, state, plan, now); err != nil {
				return err
			}
		}
	}
	for code, held := range state.workflows {
		if kept[code] {
			continue
		}
		for _, w := range held {
			if err := m.removeGeneratedWorkflow(ctx, tx, draftID, w.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *agencyManager) settleGenerated(ctx context.Context, tx *gorm.DB, state generatedState, plan generatedPlan, now string) error {
	held := state.workflows[plan.workflow.Code]
	if len(held) == 0 {
		wf := plan.workflow
		wf.ID, wf.CreatedAt, wf.LastUpdated = newID(), now, now
		if err := m.insertGeneratedRow(ctx, tx, roleWorkflow, wf); err != nil {
			return err
		}
		return m.insertGeneratedStep(ctx, tx, plan, wf.ID, now)
	}
	keep := held[0]
	for _, extra := range held[1:] {
		if err := m.removeGeneratedWorkflow(ctx, tx, plan.record.DraftID, extra.ID); err != nil {
			return err
		}
	}
	if plan.workflowDifference(keep) != "" {
		wf := plan.workflow
		wf.ID, wf.LastUpdated = keep.ID, now
		if err := m.writeUpdate(ctx, tx, roleWorkflow, keep.ID, wf, ErrWorkflowNotFound); err != nil {
			return err
		}
	}
	steps := state.steps[keep.ID]
	if len(steps) == 0 {
		return m.insertGeneratedStep(ctx, tx, plan, keep.ID, now)
	}
	for _, extra := range steps[1:] {
		if err := m.deleteInDraft(ctx, tx, roleWorkItem, plan.record.DraftID, "id = ?", extra.ID); err != nil {
			return err
		}
	}
	if plan.stepDifference(steps[0]) == "" {
		return nil
	}
	step := plan.step
	step.ID, step.WorkflowID, step.LastUpdated = steps[0].ID, keep.ID, now
	return m.writeUpdate(ctx, tx, roleWorkItem, steps[0].ID, step, ErrWorkItemNotFound)
}

func (m *agencyManager) insertGeneratedStep(ctx context.Context, tx *gorm.DB, plan generatedPlan, workflowID, now string) error {
	step := plan.step
	step.ID, step.WorkflowID, step.CreatedAt, step.LastUpdated = newID(), workflowID, now, now
	return m.insertGeneratedRow(ctx, tx, roleWorkItem, step)
}

func (m *agencyManager) EnsureGeneratedWorkflows(ctx context.Context, draftID string) error {
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := m.findDraft(ctx, tx, draftID); err != nil {
			return err
		}
		return m.ensureGenerated(ctx, tx, draftID)
	})
	if err != nil {
		return fmt.Errorf("EnsureGeneratedWorkflows: %w", err)
	}
	return nil
}

func (m *agencyManager) refuseGeneratedWorkflow(ctx context.Context, tx *gorm.DB, draftID, workflowID string) error {
	if workflowID == "" {
		return nil
	}
	var w models.Workflow
	q := scopeQuery(tx.WithContext(ctx).Table(m.table(roleWorkflow)), draftID).Where("id = ?", workflowID)
	if err := m.take(q, roleWorkflow, &w, ErrWorkflowNotFound); err != nil {
		return nil
	}
	if w.Generated {
		return fmt.Errorf("%w: %s %q", ErrGeneratedWorkflow, w.Code, w.Name)
	}
	return nil
}

func (s *issueScan) checkGeneratedWorkflows() error {
	state, err := s.m.generatedStateOf(s.ctx, s.tx, s.draftID)
	if err != nil {
		return err
	}
	workflows, err := list[models.Workflow](s.m, s.scoped(roleWorkflow), roleWorkflow)
	if err != nil {
		return err
	}
	steps, err := list[models.WorkItem](s.m, s.scoped(roleWorkItem), roleWorkItem)
	if err != nil {
		return err
	}
	expected := map[string]bool{}
	for _, record := range state.records {
		s.checkGeneratedFor(state, record, expected)
		s.checkStandIns(record, workflows, steps, state)
	}
	for code, held := range state.workflows {
		for _, w := range held {
			if !expected[code] {
				s.add(models.IssueSeverityError, "workflow", w.ID,
					"Generated workflow %s %q was made for Record %s, which does not exist", w.Code, w.Name, QuoteIfUnsafe(w.GeneratedFor))
			}
		}
	}
	return nil
}

func (s *issueScan) checkGeneratedFor(state generatedState, record models.Record, expected map[string]bool) {
	for _, plan := range generatedPlansOf(record) {
		code := plan.workflow.Code
		expected[code] = true
		held := state.workflows[code]
		switch {
		case len(held) == 0:
			s.add(models.IssueSeverityError, "record", record.ID,
				"Record %s %q has no generated workflow %s (%s)", record.Code, record.Name, code, plan.workflow.Name)
		case len(held) > 1:
			s.add(models.IssueSeverityError, "workflow", held[1].ID,
				"Record %s %q has its generated workflow %s %d times", record.Code, record.Name, code, len(held))
		default:
			s.checkGeneratedMatches(record, plan, held[0], state.steps[held[0].ID])
		}
	}
}

func (s *issueScan) checkGeneratedMatches(record models.Record, plan generatedPlan, w models.Workflow, steps []models.WorkItem) {
	problem := plan.workflowDifference(w)
	if problem == "" && len(steps) != 1 {
		problem = fmt.Sprintf("it holds %d steps, not one", len(steps))
	}
	if problem == "" {
		problem = plan.stepDifference(steps[0])
	}
	if problem != "" {
		s.add(models.IssueSeverityError, "workflow", w.ID,
			"Record %s %q: its generated workflow %s is out of date: %s", record.Code, record.Name, plan.workflow.Code, problem)
	}
}

func (s *issueScan) checkStandIns(record models.Record, workflows []models.Workflow, steps []models.WorkItem, state generatedState) {
	for _, plan := range generatedPlansOf(record) {
		standIn := plan.workflow.ReplacedBy
		if standIn == "" {
			continue
		}
		var designed *models.Workflow
		for i, w := range workflows {
			if w.Code == standIn && !w.Generated {
				designed = &workflows[i]
			}
		}
		if designed == nil {
			s.add(models.IssueSeverityError, "record", record.ID,
				"Record %s %q says Workflow %s stands in for %s, and no designed workflow has that code", record.Code, record.Name, QuoteIfUnsafe(standIn), plan.workflow.Code)
			continue
		}
		held := state.workflows[plan.workflow.Code]
		if len(held) == 1 && !stepsRun(steps, designed.ID, held[0].ID) {
			s.add(models.IssueSeverityWarning, "workflow", designed.ID,
				"Workflow %s %q stands in for %s but has no step that runs it", designed.Code, designed.Name, plan.workflow.Code)
		}
	}
}

func stepsRun(steps []models.WorkItem, inWorkflow, target string) bool {
	for _, s := range steps {
		if s.WorkflowID == inWorkflow && s.Kind == models.StepKindRunWorkflow && s.RunWorkflowID == target {
			return true
		}
	}
	return false
}

func (m *agencyManager) validateStandIns(ctx context.Context, tx *gorm.DB, r models.Record) error {
	for action, code := range map[string]string{
		models.RowActionAdd: r.AddReplacedBy, models.RowActionEdit: r.EditReplacedBy, models.RowActionDelete: r.DeleteReplacedBy,
	} {
		if code == "" {
			continue
		}
		var w models.Workflow
		q := scopeQuery(tx.WithContext(ctx).Table(m.table(roleWorkflow)), r.DraftID).Where("code = ?", code)
		if err := m.take(q, roleWorkflow, &w, ErrWorkflowNotFound); err != nil {
			return invalidRecord("the workflow %s that stands in for the %s of %q is not a workflow of this draft", QuoteIfUnsafe(code), action, r.Name)
		}
		if w.Generated {
			return invalidRecord("a generated workflow cannot stand in for the %s of %q, only a designed one", action, r.Name)
		}
	}
	return nil
}
