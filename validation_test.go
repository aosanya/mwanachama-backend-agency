package mwanachamaagency_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func findIssue(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID, entityType, entityID string) (models.Issue, bool) {
	t.Helper()
	issues, err := mgr.ListIssues(ctx, draftID, mwanachamaagency.IssueFilter{EntityType: entityType})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	for _, i := range issues {
		if i.EntityID == entityID {
			return i, true
		}
	}
	return models.Issue{}, false
}

type linkedWorld struct {
	draftID   string
	goal      models.Goal
	objective models.Objective
	workflow  models.Workflow
	step      models.WorkItem
	role      models.Role
	link      models.WorkflowObjective
}

func newLinkedWorld(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context) linkedWorld {
	t.Helper()
	w := linkedWorld{draftID: newDraftAgency(t, mgr, ctx)}
	var err error
	if w.goal, err = mgr.CreateGoal(ctx, models.Goal{DraftID: w.draftID, Title: "Grow adoption"}); err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if w.objective, err = mgr.CreateObjective(ctx, models.Objective{DraftID: w.draftID, GoalID: w.goal.ID, Title: "Grow adoption"}); err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	if w.workflow, err = mgr.CreateWorkflow(ctx, models.Workflow{DraftID: w.draftID, Name: "Onboarding"}); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	w.link = linkWorkflowToObjective(t, mgr, w.draftID, w.workflow.ID, w.objective.ID)
	if w.step, err = mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: w.draftID, WorkflowID: w.workflow.ID, Title: "Write charter"}); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	if w.role, err = mgr.CreateRole(ctx, models.Role{DraftID: w.draftID, Name: "bug-hunter"}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	return w
}

func TestDeleteObjective_RemovesItsWorkflowObjectivesAndKeepsTheWorkflow(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	w := newLinkedWorld(t, mgr, ctx)

	if err := mgr.DeleteObjective(ctx, w.objective.ID); err != nil {
		t.Fatalf("DeleteObjective: %v", err)
	}
	if _, err := mgr.GetWorkflowObjective(ctx, w.link.ID); !errors.Is(err, mwanachamaagency.ErrWorkflowObjectiveNotFound) {
		t.Fatalf("expected the cascaded WorkflowObjective to be gone, got %v", err)
	}
	if _, err := mgr.GetWorkflow(ctx, w.workflow.ID); err != nil {
		t.Fatalf("expected the Workflow to survive the Objective's deletion, got %v", err)
	}
	if _, err := mgr.GetWorkItem(ctx, w.step.ID); err != nil {
		t.Fatalf("expected the step to survive the Objective's deletion, got %v", err)
	}
}

func TestDeleteGoal_RemovesTheWorkflowObjectivesOfItsObjectives(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	w := newLinkedWorld(t, mgr, ctx)

	if err := mgr.DeleteGoal(ctx, w.goal.ID); err != nil {
		t.Fatalf("DeleteGoal: %v", err)
	}
	if _, err := mgr.GetWorkflowObjective(ctx, w.link.ID); !errors.Is(err, mwanachamaagency.ErrWorkflowObjectiveNotFound) {
		t.Fatalf("expected the cascaded WorkflowObjective to be gone, got %v", err)
	}
	if _, err := mgr.GetWorkflow(ctx, w.workflow.ID); err != nil {
		t.Fatalf("expected the Workflow to survive the Goal's deletion, got %v", err)
	}
}

func TestDeleteRole_NoRoleAssignments(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	d, _, _ := seedDraftForIssues(t, mgr, ctx)

	role, err := mgr.CreateRole(ctx, models.Role{Name: "unused", DraftID: d.ID})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if err := mgr.DeleteRole(ctx, role.ID); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}
}

func TestScan_DanglingWorkflowObjective(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	w := newLinkedWorld(t, mgr, ctx)

	if err := db.Exec("delete from "+tableOf(t, s, "workflow")+" where id = ?", w.workflow.ID).Error; err != nil {
		t.Fatalf("deleting the Workflow row directly: %v", err)
	}
	issue, ok := findIssue(t, mgr, ctx, w.draftID, "workflow_objective", w.link.ID)
	if !ok {
		t.Fatalf("expected a dangling-Workflow issue for link %s", w.link.ID)
	}
	if issue.Severity != models.IssueSeverityError || !strings.Contains(issue.Message, "does not exist") {
		t.Errorf("unexpected issue %+v", issue)
	}
}

