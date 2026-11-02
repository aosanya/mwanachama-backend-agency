package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReviewRoutes_RunningReviewsAreFilteredByTheDraftNamed(t *testing.T) {
	mgr, firstDraft := newTestManagerWithDraft(t)
	secondDraft, err := mgr.CreateDraft(t.Context(), "other", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	call := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		serve(mgr, rec, httptest.NewRequest(method, path, strings.NewReader(body)))
		return rec
	}
	for _, draftID := range []string{firstDraft, secondDraft.ID} {
		if rec := call(http.MethodPost, "/reviews", `{"draft_id":"`+draftID+`"}`); rec.Code != http.StatusCreated {
			t.Fatalf("POST /reviews for %s = %d %s", draftID, rec.Code, rec.Body.String())
		}
	}
	draftsOf := func(path string) []string {
		rec := call(http.MethodGet, path, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d %s", path, rec.Code, rec.Body.String())
		}
		var reviews []struct {
			DraftID string `json:"draft_id"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &reviews); err != nil {
			t.Fatalf("GET %s = %s, want a bare array: %v", path, rec.Body.String(), err)
		}
		out := []string{}
		for _, r := range reviews {
			out = append(out, r.DraftID)
		}
		return out
	}

	if got := draftsOf("/running-reviews"); len(got) != 2 {
		t.Fatalf("GET /running-reviews drafts = %v, want both", got)
	}
	if got := draftsOf("/running-reviews?draft_id=" + secondDraft.ID); len(got) != 1 || got[0] != secondDraft.ID {
		t.Fatalf("GET /running-reviews?draft_id=%s drafts = %v, want only that draft", secondDraft.ID, got)
	}
	if rec := call(http.MethodGet, "/running-reviews?draft_id=nope", ""); rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("GET /running-reviews?draft_id=nope = %d %s, want 200 []", rec.Code, rec.Body.String())
	}
}
