package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCreateWorkflow(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)

	req := httptest.NewRequest(http.MethodPost, "/workflows", strings.NewReader(`{"name":"Onboarding","draft_id":"`+draftID+`"}`))
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.Workflow
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID == "" || out.Name != "Onboarding" {
		t.Fatalf("unexpected workflow: %+v", out)
	}
}

func TestGetWorkflow_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	req := httptest.NewRequest(http.MethodGet, "/workflows/unknown", nil)
	req.SetPathValue("workflowID", "unknown")
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestListAndUpdateWorkflow(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	w, err := mgr.CreateWorkflow(t.Context(), models.Workflow{DraftID: draftID, Name: "Onboarding"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/workflows?draft_id="+draftID, nil)
	listRec := httptest.NewRecorder()
	serve(mgr, listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}

	updateReq := httptest.NewRequest(http.MethodPatch, "/workflows/"+w.ID, strings.NewReader(`{"name":"Onboarding v2"}`))
	updateReq.SetPathValue("workflowID", w.ID)
	updateRec := httptest.NewRecorder()
	serve(mgr, updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", updateRec.Code, updateRec.Body.String())
	}
	var updated models.Workflow
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("unmarshal updated: %v", err)
	}
	if updated.Name != "Onboarding v2" {
		t.Fatalf("unexpected updated workflow: %+v", updated)
	}
}
