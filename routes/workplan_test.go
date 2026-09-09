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

func TestCreateWorkPlan(t *testing.T) {
	mgr := newTestManager(t)
	body := `{"code":"wp-1","trigger_topic":"agency\\.published","enabled":true}`
	req := httptest.NewRequest(http.MethodPost, "/work-plans", strings.NewReader(body))
	rec := httptest.NewRecorder()
	routes.CreateWorkPlan(mgr)(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.WorkPlan
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID == "" || out.Code != "wp-1" {
		t.Fatalf("unexpected work plan: %+v", out)
	}
}

func TestCreateWorkPlan_InvalidRegex(t *testing.T) {
	mgr := newTestManager(t)
	body := `{"code":"wp-1","trigger_topic":"("}`
	req := httptest.NewRequest(http.MethodPost, "/work-plans", strings.NewReader(body))
	rec := httptest.NewRecorder()
	routes.CreateWorkPlan(mgr)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetListUpdateWorkPlan(t *testing.T) {
	mgr := newTestManager(t)
	wp, err := mgr.CreateWorkPlan(t.Context(), models.WorkPlan{Code: "wp-1", TriggerTopic: "agency.published", Enabled: true})
	if err != nil {
		t.Fatalf("CreateWorkPlan: %v", err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/work-plans/"+wp.ID, nil)
	getReq.SetPathValue("workPlanID", wp.ID)
	getRec := httptest.NewRecorder()
	routes.GetWorkPlan(mgr)(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", getRec.Code, getRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/work-plans", nil)
	listRec := httptest.NewRecorder()
	routes.ListWorkPlans(mgr)(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}

	updateReq := httptest.NewRequest(http.MethodPatch, "/work-plans/"+wp.ID, strings.NewReader(`{"code":"wp-1","trigger_topic":"agency.published","enabled":false}`))
	updateReq.SetPathValue("workPlanID", wp.ID)
	updateRec := httptest.NewRecorder()
	routes.UpdateWorkPlan(mgr)(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", updateRec.Code, updateRec.Body.String())
	}
}

func TestMatchWorkPlans(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.CreateWorkPlan(t.Context(), models.WorkPlan{Code: "wp-1", TriggerTopic: "agency\\.published", Enabled: true}); err != nil {
		t.Fatalf("CreateWorkPlan: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/work-plans/match", strings.NewReader(`{"topic":"agency.published","payload":"{}"}`))
	rec := httptest.NewRecorder()
	routes.MatchWorkPlans(mgr)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var matches []models.WorkPlanMatch
	if err := json.Unmarshal(rec.Body.Bytes(), &matches); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
}
