package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCreateRoleAssignment_InvalidLabel(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	role, _ := mgr.CreateRole(ctx, models.Role{Name: "bug-hunter"})

	_, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{RoleID: role.ID, ObjectiveID: o.ID, Label: "bogus"})
	if !errors.Is(err, mwanachamaagency.ErrInvalidRACILabel) {
		t.Fatalf("expected ErrInvalidRACILabel, got %v", err)
	}
}

func TestCreateRoleAssignment_RoleNotFound(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	_, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{RoleID: "unknown", ObjectiveID: o.ID, Label: models.RACILabelInformed})
	if !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestCreateRoleAssignment_ObjectiveNotFound(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedLiveAgency(t, mgr, ctx)
	role, _ := mgr.CreateRole(ctx, models.Role{Name: "bug-hunter"})

	_, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{RoleID: role.ID, ObjectiveID: "unknown", Label: models.RACILabelInformed})
	if !errors.Is(err, mwanachamaagency.ErrObjectiveNotFound) {
		t.Fatalf("expected ErrObjectiveNotFound, got %v", err)
	}
}

func TestCreateRoleAssignment_ScopeMismatch(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	// A draft-scoped Role cannot be assigned to a live Objective.
	d, err := mgr.CreateDraft(ctx, "d", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftRole, err := mgr.CreateRole(ctx, models.Role{DraftID: d.ID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole(draft-scoped): %v", err)
	}

	_, err = mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: d.ID, RoleID: draftRole.ID, ObjectiveID: o.ID, Label: models.RACILabelInformed})
	if !errors.Is(err, mwanachamaagency.ErrObjectiveNotFound) {
		t.Fatalf("expected ErrObjectiveNotFound assigning a draft-scoped role to a live objective, got %v", err)
	}
}

func TestCreateRoleAssignment_DuplicateAccountable(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	roleA, _ := mgr.CreateRole(ctx, models.Role{Name: "bug-hunter"})
	roleB, _ := mgr.CreateRole(ctx, models.Role{Name: "tech-lead"})

	if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{RoleID: roleA.ID, ObjectiveID: o.ID, Label: models.RACILabelAccountable}); err != nil {
		t.Fatalf("CreateRoleAssignment(A): %v", err)
	}
	_, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{RoleID: roleB.ID, ObjectiveID: o.ID, Label: models.RACILabelAccountable})
	if !errors.Is(err, mwanachamaagency.ErrDuplicateAccountable) {
		t.Fatalf("expected ErrDuplicateAccountable, got %v", err)
	}

	// A second Responsible/Consulted/Informed assignment on the same
	// Objective is fine — only Accountable is exclusive.
	if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{RoleID: roleB.ID, ObjectiveID: o.ID, Label: models.RACILabelResponsible}); err != nil {
		t.Fatalf("expected a second Responsible assignment to succeed, got %v", err)
	}
}

func TestRoleAssignment_LiveCannotBeEdited(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	role, _ := mgr.CreateRole(ctx, models.Role{Name: "bug-hunter"})
	assignment, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{RoleID: role.ID, ObjectiveID: o.ID, Label: models.RACILabelResponsible})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	assignment.Label = models.RACILabelAccountable
	if _, err := mgr.UpdateRoleAssignment(ctx, assignment); !errors.Is(err, mwanachamaagency.ErrCannotEditLiveRACI) {
		t.Fatalf("expected ErrCannotEditLiveRACI updating a live assignment, got %v", err)
	}
	if err := mgr.DeleteRoleAssignment(ctx, assignment.ID); !errors.Is(err, mwanachamaagency.ErrCannotEditLiveRACI) {
		t.Fatalf("expected ErrCannotEditLiveRACI deleting a live assignment, got %v", err)
	}
}

func TestListRoleAssignmentsForObjective(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	roleA, _ := mgr.CreateRole(ctx, models.Role{Name: "bug-hunter"})
	roleB, _ := mgr.CreateRole(ctx, models.Role{Name: "tech-lead"})

	if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{RoleID: roleA.ID, ObjectiveID: o.ID, Label: models.RACILabelResponsible}); err != nil {
		t.Fatalf("CreateRoleAssignment(A): %v", err)
	}
	if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{RoleID: roleB.ID, ObjectiveID: o.ID, Label: models.RACILabelAccountable}); err != nil {
		t.Fatalf("CreateRoleAssignment(B): %v", err)
	}

	assignments, err := mgr.ListRoleAssignmentsForObjective(ctx, o.ID, "")
	if err != nil {
		t.Fatalf("ListRoleAssignmentsForObjective: %v", err)
	}
	if len(assignments) != 2 {
		t.Fatalf("expected 2 assignments for the objective, got %d", len(assignments))
	}
}
