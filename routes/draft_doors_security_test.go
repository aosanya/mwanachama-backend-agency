package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func draftCall(t *testing.T, serveTo func(http.ResponseWriter, *http.Request), method, path, body string) (int, map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	serveTo(w, r)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func TestDraftDoorsRefuseADraftThatDoesNotExist(t *testing.T) {
	mgr, _ := newTestManagerWithDraft(t)
	to := func(w http.ResponseWriter, r *http.Request) { serve(mgr, w, r) }

	if code, _ := draftCall(t, to, http.MethodGet, "/drafts/no-such-draft", ""); code != http.StatusNotFound {
		t.Fatalf("precondition: GET /drafts/no-such-draft -> %d, want 404", code)
	}

	doors := []struct{ method, path, body string }{
		{http.MethodPost, "/publications", `{"draft_id":"no-such-draft","tag":"t1"}`},
		{http.MethodPost, "/drafts", `{"description":"fork","forked_from_id":"no-such-draft","forked_from_type":"draft"}`},
		{http.MethodPatch, "/drafts/no-such-draft", `{"description":"renamed"}`},
		{http.MethodPost, "/drafts/no-such-draft/archive", ""},
		{http.MethodPost, "/goals", `{"draft_id":"no-such-draft","title":"g"}`},
	}
	for _, d := range doors {
		if code, out := draftCall(t, to, d.method, d.path, d.body); code != http.StatusNotFound {
			t.Errorf("%s %s %s -> %d %v, want 404 draft not found", d.method, d.path, d.body, code, out)
		}
	}
}

func TestDraftCreateAndUpdateIgnoreAForgedIDStatusAndClock(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	to := func(w http.ResponseWriter, r *http.Request) { serve(mgr, w, r) }

	code, created := draftCall(t, to, http.MethodPost, "/drafts", `{"description":"f","id":"chosen","status":"archived","created_at":"1999-01-01T00:00:00Z"}`)
	if code != http.StatusCreated {
		t.Fatalf("precondition: POST /drafts -> %d %v", code, created)
	}
	if created["id"] == "chosen" || created["status"] != "open" || created["created_at"] == "1999-01-01T00:00:00Z" {
		t.Errorf("POST /drafts kept a caller-set field: %v", created)
	}

	if code, out := draftCall(t, to, http.MethodPatch, "/drafts/"+draftID, `{"description":"d2","status":"archived","id":"other"}`); code != http.StatusOK {
		t.Fatalf("precondition: PATCH /drafts/%s -> %d %v", draftID, code, out)
	}
	_, read := draftCall(t, to, http.MethodGet, "/drafts/"+draftID, "")
	if read["id"] != draftID || read["status"] != "open" || read["description"] != "d2" {
		t.Errorf("PATCH /drafts/{id} with a body id and status changed more than the description: %v", read)
	}
}
