package mcp_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestMeasureTools_SetThenListByObjective(t *testing.T) {
	mgr := newTestManager(t)
	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cs := mcpConnect(t, srv.URL)

	if _, err := mgr.CreateAgency(ctxT(t), models.Agency{Name: "Farmers"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draft, err := mgr.CreateDraft(ctxT(t), "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	goal, _ := mgr.CreateGoal(ctxT(t), models.Goal{DraftID: draft.ID, Title: "Sell"})
	objective, err := mgr.CreateObjective(ctxT(t), models.Objective{DraftID: draft.ID, GoalID: goal.ID, Title: "Suppliers post prices"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}

	for _, target := range []string{"20", "40"} {
		result, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
			Name:      "agency_set_measure",
			Arguments: map[string]any{"instance_id": "any", "objective_id": objective.ID, "name": "Suppliers posting prices", "target": target},
		})
		if err != nil || result.IsError {
			t.Fatalf("agency_set_measure %s: %v %+v", target, err, textOf(result))
		}
	}

	result, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_list_measures",
		Arguments: map[string]any{"instance_id": "any", "draft_id": draft.ID, "objective_id": objective.ID},
	})
	if err != nil || result.IsError {
		t.Fatalf("agency_list_measures: %v %+v", err, result)
	}
	structured, _ := result.StructuredContent.(map[string]any)
	items, _ := structured["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items = %+v, want the one measure after two sets of one name", structured)
	}
	item, _ := items[0].(map[string]any)
	if item["objective_id"] != objective.ID || item["name"] != "Suppliers posting prices" || item["target"] != "40" || item["id"] == "" {
		t.Fatalf("item = %+v", item)
	}
}

func TestMeasureTools_SetRefusesAnUnknownObjective(t *testing.T) {
	mgr := newTestManager(t)
	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cs := mcpConnect(t, srv.URL)

	result, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_set_measure",
		Arguments: map[string]any{"instance_id": "any", "objective_id": "no-such-objective", "name": "Shops", "target": "1"},
	})
	if err != nil || !result.IsError || !strings.Contains(textOf(result), "objective not found") {
		t.Fatalf("agency_set_measure on an unknown objective = %v %q, want a refusal saying the objective was not found", err, textOf(result))
	}
}

func textOf(r *gosdkmcp.CallToolResult) string {
	out := ""
	for _, c := range r.Content {
		if t, ok := c.(*gosdkmcp.TextContent); ok {
			out += t.Text
		}
	}
	return out
}
