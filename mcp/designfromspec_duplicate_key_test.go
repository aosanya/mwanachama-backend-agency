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

func TestAG60_DesignFromSpecRefusesADuplicateKeyNamingIt(t *testing.T) {
	cases := []struct {
		name    string
		section string
		key     string
		spec    map[string]any
	}{
		{"goals", "goals", "g1", map[string]any{
			"goals":      []map[string]any{{"key": "g1", "title": "Goal One"}, {"key": "g1", "title": "Goal Two"}},
			"objectives": []map[string]any{{"key": "o1", "goal_key": "g1", "title": "Objective"}},
		}},
		{"roles", "roles", "r1", map[string]any{
			"roles": []map[string]any{{"key": "r1", "name": "Lead"}, {"key": "r1", "name": "Other"}},
		}},
		{"objectives", "objectives", "o1", map[string]any{
			"goals": []map[string]any{{"key": "g1", "title": "Goal"}},
			"objectives": []map[string]any{
				{"key": "o1", "goal_key": "g1", "title": "First"},
				{"key": "o1", "goal_key": "g1", "title": "Second"},
			},
		}},
		{"workflows", "workflows", "w1", map[string]any{
			"workflows": []map[string]any{{"key": "w1", "name": "First"}, {"key": "w1", "name": "Second"}},
		}},
		{"work_items", "work_items", "wi1", map[string]any{
			"goals":      []map[string]any{{"key": "g1", "title": "Goal"}},
			"objectives": []map[string]any{{"key": "o1", "goal_key": "g1", "title": "Objective"}},
			"workflows":  []map[string]any{{"key": "w1", "name": "Flow"}},
			"work_items": []map[string]any{
				{"key": "wi1", "workflow_key": "w1", "objective_key": "o1", "title": "First"},
				{"key": "wi1", "workflow_key": "w1", "objective_key": "o1", "title": "Second"},
			},
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := newTestManager(t)
			server := newTestServer(mgr)
			mux := http.NewServeMux()
			mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
			srv := httptest.NewServer(mux)
			defer srv.Close()
			cs := mcpConnect(t, srv.URL)

			if _, err := mgr.CreateAgency(ctxT(t), models.Agency{Name: "AG60"}); err != nil {
				t.Fatalf("CreateAgency: %v", err)
			}
			draft, err := mgr.CreateDraft(ctxT(t), "AG60", "", "")
			if err != nil {
				t.Fatalf("CreateDraft: %v", err)
			}
			tc.spec["instance_id"] = "any"
			tc.spec["draft_id"] = draft.ID

			result, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{Name: "agency_design_from_spec", Arguments: tc.spec})
			if err != nil {
				t.Fatalf("CallTool: %v", err)
			}
			if !result.IsError {
				t.Fatalf("a spec reusing %s key %q was accepted (structured: %+v); want it refused", tc.section, tc.key, result.StructuredContent)
			}
			var text strings.Builder
			for _, c := range result.Content {
				if tc, ok := c.(*gosdkmcp.TextContent); ok {
					text.WriteString(tc.Text)
				}
			}
			if !strings.Contains(text.String(), `"`+tc.key+`"`) || !strings.Contains(text.String(), tc.section) {
				t.Errorf("refusal %q does not name the section %q and key %q", text.String(), tc.section, tc.key)
			}
			goals, err := mgr.ListGoals(ctxT(t), draft.ID, mwanachamaagency.GoalFilter{})
			if err != nil || len(goals) != 0 {
				t.Errorf("a refused spec still created goals: %+v (err %v)", goals, err)
			}
		})
	}
}
