package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestDeleteObjective_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if err := mgr.DeleteObjective(context.Background(), "unknown"); !errors.Is(err, mwanachamaagency.ErrObjectiveNotFound) {
		t.Fatalf("expected ErrObjectiveNotFound, got %v", err)
	}
}

func TestDeleteObjective_CascadesWorkflowObjectives(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, wf, step := seedLiveAgency(t, mgr, ctx)

	otherObjective, err := mgr.CreateObjective(ctx, models.Objective{DraftID: o.DraftID, GoalID: o.GoalID, Title: "Another objective"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	kept, err := mgr.CreateWorkflowObjective(ctx, models.WorkflowObjective{DraftID: o.DraftID, WorkflowID: wf.ID, ObjectiveID: otherObjective.ID})
	if err != nil {
		t.Fatalf("CreateWorkflowObjective: %v", err)
	}
	doomed, err := mgr.ListWorkflowObjectives(ctx, o.DraftID, mwanachamaagency.WorkflowObjectiveFilter{ObjectiveID: o.ID})
	if err != nil || len(doomed) != 1 {
		t.Fatalf("ListWorkflowObjectives before delete = %+v, %v", doomed, err)
	}

	if err := mgr.DeleteObjective(ctx, o.ID); err != nil {
		t.Fatalf("DeleteObjective: %v", err)
	}

	if _, err := mgr.GetObjective(ctx, o.ID); !errors.Is(err, mwanachamaagency.ErrObjectiveNotFound) {
		t.Fatalf("expected the Objective itself to be gone, got %v", err)
	}
	if _, err := mgr.GetWorkflowObjective(ctx, doomed[0].ID); !errors.Is(err, mwanachamaagency.ErrWorkflowObjectiveNotFound) {
		t.Fatalf("expected the cascaded WorkflowObjective to be gone, got %v", err)
	}
	if _, err := mgr.GetWorkflowObjective(ctx, kept.ID); err != nil {
		t.Fatalf("expected the link to the other objective to survive, got %v", err)
	}
	if _, err := mgr.GetWorkflow(ctx, wf.ID); err != nil {
		t.Fatalf("expected the Workflow to survive the Objective's deletion, got %v", err)
	}
	if _, err := mgr.GetWorkItem(ctx, step.ID); err != nil {
		t.Fatalf("expected the step to survive the Objective's deletion, got %v", err)
	}
}

func TestDeleteObjective_LeavesTheAccountableRoleOfSteps(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, step := seedLiveAgency(t, mgr, ctx)
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	step.AccountableRoleID = role.ID
	if _, err := mgr.UpdateWorkItem(ctx, step); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}

	if err := mgr.DeleteObjective(ctx, o.ID); err != nil {
		t.Fatalf("DeleteObjective: %v", err)
	}

	if got, err := mgr.GetWorkItem(ctx, step.ID); err != nil || got.AccountableRoleID != role.ID {
		t.Fatalf("expected the step's accountable role to survive the Objective's deletion, got %+v, %v", got, err)
	}
	if _, err := mgr.GetRole(ctx, role.ID); err != nil {
		t.Fatalf("expected the Role to survive the Objective's deletion, got %v", err)
	}
}

func TestDeleteRole_EmptiesTheAccountableRoleOfItsSteps(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, w, step := seedLiveAgency(t, mgr, ctx)
	doomedRole, err := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	keptRole, err := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "reviewer"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	other, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: o.DraftID, WorkflowID: w.ID, Title: "Review", AccountableRoleID: keptRole.ID})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	step.AccountableRoleID = doomedRole.ID
	if _, err := mgr.UpdateWorkItem(ctx, step); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}

	if err := mgr.DeleteRole(ctx, doomedRole.ID); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}

	if _, err := mgr.GetRole(ctx, doomedRole.ID); err == nil {
		t.Fatal("expected the Role itself to be gone")
	}
	if got, err := mgr.GetWorkItem(ctx, step.ID); err != nil || got.AccountableRoleID != "" {
		t.Fatalf("expected the step to survive with no accountable role, got %+v, %v", got, err)
	}
	if got, err := mgr.GetWorkItem(ctx, other.ID); err != nil || got.AccountableRoleID != keptRole.ID {
		t.Fatalf("expected the other Role's step to keep its accountable role, got %+v, %v", got, err)
	}
}

func TestDeleteObjective_NoWorkflowObjectives(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)
	g, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, Title: "Lonely goal"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	o, err := mgr.CreateObjective(ctx, models.Objective{DraftID: draftID, GoalID: g.ID, Title: "Lonely objective"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	if err := mgr.DeleteObjective(ctx, o.ID); err != nil {
		t.Fatalf("DeleteObjective: %v", err)
	}
	if _, err := mgr.GetObjective(ctx, o.ID); !errors.Is(err, mwanachamaagency.ErrObjectiveNotFound) {
		t.Fatalf("expected ErrObjectiveNotFound, got %v", err)
	}
}
