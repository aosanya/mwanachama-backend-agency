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

func TestCreateRoleAssignment(t *testing.T) {
	mgr := newTestManager(t)
	w, _ := mgr.CreateWorkflow(t.Context(), models.Workflow{Name: "Onboarding"})
	wi, _ := mgr.CreateWorkItem(t.Context(), models.WorkItem{WorkflowID: w.ID, Title: "Write charter"})
	role, err := mgr.CreateRole(t.Context(), models.Role{Name: "tech-lead"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	body := `{"role_id":"` + role.ID + `","work_item_id":"` + wi.ID + `","label":"accountable"}`
	req := httptest.NewRequest(http.MethodPost, "/role-assignments", strings.NewReader(body))
	rec := httptest.NewRecorder()
	routes.CreateRoleAssignment(mgr)(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.RoleAssignment
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID == "" || out.Label != models.RACILabelAccountable {
		t.Fatalf("unexpected assignment: %+v", out)
	}
}

func TestCreateRoleAssignment_DuplicateAccountable(t *testing.T) {
	mgr := newTestManager(t)
	w, _ := mgr.CreateWorkflow(t.Context(), models.Workflow{Name: "Onboarding"})
	wi, _ := mgr.CreateWorkItem(t.Context(), models.WorkItem{WorkflowID: w.ID, Title: "Write charter"})
	roleA, _ := mgr.CreateRole(t.Context(), models.Role{Name: "tech-lead"})
	roleB, _ := mgr.CreateRole(t.Context(), models.Role{Name: "developer"})

	if _, err := mgr.CreateRoleAssignment(t.Context(), models.RoleAssignment{RoleID: roleA.ID, WorkItemID: wi.ID, Label: models.RACILabelAccountable}); err != nil {
		t.Fatalf("CreateRoleAssignment(A): %v", err)
	}

	body := `{"role_id":"` + roleB.ID + `","work_item_id":"` + wi.ID + `","label":"accountable"}`
	req := httptest.NewRequest(http.MethodPost, "/role-assignments", strings.NewReader(body))
	rec := httptest.NewRecorder()
	routes.CreateRoleAssignment(mgr)(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateRoleAssignment_LiveRefused(t *testing.T) {
	mgr := newTestManager(t)
	w, _ := mgr.CreateWorkflow(t.Context(), models.Workflow{Name: "Onboarding"})
	wi, _ := mgr.CreateWorkItem(t.Context(), models.WorkItem{WorkflowID: w.ID, Title: "Write charter"})
	role, _ := mgr.CreateRole(t.Context(), models.Role{Name: "tech-lead"})
	ra, err := mgr.CreateRoleAssignment(t.Context(), models.RoleAssignment{RoleID: role.ID, WorkItemID: wi.ID, Label: models.RACILabelResponsible})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/role-assignments/"+ra.ID, strings.NewReader(`{"role_id":"`+role.ID+`","work_item_id":"`+wi.ID+`","label":"accountable"}`))
	req.SetPathValue("assignmentID", ra.ID)
	rec := httptest.NewRecorder()
	routes.UpdateRoleAssignment(mgr)(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetAndListRoleAssignments(t *testing.T) {
	mgr := newTestManager(t)
	w, _ := mgr.CreateWorkflow(t.Context(), models.Workflow{Name: "Onboarding"})
	wi, _ := mgr.CreateWorkItem(t.Context(), models.WorkItem{WorkflowID: w.ID, Title: "Write charter"})
	role, _ := mgr.CreateRole(t.Context(), models.Role{Name: "tech-lead"})
	ra, err := mgr.CreateRoleAssignment(t.Context(), models.RoleAssignment{RoleID: role.ID, WorkItemID: wi.ID, Label: models.RACILabelResponsible})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/role-assignments/"+ra.ID, nil)
	getReq.SetPathValue("assignmentID", ra.ID)
	getRec := httptest.NewRecorder()
	routes.GetRoleAssignment(mgr)(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", getRec.Code, getRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/role-assignments", nil)
	listRec := httptest.NewRecorder()
	routes.ListRoleAssignments(mgr)(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}
	var out []models.RoleAssignment
	if err := json.Unmarshal(listRec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 assignment, got %d", len(out))
	}
}
