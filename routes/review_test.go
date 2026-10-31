package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReviewRoutes_StartRecordAndReadOverHTTP(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	call := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		serve(mgr, rec, httptest.NewRequest(method, path, strings.NewReader(body)))
		return rec
	}

	rec := call(http.MethodPost, "/reviews", `{"draft_id":"`+draftID+`","started_by":"amina"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /reviews = %d %s", rec.Code, rec.Body.String())
	}
	var started struct {
		Review struct {
			ID string `json:"id"`
		} `json:"review"`
		Areas []struct {
			Area string `json:"area"`
		} `json:"areas"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &started); err != nil || started.Review.ID == "" || len(started.Areas) != 7 {
		t.Fatalf("started = %s, %v", rec.Body.String(), err)
	}

	if rec := call(http.MethodPost, "/reviews", `{"draft_id":"`+draftID+`"}`); rec.Code != http.StatusConflict ||
		!strings.Contains(rec.Body.String(), "A review of this draft is already running.") {
		t.Fatalf("second POST /reviews = %d %s, want 409 with the running sentence", rec.Code, rec.Body.String())
	}

	body := `{"status":"done","summary":"Fine.","tokens":12,"findings":[{"title":"Name the goal owner","then":[{"tool":"agency_set_raci","arguments":{"label":"accountable"}}]}]}`
	rec = call(http.MethodPut, "/reviews/"+started.Review.ID+"/areas/goals", body)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"review_status":"running"`) {
		t.Fatalf("PUT area = %d %s", rec.Code, rec.Body.String())
	}

	rec = call(http.MethodGet, "/review?draft_id="+draftID, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Name the goal owner") || !strings.Contains(rec.Body.String(), `"then":[{"tool":"agency_set_raci"`) {
		t.Fatalf("GET /review = %d %s", rec.Code, rec.Body.String())
	}

	rec = call(http.MethodGet, "/reviews?draft_id="+draftID, "")
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "Name the goal owner") {
		t.Fatalf("GET /reviews = %d %s, want the review without findings", rec.Code, rec.Body.String())
	}
}

func TestReviewRoutes_AreaReadsOnlyThatPartAndAnUnknownAreaIsRefused(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	call := func(body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		serve(mgr, rec, httptest.NewRequest(http.MethodPost, "/reviews", strings.NewReader(body)))
		return rec
	}

	rec := call(`{"draft_id":"` + draftID + `","area":"nonsense"}`)
	if rec.Code != http.StatusUnprocessableEntity && rec.Code != http.StatusBadRequest {
		t.Fatalf("POST /reviews with an unknown area = %d %s", rec.Code, rec.Body.String())
	}

	rec = call(`{"draft_id":"` + draftID + `","area":"goals"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /reviews with area goals = %d %s", rec.Code, rec.Body.String())
	}
	var started struct {
		Areas []struct {
			Area string `json:"area"`
		} `json:"areas"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &started); err != nil || len(started.Areas) != 1 || started.Areas[0].Area != "goals" {
		t.Fatalf("started = %s, %v", rec.Body.String(), err)
	}
}
