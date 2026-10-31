package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type raciRoutesSeed struct {
	mgr     mwanachamaagency.AgencyManager
	draftID string
	step    models.WorkItem
	other   models.WorkItem
	roleA   models.Role
	roleB   models.Role
}

func seedRaciRoutes(t *testing.T) raciRoutesSeed {
	t.Helper()
	mgr, draftID := newTestManagerWithDraft(t)
	ctx := t.Context()
	w, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftID, Name: "Onboarding"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	step, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: draftID, WorkflowID: w.ID, Title: "Write charter"})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	other, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: draftID, WorkflowID: w.ID, Title: "Review charter"})
	if err != nil {
		t.Fatalf("CreateWorkItem(other): %v", err)
	}
	roleA, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "tech-lead"})
	if err != nil {
		t.Fatalf("CreateRole(A): %v", err)
	}
	roleB, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "developer"})
	if err != nil {
		t.Fatalf("CreateRole(B): %v", err)
	}
	return raciRoutesSeed{mgr: mgr, draftID: draftID, step: step, other: other, roleA: roleA, roleB: roleB}
}

func (s raciRoutesSeed) body(stepID, roleID, label string) string {
	return `{"work_item_id":"` + stepID + `","role_id":"` + roleID + `","label":"` + label + `","draft_id":"` + s.draftID + `"}`
}

func (s raciRoutesSeed) post(body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	serve(s.mgr, rec, httptest.NewRequest(http.MethodPost, "/role-assignments", strings.NewReader(body)))
	return rec
}

func TestCreateRoleAssignment(t *testing.T) {
	s := seedRaciRoutes(t)

	rec := s.post(s.body(s.step.ID, s.roleA.ID, "accountable"))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.RoleAssignment
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID == "" || out.Label != models.RACILabelAccountable || out.WorkItemID != s.step.ID || out.RoleID != s.roleA.ID {
		t.Fatalf("unexpected assignment: %+v", out)
	}
}

func TestCreateRoleAssignment_TwoAccountablesOnOneStep(t *testing.T) {
	s := seedRaciRoutes(t)

	for _, role := range []string{s.roleA.ID, s.roleB.ID} {
		if rec := s.post(s.body(s.step.ID, role, "accountable")); rec.Code != http.StatusCreated {
			t.Fatalf("Accountable for role %s: status = %d, body = %s", role, rec.Code, rec.Body.String())
		}
	}
}

func TestCreateRoleAssignment_OneRoleTwoLabelsOnOneStep(t *testing.T) {
	s := seedRaciRoutes(t)

	for _, label := range []string{"responsible", "accountable"} {
		if rec := s.post(s.body(s.step.ID, s.roleA.ID, label)); rec.Code != http.StatusCreated {
			t.Fatalf("label %s: status = %d, body = %s", label, rec.Code, rec.Body.String())
		}
	}
}

func TestCreateRoleAssignment_DuplicateRoleAndLabelIsConflict(t *testing.T) {
	s := seedRaciRoutes(t)
	if rec := s.post(s.body(s.step.ID, s.roleA.ID, "accountable")); rec.Code != http.StatusCreated {
		t.Fatalf("first: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec := s.post(s.body(s.step.ID, s.roleA.ID, "accountable"))

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec := s.post(s.body(s.other.ID, s.roleA.ID, "accountable")); rec.Code != http.StatusCreated {
		t.Fatalf("same role and label on another step: status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestCreateRoleAssignment_UnknownEndsAreNotFound(t *testing.T) {
	s := seedRaciRoutes(t)

	if rec := s.post(s.body("unknown", s.roleA.ID, "accountable")); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown step: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec := s.post(s.body(s.step.ID, "unknown", "accountable")); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown role: status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestCreateRoleAssignment_BadLabelIsBadRequest(t *testing.T) {
	s := seedRaciRoutes(t)

	if rec := s.post(s.body(s.step.ID, s.roleA.ID, "bogus")); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateRoleAssignment_WithinDraft(t *testing.T) {
	s := seedRaciRoutes(t)
	ra, err := s.mgr.CreateRoleAssignment(t.Context(), models.RoleAssignment{DraftID: s.draftID, WorkItemID: s.step.ID, RoleID: s.roleA.ID, Label: models.RACILabelResponsible})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/role-assignments/"+ra.ID, strings.NewReader(`{"role_id":"`+s.roleA.ID+`","work_item_id":"`+s.step.ID+`","label":"accountable"}`))
	rec := httptest.NewRecorder()
	serve(s.mgr, rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.RoleAssignment
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Label != models.RACILabelAccountable {
		t.Fatalf("unexpected assignment: %+v", out)
	}
}

func TestGetListAndDeleteRoleAssignments(t *testing.T) {
	s := seedRaciRoutes(t)
	ra, err := s.mgr.CreateRoleAssignment(t.Context(), models.RoleAssignment{DraftID: s.draftID, WorkItemID: s.step.ID, RoleID: s.roleA.ID, Label: models.RACILabelResponsible})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	getRec := httptest.NewRecorder()
	serve(s.mgr, getRec, httptest.NewRequest(http.MethodGet, "/role-assignments/"+ra.ID, nil))
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", getRec.Code, getRec.Body.String())
	}

	listRec := httptest.NewRecorder()
	serve(s.mgr, listRec, httptest.NewRequest(http.MethodGet, "/role-assignments?draft_id="+s.draftID, nil))
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

	delRec := httptest.NewRecorder()
	serve(s.mgr, delRec, httptest.NewRequest(http.MethodDelete, "/role-assignments/"+ra.ID, nil))
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", delRec.Code, delRec.Body.String())
	}
	gone := httptest.NewRecorder()
	serve(s.mgr, gone, httptest.NewRequest(http.MethodGet, "/role-assignments/"+ra.ID, nil))
	if gone.Code != http.StatusNotFound {
		t.Fatalf("get after delete status = %d, body = %s", gone.Code, gone.Body.String())
	}
}

func TestListWorkItemRoleAssignments(t *testing.T) {
	s := seedRaciRoutes(t)
	ctx := t.Context()
	for _, role := range []string{s.roleA.ID, s.roleB.ID} {
		if _, err := s.mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: s.draftID, WorkItemID: s.step.ID, RoleID: role, Label: models.RACILabelAccountable}); err != nil {
			t.Fatalf("CreateRoleAssignment: %v", err)
		}
	}
	if _, err := s.mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: s.draftID, WorkItemID: s.other.ID, RoleID: s.roleA.ID, Label: models.RACILabelInformed}); err != nil {
		t.Fatalf("CreateRoleAssignment(other): %v", err)
	}

	rec := httptest.NewRecorder()
	serve(s.mgr, rec, httptest.NewRequest(http.MethodGet, "/work-items/"+s.step.ID+"/role-assignments?draft_id="+s.draftID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out []models.RoleAssignment
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected the step's 2 assignments, got %d: %+v", len(out), out)
	}
}
