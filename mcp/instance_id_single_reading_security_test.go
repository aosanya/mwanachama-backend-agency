package mcp_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	agencymcp "github.com/aosanya/mwanachama-backend-agency/mcp"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type memberOnlyResolver struct {
	mu     sync.Mutex
	member mwanachamaagency.AgencyManager
	asked  []string
}

func (r *memberOnlyResolver) resolve(_ context.Context, id string) (mwanachamaagency.AgencyManager, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.asked = append(r.asked, id)
	if id == "member" {
		return r.member, nil
	}
	return nil, errors.New("instance not found")
}

func (r *memberOnlyResolver) take() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.asked
	r.asked = nil
	return out
}

func TestInstanceIDReachesTheResolverOnceAndInOneSpelling(t *testing.T) {
	ctx := ctxT(t)
	member, outsider := newTestManager(t), newTestManager(t)
	if _, err := member.CreateAgency(ctx, models.Agency{Name: "member"}); err != nil {
		t.Fatalf("CreateAgency member: %v", err)
	}
	if _, err := outsider.CreateAgency(ctx, models.Agency{Name: "outsider"}); err != nil {
		t.Fatalf("CreateAgency outsider: %v", err)
	}
	outsiderDraft, err := outsider.CreateDraft(ctx, "outsider", "", "")
	if err != nil {
		t.Fatalf("CreateDraft outsider: %v", err)
	}
	if _, err := outsider.CreateGoal(ctx, models.Goal{DraftID: outsiderDraft.ID, Title: "OUTSIDER-SECRET"}); err != nil {
		t.Fatalf("CreateGoal outsider: %v", err)
	}
	if goals, err := outsider.ListGoals(ctx, outsiderDraft.ID, mwanachamaagency.GoalFilter{}); err != nil || len(goals) != 1 {
		t.Fatalf("precondition: the outsider's draft should hold one goal, got %d (%v)", len(goals), err)
	}

	r := &memberOnlyResolver{member: member}
	server := gosdkmcp.NewServer(&gosdkmcp.Implementation{Name: "test", Version: "0.0.0"}, nil)
	agencymcp.RegisterTools(server, r.resolve)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server },
		&gosdkmcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true}))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cs := mcpConnect(t, srv.URL)

	toolArgs := map[string]map[string]any{
		"agency_list_goals":             {"draft_id": outsiderDraft.ID},
		"agency_list_module_objectives": {"draft_id": outsiderDraft.ID, "module_id": "m"},
		"agency_design_from_spec":       {"draft_id": outsiderDraft.ID, "goals": []any{map[string]any{"key": "g", "title": "PLANTED"}}},
	}
	respellings := []map[string]any{
		{"instance_id": "member", "INSTANCE_ID": "outsider"},
		{"instance_id": "member", "Instance_Id": "outsider"},
		{"instance_id": "member", "instanceId": "outsider"},
		{"instance_id": "member", "instance_id ": "outsider"},
		{"INSTANCE_ID": "outsider"},
	}
	for tool, base := range toolArgs {
		for _, extra := range respellings {
			args := map[string]any{}
			for k, v := range base {
				args[k] = v
			}
			for k, v := range extra {
				args[k] = v
			}
			res, err := cs.CallTool(ctx, &gosdkmcp.CallToolParams{Name: tool, Arguments: args})
			if err != nil {
				t.Fatalf("%s %v: %v", tool, extra, err)
			}
			if asked := r.take(); len(asked) != 0 {
				t.Errorf("%s with %v reached the resolver as %v; a second spelling of instance_id must be refused before any Instance is resolved", tool, extra, asked)
			}
			if !res.IsError {
				t.Errorf("%s with %v was accepted; a second spelling of instance_id must be refused", tool, extra)
			}
		}
	}

	rawCalls := []struct {
		body string
		want string
	}{
		{`{"instance_id":"member","draft_id":"` + outsiderDraft.ID + `","instance_id":"outsider"}`, "outsider"},
		{`{"instance_id":"outsider","draft_id":"` + outsiderDraft.ID + `","instance_id":"member"}`, "member"},
	}
	for i, c := range rawCalls {
		body := `{"jsonrpc":"2.0","id":` + string(rune('1'+i)) + `,"method":"tools/call","params":{"name":"agency_list_goals","arguments":` + c.body + `}}`
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+"/mcp", bytes.NewBufferString(body))
		if err != nil {
			t.Fatalf("NewRequest: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("POST /mcp: %v", err)
		}
		answer, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		asked := r.take()
		if len(asked) != 1 || asked[0] != c.want {
			t.Errorf("duplicate instance_id keys %s reached the resolver as %v, want exactly [%s] — the id the host checks must be the only id used", c.body, asked, c.want)
		}
		if strings.Contains(string(answer), "OUTSIDER-SECRET") {
			t.Errorf("duplicate instance_id keys %s answered with the outsider's goal: %s", c.body, answer)
		}
	}

	goals, err := outsider.ListGoals(ctx, outsiderDraft.ID, mwanachamaagency.GoalFilter{})
	if err != nil {
		t.Fatalf("ListGoals outsider: %v", err)
	}
	if len(goals) != 1 {
		t.Errorf("the outsider's draft holds %d goals after the calls, want 1 — nothing may be planted through a respelt instance_id", len(goals))
	}
}
