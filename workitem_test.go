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

func TestDeleteWorkItem_LeavesTheAccountableRoleOfOtherSteps(t *testing.T) {
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
	doomed, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: draftID, WorkflowID: wf.ID, Title: "Write charter", AccountableRoleID: role.ID})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	kept, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: draftID, WorkflowID: wf.ID, Title: "Send charter", AccountableRoleID: role.ID})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}

	if err := mgr.DeleteWorkItem(ctx, doomed.ID); err != nil {
		t.Fatalf("DeleteWorkItem: %v", err)
	}

	if _, err := mgr.GetWorkItem(ctx, doomed.ID); !errors.Is(err, mwanachamaagency.ErrWorkItemNotFound) {
		t.Errorf("the deleted step: err = %v, want ErrWorkItemNotFound", err)
	}
	if got, err := mgr.GetWorkItem(ctx, kept.ID); err != nil || got.AccountableRoleID != role.ID {
		t.Errorf("the other step lost its accountable role: %+v, %v", got, err)
	}
	if _, err := mgr.GetRole(ctx, role.ID); err != nil {
		t.Errorf("the role was removed with the step: %v", err)
	}
}
