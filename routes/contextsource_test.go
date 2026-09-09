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

func TestAddListRemoveContextSource(t *testing.T) {
	mgr := newTestManager(t)
	wp, err := mgr.CreateWorkPlan(t.Context(), models.WorkPlan{Code: "wp-1", TriggerTopic: "agency.published"})
	if err != nil {
		t.Fatalf("CreateWorkPlan: %v", err)
	}

	addReq := httptest.NewRequest(http.MethodPost, "/work-plans/"+wp.ID+"/context-sources", strings.NewReader(`{"source_type":"git"}`))
	addReq.SetPathValue("workPlanID", wp.ID)
	addRec := httptest.NewRecorder()
	routes.AddContextSource(mgr)(addRec, addReq)
	if addRec.Code != http.StatusCreated {
		t.Fatalf("add status = %d, body = %s", addRec.Code, addRec.Body.String())
	}
	var cs models.ContextSource
	if err := json.Unmarshal(addRec.Body.Bytes(), &cs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/work-plans/"+wp.ID+"/context-sources", nil)
	listReq.SetPathValue("workPlanID", wp.ID)
	listRec := httptest.NewRecorder()
	routes.ListContextSources(mgr)(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}
	var sources []models.ContextSource
	if err := json.Unmarshal(listRec.Body.Bytes(), &sources); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 context source, got %d", len(sources))
	}

	removeReq := httptest.NewRequest(http.MethodDelete, "/work-plans/"+wp.ID+"/context-sources/"+cs.ID, nil)
	removeReq.SetPathValue("workPlanID", wp.ID)
	removeReq.SetPathValue("sourceID", cs.ID)
	removeRec := httptest.NewRecorder()
	routes.RemoveContextSource(mgr)(removeRec, removeReq)
	if removeRec.Code != http.StatusNoContent {
		t.Fatalf("remove status = %d, body = %s", removeRec.Code, removeRec.Body.String())
	}
}

func TestAddContextSource_WorkPlanNotFound(t *testing.T) {
	mgr := newTestManager(t)
	req := httptest.NewRequest(http.MethodPost, "/work-plans/unknown/context-sources", strings.NewReader(`{"source_type":"git"}`))
	req.SetPathValue("workPlanID", "unknown")
	rec := httptest.NewRecorder()
	routes.AddContextSource(mgr)(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
