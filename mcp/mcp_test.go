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
	"github.com/aosanya/mwanachama-backend-agency/models"
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
		"agency_get_agency", "agency_create_goal", "agency_create_objective", "agency_create_workflow",
		"agency_set_raci",
		"agency_create_value", "agency_create_rule", "agency_design_from_spec",
		"agency_list_types",
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

	// The spec's own "agency" section — creates it since none exists yet.
	// This step alone needs no draft_id: CreateAgency isn't Draft-scoped.
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

	// Everything else — Goal/Objective — belongs to a Draft (AGD-006), so
	// that needs one forked first.
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

// TestDesignFromSpec_RaciAgainstExistingObjective covers the raci
// section's escape hatch from the key-only rule: a new Role, RACI'd
// against an Objective that already existed before this call (referenced
// by its real id, not a spec key) — the one-call replacement for the
// separate agency_create_role + agency_set_raci round trip.
func TestDesignFromSpec_RaciAgainstExistingObjective(t *testing.T) {
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

	result, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name: "agency_design_from_spec",
		Arguments: map[string]any{
			"instance_id": "any",
			"draft_id":    draft.ID,
			"roles":       []map[string]any{{"key": "r1", "name": "wire-check-role"}},
			"raci":        []map[string]any{{"role_key": "r1", "objective_id": objective.ID, "label": "informed"}},
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

	list, err := mgr.ListRoleAssignmentsForObjective(ctxT(t), objective.ID, draft.ID, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil {
		t.Fatalf("ListRoleAssignmentsForObjective: %v", err)
	}
	if len(list) != 1 || list[0].Label != models.RACILabelInformed {
		t.Fatalf("ListRoleAssignmentsForObjective = %+v", list)
	}
}

// TestDesignFromSpec_DuplicateKeySilentlyMisattributesReference pins a
// known defect — board row AG33 (documentation/3. implementation/todo.md):
// composite.go's per-section key maps (out.Roles/out.Goals/out.Objectives/
// out.Workflows/out.WorkItems, each keyed by the spec author's own `key`
// string) have no duplicate check, so a second spec entry reusing an
// already-seen key silently overwrites the first entry's id in the map.
// Every later reference to that key resolves to whichever entry was
// created LAST, with zero entries in "errors". If this is fixed (a
// duplicate key rejected, or reported as an error), this test's
// assertions should invert: either the call should report a
// duplicate-key error, or the two Goals should resolve independently.
func TestDesignFromSpec_DuplicateKeySilentlyMisattributesReference(t *testing.T) {
	mgr := newTestManager(t)
	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cs := mcpConnect(t, srv.URL)

	result, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name: "agency_design_from_spec",
		Arguments: map[string]any{
			"instance_id": "any",
			"goals": []map[string]any{
				{"key": "g1", "title": "Goal One"},
				{"key": "g1", "title": "Goal Two"},
			},
			"objectives": []map[string]any{
				{"key": "o1", "goal_key": "g1", "title": "Objective referencing g1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("agency_design_from_spec: IsError = true (content: %+v)", result.Content)
	}
	got, _ := result.StructuredContent.(map[string]any)

	// pin: current behaviour reports no error for the duplicate key.
	if errs, _ := got["errors"].([]any); len(errs) != 0 {
		t.Fatalf("pin: current behaviour reports no error for a duplicate spec key, got %+v — if this now fails, AG33 is fixed and should report a duplicate-key error instead", errs)
	}

	goals, err := mgr.ListGoals(ctxT(t), "", mwanachamaagency.GoalFilter{})
	if err != nil {
		t.Fatalf("ListGoals: %v", err)
	}
	if len(goals) != 2 {
		t.Fatalf("want 2 goals, got %d", len(goals))
	}
	var goalTwoID string
	for _, g := range goals {
		if g.Title == "Goal Two" {
			goalTwoID = g.ID
		}
	}
	if goalTwoID == "" {
		t.Fatal("could not find Goal Two")
	}

	objectives, err := mgr.ListObjectives(ctxT(t), "", mwanachamaagency.ObjectiveFilter{})
	if err != nil {
		t.Fatalf("ListObjectives: %v", err)
	}
	if len(objectives) != 1 {
		t.Fatalf("want 1 objective, got %d", len(objectives))
	}
	// pin: the Objective silently resolves to whichever Goal was created
	// LAST ("Goal Two"), never "Goal One" — if this now fails, AG33 is
	// fixed.
	if objectives[0].GoalID != goalTwoID {
		t.Fatalf("pin: objective.GoalID = %q, want it silently misattributed to Goal Two (%q) — if this now fails, AG33 is fixed", objectives[0].GoalID, goalTwoID)
	}
}
