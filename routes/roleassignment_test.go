package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCreateRoleAssignment(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	g, _ := mgr.CreateGoal(t.Context(), models.Goal{DraftID: draftID, Title: "Grow adoption"})
	o, _ := mgr.CreateObjective(t.Context(), models.Objective{DraftID: draftID, GoalID: g.ID, Title: "Grow adoption"})
	role, err := mgr.CreateRole(t.Context(), models.Role{DraftID: draftID, Name: "tech-lead"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	body := `{"role_id":"` + role.ID + `","objective_id":"` + o.ID + `","label":"accountable","draft_id":"` + draftID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/role-assignments", strings.NewReader(body))
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)

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
	mgr, draftID := newTestManagerWithDraft(t)
	g, _ := mgr.CreateGoal(t.Context(), models.Goal{DraftID: draftID, Title: "Grow adoption"})
	o, _ := mgr.CreateObjective(t.Context(), models.Objective{DraftID: draftID, GoalID: g.ID, Title: "Grow adoption"})
	roleA, _ := mgr.CreateRole(t.Context(), models.Role{DraftID: draftID, Name: "tech-lead"})
	roleB, _ := mgr.CreateRole(t.Context(), models.Role{DraftID: draftID, Name: "developer"})

	if _, err := mgr.CreateRoleAssignment(t.Context(), models.RoleAssignment{DraftID: draftID, RoleID: roleA.ID, ObjectiveID: o.ID, Label: models.RACILabelAccountable}); err != nil {
		t.Fatalf("CreateRoleAssignment(A): %v", err)
	}

	body := `{"role_id":"` + roleB.ID + `","objective_id":"` + o.ID + `","label":"accountable","draft_id":"` + draftID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/role-assignments", strings.NewReader(body))
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateRoleAssignment_WithinDraft(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	g, _ := mgr.CreateGoal(t.Context(), models.Goal{DraftID: draftID, Title: "Grow adoption"})
	o, _ := mgr.CreateObjective(t.Context(), models.Objective{DraftID: draftID, GoalID: g.ID, Title: "Grow adoption"})
	role, _ := mgr.CreateRole(t.Context(), models.Role{DraftID: draftID, Name: "tech-lead"})
	ra, err := mgr.CreateRoleAssignment(t.Context(), models.RoleAssignment{DraftID: draftID, RoleID: role.ID, ObjectiveID: o.ID, Label: models.RACILabelResponsible})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/role-assignments/"+ra.ID, strings.NewReader(`{"role_id":"`+role.ID+`","objective_id":"`+o.ID+`","label":"accountable"}`))
	req.SetPathValue("assignmentID", ra.ID)
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetAndListRoleAssignments(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	g, _ := mgr.CreateGoal(t.Context(), models.Goal{DraftID: draftID, Title: "Grow adoption"})
	o, _ := mgr.CreateObjective(t.Context(), models.Objective{DraftID: draftID, GoalID: g.ID, Title: "Grow adoption"})
	role, _ := mgr.CreateRole(t.Context(), models.Role{DraftID: draftID, Name: "tech-lead"})
	ra, err := mgr.CreateRoleAssignment(t.Context(), models.RoleAssignment{DraftID: draftID, RoleID: role.ID, ObjectiveID: o.ID, Label: models.RACILabelResponsible})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/role-assignments/"+ra.ID, nil)
	getReq.SetPathValue("assignmentID", ra.ID)
	getRec := httptest.NewRecorder()
	serve(mgr, getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", getRec.Code, getRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/role-assignments?draft_id="+draftID, nil)
	listRec := httptest.NewRecorder()
	serve(mgr, listRec, listReq)
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
