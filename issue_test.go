package mwanachamaagency_test

import (
	"context"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func addUnlinkedWorkflow(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID string) models.Workflow {
	t.Helper()
	w, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftID, Name: "Unlinked flow"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	if _, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: draftID, WorkflowID: w.ID, Title: "Unlinked step"}); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	return w
}

func seedDraftForIssues(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context) (models.Draft, models.Goal, models.Objective) {
	t.Helper()
	_, g, _, _, _ := seedLiveAgency(t, mgr, ctx)
	d, err := mgr.CreateDraft(ctx, "review pass", g.DraftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	goals, err := mgr.ListGoals(ctx, d.ID, mwanachamaagency.GoalFilter{})
	if err != nil || len(goals) != 1 {
		t.Fatalf("ListGoals(draft): %v, %+v", err, goals)
	}
	objectives, err := mgr.ListObjectives(ctx, d.ID, mwanachamaagency.ObjectiveFilter{})
	if err != nil || len(objectives) != 1 {
		t.Fatalf("ListObjectives(draft): %v, %+v", err, objectives)
	}
	return d, goals[0], objectives[0]
}

func issuesFor(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID, entityType string) []models.Issue {
	t.Helper()
	issues, err := mgr.ListIssues(ctx, draftID, mwanachamaagency.IssueFilter{EntityType: entityType})
	if err != nil {
		t.Fatalf("ListIssues(%s): %v", entityType, err)
	}
	return issues
}

func TestListIssues_CleanSeedHasNoIssues(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	issues, err := mgr.ListIssues(ctx, o.DraftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected a linked workflow with a step to scan clean, got %+v", issues)
	}
}

func TestListIssues_ScopedToDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := o.DraftID
	w := addUnlinkedWorkflow(t, mgr, ctx, draftID)

	issues, err := mgr.ListIssues(ctx, draftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected exactly the workflow-without-objective issue, got %d: %+v", len(issues), issues)
	}
	if issues[0].EntityID != w.ID || issues[0].DraftID != draftID {
		t.Fatalf("expected an issue scoped to draft %s for workflow %s, got %+v", draftID, w.ID, issues[0])
	}
}

