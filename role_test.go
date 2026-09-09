package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCreateRole_RequiresName(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.CreateRole(context.Background(), models.Role{}); !errors.Is(err, mwanachamaagency.ErrInvalidRole) {
		t.Fatalf("expected ErrInvalidRole, got %v", err)
	}
}

func TestRole_LiveCannotBeEdited(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, _, _, _ := seedLiveAgency(t, mgr, ctx)

	role, err := mgr.CreateRole(ctx, models.Role{Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	role.Name = "renamed"
	if _, err := mgr.UpdateRole(ctx, role); !errors.Is(err, mwanachamaagency.ErrCannotEditLiveRACI) {
		t.Fatalf("expected ErrCannotEditLiveRACI updating a live role, got %v", err)
	}
	if err := mgr.DeleteRole(ctx, role.ID); !errors.Is(err, mwanachamaagency.ErrCannotEditLiveRACI) {
		t.Fatalf("expected ErrCannotEditLiveRACI deleting a live role, got %v", err)
	}

	d, err := mgr.CreateDraft(ctx, "rename the role", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftRoles, err := mgr.ListRoles(ctx, d.ID)
	if err != nil {
		t.Fatalf("ListRoles(draft): %v", err)
	}
	draftRole := draftRoles[0]
	draftRole.Name = "senior-bug-hunter"
	updated, err := mgr.UpdateRole(ctx, draftRole)
	if err != nil {
		t.Fatalf("expected updating a draft-scoped role to succeed, got %v", err)
	}
	if updated.Name != "senior-bug-hunter" {
		t.Fatalf("unexpected updated role: %+v", updated)
	}
}

func TestGetRole_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.GetRole(context.Background(), "unknown"); !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
	}
}
