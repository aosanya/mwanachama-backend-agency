package routes_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func readsNoPart(context.Context, string) []models.ReviewAreaName { return nil }

func TestReviewRoutes_AStartWhenNoPartIsReadAnswers409AndLeavesNothingRunning(t *testing.T) {
	mgr := newTestManager(t, mwanachamaagency.WithReviewAreas(readsNoPart))
	if _, err := mgr.CreateAgency(t.Context(), models.Agency{Name: "SWELab"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draft, err := mgr.CreateDraft(t.Context(), "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	call := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		serve(mgr, rec, httptest.NewRequest(method, path, strings.NewReader(body)))
		return rec
	}

	rec := call(http.MethodPost, "/reviews", `{"draft_id":"`+draft.ID+`"}`)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "There is no part of this draft to review.") {
		t.Fatalf("POST /reviews with no part to read = %d %s, want 409 with the no-part sentence", rec.Code, rec.Body.String())
	}

	rec = call(http.MethodGet, "/running-reviews", "")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("GET /running-reviews after the refused start = %d %s, want []", rec.Code, rec.Body.String())
	}
}
