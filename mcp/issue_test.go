package mcp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestListIssues_ViaMCP(t *testing.T) {
	mgr := newTestManager(t)
	ctx := ctxT(t)
	ag, err := mgr.CreateAgency(ctx, models.Agency{Name: "NeoLabHQ"})
	if err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	g, err := mgr.CreateGoal(ctx, models.Goal{Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if _, err := mgr.CreateObjective(ctx, models.Objective{GoalID: g.ID, Title: "Grow adoption"}); err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	d, err := mgr.CreateDraft(ctx, "review pass", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cs := mcpConnect(t, srv.URL)
	result, err := cs.CallTool(ctx, &gosdkmcp.CallToolParams{
		Name:      "agency_list_issues",
		Arguments: map[string]any{"instance_id": "test", "draft_id": d.ID},
	})
	if err != nil {
		t.Fatalf("CallTool agency_list_issues: %v", err)
	}
	if result.IsError {
		t.Fatalf("agency_list_issues: IsError = true (content: %+v)", result.Content)
	}
	sc, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structuredContent is %T, want map[string]any", result.StructuredContent)
	}
	items, ok := sc["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected exactly 1 issue, got %+v", sc["items"])
	}
}

func TestListIssues_ViaMCP_EmptyDraftIDErrors(t *testing.T) {
	mgr := newTestManager(t)
	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cs := mcpConnect(t, srv.URL)
	result, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_list_issues",
		Arguments: map[string]any{"instance_id": "test", "draft_id": ""},
	})
	if err != nil {
		t.Fatalf("CallTool agency_list_issues: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected IsError for an empty draft_id, got content: %+v", result.Content)
	}
}
