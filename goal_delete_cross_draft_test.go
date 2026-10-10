package mwanachamaagency_test

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAG75_OpenBug_DeleteGoalCascadesIntoAnotherDraftsWorkItem(t *testing.T) {
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
	wiB, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: draftB.ID, WorkflowID: wfB.ID, ObjectiveID: objA.ID, Title: "Work Item B"})
	if err != nil {
		t.Fatalf("CreateWorkItem (draft B naming draft A's objective): %v", err)
	}

	if err := mgr.DeleteGoal(ctx, goalA.ID); err != nil {
		t.Fatalf("DeleteGoal(A): %v", err)
	}

	if _, err := mgr.GetWorkItem(ctx, wiB.ID); err != nil {
		t.Errorf("AG75 open bug: deleting Goal %q in draft A cascaded into Work Item %q, which was created in — and belongs to — draft B; GetWorkItem now answers %v, want the Work Item untouched since it was never part of draft A's content to begin with", goalA.ID, wiB.ID, err)
	}
}
