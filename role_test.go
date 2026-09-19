package mwanachamaagency_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCreateRole_RequiresDraftID(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.CreateRole(context.Background(), models.Role{Name: "bug-hunter"}); !errors.Is(err, mwanachamaagency.ErrDraftRequired) {
		t.Fatalf("expected ErrDraftRequired, got %v", err)
	}
}

func TestCreateRole_RequiresName(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)
	if _, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID}); !errors.Is(err, mwanachamaagency.ErrInvalidRole) {
		t.Fatalf("expected ErrInvalidRole, got %v", err)
	}
}

// TestRoleUpdate_WithinDraft asserts a Role belonging to a Draft can be
// freely updated (and deleted) there — every Role belongs to some Draft
// (AGD-006), so this is the only path, not a special case.
func TestRoleUpdate_WithinDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, _, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := g.DraftID

	role, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	role.Name = "senior-bug-hunter"
	updated, err := mgr.UpdateRole(ctx, role)
	if err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	if updated.Name != "senior-bug-hunter" {
		t.Fatalf("unexpected updated role: %+v", updated)
	}

	if err := mgr.DeleteRole(ctx, role.ID); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}
	if _, err := mgr.GetRole(ctx, role.ID); !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound after delete, got %v", err)
	}
}

func TestGetRole_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.GetRole(context.Background(), "unknown"); !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
	}
}

// TestGetRole_NotFound_SuggestsByCodeOrName covers an AI agent guessing a
// Role's id from its Name (e.g. slugifying "Test" into "test") instead of
// using the real id — the error should still satisfy errors.Is(...,
// ErrRoleNotFound) for routes/role.go's existing 404 mapping, but now also
// names the actual match so the agent can retry with the real id instead of
// dead-ending.
func TestGetRole_NotFound_SuggestsByCodeOrName(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)

	role, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "Test"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	_, err = mgr.GetRole(ctx, "test")
	if !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
	}
	if !strings.Contains(err.Error(), role.ID) || !strings.Contains(err.Error(), role.Code) {
		t.Fatalf("expected error to name the matching role's id/code, got %q", err.Error())
	}

	// A guessed id matching by Code (case-insensitively) resolves the same way.
	_, err = mgr.GetRole(ctx, strings.ToLower(role.Code))
	if !errors.Is(err, mwanachamaagency.ErrRoleNotFound) || !strings.Contains(err.Error(), role.ID) {
		t.Fatalf("expected ErrRoleNotFound naming %s, got %v", role.ID, err)
	}
}

// TestRoleCode_SequentialAndImmutable asserts Code is minted "R-1", "R-2",
// ... in creation order, and survives an Update untouched.
func TestRoleCode_SequentialAndImmutable(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)

	r1, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if r1.Code != "R-1" {
		t.Fatalf("expected Code %q, got %q", "R-1", r1.Code)
	}

	r2, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "reviewer"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if r2.Code != "R-2" {
		t.Fatalf("expected Code %q, got %q", "R-2", r2.Code)
	}

	r1.Name = "senior-bug-hunter"
	updated, err := mgr.UpdateRole(ctx, r1)
	if err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	if updated.Code != "R-1" {
		t.Fatalf("expected Code to stay %q after update, got %q", "R-1", updated.Code)
	}
}

// TestRoleCode_SurvivesDraftFork asserts a Role's Code carries unchanged
// from the source row, through its draft-scoped copy (a new ID minted at
// fork time, AG16), through an edit of that copy.
func TestRoleCode_SurvivesDraftFork(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, _, _, _ := seedLiveAgency(t, mgr, ctx)

	role, err := mgr.CreateRole(ctx, models.Role{DraftID: g.DraftID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	originalCode := role.Code
	if originalCode == "" {
		t.Fatal("expected created role to have a non-empty Code")
	}

	d, err := mgr.CreateDraft(ctx, "rename the role", g.DraftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftRoles, err := mgr.ListRoles(ctx, d.ID, mwanachamaagency.RoleFilter{})
	if err != nil {
		t.Fatalf("ListRoles(draft): %v", err)
	}
	edited := draftRoles[0]
	if edited.ID == role.ID {
		t.Fatal("expected the draft-scoped copy to have a new id")
	}
	if edited.Code != originalCode {
		t.Fatalf("expected draft-scoped copy to carry the same Code %q, got %q", originalCode, edited.Code)
	}
	edited.Name = "senior-bug-hunter"
	updated, err := mgr.UpdateRole(ctx, edited)
	if err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	if updated.Code != originalCode {
		t.Fatalf("expected edited copy's Code to survive unchanged, got %q want %q", updated.Code, originalCode)
	}
}
