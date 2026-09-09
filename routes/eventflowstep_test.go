package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/aosanya/mwanachama-backend-agency/routes"
)

func TestLookupAndListEventFlowSteps(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.CreateEventFlowStep(t.Context(), models.EventFlowStep{
		Code: "planning:1", WorkflowCode: "planning", Step: "1",
		HandlerCode: "wp-planning-1", Consumer: "codevaldwork",
	}); err != nil {
		t.Fatalf("CreateEventFlowStep: %v", err)
	}

	lookupReq := httptest.NewRequest(http.MethodGet, "/event-flow-steps/lookup?handler_code=wp-planning-1", nil)
	lookupRec := httptest.NewRecorder()
	routes.LookupFlowStep(mgr)(lookupRec, lookupReq)
	if lookupRec.Code != http.StatusOK {
		t.Fatalf("lookup status = %d, body = %s", lookupRec.Code, lookupRec.Body.String())
	}
	var step models.EventFlowStep
	if err := json.Unmarshal(lookupRec.Body.Bytes(), &step); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if step.Code != "planning:1" {
		t.Fatalf("unexpected step: %+v", step)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/event-flow-steps?workflow_code=planning", nil)
	listRec := httptest.NewRecorder()
	routes.ListEventFlowSteps(mgr)(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}
}

func TestLookupFlowStep_InvalidLookup(t *testing.T) {
	mgr := newTestManager(t)
	req := httptest.NewRequest(http.MethodGet, "/event-flow-steps/lookup", nil)
	rec := httptest.NewRecorder()
	routes.LookupFlowStep(mgr)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
