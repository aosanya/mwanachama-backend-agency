package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCreateObjective(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	g, err := mgr.CreateGoal(t.Context(), models.Goal{DraftID: draftID, Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}

	body := `{"goal_id":"` + g.ID + `","title":"Grow adoption","draft_id":"` + draftID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/objectives", strings.NewReader(body))
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.Objective
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID == "" || out.GoalID != g.ID {
		t.Fatalf("unexpected objective: %+v", out)
	}
}

func TestCreateObjective_MultiplePerGoal(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	g, _ := mgr.CreateGoal(t.Context(), models.Goal{DraftID: draftID, Title: "Grow adoption"})
	if _, err := mgr.CreateObjective(t.Context(), models.Objective{DraftID: draftID, GoalID: g.ID, Title: "First"}); err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}

	body := `{"goal_id":"` + g.ID + `","title":"Second","draft_id":"` + draftID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/objectives", strings.NewReader(body))
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/objectives?draft_id="+draftID, nil)
	listRec := httptest.NewRecorder()
	serve(mgr, listRec, listReq)
	var out []models.Objective
	if err := json.Unmarshal(listRec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 objectives for goal %q, got %d: %+v", g.ID, len(out), out)
	}
}

func TestListRoleAssignmentsForObjective(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	g, _ := mgr.CreateGoal(t.Context(), models.Goal{DraftID: draftID, Title: "Grow adoption"})
	o, _ := mgr.CreateObjective(t.Context(), models.Objective{DraftID: draftID, GoalID: g.ID, Title: "Grow adoption"})
	role, err := mgr.CreateRole(t.Context(), models.Role{DraftID: draftID, Name: "tech-lead"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := mgr.CreateRoleAssignment(t.Context(), models.RoleAssignment{DraftID: draftID, RoleID: role.ID, ObjectiveID: o.ID, Label: models.RACILabelAccountable}); err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/objectives/"+o.ID+"/role-assignments?draft_id="+draftID, nil)
	listReq.SetPathValue("objectiveID", o.ID)
	listRec := httptest.NewRecorder()
	serve(mgr, listRec, listReq)
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
