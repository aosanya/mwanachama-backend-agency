package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAG72_OpenHole_RowsInAnArchivedDraftStillTakeUpdatesAndDeletes(t *testing.T) {
	mgr, draft := newTestManagerWithDraft(t)

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

	goal := create("/goals", `{"title":"G","draft_id":"`+draft+`"}`)
	objective := create("/objectives", `{"title":"O","goal_id":"`+goal+`","draft_id":"`+draft+`"}`)
	workflow := create("/workflows", `{"name":"W","draft_id":"`+draft+`"}`)
	workItem := create("/work-items", `{"title":"WI","workflow_id":"`+workflow+`","objective_id":"`+objective+`","draft_id":"`+draft+`"}`)
	role := create("/roles", `{"name":"R","draft_id":"`+draft+`"}`)
	assignment := create("/role-assignments", `{"role_id":"`+role+`","objective_id":"`+objective+`","label":"consulted","draft_id":"`+draft+`"}`)

	if _, err := mgr.ArchiveDraft(t.Context(), draft); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}
	if code, out := call(http.MethodGet, "/drafts/"+draft, ""); code != http.StatusOK || out["status"] != "archived" {
		t.Fatalf("precondition: the draft should read back archived, got %d %v", code, out)
	}
	if code, _ := call(http.MethodPost, "/goals", `{"title":"late","draft_id":"`+draft+`"}`); code != http.StatusConflict {
		t.Fatalf("precondition: creating into the archived draft should be refused 409, got %d", code)
	}

	cases := []struct{ name, method, path, body, readBack string }{
		{"rename a role", http.MethodPatch, "/roles/" + role, `{"name":"R-edited"}`, "/roles/" + role},
		{"delete a RACI link", http.MethodDelete, "/role-assignments/" + assignment, ``, "/role-assignments/" + assignment},
		{"delete a work item", http.MethodDelete, "/work-items/" + workItem, ``, "/work-items/" + workItem},
		{"delete a role", http.MethodDelete, "/roles/" + role, ``, "/roles/" + role},
		{"delete a goal", http.MethodDelete, "/goals/" + goal, ``, "/goals/" + goal},
	}
	for _, c := range cases {
		code, out := call(c.method, c.path, c.body)
		if code != http.StatusConflict {
			t.Errorf("AG72 (P2) open hole: %s %s in archived draft %s -> %d %v; the draft is archived and creating into it answers 409, so this %s should be refused 409 too", c.method, c.path, draft, code, out, c.name)
		}
		if got, row := call(http.MethodGet, c.readBack, ""); got != http.StatusOK || row["name"] == "R-edited" {
			t.Errorf("AG72 (P2) open hole: after %s %s the row in archived draft %s reads back %d %v; it should still read back unchanged", c.method, c.path, draft, got, row)
		}
	}
}