func TestListIssues_ScopesAreIndependent(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	draftA := o.DraftID
	addUnlinkedWorkflow(t, mgr, ctx, draftA)

	draftB, err := mgr.CreateDraft(ctx, "second draft", draftA, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	objectivesB, err := mgr.ListObjectives(ctx, draftB.ID, mwanachamaagency.ObjectiveFilter{})
	if err != nil || len(objectivesB) != 1 {
		t.Fatalf("ListObjectives(draftB): %v, %+v", err, objectivesB)
	}
	workflowsB, err := mgr.ListWorkflows(ctx, draftB.ID, mwanachamaagency.WorkflowFilter{})
	if err != nil || len(workflowsB) != 2 {
		t.Fatalf("ListWorkflows(draftB): %v, %+v", err, workflowsB)
	}
	for _, w := range workflowsB {
		links, err := mgr.ListWorkflowObjectives(ctx, draftB.ID, mwanachamaagency.WorkflowObjectiveFilter{WorkflowID: w.ID})
		if err != nil {
			t.Fatalf("ListWorkflowObjectives: %v", err)
		}
		if len(links) == 0 {
			if _, err := mgr.CreateWorkflowObjective(ctx, models.WorkflowObjective{
				DraftID: draftB.ID, WorkflowID: w.ID, ObjectiveID: objectivesB[0].ID,
			}); err != nil {
				t.Fatalf("CreateWorkflowObjective: %v", err)
			}
		}
	}

	issuesA, err := mgr.ListIssues(ctx, draftA, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues(draftA): %v", err)
	}
	if len(issuesA) != 1 {
		t.Fatalf("expected draftA's unlinked-workflow issue to still be there, got %+v", issuesA)
	}
	for _, i := range issuesA {
		if i.DraftID != draftA {
			t.Fatalf("draftA scan returned an out-of-scope issue: %+v", i)
		}
	}

	issuesB, err := mgr.ListIssues(ctx, draftB.ID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues(draftB): %v", err)
	}
	if len(issuesB) != 0 {
		t.Fatalf("expected draftB's workflows (now linked) to have no issue, got %+v", issuesB)
	}

	again, err := mgr.ListIssues(ctx, draftA, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues(draftA, again): %v", err)
	}
	if len(again) != 1 {
		t.Fatalf("draftB's scan affected draftA's issues: %+v", again)
	}
}

func TestListIssues_WorkflowWithNoObjective(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	d, _, _ := seedDraftForIssues(t, mgr, ctx)
	w := addUnlinkedWorkflow(t, mgr, ctx, d.ID)

	found := issuesFor(t, mgr, ctx, d.ID, "workflow")
	if len(found) != 1 || found[0].EntityID != w.ID {
		t.Fatalf("expected one workflow issue for %s, got %+v", w.ID, found)
	}
	if found[0].Severity != models.IssueSeverityError {
		t.Errorf("expected error severity, got %q", found[0].Severity)
	}
	if !strings.Contains(found[0].Message, "no Objective") {
		t.Errorf("expected the message to name the missing objective, got %q", found[0].Message)
	}
}

func TestListIssues_LinkingAnObjectiveClearsTheWorkflowIssue(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	d, _, o := seedDraftForIssues(t, mgr, ctx)
	w := addUnlinkedWorkflow(t, mgr, ctx, d.ID)

	if len(issuesFor(t, mgr, ctx, d.ID, "workflow")) != 1 {
		t.Fatal("precondition: the unlinked workflow should be flagged")
	}
	if _, err := mgr.CreateWorkflowObjective(ctx, models.WorkflowObjective{DraftID: d.ID, WorkflowID: w.ID, ObjectiveID: o.ID}); err != nil {
		t.Fatalf("CreateWorkflowObjective: %v", err)
	}
	if after := issuesFor(t, mgr, ctx, d.ID, "workflow"); len(after) != 0 {
		t.Fatalf("expected the issue gone once the workflow has an objective, got %+v", after)
	}
}

func TestListIssues_UnlinkingTheLastObjectiveFlagsTheWorkflow(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	d, _, o := seedDraftForIssues(t, mgr, ctx)

	links, err := mgr.ListWorkflowObjectives(ctx, d.ID, mwanachamaagency.WorkflowObjectiveFilter{ObjectiveID: o.ID})
	if err != nil || len(links) != 1 {
		t.Fatalf("ListWorkflowObjectives: %v, %+v", err, links)
	}
	if err := mgr.DeleteWorkflowObjective(ctx, links[0].ID); err != nil {
		t.Fatalf("DeleteWorkflowObjective: %v", err)
	}
	found := issuesFor(t, mgr, ctx, d.ID, "workflow")
	if len(found) != 1 || found[0].Severity != models.IssueSeverityError {
		t.Fatalf("expected an error for the workflow left without an objective, got %+v", found)
	}
}

func TestListIssues_AccountableIsNotPoliced(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	d, _, _ := seedDraftForIssues(t, mgr, ctx)
	items, err := mgr.ListWorkItems(ctx, d.ID, mwanachamaagency.WorkItemFilter{})
	if err != nil || len(items) != 1 {
		t.Fatalf("ListWorkItems: %v, %+v", err, items)
	}

	none, err := mgr.ListIssues(ctx, d.ID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("a step with no Accountable at all must scan clean, got %+v", none)
	}

	for _, name := range []string{"first-owner", "second-owner"} {
		role, err := mgr.CreateRole(ctx, models.Role{Name: name, DraftID: d.ID})
		if err != nil {
			t.Fatalf("CreateRole: %v", err)
		}
		if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
			RoleID: role.ID, WorkItemID: items[0].ID, Label: models.RACILabelAccountable, DraftID: d.ID,
		}); err != nil {
			t.Fatalf("CreateRoleAssignment(%s): %v", name, err)
		}
	}

	two, err := mgr.ListIssues(ctx, d.ID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(two) != 0 {
		t.Fatalf("two Accountables on one step must scan clean, got %+v", two)
	}
	for _, i := range two {
		if strings.Contains(i.Message, "Accountable") {
			t.Errorf("the scan still talks about Accountable: %q", i.Message)
		}
	}
}

func TestListIssues_GoalWithNoObjectives(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	d, _, _ := seedDraftForIssues(t, mgr, ctx)

	lonely, err := mgr.CreateGoal(ctx, models.Goal{Title: "Lonely goal", DraftID: d.ID})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}

	issues, err := mgr.ListIssues(ctx, d.ID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	found := false
	for _, i := range issues {
		if i.EntityType == "goal" && i.EntityID == lonely.ID {
			found = true
			if i.Severity != models.IssueSeverityError {
				t.Errorf("expected error severity, got %q", i.Severity)
			}
		}
	}
	if !found {
		t.Fatalf("expected an issue for goal %s, got %+v", lonely.ID, issues)
	}
}

func TestListIssues_Rescans(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	d, _, o := seedDraftForIssues(t, mgr, ctx)
	w := addUnlinkedWorkflow(t, mgr, ctx, d.ID)

	first, err := mgr.ListIssues(ctx, d.ID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues (1st): %v", err)
	}
	if len(first) != 1 {
		t.Fatalf("expected exactly 1 issue before linking the workflow, got %d: %+v", len(first), first)
	}

	if _, err := mgr.CreateWorkflowObjective(ctx, models.WorkflowObjective{DraftID: d.ID, WorkflowID: w.ID, ObjectiveID: o.ID}); err != nil {
		t.Fatalf("CreateWorkflowObjective: %v", err)
	}

	second, err := mgr.ListIssues(ctx, d.ID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues (2nd): %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("expected the issue to be gone on rescan, got %d: %+v", len(second), second)
	}
}

