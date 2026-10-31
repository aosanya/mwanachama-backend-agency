package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestConsequenceRoutes_BelongToOneHazardAndListByIt(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	handle := func(w http.ResponseWriter, r *http.Request) { serve(mgr, w, r) }

	var hazards []models.Hazard
	for _, title := range []string{"Fire", "Flood"} {
		rec := postJSON(t, handle, "/hazards", `{"title":"`+title+`","draft_id":"`+draftID+`"}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create hazard %s = %d %s", title, rec.Code, rec.Body.String())
		}
		var h models.Hazard
		if err := json.Unmarshal(rec.Body.Bytes(), &h); err != nil || h.Code == "" {
			t.Fatalf("hazard = %+v, %v", h, err)
		}
		hazards = append(hazards, h)
	}

	if rec := postJSON(t, handle, "/consequences", `{"title":"Produce spoils","draft_id":"`+draftID+`"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("consequence without hazard_code = %d %s, want 400", rec.Code, rec.Body.String())
	}
	if rec := postJSON(t, handle, "/consequences", `{"title":"Produce spoils","hazard_code":"HZ-999","draft_id":"`+draftID+`"}`); rec.Code != http.StatusNotFound && rec.Code != http.StatusBadRequest {
		t.Fatalf("consequence on an unknown hazard = %d %s, want 404 or 400", rec.Code, rec.Body.String())
	}

	created := postJSON(t, handle, "/consequences", `{"title":"Produce spoils","hazard_code":"`+hazards[0].Code+`","draft_id":"`+draftID+`"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create consequence = %d %s", created.Code, created.Body.String())
	}
	var made models.Consequence
	if err := json.Unmarshal(created.Body.Bytes(), &made); err != nil || made.HazardCode != hazards[0].Code {
		t.Fatalf("consequence = %+v, %v", made, err)
	}
	if rec := postJSON(t, handle, "/consequences", `{"title":"Barn lost","hazard_code":"`+hazards[1].Code+`","draft_id":"`+draftID+`"}`); rec.Code != http.StatusCreated {
		t.Fatalf("second consequence = %d %s", rec.Code, rec.Body.String())
	}

	for hazard, want := range map[string]int{hazards[0].Code: 1, hazards[1].Code: 1, "": 2} {
		path := "/consequences?draft_id=" + draftID
		if hazard != "" {
			path += "&hazard_code=" + hazard
		}
		rec := httptest.NewRecorder()
		handle(rec, httptest.NewRequest(http.MethodGet, path, nil))
		var listed []models.Consequence
		if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil || len(listed) != want {
			t.Fatalf("list %q = %s, %v, want %d", hazard, rec.Body.String(), err, want)
		}
		for _, c := range listed {
			if hazard != "" && c.HazardCode != hazard {
				t.Fatalf("list %q returned %+v", hazard, c)
			}
		}
	}
}
