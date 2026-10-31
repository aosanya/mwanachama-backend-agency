package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAG48_UpdateKeepsARowInItsDraft(t *testing.T) {
	mgr, openDraft := newTestManagerWithDraft(t)
	archived, err := mgr.CreateDraft(t.Context(), "closed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if _, err := mgr.ArchiveDraft(t.Context(), archived.ID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}

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

	if code, _ := call(http.MethodPost, "/goals", `{"title":"direct","draft_id":"`+archived.ID+`"}`); code != http.StatusConflict {
		t.Fatalf("precondition: creating straight into an archived draft should be refused 409, got %d", code)
	}

	goal := create("/goals", `{"title":"G","draft_id":"`+openDraft+`"}`)
	objective := create("/objectives", `{"title":"O","goal_id":"`+goal+`","draft_id":"`+openDraft+`"}`)
	workflow := create("/workflows", `{"name":"W","draft_id":"`+openDraft+`"}`)
	workItem := create("/work-items", `{"title":"WI","workflow_id":"`+workflow+`","draft_id":"`+openDraft+`"}`)

	cases := []struct{ name, path, body string }{
		{"goal", "/goals/" + goal, `{"title":"G2","draft_id":"` + archived.ID + `"}`},
		{"objective", "/objectives/" + objective, `{"title":"O2","goal_id":"` + goal + `","draft_id":"` + archived.ID + `"}`},
		{"workflow", "/workflows/" + workflow, `{"name":"W2","draft_id":"` + archived.ID + `"}`},
		{"work item", "/work-items/" + workItem, `{"title":"WI2","workflow_id":"` + workflow + `","draft_id":"` + archived.ID + `"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			call(http.MethodPatch, c.path, c.body)
			code, got := call(http.MethodGet, c.path, "")
			if code != http.StatusOK {
				t.Fatalf("GET %s after PATCH -> %d %v", c.path, code, got)
			}
			if got["draft_id"] != openDraft {
				t.Errorf("AG48 (P1) open hole: PATCH %s with a body draft_id moved the %s into draft %v (archived, where create is refused 409); its draft should have stayed %s", c.path, c.name, got["draft_id"], openDraft)
			}
		})
	}
}

func TestRoleUpdateIgnoresABodyDraftID(t *testing.T) {
	mgr, openDraft := newTestManagerWithDraft(t)
	other, err := mgr.CreateDraft(t.Context(), "other", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	send := func(method, path, body string) (int, map[string]any) {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		serve(mgr, w, r)
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}

	code, role := send(http.MethodPost, "/roles", `{"name":"R","draft_id":"`+openDraft+`"}`)
	id, _ := role["id"].(string)
	if code != http.StatusCreated || id == "" {
		t.Fatalf("precondition: POST /roles -> %d %v", code, role)
	}
	if code, out := send(http.MethodPatch, "/roles/"+id, `{"name":"R2","draft_id":"`+other.ID+`"}`); code != http.StatusOK {
		t.Fatalf("PATCH /roles/%s -> %d %v", id, code, out)
	}
	_, got := send(http.MethodGet, "/roles/"+id, "")
	if got["name"] != "R2" {
		t.Fatalf("precondition: the PATCH should have renamed the role, got %v", got)
	}
	if got["draft_id"] != openDraft {
		t.Fatalf("a role's draft moved to %v on PATCH; it must stay %s", got["draft_id"], openDraft)
	}
}
