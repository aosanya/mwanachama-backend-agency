package mwanachamaagency_test

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAG75_DeleteGoalLeavesAnotherDraftsWorkItem(t *testing.T) {
	mgr := newTestManager(t)
	ctx := t.Context()

	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "Probe Agency", Enabled: true}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}

	draftA, err := mgr.CreateDraft(ctx, "draft A", "", "")
	if err != nil {
		t.Fatalf("CreateDraft A: %v", err)
	}
	draftB, err := mgr.CreateDraft(ctx, "draft B", "", "")
	if err != nil {
		t.Fatalf("CreateDraft B: %v", err)
	}

	goalA, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draftA.ID, Title: "Goal A"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	objA, err := mgr.CreateObjective(ctx, models.Objective{DraftID: draftA.ID, GoalID: goalA.ID, Title: "Objective A"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	wfB, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftB.ID, Name: "Workflow B"})
	if err != nil {
		t.Fatalf("CreateWorkflow B: %v", err)
	}
	objB, err := mgr.CreateObjective(ctx, models.Objective{DraftID: draftB.ID, GoalID: goalA.ID, Title: "Objective B"})
	if err != nil {
		t.Fatalf("CreateObjective B: %v", err)
	}
	linkB, err := mgr.CreateWorkflowObjective(ctx, models.WorkflowObjective{DraftID: draftB.ID, WorkflowID: wfB.ID, ObjectiveID: objB.ID})
	if err != nil {
		t.Fatalf("CreateWorkflowObjective B: %v", err)
	}
	wiB, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: draftB.ID, WorkflowID: wfB.ID, Title: "Work Item B"})
	if err != nil {
		t.Fatalf("CreateWorkItem B: %v", err)
	}

	if err := mgr.DeleteGoal(ctx, goalA.ID); err != nil {
		t.Fatalf("DeleteGoal(A): %v", err)
	}

	if _, err := mgr.GetWorkItem(ctx, wiB.ID); err != nil {
		t.Errorf("AG75 open bug: deleting Goal %q in draft A cascaded into Work Item %q, which was created in — and belongs to — draft B; GetWorkItem now answers %v, want the Work Item untouched since it was never part of draft A's content to begin with", goalA.ID, wiB.ID, err)
	}
	if _, err := mgr.GetObjective(ctx, objA.ID); err == nil {
		t.Errorf("deleting Goal %q in draft A left its own Objective %q behind", goalA.ID, objA.ID)
	}
	if _, err := mgr.GetObjective(ctx, objB.ID); err != nil {
		t.Errorf("deleting Goal %q in draft A removed draft B's Objective %q: %v", goalA.ID, objB.ID, err)
	}
	if _, err := mgr.GetWorkflowObjective(ctx, linkB.ID); err != nil {
		t.Errorf("deleting Goal %q in draft A removed draft B's WorkflowObjective %q: %v", goalA.ID, linkB.ID, err)
	}
}

func TestAG75_DeleteObjectiveAndWorkflowLeaveAnotherDraftsWorkItem(t *testing.T) {
	mgr := newTestManager(t)
	ctx := t.Context()

	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "Probe Agency", Enabled: true}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draftA, err := mgr.CreateDraft(ctx, "draft A", "", "")
	if err != nil {
		t.Fatalf("CreateDraft A: %v", err)
	}
	draftB, err := mgr.CreateDraft(ctx, "draft B", "", "")
	if err != nil {
		t.Fatalf("CreateDraft B: %v", err)
	}
	goalA, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draftA.ID, Title: "Goal A"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	objA, err := mgr.CreateObjective(ctx, models.Objective{DraftID: draftA.ID, GoalID: goalA.ID, Title: "Objective A"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	wfA, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftA.ID, Name: "Workflow A"})
	if err != nil {
		t.Fatalf("CreateWorkflow A: %v", err)
	}
	wfB, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftB.ID, Name: "Workflow B"})
	if err != nil {
		t.Fatalf("CreateWorkflow B: %v", err)
	}
	objB, err := mgr.CreateObjective(ctx, models.Objective{DraftID: draftB.ID, GoalID: goalA.ID, Title: "Objective B"})
	if err != nil {
		t.Fatalf("CreateObjective B: %v", err)
	}
	linkB, err := mgr.CreateWorkflowObjective(ctx, models.WorkflowObjective{DraftID: draftB.ID, WorkflowID: wfB.ID, ObjectiveID: objB.ID})
	if err != nil {
		t.Fatalf("CreateWorkflowObjective B: %v", err)
	}
	inB, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: draftB.ID, WorkflowID: wfB.ID, Title: "in B's workflow"})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	viaWorkflow, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: draftB.ID, WorkflowID: wfA.ID, Title: "names A's workflow"})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}

	if err := mgr.DeleteObjective(ctx, objA.ID); err != nil {
		t.Fatalf("DeleteObjective(A): %v", err)
	}
	if err := mgr.DeleteWorkflow(ctx, wfA.ID); err != nil {
		t.Fatalf("DeleteWorkflow(A): %v", err)
	}

	if _, err := mgr.GetWorkItem(ctx, inB.ID); err != nil {
		t.Errorf("deleting Objective %q in draft A removed draft B's Work Item %q: %v", objA.ID, inB.ID, err)
	}
	if _, err := mgr.GetWorkflowObjective(ctx, linkB.ID); err != nil {
		t.Errorf("deleting Objective %q and Workflow %q in draft A removed draft B's WorkflowObjective %q: %v", objA.ID, wfA.ID, linkB.ID, err)
	}
	if _, err := mgr.GetWorkItem(ctx, viaWorkflow.ID); err != nil {
		t.Errorf("deleting Workflow %q in draft A removed draft B's Work Item %q: %v", wfA.ID, viaWorkflow.ID, err)
	}
}
