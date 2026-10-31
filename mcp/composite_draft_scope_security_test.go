package mcp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestDesignFromSpec_RaciAndObjectiveLinksNeverReachAnotherDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := ctxT(t)
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "SWELab"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	other, err := mgr.CreateDraft(ctx, "other", "", "")
	if err != nil {
		t.Fatalf("CreateDraft other: %v", err)
	}
	target, err := mgr.CreateDraft(ctx, "target", "", "")
	if err != nil {
		t.Fatalf("CreateDraft target: %v", err)
	}
	foreignRole, err := mgr.CreateRole(ctx, models.Role{DraftID: other.ID, Name: "foreign"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	g, err := mgr.CreateGoal(ctx, models.Goal{DraftID: other.ID, Title: "foreign goal"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	foreignObjective, err := mgr.CreateObjective(ctx, models.Objective{DraftID: other.ID, GoalID: g.ID, Title: "foreign objective"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	foreignFlow, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: other.ID, Name: "foreign flow"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	foreignStep, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: other.ID, WorkflowID: foreignFlow.ID, Title: "foreign step"})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}

	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cs := mcpConnect(t, srv.URL)
	result, err := cs.CallTool(ctx, &gosdkmcp.CallToolParams{
		Name: "agency_design_from_spec",
		Arguments: map[string]any{
			"instance_id": "test",
			"draft_id":    target.ID,
			"roles":       []any{map[string]any{"key": "local", "name": "local"}},
			"workflows":   []any{map[string]any{"key": "w", "name": "local flow", "objective_ids": []any{foreignObjective.ID}}},
			"raci": []any{
				map[string]any{"role_key": "local", "work_item_id": foreignStep.ID, "label": "accountable"},
				map[string]any{"role_id": foreignRole.ID, "work_item_id": foreignStep.ID, "label": "consulted"},
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool agency_design_from_spec: %v", err)
	}
	sc, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structuredContent is %T, want map[string]any", result.StructuredContent)
	}
	if roles, _ := sc["roles"].(map[string]any); roles["local"] == nil {
		t.Fatalf("precondition: the spec's own role was not created in the target draft: %+v", sc)
	}
	if errs, _ := sc["errors"].([]any); len(errs) != 3 {
		t.Errorf("want the cross-draft objective link and both cross-draft raci entries refused, got errors %+v", sc["errors"])
	}
	if linked, _ := sc["workflow_objectives"].([]any); len(linked) != 0 {
		t.Errorf("a workflow objective link naming another draft's objective was reported created: %+v", linked)
	}

	for _, d := range []string{other.ID, target.ID} {
		assigned, err := mgr.ListRoleAssignments(ctx, d, mwanachamaagency.RoleAssignmentFilter{})
		if err != nil {
			t.Fatalf("ListRoleAssignments %s: %v", d, err)
		}
		if len(assigned) != 0 {
			t.Errorf("a raci entry naming another draft's role or step was stored in draft %s: %+v", d, assigned)
		}
		links, err := mgr.ListWorkflowObjectives(ctx, d, mwanachamaagency.WorkflowObjectiveFilter{})
		if err != nil {
			t.Fatalf("ListWorkflowObjectives %s: %v", d, err)
		}
		if len(links) != 0 {
			t.Errorf("a workflow objective link naming another draft's objective was stored in draft %s: %+v", d, links)
		}
	}
}
