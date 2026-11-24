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

func TestDesignFromSpec_StoresNothingInAnArchivedUnknownOrUnnamedDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := ctxT(t)
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "SWELab"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	archived, err := mgr.CreateDraft(ctx, "archived", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	goal, err := mgr.CreateGoal(ctx, models.Goal{DraftID: archived.ID, Title: "kept"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	objective, err := mgr.CreateObjective(ctx, models.Objective{DraftID: archived.ID, GoalID: goal.ID, Title: "kept"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: archived.ID, Name: "kept"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := mgr.ArchiveDraft(ctx, archived.ID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}

	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cs := mcpConnect(t, srv.URL)

	type counts struct{ workflows, steps, links int }
	count := func(draftID string) counts {
		workflows, _ := mgr.ListWorkflows(ctx, draftID, mwanachamaagency.WorkflowFilter{})
		steps, _ := mgr.ListWorkItems(ctx, draftID, mwanachamaagency.WorkItemFilter{})
		links, _ := mgr.ListWorkflowObjectives(ctx, draftID, mwanachamaagency.WorkflowObjectiveFilter{})
		return counts{len(workflows), len(steps), len(links)}
	}
	before := map[string]counts{}
	for _, draftID := range []string{archived.ID, "no-such-draft", ""} {
		before[draftID] = count(draftID)
	}

	cases := map[string]string{"archived": archived.ID, "unknown": "no-such-draft", "unnamed": ""}
	for name, draftID := range cases {
		result, err := cs.CallTool(ctx, &gosdkmcp.CallToolParams{
			Name: "agency_design_from_spec",
			Arguments: map[string]any{
				"instance_id": "test",
				"draft_id":    draftID,
				"roles":       []any{map[string]any{"key": "r", "name": "planted role"}},
				"goals":       []any{map[string]any{"key": "g", "title": "planted goal"}},
				"objectives":  []any{map[string]any{"key": "o", "goal_key": "g", "title": "planted objective"}},
				"workflows":   []any{map[string]any{"key": "w", "name": "planted workflow", "objective_keys": []any{"o"}, "objective_ids": []any{objective.ID}}},
				"work_items":  []any{map[string]any{"key": "wi", "workflow_key": "w", "title": "planted item", "accountable_role_id": role.ID}},
			},
		})
		if err != nil {
			t.Fatalf("%s: CallTool agency_design_from_spec: %v", name, err)
		}
		sc, ok := result.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("%s: structuredContent is %T, want map[string]any", name, result.StructuredContent)
		}
		errs, _ := sc["errors"].([]any)
		if len(errs) != 5 {
			t.Errorf("%s draft: want every one of the five spec entries refused, got errors %+v", name, errs)
		}
		for _, section := range []string{"roles", "goals", "objectives", "workflows", "work_items"} {
			if created, _ := sc[section].(map[string]any); len(created) != 0 {
				t.Errorf("%s draft: the spec's %s were reported created: %+v", name, section, created)
			}
		}
		if linked, _ := sc["workflow_objectives"].([]any); len(linked) != 0 {
			t.Errorf("%s draft: a workflow objective link was reported created: %+v", name, linked)
		}
	}

	for _, draftID := range []string{archived.ID, "no-such-draft", ""} {
		goals, _ := mgr.ListGoals(ctx, draftID, mwanachamaagency.GoalFilter{})
		roles, _ := mgr.ListRoles(ctx, draftID, mwanachamaagency.RoleFilter{})
		for _, g := range goals {
			if strings.HasPrefix(g.Title, "planted") {
				t.Errorf("draft %q holds a goal the spec planted: %+v", draftID, g)
			}
		}
		for _, r := range roles {
			if strings.HasPrefix(r.Name, "planted") {
				t.Errorf("draft %q holds a role the spec planted: %+v", draftID, r)
			}
		}
		if after := count(draftID); after != before[draftID] {
			t.Errorf("draft %q held %+v before the refused spec and holds %+v after: workflows, RACI assignments and workflow objective links must not move", draftID, before[draftID], after)
		}
	}
}
