package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCreateGoal(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)

	req := httptest.NewRequest(http.MethodPost, "/goals", strings.NewReader(`{"title":"Grow adoption","draft_id":"`+draftID+`"}`))
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.Goal
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID == "" || out.Title != "Grow adoption" {
		t.Fatalf("unexpected goal: %+v", out)
	}
}

// TestCreateGoal_IgnoresForgedFieldsOverHTTP is WK17's HTTP-level
// regression: a caller-supplied id/created_at/draft_id used to be written
// verbatim by this exact handler.
func TestCreateGoal_IgnoresForgedFieldsOverHTTP(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)

	body := `{"id":"forged-goal-id","title":"Probe","created_at":"1999-01-01T00:00:00Z","draft_id":"unknown-draft"}`
	req := httptest.NewRequest(http.MethodPost, "/goals", strings.NewReader(body))
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)

	// An unknown draft_id is refused outright — see TestCreateGoal_UnknownDraftIDIs404.
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	body = `{"id":"forged-goal-id","title":"Probe","created_at":"1999-01-01T00:00:00Z","draft_id":"` + draftID + `"}`
	req = httptest.NewRequest(http.MethodPost, "/goals", strings.NewReader(body))
	rec = httptest.NewRecorder()
	serve(mgr, rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.Goal
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID == "forged-goal-id" {
		t.Fatal("the response honoured a caller-supplied id")
	}
	if out.CreatedAt == "1999-01-01T00:00:00Z" {
		t.Fatal("the response honoured a caller-supplied created_at")
	}
}

// TestCreateGoal_UnknownDraftIDIs404 confirms a forged/nonexistent
// draft_id is refused with a clean 404, not the 500 "internal error" it
// would have fallen through to before goalStatusFor learned about
// ErrDraftNotFound.
func TestCreateGoal_UnknownDraftIDIs404(t *testing.T) {
	mgr := newTestManager(t)

	req := httptest.NewRequest(http.MethodPost, "/goals", strings.NewReader(`{"title":"Probe","draft_id":"unknown-draft"}`))
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetGoal_NotFound(t *testing.T) {
	mgr := newTestManager(t)

	req := httptest.NewRequest(http.MethodGet, "/goals/unknown", nil)
	req.SetPathValue("goalID", "unknown")
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestListAndUpdateGoal(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	g, err := mgr.CreateGoal(t.Context(), models.Goal{DraftID: draftID, Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/goals?draft_id="+draftID, nil)
	listRec := httptest.NewRecorder()
	serve(mgr, listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}
	var list []models.Goal
	if err := json.Unmarshal(listRec.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(list))
	}

	updateReq := httptest.NewRequest(http.MethodPatch, "/goals/"+g.ID, strings.NewReader(`{"title":"Grow adoption worldwide"}`))
	updateReq.SetPathValue("goalID", g.ID)
	updateRec := httptest.NewRecorder()
	serve(mgr, updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", updateRec.Code, updateRec.Body.String())
	}
	var updated models.Goal
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("unmarshal updated: %v", err)
	}
	if updated.Title != "Grow adoption worldwide" {
		t.Fatalf("unexpected updated goal: %+v", updated)
	}
}
