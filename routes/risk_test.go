package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func postJSON(t *testing.T, mgrServe func(http.ResponseWriter, *http.Request), path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	mgrServe(rec, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
	return rec
}

func TestRiskRoutes_CreateListAndDeleteWithItsPathways(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	handle := func(w http.ResponseWriter, r *http.Request) { serve(mgr, w, r) }

	created := postJSON(t, handle, "/risks", `{"title":"Unsold harvest","top_event":"A harvest goes unsold","draft_id":"`+draftID+`"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create risk = %d %s", created.Code, created.Body.String())
	}
	var risk models.Risk
	if err := json.Unmarshal(created.Body.Bytes(), &risk); err != nil || risk.ID == "" || risk.Code != "RK-1" {
		t.Fatalf("risk = %+v, %v", risk, err)
	}

	arm := postJSON(t, handle, "/risk-pathways", `{"risk_id":"`+risk.ID+`","side":"threat","title":"No buyer is listed","draft_id":"`+draftID+`"}`)
	if arm.Code != http.StatusCreated {
		t.Fatalf("create pathway = %d %s", arm.Code, arm.Body.String())
	}
	var pathway models.RiskPathway
	if err := json.Unmarshal(arm.Body.Bytes(), &pathway); err != nil || pathway.Code != "RP-1" || pathway.RiskID != risk.ID {
		t.Fatalf("pathway = %+v, %v", pathway, err)
	}

	listRec := httptest.NewRecorder()
	handle(listRec, httptest.NewRequest(http.MethodGet, "/risk-pathways?draft_id="+draftID+"&risk_id="+risk.ID+"&side=threat", nil))
	var listed []models.RiskPathway
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil || len(listed) != 1 || listed[0].ID != pathway.ID {
		t.Fatalf("list = %s, %v", listRec.Body.String(), err)
	}

	del := httptest.NewRecorder()
	handle(del, httptest.NewRequest(http.MethodDelete, "/risks/"+risk.ID, nil))
	if del.Code != http.StatusNoContent {
		t.Fatalf("delete risk = %d %s", del.Code, del.Body.String())
	}
	gone := httptest.NewRecorder()
	handle(gone, httptest.NewRequest(http.MethodGet, "/risk-pathways/"+pathway.ID, nil))
	if gone.Code != http.StatusNotFound {
		t.Fatalf("pathway after the risk was deleted = %d, want 404", gone.Code)
	}
}

func TestRiskRoutes_RefuseWhatTheManagerRefuses(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	handle := func(w http.ResponseWriter, r *http.Request) { serve(mgr, w, r) }

	if rec := postJSON(t, handle, "/risks", `{"title":"","top_event":"x","draft_id":"`+draftID+`"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("blank title = %d %s, want 400", rec.Code, rec.Body.String())
	}
	if rec := postJSON(t, handle, "/risk-pathways", `{"risk_id":"nope","side":"threat","title":"x","draft_id":"`+draftID+`"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown risk = %d %s, want 404", rec.Code, rec.Body.String())
	}
	rec := httptest.NewRecorder()
	handle(rec, httptest.NewRequest(http.MethodGet, "/risks/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown risk read = %d, want 404", rec.Code)
	}
}
