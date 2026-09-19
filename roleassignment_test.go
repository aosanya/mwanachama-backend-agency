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
	role, _ := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "bug-hunter"})

	_, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: o.DraftID, RoleID: role.ID, ObjectiveID: o.ID, Label: "bogus"})
	if !errors.Is(err, mwanachamaagency.ErrInvalidRACILabel) {
		t.Fatalf("expected ErrInvalidRACILabel, got %v", err)
	}
}

func TestCreateRoleAssignment_RoleNotFound(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	_, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: o.DraftID, RoleID: "unknown", ObjectiveID: o.ID, Label: models.RACILabelInformed})
	if !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestCreateRoleAssignment_ObjectiveNotFound(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	role, _ := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "bug-hunter"})

	_, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: o.DraftID, RoleID: role.ID, ObjectiveID: "unknown", Label: models.RACILabelInformed})
	if !errors.Is(err, mwanachamaagency.ErrObjectiveNotFound) {
		t.Fatalf("expected ErrObjectiveNotFound, got %v", err)
	}
}

// TestCreateRoleAssignment_ScopeMismatch: a Role forked into one Draft
// cannot be assigned to an Objective that only exists in a different
// Draft (or in neither).
func TestCreateRoleAssignment_ScopeMismatch(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	seedDraftID := o.DraftID

	d, err := mgr.CreateDraft(ctx, "second draft", seedDraftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftRole, err := mgr.CreateRole(ctx, models.Role{DraftID: d.ID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole(draft-scoped): %v", err)
	}

	// draftRole belongs to d, but o only exists in the seed draft.
	_, err = mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: d.ID, RoleID: draftRole.ID, ObjectiveID: o.ID, Label: models.RACILabelInformed})
	if !errors.Is(err, mwanachamaagency.ErrObjectiveNotFound) {
		t.Fatalf("expected ErrObjectiveNotFound assigning across two different drafts, got %v", err)
	}
}

func TestCreateRoleAssignment_DuplicateAccountable(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	roleA, _ := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "bug-hunter"})
	roleB, _ := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "tech-lead"})

	if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: o.DraftID, RoleID: roleA.ID, ObjectiveID: o.ID, Label: models.RACILabelAccountable}); err != nil {
		t.Fatalf("CreateRoleAssignment(A): %v", err)
	}
	_, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: o.DraftID, RoleID: roleB.ID, ObjectiveID: o.ID, Label: models.RACILabelAccountable})
	if !errors.Is(err, mwanachamaagency.ErrDuplicateAccountable) {
		t.Fatalf("expected ErrDuplicateAccountable, got %v", err)
	}

	// A second Responsible/Consulted/Informed assignment on the same
	// Objective is fine — only Accountable is exclusive.
	if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: o.DraftID, RoleID: roleB.ID, ObjectiveID: o.ID, Label: models.RACILabelResponsible}); err != nil {
		t.Fatalf("expected a second Responsible assignment to succeed, got %v", err)
	}
}

// TestRoleAssignment_UpdateAndDelete asserts a RoleAssignment belonging to
// a Draft can be freely updated and deleted there — every RoleAssignment
// belongs to some Draft (AGD-006), so this is the only path.
func TestRoleAssignment_UpdateAndDelete(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	role, _ := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "bug-hunter"})
	assignment, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: o.DraftID, RoleID: role.ID, ObjectiveID: o.ID, Label: models.RACILabelResponsible})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	assignment.Label = models.RACILabelAccountable
	updated, err := mgr.UpdateRoleAssignment(ctx, assignment)
	if err != nil {
		t.Fatalf("UpdateRoleAssignment: %v", err)
	}
	if updated.Label != models.RACILabelAccountable {
		t.Fatalf("unexpected updated assignment: %+v", updated)
	}

	if err := mgr.DeleteRoleAssignment(ctx, assignment.ID); err != nil {
		t.Fatalf("DeleteRoleAssignment: %v", err)
	}
	if _, err := mgr.GetRoleAssignment(ctx, assignment.ID); !errors.Is(err, mwanachamaagency.ErrRoleAssignmentNotFound) {
		t.Fatalf("expected ErrRoleAssignmentNotFound after delete, got %v", err)
	}
}

func TestListRoleAssignmentsForObjective(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	roleA, _ := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "bug-hunter"})
	roleB, _ := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "tech-lead"})

	if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: o.DraftID, RoleID: roleA.ID, ObjectiveID: o.ID, Label: models.RACILabelResponsible}); err != nil {
		t.Fatalf("CreateRoleAssignment(A): %v", err)
	}
	if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: o.DraftID, RoleID: roleB.ID, ObjectiveID: o.ID, Label: models.RACILabelAccountable}); err != nil {
		t.Fatalf("CreateRoleAssignment(B): %v", err)
	}

	assignments, err := mgr.ListRoleAssignmentsForObjective(ctx, o.ID, o.DraftID, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil {
		t.Fatalf("ListRoleAssignmentsForObjective: %v", err)
	}
	if len(assignments) != 2 {
		t.Fatalf("expected 2 assignments for the objective, got %d", len(assignments))
	}
}
