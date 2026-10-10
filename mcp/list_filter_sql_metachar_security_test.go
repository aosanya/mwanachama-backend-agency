package mcp_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestListGoals_SQLMetacharacterFiltersNarrowToNothingAndLeaveTheTableIntact(t *testing.T) {
	mgr := newTestManager(t)
	ctx := ctxT(t)
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "MCP05"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draft, err := mgr.CreateDraft(ctx, "mcp05", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	for _, title := range []string{"Ship reliable software", "Grow the membership"} {
		if _, err := mgr.CreateGoal(ctx, models.Goal{Title: title, DraftID: draft.ID}); err != nil {
			t.Fatalf("CreateGoal(%q): %v", title, err)
		}
	}

	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cs := mcpConnect(t, srv.URL)

	count := func(args map[string]any) int {
		t.Helper()
		args["instance_id"], args["draft_id"] = "any", draft.ID
		res, err := cs.CallTool(ctx, &gosdkmcp.CallToolParams{Name: "agency_list_goals", Arguments: args})
		if err != nil || res.IsError {
			t.Fatalf("agency_list_goals(%v): err=%v isError=%v content=%v", args, err, res != nil && res.IsError, res)
		}
		raw, _ := json.Marshal(res.StructuredContent)
		var out struct {
			Items []json.RawMessage `json:"items"`
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("agency_list_goals structured content %s: %v", raw, err)
		}
		return len(out.Items)
	}

	if n := count(map[string]any{}); n != 2 {
		t.Fatalf("precondition: an unfiltered listing should hold both goals, got %d", n)
	}
	payloads := []string{
		"' or '1'='1",
		"x') or 1=1 --",
		"x'; delete from goals; --",
		"\" or \"\"=\"",
		"\\' or 1=1 /*",
	}
	for _, field := range []string{"code", "title", "description"} {
		for _, p := range payloads {
			if n := count(map[string]any{field: p}); n != 0 {
				t.Errorf("agency_list_goals %s=%q listed %d goals; a filter value is a literal substring and must narrow to none, not widen to every row", field, p, n)
			}
		}
	}
	if n := count(map[string]any{}); n != 2 {
		t.Errorf("after the metacharacter filters an unfiltered listing held %d goals, want both still stored", n)
	}
}
