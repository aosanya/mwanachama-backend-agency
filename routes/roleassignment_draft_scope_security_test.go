package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoleAssignmentRefusesARoleOrStepFromAnotherDraft(t *testing.T) {
	mgr, draftA := newTestManagerWithDraft(t)
	b, err := mgr.CreateDraft(t.Context(), "other", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftB := b.ID

	call := func(method, path, body string) (int, map[string]any) {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		serve(mgr, w, r)
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}
	create := func(path, body string) string {
		t.Helper()
		code, out := call(http.MethodPost, path, body)
		id, _ := out["id"].(string)
		if code != http.StatusCreated || id == "" {
			t.Fatalf("precondition: POST %s %s -> %d %v", path, body, code, out)
		}
		return id
	}

	workflowA := create("/workflows", `{"name":"WA","draft_id":"`+draftA+`"}`)
	stepA := create("/work-items", `{"title":"SA","workflow_id":"`+workflowA+`","draft_id":"`+draftA+`"}`)
	roleA := create("/roles", `{"name":"RA","draft_id":"`+draftA+`"}`)
	workflowB := create("/workflows", `{"name":"WB","draft_id":"`+draftB+`"}`)
	stepB := create("/work-items", `{"title":"SB","workflow_id":"`+workflowB+`","draft_id":"`+draftB+`"}`)
	roleB := create("/roles", `{"name":"RB","draft_id":"`+draftB+`"}`)
	assignment := create("/role-assignments", `{"role_id":"`+roleA+`","work_item_id":"`+stepA+`","label":"responsible","draft_id":"`+draftA+`"}`)

	refusals := []struct{ name, method, path, body string }{
		{"create with a role from another draft", http.MethodPost, "/role-assignments",
			`{"role_id":"` + roleB + `","work_item_id":"` + stepA + `","label":"accountable","draft_id":"` + draftA + `"}`},
		{"create with a step from another draft", http.MethodPost, "/role-assignments",
			`{"role_id":"` + roleA + `","work_item_id":"` + stepB + `","label":"accountable","draft_id":"` + draftA + `"}`},
		{"update onto a role from another draft", http.MethodPatch, "/role-assignments/" + assignment,
			`{"role_id":"` + roleB + `","work_item_id":"` + stepA + `","label":"responsible"}`},
		{"update onto a step from another draft", http.MethodPatch, "/role-assignments/" + assignment,
			`{"role_id":"` + roleA + `","work_item_id":"` + stepB + `","label":"responsible"}`},
	}
	for _, c := range refusals {
		t.Run(c.name, func(t *testing.T) {
			if code, out := call(c.method, c.path, c.body); code != http.StatusNotFound {
				t.Errorf("%s %s %s -> %d %v; a RoleAssignment must name a Role and a step of its own draft (404)", c.method, c.path, c.body, code, out)
			}
		})
	}

	code, got := call(http.MethodGet, "/role-assignments/"+assignment, "")
	if code != http.StatusOK || got["role_id"] != roleA || got["work_item_id"] != stepA || got["draft_id"] != draftA {
		t.Errorf("GET /role-assignments/%s -> %d %v; the refused updates should have left role %s, step %s, draft %s", assignment, code, got, roleA, stepA, draftA)
	}
	for _, d := range []string{draftA, draftB} {
		w := httptest.NewRecorder()
		serve(mgr, w, httptest.NewRequest(http.MethodGet, "/role-assignments?draft_id="+d, nil))
		var list []map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
			var wrapped struct {
				Items []map[string]any `json:"items"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &wrapped)
			list = wrapped.Items
		}
		want := 0
		if d == draftA {
			want = 1
		}
		if w.Code != http.StatusOK || len(list) != want {
			t.Errorf("GET /role-assignments?draft_id=%s -> %d with %d rows; want %d — a refused create stored a row", d, w.Code, len(list), want)
		}
	}
}
