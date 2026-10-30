package routes_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAG50_IssueScanRefusesADraftThatDoesNotExist(t *testing.T) {
	mgr, _ := newTestManagerWithDraft(t)

	get := func(path string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		serve(mgr, w, httptest.NewRequest(http.MethodGet, path, nil))
		return w
	}

	if w := get("/drafts/no-such-draft"); w.Code != http.StatusNotFound {
		t.Fatalf("precondition: GET /drafts/no-such-draft -> %d %s, want 404", w.Code, w.Body.String())
	}

	w := get("/drafts/no-such-draft/issues")
	if w.Code != http.StatusNotFound {
		t.Fatalf("AG50 (P2): GET /drafts/no-such-draft/issues answered %d %s — the scan wrote issue rows under a draft that does not exist; want 404 draft not found, as GET /drafts/{id} and POST /goals already answer for the same id", w.Code, w.Body.String())
	}
}
