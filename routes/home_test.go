package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestHomeRoutes_CreateNestedThenListTheDraft(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	call := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		serve(mgr, rec, httptest.NewRequest(method, path, strings.NewReader(body)))
		return rec
	}

	rec := call(http.MethodPost, "/home-entries", `{"draft_id":"`+draftID+`","key":"registers","title":"Registers"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /home-entries = %d %s", rec.Code, rec.Body.String())
	}
	var section models.HomeEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &section); err != nil || section.ID == "" {
		t.Fatalf("section = %+v, %v", section, err)
	}
	rec = call(http.MethodPost, "/home-entries", `{"draft_id":"`+draftID+`","parent_id":"`+section.ID+`","key":"farms","title":"Farms","target_type":"record","target":"RC-1"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /home-entries inside = %d %s", rec.Code, rec.Body.String())
	}

	rec = call(http.MethodGet, "/home-entries?draft_id="+draftID, "")
	var listed []models.HomeEntry
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &listed) != nil || len(listed) != 2 {
		t.Fatalf("GET /home-entries = %d %s, want both rows", rec.Code, rec.Body.String())
	}
}

func TestHomeRoutes_Refusals(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	cases := []struct {
		name   string
		body   string
		status int
	}{
		{"no title", `{"draft_id":"` + draftID + `","key":"a"}`, http.StatusBadRequest},
		{"unknown target type", `{"draft_id":"` + draftID + `","key":"a","title":"A","target_type":"page","target":"x"}`, http.StatusBadRequest},
		{"unknown parent", `{"draft_id":"` + draftID + `","parent_id":"nope","key":"a","title":"A"}`, http.StatusNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			serve(mgr, rec, httptest.NewRequest(http.MethodPost, "/home-entries", strings.NewReader(c.body)))
			if rec.Code != c.status {
				t.Fatalf("POST /home-entries = %d %s, want %d", rec.Code, rec.Body.String(), c.status)
			}
		})
	}
}
