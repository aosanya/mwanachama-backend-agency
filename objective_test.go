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

func TestDeleteObjective_LeavesTheRoleAssignmentsOfSteps(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, step := seedLiveAgency(t, mgr, ctx)
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	assignment, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		DraftID: o.DraftID, WorkItemID: step.ID, RoleID: role.ID, Label: models.RACILabelAccountable,
	})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	if err := mgr.DeleteObjective(ctx, o.ID); err != nil {
		t.Fatalf("DeleteObjective: %v", err)
	}

	if _, err := mgr.GetRoleAssignment(ctx, assignment.ID); err != nil {
		t.Fatalf("expected the step's RoleAssignment to survive the Objective's deletion, got %v", err)
	}
	if _, err := mgr.GetRole(ctx, role.ID); err != nil {
		t.Fatalf("expected the Role to survive the Objective's deletion, got %v", err)
	}
}

func TestDeleteRole_CascadesRoleAssignments(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, step := seedLiveAgency(t, mgr, ctx)
	doomedRole, err := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	keptRole, err := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "reviewer"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	doomed, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		DraftID: o.DraftID, WorkItemID: step.ID, RoleID: doomedRole.ID, Label: models.RACILabelResponsible,
	})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}
	kept, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		DraftID: o.DraftID, WorkItemID: step.ID, RoleID: keptRole.ID, Label: models.RACILabelResponsible,
	})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	if err := mgr.DeleteRole(ctx, doomedRole.ID); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}

	if _, err := mgr.GetRole(ctx, doomedRole.ID); err == nil {
		t.Fatal("expected the Role itself to be gone")
	}
	if _, err := mgr.GetRoleAssignment(ctx, doomed.ID); !errors.Is(err, mwanachamaagency.ErrRoleAssignmentNotFound) {
		t.Fatalf("expected the cascaded RoleAssignment to be gone, got %v", err)
	}
	if _, err := mgr.GetRoleAssignment(ctx, kept.ID); err != nil {
		t.Fatalf("expected the other Role's assignment to survive, got %v", err)
	}
	if _, err := mgr.GetWorkItem(ctx, step.ID); err != nil {
		t.Fatalf("expected the step to survive the Role's deletion, got %v", err)
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
