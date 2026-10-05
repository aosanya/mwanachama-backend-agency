package routes_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAG67_OpenHole_PublicationListingCarriesEveryWholeDocument(t *testing.T) {
	mgr := newTestManager(t)
	draftID := seedPublishableAgency(t, mgr)
	call := func(method, path, body string) (int, string) {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		serve(mgr, w, r)
		return w.Code, w.Body.String()
	}

	const publications = 12
	text := strings.Repeat("v", 50_000)
	for i := 1; i <= publications; i++ {
		if code, got := call(http.MethodPost, "/values", fmt.Sprintf(`{"text":"%s-%d","draft_id":"%s"}`, text, i, draftID)); code != http.StatusCreated {
			t.Fatalf("precondition: value %d answered %d %.200s", i, code, got)
		}
		if code, got := call(http.MethodPost, "/publications", fmt.Sprintf(`{"draft_id":"%s","tag":"t%d"}`, draftID, i)); code != http.StatusCreated {
			t.Fatalf("precondition: publish %d answered %d %.200s", i, code, got)
		}
	}

	code, newest := call(http.MethodGet, fmt.Sprintf("/publications/%d", publications), "")
	if code != http.StatusOK || !strings.Contains(newest, `"document"`) {
		t.Fatalf("precondition: GET /publications/%d answered %d %.200s", publications, code, newest)
	}

	code, listing := call(http.MethodGet, "/publications?limit=1", "")
	if code != http.StatusOK {
		t.Fatalf("precondition: GET /publications answered %d %.200s", code, listing)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(listing), &rows); err != nil || len(rows) != publications {
		t.Fatalf("precondition: expected %d listed publications, got %d (%v)", publications, len(rows), err)
	}

	if len(listing) >= len(newest) {
		t.Errorf("AG67 (P2): GET /publications?limit=1 over %d publications answered %d bytes, more than the newest whole publication (%d bytes); "+
			"a listing should name each publication without carrying its document (get_publication is the document read), so it stays smaller than one document read",
			publications, len(listing), len(newest))
	}
}
