package mcp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestScenarioTools_ACreateCarriesActorsInputsAndThensWithoutIdsOfARowThatDoesNotExistYet(t *testing.T) {
	mgr := newTestManager(t)
	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cs := mcpConnect(t, srv.URL)

	ctx := ctxT(t)
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "Farmers"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draft, err := mgr.CreateDraft(ctx, "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	goal, _ := mgr.CreateGoal(ctx, models.Goal{DraftID: draft.ID, Title: "Sell"})
	objective, _ := mgr.CreateObjective(ctx, models.Objective{DraftID: draft.ID, GoalID: goal.ID, Title: "Sell"})
	flow, _ := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draft.ID, Name: "Buy now"})
	farmer, _ := mgr.CreateRole(ctx, models.Role{DraftID: draft.ID, Name: "Farmer"})
	ask, err := mgr.CreateWorkItem(ctx, models.WorkItem{
		DraftID: draft.ID, WorkflowID: flow.ID, ObjectiveID: objective.ID, Title: "Ask", Kind: models.StepKindFillForm,
		RoleID: farmer.ID, FormName: "Buy now form",
		Questions: []models.StepQuestion{{Prompt: "Enter the product", AnswerType: models.AnswerFreeText, Required: true}},
	})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}

	result, err := cs.CallTool(ctx, &gosdkmcp.CallToolParams{
		Name: "agency_create_scenario",
		Arguments: map[string]any{
			"instance_id": "any", "draft_id": draft.ID, "name": "Shop says yes", "workflow_id": flow.ID,
			"actors": []any{map[string]any{"name": "Farmer 1", "role_id": farmer.ID}},
			"inputs": []any{map[string]any{"step_id": ask.ID, "field": "Enter the product", "value": "urea"}},
			"thens":  []any{map[string]any{"step_id": ask.ID, "then": "The order shows urea"}},
		},
	})
	if err != nil || result.IsError {
		t.Fatalf("agency_create_scenario = %v %q, want the scenario written with its people, fixed answer and Then line", err, textOf(result))
	}

	structured, _ := result.StructuredContent.(map[string]any)
	blocks, _ := structured["blocks"].([]any)
	if len(blocks) != 1 {
		t.Fatalf("scenario = %+v, want one block", structured)
	}
	block, _ := blocks[0].(map[string]any)
	fields, _ := block["fields"].([]any)
	field, _ := fields[0].(map[string]any)
	if block["actor"] != "Farmer 1" || block["then"] != "The order shows urea" || field["value"] != "urea" || field["free"] == true {
		t.Errorf("block = %+v, want actor Farmer 1, the Then line and the fixed answer urea", block)
	}
}

func TestScenarioTools_AnUpdateCarriesActorsAndInputsWithoutTheScenarioAndDraftIdsOfEachRow(t *testing.T) {
	mgr := newTestManager(t)
	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cs := mcpConnect(t, srv.URL)

	ctx := ctxT(t)
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "Farmers"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draft, _ := mgr.CreateDraft(ctx, "seed", "", "")
	goal, _ := mgr.CreateGoal(ctx, models.Goal{DraftID: draft.ID, Title: "Sell"})
	objective, _ := mgr.CreateObjective(ctx, models.Objective{DraftID: draft.ID, GoalID: goal.ID, Title: "Sell"})
	flow, _ := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draft.ID, Name: "Buy now"})
	farmer, _ := mgr.CreateRole(ctx, models.Role{DraftID: draft.ID, Name: "Farmer"})
	ask, _ := mgr.CreateWorkItem(ctx, models.WorkItem{
		DraftID: draft.ID, WorkflowID: flow.ID, ObjectiveID: objective.ID, Title: "Ask", Kind: models.StepKindFillForm,
		RoleID: farmer.ID, FormName: "Buy now form",
		Questions: []models.StepQuestion{{Prompt: "Enter the product", AnswerType: models.AnswerFreeText, Required: true}},
	})
	sc, err := mgr.CreateScenario(ctx, models.Scenario{DraftID: draft.ID, Name: "Shop says yes", WorkflowID: flow.ID})
	if err != nil {
		t.Fatalf("CreateScenario: %v", err)
	}

	result, err := cs.CallTool(ctx, &gosdkmcp.CallToolParams{
		Name: "agency_update_scenario",
		Arguments: map[string]any{
			"instance_id": "any", "scenario_id": sc.ID, "draft_id": draft.ID, "workflow_id": flow.ID, "name": "Shop says yes",
			"actors": []any{map[string]any{"name": "Farmer 1", "role_id": farmer.ID}},
			"inputs": []any{map[string]any{"step_id": ask.ID, "field": "Enter the product", "value": "urea"}},
		},
	})
	if err != nil || result.IsError {
		t.Fatalf("agency_update_scenario = %v %q, want the set accepted", err, textOf(result))
	}
}
