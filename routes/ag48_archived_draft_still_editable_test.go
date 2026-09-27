package routes_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAG48_PinsKnownDefect_ArchivedDraftRoleStillPatchable(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	role, err := mgr.CreateRole(t.Context(), models.Role{DraftID: draftID, Name: "before-archive"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := mgr.ArchiveDraft(t.Context(), draftID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/roles/"+role.ID, strings.NewReader(`{"name":"renamed-after-archive"}`))
	req.SetPathValue("roleID", role.ID)
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("AG48 should be fixed to refuse this: got status %d, body %s", rec.Code, rec.Body.String())
	}
	got, err := mgr.GetRole(t.Context(), role.ID)
	if err != nil {
		t.Fatalf("GetRole: %v", err)
	}
	if got.Name != "renamed-after-archive" {
		t.Fatalf("AG48 should be fixed to leave the row alone: got name %q", got.Name)
	}
}

func TestAG48_PinsKnownDefect_ArchivedDraftGoalStillDeletable(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	goal, err := mgr.CreateGoal(t.Context(), models.Goal{DraftID: draftID, Title: "before-archive goal"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if _, err := mgr.ArchiveDraft(t.Context(), draftID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/goals/"+goal.ID, nil)
	req.SetPathValue("goalID", goal.ID)
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("AG48 should be fixed to refuse this: got status %d, body %s", rec.Code, rec.Body.String())
	}
	if _, err := mgr.GetGoal(t.Context(), goal.ID); err == nil {
		t.Fatalf("AG48 should be fixed to leave the row in place: goal was actually deleted")
	}
}

func TestAG48_PinsKnownDefect_ArchivedDraftRoleAssignmentStillPatchable(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	goal, err := mgr.CreateGoal(t.Context(), models.Goal{DraftID: draftID, Title: "g1"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	obj, err := mgr.CreateObjective(t.Context(), models.Objective{DraftID: draftID, GoalID: goal.ID, Title: "o1"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	role, err := mgr.CreateRole(t.Context(), models.Role{DraftID: draftID, Name: "r1"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	ra, err := mgr.CreateRoleAssignment(t.Context(), models.RoleAssignment{DraftID: draftID, RoleID: role.ID, ObjectiveID: obj.ID, Label: models.RACILabelAccountable})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}
	if _, err := mgr.ArchiveDraft(t.Context(), draftID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/role-assignments/"+ra.ID, strings.NewReader(
		`{"role_id":"`+role.ID+`","objective_id":"`+obj.ID+`","label":"consulted"}`))
	req.SetPathValue("assignmentID", ra.ID)
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("AG48 should be fixed to refuse this: got status %d, body %s", rec.Code, rec.Body.String())
	}
	got, err := mgr.GetRoleAssignment(t.Context(), ra.ID)
	if err != nil {
		t.Fatalf("GetRoleAssignment: %v", err)
	}
	if got.Label != models.RACILabelConsulted {
		t.Fatalf("AG48 should be fixed to leave the row alone: got label %q", got.Label)
	}
}
