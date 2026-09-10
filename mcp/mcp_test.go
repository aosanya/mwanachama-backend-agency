package mcp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	agencymcp "github.com/aosanya/mwanachama-backend-agency/mcp"
)

func ctxT(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// newTestServer registers every tool onto a fresh mcp.Server, resolving
// every id to the same single manager — this package has no notion of
// multiple Agencies itself (that's a mounting host's own concern, e.g.
// mwanachama-wakala-api's Registry), so a single-manager resolver is
// enough to prove the tools work.
func newTestServer(mgr mwanachamaagency.AgencyManager) *gosdkmcp.Server {
	server := gosdkmcp.NewServer(&gosdkmcp.Implementation{Name: "test", Version: "0.0.0"}, nil)
	agencymcp.RegisterTools(server, func(ctx context.Context, id string) (mwanachamaagency.AgencyManager, error) {
		return mgr, nil
	})
	return server
}

func mcpConnect(t *testing.T, base string) *gosdkmcp.ClientSession {
	t.Helper()
	client := gosdkmcp.NewClient(&gosdkmcp.Implementation{Name: "test-client", Version: "0.0.0"}, nil)
	transport := &gosdkmcp.StreamableClientTransport{
		Endpoint:             base + "/mcp",
		DisableStandaloneSSE: true,
	}
	cs, err := client.Connect(ctxT(t), transport, nil)
	if err != nil {
		t.Fatalf("mcp connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func TestRegisterTools_ListsExpectedTools(t *testing.T) {
	server := newTestServer(newTestManager(t))
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cs := mcpConnect(t, srv.URL)
	tools, err := cs.ListTools(ctxT(t), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	names := map[string]bool{}
	for _, tl := range tools.Tools {
		names[tl.Name] = true
	}
	want := []string{
		"agency_get_agency", "agency_create_goal", "agency_create_objective", "agency_create_department",
		"agency_set_raci", "agency_create_work_plan", "agency_add_context_source",
		"agency_create_value", "agency_create_rule", "agency_design_from_spec",
	}
	for _, w := range want {
		if !names[w] {
			t.Errorf("tools/list missing %q", w)
		}
	}
}

func TestAgencyAndGoalLifecycle(t *testing.T) {
	server := newTestServer(newTestManager(t))
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cs := mcpConnect(t, srv.URL)

	result, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_create_agency",
		Arguments: map[string]any{"instance_id": "any", "name": "NeoLabHQ"},
	})
	if err != nil {
		t.Fatalf("CallTool agency_create_agency: %v", err)
	}
	if result.IsError {
		t.Fatalf("agency_create_agency: IsError = true (content: %+v)", result.Content)
	}

	result, err = cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_create_goal",
		Arguments: map[string]any{"instance_id": "any", "title": "Grow membership"},
	})
	if err != nil {
		t.Fatalf("CallTool agency_create_goal: %v", err)
	}
	if result.IsError {
		t.Fatalf("agency_create_goal: IsError = true (content: %+v)", result.Content)
	}

	result, err = cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_list_goals",
		Arguments: map[string]any{"instance_id": "any"},
	})
	if err != nil {
		t.Fatalf("CallTool agency_list_goals: %v", err)
	}
	got, ok := result.StructuredContent.(map[string]any)
	items, _ := got["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("agency_list_goals structured content = %+v", result.StructuredContent)
	}
}

func TestDesignFromSpec(t *testing.T) {
	server := newTestServer(newTestManager(t))
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cs := mcpConnect(t, srv.URL)

	result, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name: "agency_design_from_spec",
		Arguments: map[string]any{
			"instance_id": "any",
			"agency":      map[string]any{"name": "NeoLabHQ"},
			"goals":       []map[string]any{{"key": "g1", "title": "Grow membership"}},
			"objectives":  []map[string]any{{"key": "o1", "goal_key": "g1", "title": "Grow membership by 10%"}},
		},
	})
	if err != nil {
		t.Fatalf("CallTool agency_design_from_spec: %v", err)
	}
	if result.IsError {
		t.Fatalf("agency_design_from_spec: IsError = true (content: %+v)", result.Content)
	}
	got, _ := result.StructuredContent.(map[string]any)
	if got["agency_created"] != true {
		t.Fatalf("agency_design_from_spec structured content = %+v", result.StructuredContent)
	}
}
