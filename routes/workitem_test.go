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
	mgr := newTestManager(t)
	w, err := mgr.CreateWorkflow(t.Context(), models.Workflow{Name: "Onboarding"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	handler := routes.CreateWorkItem(mgr)
	body := `{"workflow_id":"` + w.ID + `","title":"Write charter"}`
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

func TestUpdateWorkItemAndListRoleAssignmentsForWorkItem(t *testing.T) {
	mgr := newTestManager(t)
	w, _ := mgr.CreateWorkflow(t.Context(), models.Workflow{Name: "Onboarding"})
	wi, err := mgr.CreateWorkItem(t.Context(), models.WorkItem{WorkflowID: w.ID, Title: "Write charter"})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}

	updateReq := httptest.NewRequest(http.MethodPatch, "/work-items/"+wi.ID, strings.NewReader(`{"workflow_id":"`+w.ID+`","title":"Write charter v2"}`))
	updateReq.SetPathValue("workItemID", wi.ID)
	updateRec := httptest.NewRecorder()
	routes.UpdateWorkItem(mgr)(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", updateRec.Code, updateRec.Body.String())
	}

	role, err := mgr.CreateRole(t.Context(), models.Role{Name: "tech-lead"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := mgr.CreateRoleAssignment(t.Context(), models.RoleAssignment{RoleID: role.ID, WorkItemID: wi.ID, Label: models.RACILabelAccountable}); err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/work-items/"+wi.ID+"/role-assignments", nil)
	listReq.SetPathValue("workItemID", wi.ID)
	listRec := httptest.NewRecorder()
	routes.ListRoleAssignmentsForWorkItem(mgr)(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}
	var assignments []models.RoleAssignment
	if err := json.Unmarshal(listRec.Body.Bytes(), &assignments); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(assignments) != 1 || assignments[0].Label != models.RACILabelAccountable {
		t.Fatalf("unexpected assignments: %+v", assignments)
	}
}
