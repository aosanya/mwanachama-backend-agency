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
	ag, _, _, _, _ := seedLiveAgency(t, mgr, ctx)

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

// TestRoleCode_SequentialAndImmutable asserts Code is minted "R-1", "R-2",
// ... in creation order, and survives a draft-scoped Update untouched.
func TestRoleCode_SequentialAndImmutable(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, _, _, _, _ := seedLiveAgency(t, mgr, ctx)

	r1, err := mgr.CreateRole(ctx, models.Role{Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if r1.Code != "R-1" {
		t.Fatalf("expected Code %q, got %q", "R-1", r1.Code)
	}

	r2, err := mgr.CreateRole(ctx, models.Role{Name: "reviewer"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if r2.Code != "R-2" {
		t.Fatalf("expected Code %q, got %q", "R-2", r2.Code)
	}

	// Roles are draft-immutable while live (TestRole_LiveCannotBeEdited),
	// so exercising the Update path needs a draft-scoped copy.
	d, err := mgr.CreateDraft(ctx, "rename r1", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftRoles, err := mgr.ListRoles(ctx, d.ID)
	if err != nil {
		t.Fatalf("ListRoles(draft): %v", err)
	}
	var draftR1 models.Role
	for _, dr := range draftRoles {
		if dr.Code == "R-1" {
			draftR1 = dr
		}
	}
	if draftR1.ID == "" {
		t.Fatalf("expected a draft-scoped copy carrying Code %q, got %+v", "R-1", draftRoles)
	}
	draftR1.Name = "senior-bug-hunter"
	updated, err := mgr.UpdateRole(ctx, draftR1)
	if err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	if updated.Code != "R-1" {
		t.Fatalf("expected Code to stay %q after update, got %q", "R-1", updated.Code)
	}
}

// TestRoleCode_SurvivesDraftPromote asserts a Role's Code carries unchanged
// from the live row, through its draft-scoped copy (a new ID minted at
// fork time), through an edit of that copy, to the row that ends up live
// after PromoteDraft (which replaces the original live row's ID).
func TestRoleCode_SurvivesDraftPromote(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, _, _, _, _ := seedLiveAgency(t, mgr, ctx)

	role, err := mgr.CreateRole(ctx, models.Role{Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	originalCode := role.Code
	if originalCode == "" {
		t.Fatal("expected created role to have a non-empty Code")
	}

	d, err := mgr.CreateDraft(ctx, "rename the role", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftRoles, err := mgr.ListRoles(ctx, d.ID)
	if err != nil {
		t.Fatalf("ListRoles(draft): %v", err)
	}
	edited := draftRoles[0]
	if edited.ID == role.ID {
		t.Fatal("expected the draft-scoped copy to have a new id")
	}
	if edited.Code != originalCode {
		t.Fatalf("expected draft-scoped copy to carry the live Code %q, got %q", originalCode, edited.Code)
	}
	edited.Name = "senior-bug-hunter"
	if _, err := mgr.UpdateRole(ctx, edited); err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}

	if _, err := mgr.PromoteDraft(ctx, d.ID); err != nil {
		t.Fatalf("PromoteDraft: %v", err)
	}

	liveRoles, err := mgr.ListRoles(ctx, "")
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	if len(liveRoles) != 1 {
		t.Fatalf("expected exactly 1 live role after promotion, got %d", len(liveRoles))
	}
	if liveRoles[0].ID == role.ID {
		t.Fatal("expected the old live role id to have been replaced, not reused")
	}
	if liveRoles[0].Code != originalCode {
		t.Fatalf("expected promoted role's Code to survive unchanged, got %q want %q", liveRoles[0].Code, originalCode)
	}
}
