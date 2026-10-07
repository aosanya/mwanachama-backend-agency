package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTheServedAgencyIsTheFirstCreatedAndNoBodyReachesItsServerOwnedSettings(t *testing.T) {
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
	if code != http.StatusOK || served["name"] != "SWELab" {
		t.Fatalf("precondition: GET /agency should serve the seeded SWELab agency, got %d %v", code, served)
	}
	servedID, servedCreatedAt := served["id"], served["created_at"]

	_, late := call(http.MethodPost, `{"name":"Takeover","id":"forged","created_at":"1999-01-01T00:00:00Z","enabled":true,"deleted":true}`)
	if late["id"] == "forged" || late["created_at"] == "1999-01-01T00:00:00Z" {
		t.Errorf("a second create kept the body's id or clock: %v", late)
	}

	code, after := call(http.MethodGet, "")
	if code != http.StatusOK || after["id"] != servedID || after["name"] != "SWELab" {
		t.Errorf("a later create displaced the served agency: GET /agency now serves %v, want the first agency %v", after, servedID)
	}

	code, patched := call(http.MethodPatch, `{"name":"Renamed","enabled":true,"deleted":true,"inactivity_timeout_seconds":5,"default_failure_pipeline_budget":9,"event_flows":"[1]","id":"forged","created_at":"1999-01-01T00:00:00Z"}`)
	if code != http.StatusOK {
		t.Fatalf("precondition: PATCH /agency answered %d %v", code, patched)
	}
	_, reread := call(http.MethodGet, "")
	if reread["name"] != "Renamed" {
		t.Fatalf("precondition: the rename should have landed, got %v", reread)
	}
	if reread["id"] != servedID || reread["created_at"] != servedCreatedAt {
		t.Errorf("a rename moved the agency's id or clock: %v", reread)
	}
	if reread["enabled"] != false || reread["deleted"] != false ||
		reread["inactivity_timeout_seconds"] != float64(0) || reread["default_failure_pipeline_budget"] != float64(0) ||
		reread["event_flows"] != nil {
		t.Errorf("a rename wrote a setting the body named, beyond name and description: %v", reread)
	}
}
