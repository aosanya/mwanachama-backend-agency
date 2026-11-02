package mwanachamaagency_test

import (
	"context"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func cancelsOrderByName(workflows []models.Workflow) map[string]bool {
	out := map[string]bool{}
	for _, wf := range workflows {
		out[wf.Name] = wf.CancelsOrder
	}
	return out
}

func TestWorkflow_CancelsOrderSurvivesReadUpdatePublicationAndFork(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	const cancelName, otherName = "Cancel after the Supplier says yes", "Group order"
	cancel, err := w.mgr.CreateWorkflow(ctx, models.Workflow{DraftID: w.draftID, Name: cancelName, CancelsOrder: true})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	linkWorkflowToObjective(t, w.mgr, w.draftID, cancel.ID, w.objective.ID)
	w.build(t, cancel, groupOrderSteps[:1])
	w.build(t, w.workflow(t, otherName), groupOrderSteps[:1])

	got, err := w.mgr.GetWorkflow(ctx, cancel.ID)
	if err != nil {
		t.Fatalf("GetWorkflow: %v", err)
	}
	if !got.CancelsOrder {
		t.Errorf("GetWorkflow cancels_order = false, want true")
	}
	listed, err := w.mgr.ListWorkflows(ctx, w.draftID, mwanachamaagency.WorkflowFilter{})
	if err != nil {
		t.Fatalf("ListWorkflows: %v", err)
	}
	if marks := cancelsOrderByName(listed); !marks[cancelName] || marks[otherName] {
		t.Errorf("ListWorkflows cancels_order = %v, want only %q", marks, cancelName)
	}

	pub, err := w.mgr.PublishAgency(ctx, w.draftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	inDocument := map[string]bool{}
	for _, wf := range pub.Document.Workflows {
		inDocument[wf.Name] = wf.CancelsOrder
	}
	if !inDocument[cancelName] || inDocument[otherName] {
		t.Errorf("the publication document cancels_order = %v, want only %q", inDocument, cancelName)
	}

	fork, err := w.mgr.CreateDraft(ctx, "from the draft", w.draftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	forked, err := w.mgr.ListWorkflows(ctx, fork.ID, mwanachamaagency.WorkflowFilter{})
	if err != nil {
		t.Fatalf("ListWorkflows on the fork: %v", err)
	}
	if marks := cancelsOrderByName(forked); !marks[cancelName] || marks[otherName] {
		t.Errorf("the fork's cancels_order = %v, want only %q", marks, cancelName)
	}

	if _, err := w.mgr.UpdateWorkflow(ctx, models.Workflow{ID: cancel.ID, Name: cancelName, CancelsOrder: false}); err != nil {
		t.Fatalf("UpdateWorkflow: %v", err)
	}
	after, err := w.mgr.GetWorkflow(ctx, cancel.ID)
	if err != nil {
		t.Fatalf("GetWorkflow after the update: %v", err)
	}
	if after.CancelsOrder {
		t.Errorf("cancels_order after unticking = true, want false")
	}
}
