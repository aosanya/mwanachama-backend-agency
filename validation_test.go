package mwanachamaagency_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// findIssue returns the first live-scope Issue of entityType naming
// entityID, and whether one was found.
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

// --- AG27: the delete cascades that used to produce the orphans below ---

// TestDeleteObjective_CascadesWorkItems covers the gap AGD-004 found:
// WorkItem.ObjectiveID is required, so a surviving WorkItem is
// unreachable, not merely untagged.
func TestDeleteObjective_CascadesWorkItems(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, wi := seedLiveAgency(t, mgr, ctx)

	if err := mgr.DeleteObjective(ctx, o.ID); err != nil {
		t.Fatalf("DeleteObjective: %v", err)
	}
	if _, err := mgr.GetWorkItem(ctx, wi.ID); !errors.Is(err, mwanachamaagency.ErrWorkItemNotFound) {
		t.Fatalf("expected the cascaded WorkItem to be gone, got %v", err)
	}
}

// TestDeleteGoal_CascadesWorkItems is the same cascade one level up — the
// Goal's Objectives go, and so must the WorkItems under them.
func TestDeleteGoal_CascadesWorkItems(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, _, _, wi := seedLiveAgency(t, mgr, ctx)

	if err := mgr.DeleteGoal(ctx, g.ID); err != nil {
		t.Fatalf("DeleteGoal: %v", err)
	}
	if _, err := mgr.GetWorkItem(ctx, wi.ID); !errors.Is(err, mwanachamaagency.ErrWorkItemNotFound) {
		t.Fatalf("expected the cascaded WorkItem to be gone, got %v", err)
	}
}

// TestDeleteRole_CascadesRoleAssignments is DeleteObjective's cascade seen
// from the other side of the same join table: deleting a Role must not
// leave assignments naming it behind.
func TestDeleteRole_CascadesRoleAssignments(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	d, _, o := seedDraftForIssues(t, mgr, ctx)

	role, err := mgr.CreateRole(ctx, models.Role{Name: "bug-hunter", DraftID: d.ID})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	assignment, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		RoleID: role.ID, ObjectiveID: o.ID, Label: models.RACILabelAccountable, DraftID: d.ID,
	})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	if err := mgr.DeleteRole(ctx, role.ID); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}
	if _, err := mgr.GetRoleAssignment(ctx, assignment.ID); !errors.Is(err, mwanachamaagency.ErrRoleAssignmentNotFound) {
		t.Fatalf("expected the cascaded RoleAssignment to be gone, got %v", err)
	}
	// The Objective is a separate entity — only the assignment referenced
	// the Role, so it must survive.
	if _, err := mgr.GetObjective(ctx, o.ID); err != nil {
		t.Fatalf("expected the Objective to survive the Role's deletion, got %v", err)
	}
}

// TestDeleteRole_NoRoleAssignments asserts the cascade's delete call does
// not choke on a Role nothing references.
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

// --- AG26: the dangling-reference checks, driven through the one delete
// path that can still strand a row ---

// TestScan_DanglingWorkItem uses the path that is still open after AG27:
// CreateWorkItem only rejects an *empty* ObjectiveID, so a caller can
// still name one that does not exist.
func TestScan_DanglingWorkItem(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, _, w, _ := seedLiveAgency(t, mgr, ctx)

	orphan, err := mgr.CreateWorkItem(ctx, models.WorkItem{
		WorkflowID: w.ID, ObjectiveID: "no-such-objective", Title: "Orphan",
	})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}

	issue, ok := findIssue(t, mgr, ctx, "", "work_item", orphan.ID)
	if !ok {
		t.Fatalf("expected a dangling-Objective issue for work item %s", orphan.ID)
	}
	if issue.Severity != models.IssueSeverityError {
		t.Errorf("expected error severity, got %q", issue.Severity)
	}
	if !strings.Contains(issue.Message, "does not exist") {
		t.Errorf("unexpected message %q", issue.Message)
	}
}

// TestScan_DanglingRoleAssignment covers the other half of the same
// family: an assignment whose Role row is gone. CreateRoleAssignment
// validates both ends and AG27 now cascades, so the only way to produce
// one is the way it was produced before AG27 — the Role row vanishing
// underneath it. That data still exists in any database that ran the old
// DeleteRole, which is exactly what the scan is for.
func TestScan_DanglingRoleAssignment(t *testing.T) {
	mgr, db, tables := newTestManagerDB(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	role, err := mgr.CreateRole(ctx, models.Role{Name: "owner"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	assignment, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		RoleID: role.ID, ObjectiveID: o.ID, Label: models.RACILabelAccountable,
	})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	if err := db.Table(tables.Roles).Where("id = ?", role.ID).Delete(nil).Error; err != nil {
		t.Fatalf("deleting the Role row directly: %v", err)
	}

	issue, ok := findIssue(t, mgr, ctx, "", "role_assignment", assignment.ID)
	if !ok {
		t.Fatalf("expected a dangling-Role issue for assignment %s", assignment.ID)
	}
	if issue.Severity != models.IssueSeverityError {
		t.Errorf("expected error severity, got %q", issue.Severity)
	}
}

// TestScan_DanglingAttachment covers AG22's Code-keyed attachments: the
// attachment row outlives its target, since attaching is live-editable and
// sits outside the fork/promote lifecycle entirely.
func TestScan_DanglingAttachment(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, _, _, _ := seedLiveAgency(t, mgr, ctx)

	rule, err := mgr.CreateRule(ctx, models.Rule{Text: "never touch production data"})
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

	issue, ok := findIssue(t, mgr, ctx, "", "rule_attachment", attachment.ID)
	if !ok {
		t.Fatalf("expected a dangling-target issue for rule attachment %s", attachment.ID)
	}
	if issue.Severity != models.IssueSeverityError {
		t.Errorf("expected error severity, got %q", issue.Severity)
	}
}

// TestScan_EmptyWorkflowAndAgency covers the two structural checks that do
// not involve a broken reference at all.
func TestScan_EmptyWorkflowAndAgency(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "Empty"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	if _, ok := findIssue(t, mgr, ctx, "", "agency", ""); !ok {
		t.Fatal("expected an issue for an Agency with no Goals")
	}

	_, _, _, _, _ = seedLiveAgency(t, mgr, ctx)
	empty, err := mgr.CreateWorkflow(ctx, models.Workflow{Name: "Empty flow"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	issue, ok := findIssue(t, mgr, ctx, "", "workflow", empty.ID)
	if !ok {
		t.Fatalf("expected an issue for workflow %s with no work items", empty.ID)
	}
	if issue.Severity != models.IssueSeverityError {
		t.Errorf("expected error severity, got %q", issue.Severity)
	}
}

// TestScan_DanglingEventFlowStep covers the workflow_code business-key
// join, which nothing enforces at write time.
func TestScan_DanglingEventFlowStep(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedLiveAgency(t, mgr, ctx)

	step, err := mgr.CreateEventFlowStep(ctx, models.EventFlowStep{
		Code: "step-1", WorkflowCode: "W-404", Step: "1", Name: "Orphan step",
	})
	if err != nil {
		t.Fatalf("CreateEventFlowStep: %v", err)
	}

	issue, ok := findIssue(t, mgr, ctx, "", "event_flow_step", step.ID)
	if !ok {
		t.Fatalf("expected a dangling-Workflow issue for step %s", step.ID)
	}
	if issue.Severity != models.IssueSeverityError {
		t.Errorf("expected error severity, got %q", issue.Severity)
	}
}
