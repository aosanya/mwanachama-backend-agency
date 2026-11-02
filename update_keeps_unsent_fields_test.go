package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func consoleBody(t *testing.T, into any, body map[string]any) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal the console's body: %v", err)
	}
	if err := json.Unmarshal(raw, into); err != nil {
		t.Fatalf("decode the console's body: %v", err)
	}
}

func TestUpdateWorkflow_TheConsolesBodyKeepsItsOrdinalityAndEventFlows(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	made, err := w.mgr.CreateWorkflow(ctx, models.Workflow{DraftID: w.draftID, Name: "Group order", Description: "Farmers buy together", Ordinality: 3, EventFlows: "ask,answer"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	var update models.Workflow
	consoleBody(t, &update, map[string]any{
		"draft_id":    w.draftID,
		"workflow_id": made.ID,
		"name":        "Group order renamed",
		"description": "Farmers buy together",
	})
	update.ID = made.ID
	if _, err := w.mgr.UpdateWorkflow(ctx, update); err != nil {
		t.Fatalf("UpdateWorkflow: %v", err)
	}

	got, err := w.mgr.GetWorkflow(ctx, made.ID)
	if err != nil {
		t.Fatalf("GetWorkflow: %v", err)
	}
	if got.Name != "Group order renamed" {
		t.Fatalf("name after the update = %q", got.Name)
	}
	if got.Ordinality != 3 {
		t.Errorf("ordinality after the console's update = %d, want 3", got.Ordinality)
	}
	if got.EventFlows != "ask,answer" {
		t.Errorf("event_flows after the console's update = %q, want ask,answer", got.EventFlows)
	}
}

func TestUpdateWorkItem_TheConsolesBodyKeepsItsTool(t *testing.T) {
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
		"draft_id":           w.draftID,
		"workflow_id":        wf.ID,
		"work_item_id":       made.ID,
		"title":              "Ask the shop",
		"description":        "",
		"ordinality":         1,
		"prompt":             "",
		"kind":               string(models.StepKindFillForm),
		"role_id":            w.roles[farmer].ID,
		"run_workflow_id":    "",
		"form_name":          made.FormName,
		"questions":          made.Questions,
		"recipient_role_id":  "",
		"sent_step_id":       "",
		"routes":             []any{},
		"otherwise_step_id":  "",
		"time_limit_minutes": 0,
		"cancel_workflow_id": "",
	})
	update.ID = made.ID
	if _, err := w.mgr.UpdateWorkItem(ctx, update); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}

	got, err := w.mgr.GetWorkItem(ctx, made.ID)
	if err != nil {
		t.Fatalf("GetWorkItem: %v", err)
	}
	if got.Title != "Ask the shop" {
		t.Fatalf("title after the update = %q", got.Title)
	}
	if got.Tool != "market_ask" {
		t.Errorf("tool after the console's update = %q, want market_ask", got.Tool)
	}
}
