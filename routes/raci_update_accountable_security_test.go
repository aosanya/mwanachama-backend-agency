package routes_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestRACIUpdateMaySeatASecondAccountable(t *testing.T) {
	s := seedRaciRoutes(t)
	ctx := t.Context()
	if _, err := s.mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: s.draftID, WorkItemID: s.step.ID, RoleID: s.roleA.ID, Label: models.RACILabelAccountable}); err != nil {
		t.Fatalf("precondition: seating the first Accountable: %v", err)
	}
	consulted, err := s.mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: s.draftID, WorkItemID: s.step.ID, RoleID: s.roleB.ID, Label: models.RACILabelConsulted})
	if err != nil {
		t.Fatalf("precondition: a Consulted seat on the step: %v", err)
	}
	roleC, err := s.mgr.CreateRole(ctx, models.Role{DraftID: s.draftID, Name: "reviewer"})
	if err != nil {
		t.Fatalf("precondition: CreateRole: %v", err)
	}
	elsewhere, err := s.mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: s.draftID, WorkItemID: s.other.ID, RoleID: roleC.ID, Label: models.RACILabelAccountable})
	if err != nil {
		t.Fatalf("precondition: an Accountable seat on another step: %v", err)
	}

	cases := []struct{ name, id, body string }{
		{"relabel a Consulted seat to Accountable", consulted.ID,
			`{"role_id":"` + s.roleB.ID + `","work_item_id":"` + s.step.ID + `","label":"accountable"}`},
		{"re-point another step's Accountable onto the held step", elsewhere.ID,
			`{"role_id":"` + roleC.ID + `","work_item_id":"` + s.step.ID + `","label":"accountable"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPatch, "/role-assignments/"+c.id, strings.NewReader(c.body))
			rec := httptest.NewRecorder()
			serve(s.mgr, rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("PATCH /role-assignments/%s %s: got %d %s, want 200 — a step may hold any number of Accountable seats", c.id, c.body, rec.Code, rec.Body.String())
			}
		})
	}

	seats, err := s.mgr.ListRoleAssignmentsForWorkItem(ctx, s.step.ID, s.draftID, mwanachamaagency.RoleAssignmentFilter{Label: string(models.RACILabelAccountable)})
	if err != nil {
		t.Fatalf("ListRoleAssignmentsForWorkItem: %v", err)
	}
	if len(seats) != 3 {
		t.Errorf("the step has %d Accountable seats after the updates, want 3", len(seats))
	}
}

func TestRACIUpdateOntoAHeldRoleAndLabelIsConflict(t *testing.T) {
	s := seedRaciRoutes(t)
	ctx := t.Context()
	if _, err := s.mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: s.draftID, WorkItemID: s.step.ID, RoleID: s.roleA.ID, Label: models.RACILabelAccountable}); err != nil {
		t.Fatalf("precondition: first seat: %v", err)
	}
	consulted, err := s.mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: s.draftID, WorkItemID: s.step.ID, RoleID: s.roleA.ID, Label: models.RACILabelConsulted})
	if err != nil {
		t.Fatalf("precondition: second seat: %v", err)
	}

	cases := []struct{ name, body string }{
		{"relabel onto the held label", `{"role_id":"` + s.roleA.ID + `","work_item_id":"` + s.step.ID + `","label":"accountable"}`},
		{"relabel while claiming server-owned fields", `{"role_id":"` + s.roleA.ID + `","work_item_id":"` + s.step.ID + `","label":"accountable","draft_id":"elsewhere","id":"forged","created_at":"1999-01-01T00:00:00Z"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPatch, "/role-assignments/"+consulted.ID, strings.NewReader(c.body))
			rec := httptest.NewRecorder()
			serve(s.mgr, rec, req)
			if rec.Code != http.StatusConflict {
				t.Errorf("PATCH %s: got %d %s, want 409 — the step already holds that role with that label", c.body, rec.Code, rec.Body.String())
			}
		})
	}

	seats, err := s.mgr.ListRoleAssignmentsForWorkItem(ctx, s.step.ID, s.draftID, mwanachamaagency.RoleAssignmentFilter{RoleID: s.roleA.ID})
	if err != nil {
		t.Fatalf("ListRoleAssignmentsForWorkItem: %v", err)
	}
	if len(seats) != 2 {
		t.Errorf("the role holds %d seats on the step after the refused updates, want 2", len(seats))
	}
}
