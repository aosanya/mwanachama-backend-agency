package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestWorkItemNestedListingsNeverReachAnotherDraftsRows(t *testing.T) {
	mgr := newTestManager(t)
	ctx := t.Context()
	draftA := bootstrapDraftID(t, mgr)
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "SWELab"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	other, err := mgr.CreateDraft(ctx, "other", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftB := other.ID

	call := func(method, path, body string) (int, []byte) {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		serve(mgr, w, r)
		return w.Code, w.Body.Bytes()
	}
	create := func(path, body string) string {
		t.Helper()
		code, raw := call(http.MethodPost, path, body)
		var out map[string]any
		_ = json.Unmarshal(raw, &out)
		id, _ := out["id"].(string)
		if code != http.StatusCreated || id == "" {
			t.Fatalf("precondition: POST %s %s -> %d %s", path, body, code, raw)
		}
		return id
	}
	rows := func(path string) []map[string]any {
		t.Helper()
		code, raw := call(http.MethodGet, path, "")
		if code != http.StatusOK {
			t.Fatalf("GET %s -> %d %s, want 200", path, code, raw)
		}
		var list []map[string]any
		if err := json.Unmarshal(raw, &list); err != nil {
			t.Fatalf("GET %s: %v in %s", path, err, raw)
		}
		return list
	}

	goal := create("/goals", `{"title":"GA","draft_id":"`+draftA+`"}`)
	objective := create("/objectives", `{"title":"OA","goal_id":"`+goal+`","draft_id":"`+draftA+`"}`)
	role := create("/roles", `{"name":"RA","draft_id":"`+draftA+`"}`)
	workflow := create("/workflows", `{"name":"WA","draft_id":"`+draftA+`"}`)
	step := create("/work-items", `{"title":"SA","workflow_id":"`+workflow+`","draft_id":"`+draftA+`"}`)
	create("/role-assignments", `{"role_id":"`+role+`","work_item_id":"`+step+`","label":"accountable","draft_id":"`+draftA+`"}`)

	for _, nested := range []string{"role-assignments"} {
		base := "/work-items/" + step + "/" + nested
		if got := rows(base + "?draft_id=" + draftA); len(got) != 1 || got[0]["draft_id"] != draftA {
			t.Fatalf("precondition: GET %s?draft_id=<A> -> %v, want the one row of draft A", base, got)
		}
		for _, c := range []struct{ name, path string }{
			{"another open draft", base + "?draft_id=" + draftB},
			{"no draft at all", base},
			{"a draft that does not exist", base + "?draft_id=no-such-draft"},
		} {
			if got := rows(c.path); len(got) != 0 {
				t.Errorf("GET %s (%s) -> %v; a nested listing must answer only the draft it names, never draft A's %s", c.path, c.name, got, nested)
			}
		}
	}

	if _, err := mgr.GetObjective(ctx, objective); err != nil {
		t.Fatalf("postcondition: GetObjective: %v", err)
	}
	assignments, err := mgr.ListRoleAssignmentsForWorkItem(ctx, step, draftA, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil || len(assignments) != 1 {
		t.Fatalf("postcondition: draft A's assignment changed: %+v %v", assignments, err)
	}
}
