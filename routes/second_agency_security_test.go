package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAG73_OpenHole_ASecondCreateAgencyStoresAnOrganizationNothingCanReadOrDelete(t *testing.T) {
	mgr, _ := newTestManagerWithDraft(t)

	call := func(method, body string) (int, map[string]any) {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/agency", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		serve(mgr, w, r)
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}

	code, served := call(http.MethodGet, "")
	if code != http.StatusOK || served["id"] == nil {
		t.Fatalf("precondition: GET /agency should serve the one seeded agency, got %d %v", code, served)
	}

	for i := 0; i < 3; i++ {
		code, out := call(http.MethodPost, `{"name":"Another organization"}`)
		if code == http.StatusCreated {
			t.Fatalf("AG73 (P2) open hole: POST /agency on an instance that already has its one agency answered 201 and stored another organization (id %v) beside the served one (id %v) — no route lists or deletes it and GET /agency never serves it; the blueprint declares exactly one agency per mounted instance, so a second create should be refused (409), not stored", out["id"], served["id"])
		}
	}

	code, after := call(http.MethodGet, "")
	if code != http.StatusOK || after["id"] != served["id"] {
		t.Fatalf("precondition: GET /agency should still serve %v, got %d %v", served["id"], code, after)
	}
}