func TestListIssues_DanglingWorkflowObjective(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	d, _, o := seedDraftForIssues(t, mgr, ctx)
	links, err := mgr.ListWorkflowObjectives(ctx, d.ID, mwanachamaagency.WorkflowObjectiveFilter{ObjectiveID: o.ID})
	if err != nil || len(links) != 1 {
		t.Fatalf("ListWorkflowObjectives: %v, %+v", err, links)
	}
	table := tableOf(t, s, "workflow_objective")

	if err := db.Exec("update "+table+" set objective_id = ? where id = ?", "ghost-objective", links[0].ID).Error; err != nil {
		t.Fatalf("pointing the join at a missing objective: %v", err)
	}
	found := issuesFor(t, mgr, ctx, d.ID, "workflow_objective")
	if len(found) != 1 || found[0].EntityID != links[0].ID || found[0].Severity != models.IssueSeverityError {
		t.Fatalf("expected one error for the join with a missing objective, got %+v", found)
	}
	if !strings.Contains(found[0].Message, "ghost-objective") {
		t.Errorf("expected the message to name the missing objective, got %q", found[0].Message)
	}

	if err := db.Exec("update "+table+" set objective_id = ?, workflow_id = ? where id = ?", o.ID, "ghost-workflow", links[0].ID).Error; err != nil {
		t.Fatalf("pointing the join at a missing workflow: %v", err)
	}
	found = issuesFor(t, mgr, ctx, d.ID, "workflow_objective")
	if len(found) != 1 || !strings.Contains(found[0].Message, "ghost-workflow") {
		t.Fatalf("expected one error naming the missing workflow, got %+v", found)
	}
}

func TestListIssues_DanglingRoleAssignment(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	d, _, _ := seedDraftForIssues(t, mgr, ctx)
	items, err := mgr.ListWorkItems(ctx, d.ID, mwanachamaagency.WorkItemFilter{})
	if err != nil || len(items) != 1 {
		t.Fatalf("ListWorkItems: %v, %+v", err, items)
	}
	role, err := mgr.CreateRole(ctx, models.Role{Name: "owner", DraftID: d.ID})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	a, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		RoleID: role.ID, WorkItemID: items[0].ID, Label: models.RACILabelResponsible, DraftID: d.ID,
	})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}
	table := tableOf(t, s, "role_assignment")

	if err := db.Exec("update "+table+" set work_item_id = ? where id = ?", "ghost-step", a.ID).Error; err != nil {
		t.Fatalf("pointing the assignment at a missing step: %v", err)
	}
	found := issuesFor(t, mgr, ctx, d.ID, "role_assignment")
	if len(found) != 1 || found[0].EntityID != a.ID || found[0].Severity != models.IssueSeverityError {
		t.Fatalf("expected one error for the assignment with a missing step, got %+v", found)
	}
	if !strings.Contains(found[0].Message, "ghost-step") {
		t.Errorf("expected the message to name the missing step, got %q", found[0].Message)
	}

	if err := db.Exec("update "+table+" set work_item_id = ?, role_id = ? where id = ?", items[0].ID, "ghost-role", a.ID).Error; err != nil {
		t.Fatalf("pointing the assignment at a missing role: %v", err)
	}
	found = issuesFor(t, mgr, ctx, d.ID, "role_assignment")
	if len(found) != 1 || !strings.Contains(found[0].Message, "ghost-role") {
		t.Fatalf("expected one error naming the missing role, got %+v", found)
	}
}

func TestListIssues_FilterBySeverity(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	d, _, _ := seedDraftForIssues(t, mgr, ctx)
	if _, err := mgr.CreateGoal(ctx, models.Goal{Title: "Lonely goal", DraftID: d.ID}); err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}

	errs, err := mgr.ListIssues(ctx, d.ID, mwanachamaagency.IssueFilter{Severity: "error"})
	if err != nil {
		t.Fatalf("ListIssues(error): %v", err)
	}
	for _, i := range errs {
		if i.Severity != models.IssueSeverityError {
			t.Errorf("filter leaked a %q severity issue", i.Severity)
		}
	}
	if len(errs) == 0 {
		t.Fatal("expected at least one error (the goal with no objective)")
	}

	warnings, err := mgr.ListIssues(ctx, d.ID, mwanachamaagency.IssueFilter{Severity: "warning"})
	if err != nil {
		t.Fatalf("ListIssues(warning): %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("no check emits a warning yet, got %+v", warnings)
	}
}
