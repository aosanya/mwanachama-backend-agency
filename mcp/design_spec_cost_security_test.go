package mcp_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAG58_OpenHole_OneDesignFromSpecCallAppliesUnboundedEntities(t *testing.T) {
	mgr := newTestManager(t)
	ctx := ctxT(t)
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "AG58"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draft, err := mgr.CreateDraft(ctx, "ag58", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cs := mcpConnect(t, srv.URL)

	const goals = 5000
	spec := make([]any, goals)
	for i := range spec {
		spec[i] = map[string]any{"key": fmt.Sprintf("g%d", i), "title": "t"}
	}
	start := time.Now()
	result, err := cs.CallTool(ctx, &gosdkmcp.CallToolParams{
		Name:      "agency_design_from_spec",
		Arguments: map[string]any{"instance_id": "test", "draft_id": draft.ID, "goals": spec},
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("CallTool agency_design_from_spec: %v", err)
	}

	stored, err := mgr.ListGoals(ctx, draft.ID, mwanachamaagency.GoalFilter{})
	if err != nil {
		t.Fatalf("ListGoals: %v", err)
	}
	if !result.IsError && len(stored) >= goals {
		t.Errorf("AG58 (P2) open hole: one agency_design_from_spec call carrying %d goals was applied whole — %d goal rows "+
			"written in %s with no refusal; a per-call entity cap should refuse the spec with an error before anything is "+
			"written (this test assumes a cap below %d)", goals, len(stored), elapsed.Round(time.Millisecond), goals)
	}
}
