package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/aosanya/mwanachama-backend-agency/routes"
)

func TestCreateGoal(t *testing.T) {
	mgr := newTestManager(t)
	handler := routes.CreateGoal(mgr)

	req := httptest.NewRequest(http.MethodPost, "/goals", strings.NewReader(`{"title":"Grow adoption"}`))
	rec := httptest.NewRecorder()
	handler(rec, req)

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

func TestGetGoal_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	handler := routes.GetGoal(mgr)

	req := httptest.NewRequest(http.MethodGet, "/goals/unknown", nil)
	req.SetPathValue("goalID", "unknown")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestListAndUpdateGoal(t *testing.T) {
	mgr := newTestManager(t)
	g, err := mgr.CreateGoal(t.Context(), models.Goal{Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/goals", nil)
	listRec := httptest.NewRecorder()
	routes.ListGoals(mgr)(listRec, listReq)
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
	routes.UpdateGoal(mgr)(updateRec, updateReq)
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
