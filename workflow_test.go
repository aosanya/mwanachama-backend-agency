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

// TestDeleteWorkflow_CascadesWorkItems asserts DeleteWorkflow takes its
// WorkItems down with it.
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

// TestDeleteWorkflow_NoWorkItems asserts deleting a Workflow with nothing
// under it is just a plain delete — the cascade's empty case must not
// choke.
func TestDeleteWorkflow_NoWorkItems(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	w, err := mgr.CreateWorkflow(ctx, models.Workflow{Name: "Lonely workflow"})
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
