package mwanachamaagency_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestUpdateWorkflow_ASentEmptyEventFlowsAndZeroOrdinalityClearThem(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	made, err := w.mgr.CreateWorkflow(ctx, models.Workflow{DraftID: w.draftID, Name: "Group order", Ordinality: 3, EventFlows: "ask,answer"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	var update models.Workflow
	consoleBody(t, &update, map[string]any{
		"draft_id":    w.draftID,
		"workflow_id": made.ID,
		"name":        "Group order",
		"ordinality":  0,
		"event_flows": "",
	})
	update.ID = made.ID
	if _, err := w.mgr.UpdateWorkflow(ctx, update); err != nil {
		t.Fatalf("UpdateWorkflow: %v", err)
	}

	got, err := w.mgr.GetWorkflow(ctx, made.ID)
	if err != nil {
		t.Fatalf("GetWorkflow: %v", err)
	}
	if got.Ordinality != 0 || got.EventFlows != "" {
		t.Errorf("ordinality = %d, event_flows = %q, want both cleared", got.Ordinality, got.EventFlows)
	}
}

func TestUpdateWorkflow_ABodyBuiltInCodeStillReplacesTheWholeWorkflow(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	made, err := w.mgr.CreateWorkflow(ctx, models.Workflow{DraftID: w.draftID, Name: "Group order", Ordinality: 3, EventFlows: "ask,answer"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	if _, err := w.mgr.UpdateWorkflow(ctx, models.Workflow{ID: made.ID, DraftID: w.draftID, Name: "Group order"}); err != nil {
		t.Fatalf("UpdateWorkflow: %v", err)
	}

	got, err := w.mgr.GetWorkflow(ctx, made.ID)
	if err != nil {
		t.Fatalf("GetWorkflow: %v", err)
	}
	if got.Ordinality != 0 || got.EventFlows != "" {
		t.Errorf("ordinality = %d, event_flows = %q, want both replaced", got.Ordinality, got.EventFlows)
	}
}

func TestUpdateWorkflow_TheConsolesBodyKeepsItsOpeningAndCancelMarks(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	made, err := w.mgr.CreateWorkflow(ctx, models.Workflow{DraftID: w.draftID, Name: "Talk", OpensIn: models.OpensInChat, CancelsOrder: true})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	var update models.Workflow
	consoleBody(t, &update, map[string]any{"draft_id": w.draftID, "workflow_id": made.ID, "name": "Talk renamed"})
	update.ID = made.ID
	if _, err := w.mgr.UpdateWorkflow(ctx, update); err != nil {
		t.Fatalf("UpdateWorkflow: %v", err)
	}

	got, err := w.mgr.GetWorkflow(ctx, made.ID)
	if err != nil {
		t.Fatalf("GetWorkflow: %v", err)
	}
	if got.OpensIn != models.OpensInChat || !got.CancelsOrder {
		t.Errorf("opens_in = %q, cancels_order = %v, want chat and true", got.OpensIn, got.CancelsOrder)
	}
}

func TestUpdateWorkItem_ASentEmptyToolClearsIt(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Buy now")
	in := w.item(wf, 1, buyNowSteps[0])
	in.Tool = "market_ask"
	made, err := w.mgr.CreateWorkItem(ctx, in)
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}

	var update models.WorkItem
	consoleBody(t, &update, map[string]any{
		"draft_id":     w.draftID,
		"workflow_id":  wf.ID,
		"work_item_id": made.ID,
		"title":        made.Title,
		"ordinality":   1,
		"kind":         string(models.StepKindFillForm),
		"role_id":      w.roles[farmer].ID,
		"form_name":    made.FormName,
		"questions":    made.Questions,
		"tool":         "",
	})
	update.ID = made.ID
	if _, err := w.mgr.UpdateWorkItem(ctx, update); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}

	got, err := w.mgr.GetWorkItem(ctx, made.ID)
	if err != nil {
		t.Fatalf("GetWorkItem: %v", err)
	}
	if got.Tool != "" {
		t.Errorf("tool = %q, want it cleared", got.Tool)
	}
}
