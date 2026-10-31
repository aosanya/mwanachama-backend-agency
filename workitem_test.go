package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestDeleteWorkItem_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if err := mgr.DeleteWorkItem(context.Background(), "unknown"); !errors.Is(err, mwanachamaagency.ErrWorkItemNotFound) {
		t.Fatalf("expected ErrWorkItemNotFound, got %v", err)
	}
}

func TestDeleteWorkItem(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, _, _, wi := seedLiveAgency(t, mgr, ctx)

	if err := mgr.DeleteWorkItem(ctx, wi.ID); err != nil {
		t.Fatalf("DeleteWorkItem: %v", err)
	}
	if _, err := mgr.GetWorkItem(ctx, wi.ID); !errors.Is(err, mwanachamaagency.ErrWorkItemNotFound) {
		t.Fatalf("expected ErrWorkItemNotFound, got %v", err)
	}
}

func TestDeleteWorkItem_RemovesItsRoleAssignmentsAndNoOthers(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)
	wf, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftID, Name: "Onboarding"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "Coordinator"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	doomed, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: draftID, WorkflowID: wf.ID, Title: "Write charter"})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	kept, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: draftID, WorkflowID: wf.ID, Title: "Send charter"})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	var doomedAssignments []models.RoleAssignment
	for _, label := range []models.RACILabel{models.RACILabelAccountable, models.RACILabelResponsible} {
		a, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: draftID, WorkItemID: doomed.ID, RoleID: role.ID, Label: label})
		if err != nil {
			t.Fatalf("CreateRoleAssignment %s: %v", label, err)
		}
		doomedAssignments = append(doomedAssignments, a)
	}
	survivor, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: draftID, WorkItemID: kept.ID, RoleID: role.ID, Label: models.RACILabelAccountable})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	if err := mgr.DeleteWorkItem(ctx, doomed.ID); err != nil {
		t.Fatalf("DeleteWorkItem: %v", err)
	}

	for _, a := range doomedAssignments {
		if _, err := mgr.GetRoleAssignment(ctx, a.ID); !errors.Is(err, mwanachamaagency.ErrRoleAssignmentNotFound) {
			t.Errorf("assignment %s (%s) of the deleted step: err = %v, want ErrRoleAssignmentNotFound", a.ID, a.Label, err)
		}
	}
	if _, err := mgr.GetRoleAssignment(ctx, survivor.ID); err != nil {
		t.Errorf("the assignment of another step was removed: %v", err)
	}
	left, err := mgr.ListRoleAssignmentsForWorkItem(ctx, doomed.ID, draftID, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil || len(left) != 0 {
		t.Errorf("assignments left on the deleted step = %+v, %v", left, err)
	}
}
