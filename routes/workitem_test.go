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

func TestCreateWorkItem(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	w, err := mgr.CreateWorkflow(t.Context(), models.Workflow{DraftID: draftID, Name: "Onboarding"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	g, _ := mgr.CreateGoal(t.Context(), models.Goal{DraftID: draftID, Title: "Grow adoption"})
	o, _ := mgr.CreateObjective(t.Context(), models.Objective{DraftID: draftID, GoalID: g.ID, Title: "Grow adoption"})

	handler := routes.CreateWorkItem(mgr)
	body := `{"workflow_id":"` + w.ID + `","objective_id":"` + o.ID + `","title":"Write charter","draft_id":"` + draftID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/work-items", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.WorkItem
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID == "" || out.Title != "Write charter" {
		t.Fatalf("unexpected work item: %+v", out)
	}
}

func TestCreateWorkItem_RequiresObjective(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	w, _ := mgr.CreateWorkflow(t.Context(), models.Workflow{DraftID: draftID, Name: "Onboarding"})

	handler := routes.CreateWorkItem(mgr)
	body := `{"workflow_id":"` + w.ID + `","title":"Write charter","draft_id":"` + draftID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/work-items", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetWorkItem_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	req := httptest.NewRequest(http.MethodGet, "/work-items/unknown", nil)
	req.SetPathValue("workItemID", "unknown")
	rec := httptest.NewRecorder()
	routes.GetWorkItem(mgr)(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateWorkItem(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	w, _ := mgr.CreateWorkflow(t.Context(), models.Workflow{DraftID: draftID, Name: "Onboarding"})
	g, _ := mgr.CreateGoal(t.Context(), models.Goal{DraftID: draftID, Title: "Grow adoption"})
	o, _ := mgr.CreateObjective(t.Context(), models.Objective{DraftID: draftID, GoalID: g.ID, Title: "Grow adoption"})
	wi, err := mgr.CreateWorkItem(t.Context(), models.WorkItem{DraftID: draftID, WorkflowID: w.ID, ObjectiveID: o.ID, Title: "Write charter"})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}

	updateReq := httptest.NewRequest(http.MethodPatch, "/work-items/"+wi.ID, strings.NewReader(`{"workflow_id":"`+w.ID+`","objective_id":"`+o.ID+`","title":"Write charter v2"}`))
	updateReq.SetPathValue("workItemID", wi.ID)
	updateRec := httptest.NewRecorder()
	routes.UpdateWorkItem(mgr)(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", updateRec.Code, updateRec.Body.String())
	}
}
