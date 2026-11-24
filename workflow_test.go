package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestDeleteWorkflow_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if err := mgr.DeleteWorkflow(context.Background(), "unknown"); !errors.Is(err, mwanachamaagency.ErrWorkflowNotFound) {
		t.Fatalf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestDeleteWorkflow_CascadesWorkItems(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, _, w, wi := seedLiveAgency(t, mgr, ctx)

	if err := mgr.DeleteWorkflow(ctx, w.ID); err != nil {
		t.Fatalf("DeleteWorkflow: %v", err)
	}

	if _, err := mgr.GetWorkflow(ctx, w.ID); !errors.Is(err, mwanachamaagency.ErrWorkflowNotFound) {
		t.Fatalf("expected the Workflow itself to be gone, got %v", err)
	}
	if _, err := mgr.GetWorkItem(ctx, wi.ID); !errors.Is(err, mwanachamaagency.ErrWorkItemNotFound) {
		t.Fatalf("expected the cascaded WorkItem to be gone, got %v", err)
	}
}

func TestDeleteWorkflow_NoWorkItems(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)
	w, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftID, Name: "Lonely workflow"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	if err := mgr.DeleteWorkflow(ctx, w.ID); err != nil {
		t.Fatalf("DeleteWorkflow: %v", err)
	}
	if _, err := mgr.GetWorkflow(ctx, w.ID); !errors.Is(err, mwanachamaagency.ErrWorkflowNotFound) {
		t.Fatalf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestDeleteWorkflow_RemovesItsObjectiveLinksAndLeavesTheAccountableRoleOfOtherWorkflows(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)
	g, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	o, err := mgr.CreateObjective(ctx, models.Objective{DraftID: draftID, GoalID: g.ID, Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "Coordinator"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	doomed, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftID, Name: "Onboarding"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	kept, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftID, Name: "Offboarding"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	doomedLink := linkWorkflowToObjective(t, mgr, draftID, doomed.ID, o.ID)
	keptLink := linkWorkflowToObjective(t, mgr, draftID, kept.ID, o.ID)

	for _, title := range []string{"Write charter", "Send charter"} {
		if _, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: draftID, WorkflowID: doomed.ID, Title: title, AccountableRoleID: role.ID}); err != nil {
			t.Fatalf("CreateWorkItem %s: %v", title, err)
		}
	}
	keptStep, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: draftID, WorkflowID: kept.ID, Title: "Close account", AccountableRoleID: role.ID})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}

	if err := mgr.DeleteWorkflow(ctx, doomed.ID); err != nil {
		t.Fatalf("DeleteWorkflow: %v", err)
	}

	if _, err := mgr.GetWorkflowObjective(ctx, doomedLink.ID); !errors.Is(err, mwanachamaagency.ErrWorkflowObjectiveNotFound) {
		t.Errorf("link of the deleted workflow: err = %v, want ErrWorkflowObjectiveNotFound", err)
	}
	if _, err := mgr.GetWorkflowObjective(ctx, keptLink.ID); err != nil {
		t.Errorf("the link of another workflow was removed: %v", err)
	}
	if got, err := mgr.GetWorkItem(ctx, keptStep.ID); err != nil || got.AccountableRoleID != role.ID {
		t.Errorf("the step of another workflow was changed: %+v, %v", got, err)
	}
	if _, err := mgr.GetObjective(ctx, o.ID); err != nil {
		t.Errorf("the objective was removed with the workflow: %v", err)
	}
	if _, err := mgr.GetRole(ctx, role.ID); err != nil {
		t.Errorf("the role was removed with the workflow: %v", err)
	}
}
