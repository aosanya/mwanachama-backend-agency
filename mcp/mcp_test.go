package mcp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	agencymcp "github.com/aosanya/mwanachama-backend-agency/mcp"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func ctxT(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

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
		"agency_get_agency", "agency_create_goal", "agency_create_objective", "agency_create_workflow",
		"agency_set_raci",
		"agency_create_value", "agency_create_rule", "agency_design_from_spec",
		"agency_list_types",
		"agency_set_module_enabled", "agency_list_modules", "agency_get_module",
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
		Arguments: map[string]any{"instance_id": "any", "name": "SWELab"},
	})
	if err != nil {
		t.Fatalf("CallTool agency_create_agency: %v", err)
	}
	if result.IsError {
		t.Fatalf("agency_create_agency: IsError = true (content: %+v)", result.Content)
	}

	result, err = cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_create_draft",
		Arguments: map[string]any{"instance_id": "any", "description": "seed"},
	})
	if err != nil {
		t.Fatalf("CallTool agency_create_draft: %v", err)
	}
	if result.IsError {
		t.Fatalf("agency_create_draft: IsError = true (content: %+v)", result.Content)
	}
	draftSC, _ := result.StructuredContent.(map[string]any)
	draftID, _ := draftSC["id"].(string)
	if draftID == "" {
		t.Fatalf("agency_create_draft structured content = %+v", result.StructuredContent)
	}

	result, err = cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_create_goal",
		Arguments: map[string]any{"instance_id": "any", "title": "Grow membership", "draft_id": draftID},
	})
	if err != nil {
		t.Fatalf("CallTool agency_create_goal: %v", err)
	}
	if result.IsError {
		t.Fatalf("agency_create_goal: IsError = true (content: %+v)", result.Content)
	}

	result, err = cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_list_goals",
		Arguments: map[string]any{"instance_id": "any", "draft_id": draftID},
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
			"agency":      map[string]any{"name": "SWELab"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool agency_design_from_spec (agency only): %v", err)
	}
	if result.IsError {
		t.Fatalf("agency_design_from_spec (agency only): IsError = true (content: %+v)", result.Content)
	}
	got, _ := result.StructuredContent.(map[string]any)
	if got["agency_created"] != true {
		t.Fatalf("agency_design_from_spec structured content = %+v", result.StructuredContent)
	}

	result, err = cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_create_draft",
		Arguments: map[string]any{"instance_id": "any", "description": "seed"},
	})
	if err != nil {
		t.Fatalf("CallTool agency_create_draft: %v", err)
	}
	draftSC, _ := result.StructuredContent.(map[string]any)
	draftID, _ := draftSC["id"].(string)
	if draftID == "" {
		t.Fatalf("agency_create_draft structured content = %+v", result.StructuredContent)
	}

	result, err = cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name: "agency_design_from_spec",
		Arguments: map[string]any{
			"instance_id": "any",
			"draft_id":    draftID,
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
	got, _ = result.StructuredContent.(map[string]any)
	if errs, _ := got["errors"].([]any); len(errs) != 0 {
		t.Fatalf("agency_design_from_spec unexpected errors: %+v", errs)
	}
	goals, _ := got["goals"].(map[string]any)
	if len(goals) != 1 {
		t.Fatalf("agency_design_from_spec structured content = %+v", result.StructuredContent)
	}
}

