package mcp_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCreateToolRefusesCaseVariantsOfServerOwnedFields(t *testing.T) {
	mgr := newTestManager(t)
	ctx := ctxT(t)
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "case"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draft, err := mgr.CreateDraft(ctx, "case", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	other, err := mgr.CreateDraft(ctx, "other", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cs := mcpConnect(t, srv.URL)

	smuggled := []map[string]any{
		{"Deleted": true},
		{"DELETED": true},
		{"ID": "forged-id"},
		{"Created_At": "1999-01-01T00:00:00Z"},
		{"CreatedAt": "1999-01-01T00:00:00Z"},
		{"Draft_ID": other.ID},
		{"DraftID": other.ID},
		{"Code": "G-999"},
	}
	for _, extra := range smuggled {
		args := map[string]any{"instance_id": "any", "draft_id": draft.ID, "title": "smuggled"}
		for k, v := range extra {
			args[k] = v
		}
		res, err := cs.CallTool(ctx, &gosdkmcp.CallToolParams{Name: "agency_create_goal", Arguments: args})
		if err != nil {
			t.Fatalf("agency_create_goal %v: %v", extra, err)
		}
		if !res.IsError {
			t.Errorf("agency_create_goal with %v should be refused as an unexpected property, was accepted", extra)
			continue
		}
		text := ""
		for _, c := range res.Content {
			if tc, ok := c.(*gosdkmcp.TextContent); ok {
				text += tc.Text
			}
		}
		if !strings.Contains(text, "unexpected additional properties") {
			t.Errorf("agency_create_goal with %v refused for the wrong reason: %s", extra, text)
		}
	}

	for _, d := range []string{draft.ID, other.ID} {
		goals, err := mgr.ListGoals(ctx, d, mwanachamaagency.GoalFilter{})
		if err != nil {
			t.Fatalf("ListGoals: %v", err)
		}
		if len(goals) != 0 {
			t.Errorf("no refused call should store a goal; draft %s holds %d", d, len(goals))
		}
	}
}
