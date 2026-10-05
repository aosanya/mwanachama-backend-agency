package routes_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestRACIUpdateNeverSeatsASecondAccountable(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	ctx := t.Context()
	g, _ := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, Title: "G"})
	held, _ := mgr.CreateObjective(ctx, models.Objective{DraftID: draftID, GoalID: g.ID, Title: "held"})
	other, _ := mgr.CreateObjective(ctx, models.Objective{DraftID: draftID, GoalID: g.ID, Title: "other"})
	owner, _ := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "owner"})
	rival, _ := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "rival"})
	if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: draftID, RoleID: owner.ID, ObjectiveID: held.ID, Label: models.RACILabelAccountable}); err != nil {
		t.Fatalf("precondition: seating the first Accountable: %v", err)
	}
	consulted, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: draftID, RoleID: rival.ID, ObjectiveID: held.ID, Label: models.RACILabelConsulted})
	if err != nil {
		t.Fatalf("precondition: a Consulted seat on the held objective: %v", err)
	}
	elsewhere, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: draftID, RoleID: rival.ID, ObjectiveID: other.ID, Label: models.RACILabelAccountable})
	if err != nil {
		t.Fatalf("precondition: an Accountable seat on another objective: %v", err)
	}

	cases := []struct{ name, id, body string }{
		{"relabel a Consulted seat to Accountable", consulted.ID,
			`{"role_id":"` + rival.ID + `","objective_id":"` + held.ID + `","label":"accountable"}`},
		{"re-point another objective's Accountable onto the held one", elsewhere.ID,
			`{"role_id":"` + rival.ID + `","objective_id":"` + held.ID + `","label":"accountable"}`},
		{"relabel while claiming server-owned fields", consulted.ID,
			`{"role_id":"` + rival.ID + `","objective_id":"` + held.ID + `","label":"accountable","draft_id":"elsewhere","id":"forged","created_at":"1999-01-01T00:00:00Z"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPatch, "/role-assignments/"+c.id, strings.NewReader(c.body))
			rec := httptest.NewRecorder()
			serve(mgr, rec, req)
			if rec.Code != http.StatusConflict {
				t.Errorf("PATCH /role-assignments/%s %s: got %d %s, want 409 — an objective holds at most one Accountable", c.id, c.body, rec.Code, rec.Body.String())
			}
		})
	}

	seats, err := mgr.ListRoleAssignmentsForObjective(ctx, held.ID, draftID, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil {
		t.Fatalf("ListRoleAssignmentsForObjective: %v", err)
	}
	accountable := 0
	for _, s := range seats {
		if s.Label == models.RACILabelAccountable {
			accountable++
			if s.RoleID != owner.ID {
				t.Errorf("the held objective's Accountable moved to role %s, want the original %s", s.RoleID, owner.ID)
			}
		}
	}
	if accountable != 1 {
		t.Errorf("the held objective has %d Accountable seats after the refused updates, want exactly 1", accountable)
	}
}