func TestDesignFromSpec_RaciAgainstExistingWorkItem(t *testing.T) {
	mgr := newTestManager(t)
	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cs := mcpConnect(t, srv.URL)

	_, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_create_agency",
		Arguments: map[string]any{"instance_id": "any", "name": "SWELab"},
	})
	if err != nil {
		t.Fatalf("CallTool agency_create_agency: %v", err)
	}
	draft, err := mgr.CreateDraft(ctxT(t), "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	goal, err := mgr.CreateGoal(ctxT(t), models.Goal{DraftID: draft.ID, Title: "Ship reliable software"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	objective, err := mgr.CreateObjective(ctxT(t), models.Objective{DraftID: draft.ID, GoalID: goal.ID, Title: "Ship reliable software"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	workflow, err := mgr.CreateWorkflow(ctxT(t), models.Workflow{DraftID: draft.ID, Name: "Release"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	if _, err := mgr.CreateWorkflowObjective(ctxT(t), models.WorkflowObjective{DraftID: draft.ID, WorkflowID: workflow.ID, ObjectiveID: objective.ID}); err != nil {
		t.Fatalf("CreateWorkflowObjective: %v", err)
	}
	workItem, err := mgr.CreateWorkItem(ctxT(t), models.WorkItem{DraftID: draft.ID, WorkflowID: workflow.ID, Title: "Cut the release"})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}

	result, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name: "agency_design_from_spec",
		Arguments: map[string]any{
			"instance_id": "any",
			"draft_id":    draft.ID,
			"roles":       []map[string]any{{"key": "r1", "name": "wire-check-role"}},
			"raci":        []map[string]any{{"role_key": "r1", "work_item_id": workItem.ID, "label": "informed"}},
		},
	})
	if err != nil {
		t.Fatalf("CallTool agency_design_from_spec: %v", err)
	}
	if result.IsError {
		t.Fatalf("agency_design_from_spec: IsError = true (content: %+v)", result.Content)
	}
	got, _ := result.StructuredContent.(map[string]any)
	if errs, _ := got["errors"].([]any); len(errs) != 0 {
		t.Fatalf("agency_design_from_spec unexpected errors: %+v", errs)
	}
	assignments, _ := got["role_assignments"].([]any)
	if len(assignments) != 1 {
		t.Fatalf("agency_design_from_spec role_assignments = %+v", got["role_assignments"])
	}

	list, err := mgr.ListRoleAssignmentsForWorkItem(ctxT(t), workItem.ID, draft.ID, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil {
		t.Fatalf("ListRoleAssignmentsForWorkItem: %v", err)
	}
	if len(list) != 1 || list[0].Label != models.RACILabelInformed || list[0].WorkItemID != workItem.ID {
		t.Fatalf("ListRoleAssignmentsForWorkItem = %+v", list)
	}
}

func TestModuleSwitchTools_SetListAndRefuseLocked(t *testing.T) {
	mgr := newTestManager(t)
	ctx := ctxT(t)
	drafts, err := mgr.ListDrafts(ctx, mwanachamaagency.DraftFilter{})
	if err != nil || len(drafts) != 1 {
		t.Fatalf("ListDrafts: %+v err=%v", drafts, err)
	}
	draftID := drafts[0].ID
	git, err := seededModule(mgr, ctx, draftID, "git")
	if err != nil {
		t.Fatalf("seededModule: %v", err)
	}
	auth, err := seededModule(mgr, ctx, draftID, "auth")
	if err != nil {
		t.Fatalf("seededModule: %v", err)
	}

	mux := http.NewServeMux()
	server := newTestServer(mgr)
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cs := mcpConnect(t, srv.URL)

	on, err := cs.CallTool(ctx, &gosdkmcp.CallToolParams{Name: "agency_set_module_enabled",
		Arguments: map[string]any{"instance_id": "any", "draft_id": draftID, "module_id": git.ID, "enabled": true}})
	if err != nil || on.IsError {
		t.Fatalf("switch on: err=%v result=%+v", err, on)
	}
	listed, err := cs.CallTool(ctx, &gosdkmcp.CallToolParams{Name: "agency_list_modules",
		Arguments: map[string]any{"instance_id": "any", "draft_id": draftID, "code": "git"}})
	if err != nil || listed.IsError {
		t.Fatalf("list: err=%v result=%+v", err, listed)
	}
	b, _ := json.Marshal(listed.StructuredContent)
	for _, want := range []string{`"enabled":true`, `"explicit":true`, `"needed_by_steps":false`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("list result missing %s: %s", want, b)
		}
	}

	locked, err := cs.CallTool(ctx, &gosdkmcp.CallToolParams{Name: "agency_set_module_enabled",
		Arguments: map[string]any{"instance_id": "any", "draft_id": draftID, "module_id": auth.ID, "enabled": false}})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !locked.IsError {
		t.Fatalf("switching a locked module off should be a tool error")
	}

	elsewhere, err := cs.CallTool(ctx, &gosdkmcp.CallToolParams{Name: "agency_set_module_enabled",
		Arguments: map[string]any{"instance_id": "any", "draft_id": "no-such-draft", "module_id": git.ID, "enabled": false}})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !elsewhere.IsError {
		t.Fatalf("a module of another draft should not be found")
	}
	if got, err := mgr.GetModule(ctx, git.ID); err != nil || !got.Explicit {
		t.Fatalf("a refused call changed git: %+v %v", got, err)
	}
}
