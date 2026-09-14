package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestDeleteObjective_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if err := mgr.DeleteObjective(context.Background(), "unknown"); !errors.Is(err, mwanachamaagency.ErrObjectiveNotFound) {
		t.Fatalf("expected ErrObjectiveNotFound, got %v", err)
	}
}

// TestDeleteObjective_CascadesRoleAssignments asserts DeleteObjective takes
// its RoleAssignments down with it, even though they're live —
// DeleteRoleAssignment on its own would refuse that with
// ErrCannotEditLiveRACI, but the cascade bypasses that per-row check (see
// AgencyManager's DeleteObjective doc, mirroring DeleteGoal's own cascade).
func TestDeleteObjective_CascadesRoleAssignments(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	role, err := mgr.CreateRole(ctx, models.Role{Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	assignment, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		RoleID: role.ID, ObjectiveID: o.ID, Label: models.RACILabelAccountable,
	})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	if err := mgr.DeleteObjective(ctx, o.ID); err != nil {
		t.Fatalf("DeleteObjective: %v", err)
	}

	if _, err := mgr.GetObjective(ctx, o.ID); !errors.Is(err, mwanachamaagency.ErrObjectiveNotFound) {
		t.Fatalf("expected the Objective itself to be gone, got %v", err)
	}
	if _, err := mgr.GetRoleAssignment(ctx, assignment.ID); !errors.Is(err, mwanachamaagency.ErrRoleAssignmentNotFound) {
		t.Fatalf("expected the cascaded RoleAssignment to be gone, got %v", err)
	}
	// The Role itself is a separate entity, only referenced by the
	// assignment — it must survive the Objective's deletion untouched.
	if _, err := mgr.GetRole(ctx, role.ID); err != nil {
		t.Fatalf("expected the Role to survive the Objective's deletion, got %v", err)
	}
}

// TestDeleteObjective_NoRoleAssignments asserts deleting an Objective with
// nothing assigned to it is just a plain delete — the cascade's delete
// call must not choke on the empty case.
func TestDeleteObjective_NoRoleAssignments(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, err := mgr.CreateGoal(ctx, models.Goal{Title: "Lonely goal"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	o, err := mgr.CreateObjective(ctx, models.Objective{GoalID: g.ID, Title: "Lonely objective"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	if err := mgr.DeleteObjective(ctx, o.ID); err != nil {
		t.Fatalf("DeleteObjective: %v", err)
	}
	if _, err := mgr.GetObjective(ctx, o.ID); !errors.Is(err, mwanachamaagency.ErrObjectiveNotFound) {
		t.Fatalf("expected ErrObjectiveNotFound, got %v", err)
	}
}