func TestScan_WorkflowObjectiveWhoseObjectiveIsGone(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	w := newLinkedWorld(t, mgr, ctx)

	if err := db.Exec("delete from "+tableOf(t, s, "objective")+" where id = ?", w.objective.ID).Error; err != nil {
		t.Fatalf("deleting the Objective row directly: %v", err)
	}
	issue, ok := findIssue(t, mgr, ctx, w.draftID, "workflow_objective", w.link.ID)
	if !ok {
		t.Fatalf("expected a dangling-Objective issue for link %s", w.link.ID)
	}
	if issue.Severity != models.IssueSeverityError || !strings.Contains(issue.Message, "Objective") {
		t.Errorf("unexpected issue %+v", issue)
	}
}

func TestScan_AccountableRoleWhoseRoleIsGone(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	w := newLinkedWorld(t, mgr, ctx)

	step := w.step
	step.AccountableRoleID = w.role.ID
	if _, err := mgr.UpdateWorkItem(ctx, step); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}

	if err := db.Exec("delete from "+tableOf(t, s, "role")+" where id = ?", w.role.ID).Error; err != nil {
		t.Fatalf("deleting the Role row directly: %v", err)
	}

	issue, ok := findIssue(t, mgr, ctx, w.draftID, "work_item", step.ID)
	if !ok {
		t.Fatalf("expected a dangling-Role issue for step %s", step.ID)
	}
	if issue.Severity != models.IssueSeverityError || !strings.Contains(issue.Message, "accountable") {
		t.Errorf("unexpected issue %+v", issue)
	}
}

func TestScan_DanglingAttachment(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, _, _, _ := seedLiveAgency(t, mgr, ctx)

	rule, err := mgr.CreateRule(ctx, models.Rule{DraftID: g.DraftID, Text: "never touch production data"})
	if err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	attachment, err := mgr.CreateRuleAttachment(ctx, rule.ID, models.TargetTypeGoal, g.ID)
	if err != nil {
		t.Fatalf("CreateRuleAttachment: %v", err)
	}

	if err := mgr.DeleteGoal(ctx, g.ID); err != nil {
		t.Fatalf("DeleteGoal: %v", err)
	}

	issue, ok := findIssue(t, mgr, ctx, g.DraftID, "rule_attachment", attachment.ID)
	if !ok {
		t.Fatalf("expected a dangling-target issue for rule attachment %s", attachment.ID)
	}
	if issue.Severity != models.IssueSeverityError {
		t.Errorf("expected error severity, got %q", issue.Severity)
	}
}

func TestScan_EmptyWorkflowAndAgency(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)

	if _, ok := findIssue(t, mgr, ctx, draftID, "agency", ""); !ok {
		t.Fatal("expected an issue for an Agency with no Goals")
	}

	g, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if _, err := mgr.CreateObjective(ctx, models.Objective{DraftID: draftID, GoalID: g.ID, Title: "Grow adoption"}); err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	empty, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftID, Name: "Empty flow"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	issue, ok := findIssue(t, mgr, ctx, draftID, "workflow", empty.ID)
	if !ok {
		t.Fatalf("expected an issue for workflow %s with no work items", empty.ID)
	}
	if issue.Severity != models.IssueSeverityError {
		t.Errorf("expected error severity, got %q", issue.Severity)
	}
}

func TestScan_DanglingEventFlowStep(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, _, _, _ := seedLiveAgency(t, mgr, ctx)

	step, err := mgr.CreateEventFlowStep(ctx, models.EventFlowStep{
		DraftID: g.DraftID, Code: "step-1", WorkflowCode: "W-404", Step: "1", Name: "Orphan step",
	})
	if err != nil {
		t.Fatalf("CreateEventFlowStep: %v", err)
	}

	issue, ok := findIssue(t, mgr, ctx, g.DraftID, "event_flow_step", step.ID)
	if !ok {
		t.Fatalf("expected a dangling-Workflow issue for step %s", step.ID)
	}
	if issue.Severity != models.IssueSeverityError {
		t.Errorf("expected error severity, got %q", issue.Severity)
	}
}
