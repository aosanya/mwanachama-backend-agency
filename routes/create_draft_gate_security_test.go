package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEveryDraftScopedCreateRefusesAnArchivedMissingOrAbsentDraft(t *testing.T) {
	mgr, open := newTestManagerWithDraft(t)
	archived, err := mgr.CreateDraft(t.Context(), "closed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if _, err := mgr.ArchiveDraft(t.Context(), archived.ID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}

	call := func(method, path, body string) (int, string) {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		serve(mgr, w, r)
		return w.Code, strings.TrimSpace(w.Body.String())
	}
	create := func(path, body string) string {
		t.Helper()
		code, got := call(http.MethodPost, path, body)
		var out map[string]any
		_ = json.Unmarshal([]byte(got), &out)
		id, _ := out["id"].(string)
		if code != http.StatusCreated || id == "" {
			t.Fatalf("precondition: POST %s %s in the open draft -> %d %s", path, body, code, got)
		}
		return id
	}

	goal := create("/goals", `{"title":"G","draft_id":"`+open+`"}`)
	objective := create("/objectives", `{"title":"O","goal_id":"`+goal+`","draft_id":"`+open+`"}`)
	workflow := create("/workflows", `{"name":"W","draft_id":"`+open+`"}`)
	role := create("/roles", `{"name":"R","draft_id":"`+open+`"}`)
	step := create("/work-items", `{"title":"S","workflow_id":"`+workflow+`","draft_id":"`+open+`"}`)

	bodies := []struct{ path, body string }{
		{"/goals", `{"title":"late","draft_id":%q}`},
		{"/objectives", `{"title":"late","goal_id":"` + goal + `","draft_id":%q}`},
		{"/workflows", `{"name":"late","draft_id":%q}`},
		{"/work-items", `{"title":"late","workflow_id":"` + workflow + `","draft_id":%q}`},
		{"/workflow-objectives", `{"workflow_id":"` + workflow + `","objective_id":"` + objective + `","draft_id":%q}`},
		{"/roles", `{"name":"late","draft_id":%q}`},
		{"/role-assignments", `{"role_id":"` + role + `","work_item_id":"` + step + `","label":"consulted","draft_id":%q}`},
		{"/values", `{"text":"late","draft_id":%q}`},
		{"/rules", `{"text":"late","draft_id":%q}`},
	}
	drafts := []struct {
		name, id string
		want     int
	}{
		{"archived", archived.ID, http.StatusConflict},
		{"missing", "no-such-draft", http.StatusNotFound},
		{"absent", "", http.StatusBadRequest},
	}

	for _, d := range drafts {
		for _, b := range bodies {
			t.Run(d.name+b.path, func(t *testing.T) {
				body := strings.Replace(b.body, "%q", `"`+d.id+`"`, 1)
				code, got := call(http.MethodPost, b.path, body)
				if code != d.want {
					t.Errorf("POST %s naming the %s draft %q -> %d %s, want %d and nothing stored", b.path, d.name, d.id, code, got, d.want)
				}
			})
		}
	}

	for _, b := range bodies {
		for _, scope := range []string{archived.ID, "no-such-draft"} {
			code, got := call(http.MethodGet, b.path+"?draft_id="+scope, "")
			if code != http.StatusOK || got != "[]" {
				t.Errorf("GET %s?draft_id=%s -> %d %s, want 200 [] — a refused create must store nothing", b.path, scope, code, got)
			}
		}
	}
}
