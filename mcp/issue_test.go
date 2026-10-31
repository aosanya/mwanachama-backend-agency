package mcp_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestListIssues_ViaMCP(t *testing.T) {
	mgr := newTestManager(t)
	ctx := ctxT(t)
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "SWELab"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	d, err := mgr.CreateDraft(ctx, "review pass", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	g, err := mgr.CreateGoal(ctx, models.Goal{DraftID: d.ID, Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if _, err := mgr.CreateObjective(ctx, models.Objective{DraftID: d.ID, GoalID: g.ID, Title: "Grow adoption"}); err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	flow, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: d.ID, Name: "Onboard"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	if _, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: d.ID, WorkflowID: flow.ID, Title: "Welcome"}); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
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
	issue, _ := items[0].(map[string]any)
	if issue["entity_type"] != "workflow" || !strings.Contains(fmt.Sprint(issue["message"]), "has no Objective") {
		t.Fatalf("the one issue should be the workflow that serves no objective, got %+v", issue)
	}
}

func TestListIssues_ViaMCP_OmittedDraftIDIsError(t *testing.T) {
	mgr := newTestManager(t)
	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cs := mcpConnect(t, srv.URL)
	result, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_list_issues",
		Arguments: map[string]any{"instance_id": "test"},
	})
	if err != nil {
		t.Fatalf("CallTool agency_list_issues: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected an omitted draft_id to fail, got content: %+v", result.Content)
	}
}
