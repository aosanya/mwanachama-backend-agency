package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestGoalCRUD(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	g, err := mgr.CreateGoal(ctx, models.Goal{Title: "Grow adoption", Ordinality: 1})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if g.ID == "" {
		t.Fatal("expected minted ID")
	}

	got, err := mgr.GetGoal(ctx, g.ID)
	if err != nil {
		t.Fatalf("GetGoal: %v", err)
	}
	if got.Title != "Grow adoption" {
		t.Fatalf("unexpected goal: %+v", got)
	}

	g.Title = "Grow adoption globally"
	g.Ordinality = 0 // exercise the zero-value update path
	updated, err := mgr.UpdateGoal(ctx, g)
	if err != nil {
		t.Fatalf("UpdateGoal: %v", err)
	}
	if updated.Title != "Grow adoption globally" || updated.Ordinality != 0 {
		t.Fatalf("update did not apply zero-value field: %+v", updated)
	}

	live, err := mgr.ListGoals(ctx, "")
	if err != nil {
		t.Fatalf("ListGoals: %v", err)
	}
	if len(live) != 1 {
		t.Fatalf("expected 1 live goal, got %d", len(live))
	}
}

func TestGetGoal_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.GetGoal(context.Background(), "unknown"); !errors.Is(err, mwanachamaagency.ErrGoalNotFound) {
		t.Fatalf("expected ErrGoalNotFound, got %v", err)
	}
}

func TestDeleteGoal_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if err := mgr.DeleteGoal(context.Background(), "unknown"); !errors.Is(err, mwanachamaagency.ErrGoalNotFound) {
		t.Fatalf("expected ErrGoalNotFound, got %v", err)
	}
}

// TestDeleteGoal_CascadesObjectivesAndRoleAssignments asserts DeleteGoal
// takes its Objectives (and those Objectives' RoleAssignments) down with
// it, even though the RoleAssignment is live — DeleteRoleAssignment on its
// own would refuse that with ErrCannotEditLiveRACI, but the cascade
// bypasses that per-row check (see AgencyManager's DeleteGoal doc).
func TestDeleteGoal_CascadesObjectivesAndRoleAssignments(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, o, _, _ := seedLiveAgency(t, mgr, ctx)

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

	if err := mgr.DeleteGoal(ctx, g.ID); err != nil {
		t.Fatalf("DeleteGoal: %v", err)
	}

	if _, err := mgr.GetGoal(ctx, g.ID); !errors.Is(err, mwanachamaagency.ErrGoalNotFound) {
		t.Fatalf("expected the Goal itself to be gone, got %v", err)
	}
	if _, err := mgr.GetObjective(ctx, o.ID); !errors.Is(err, mwanachamaagency.ErrObjectiveNotFound) {
		t.Fatalf("expected the cascaded Objective to be gone, got %v", err)
	}
	if _, err := mgr.GetRoleAssignment(ctx, assignment.ID); !errors.Is(err, mwanachamaagency.ErrRoleAssignmentNotFound) {
		t.Fatalf("expected the cascaded RoleAssignment to be gone, got %v", err)
	}
	// The Role itself is a separate entity, only referenced by the
	// assignment — it must survive the Goal's deletion untouched.
	if _, err := mgr.GetRole(ctx, role.ID); err != nil {
		t.Fatalf("expected the Role to survive the Goal's deletion, got %v", err)
	}
}

// TestDeleteGoal_NoObjectives asserts deleting a Goal with nothing under
// it is just a plain delete — the cascade's "any objectives?" branch must
// not choke on the empty case.
func TestDeleteGoal_NoObjectives(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, err := mgr.CreateGoal(ctx, models.Goal{Title: "Lonely goal"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if err := mgr.DeleteGoal(ctx, g.ID); err != nil {
		t.Fatalf("DeleteGoal: %v", err)
	}
	if _, err := mgr.GetGoal(ctx, g.ID); !errors.Is(err, mwanachamaagency.ErrGoalNotFound) {
		t.Fatalf("expected ErrGoalNotFound, got %v", err)
	}
}

// TestGoalCode_SequentialAndImmutable asserts Code is minted "G-1", "G-2",
// ... in creation order, and survives an Update untouched even when the
// caller's struct carries a stale/empty Code (see UpdateGoal's doc for why
// it preserves the persisted Code rather than trusting the input).
func TestGoalCode_SequentialAndImmutable(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	g1, err := mgr.CreateGoal(ctx, models.Goal{Title: "First"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if g1.Code != "G-1" {
		t.Fatalf("expected Code %q, got %q", "G-1", g1.Code)
	}

	g2, err := mgr.CreateGoal(ctx, models.Goal{Title: "Second"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if g2.Code != "G-2" {
		t.Fatalf("expected Code %q, got %q", "G-2", g2.Code)
	}

	g1.Title = "First, renamed"
	updated, err := mgr.UpdateGoal(ctx, g1)
	if err != nil {
		t.Fatalf("UpdateGoal: %v", err)
	}
	if updated.Code != "G-1" {
		t.Fatalf("expected Code to stay %q after update, got %q", "G-1", updated.Code)
	}

	got, err := mgr.GetGoal(ctx, g1.ID)
	if err != nil {
		t.Fatalf("GetGoal: %v", err)
	}
	if got.Code != "G-1" {
		t.Fatalf("expected persisted Code to stay %q after update, got %q", "G-1", got.Code)
	}
}

// TestGoalCode_SurvivesDraftPromote asserts a Goal's Code carries unchanged
// from the live row, through its draft-scoped copy (which gets a brand-new
// ID at fork time), through an edit of that copy, to the row that ends up
// live after PromoteDraft (which itself gets a new ID relative to the
// original live row — see draft_impl.go's CreateDraft/PromoteDraft).
func TestGoalCode_SurvivesDraftPromote(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, g, _, _, _ := seedLiveAgency(t, mgr, ctx)
	originalCode := g.Code
	if originalCode == "" {
		t.Fatal("expected seeded goal to have a non-empty Code")
	}

	d, err := mgr.CreateDraft(ctx, "rename the goal", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftGoals, err := mgr.ListGoals(ctx, d.ID)
	if err != nil {
		t.Fatalf("ListGoals(draft): %v", err)
	}
	edited := draftGoals[0]
	if edited.ID == g.ID {
		t.Fatal("expected the draft-scoped copy to have a new id")
	}
	if edited.Code != originalCode {
		t.Fatalf("expected draft-scoped copy to carry the live Code %q, got %q", originalCode, edited.Code)
	}
	edited.Title = "Grow adoption worldwide"
	if _, err := mgr.UpdateGoal(ctx, edited); err != nil {
		t.Fatalf("UpdateGoal: %v", err)
	}

	if _, err := mgr.PromoteDraft(ctx, d.ID); err != nil {
		t.Fatalf("PromoteDraft: %v", err)
	}

	liveGoals, err := mgr.ListGoals(ctx, "")
	if err != nil {
		t.Fatalf("ListGoals: %v", err)
	}
	if len(liveGoals) != 1 {
		t.Fatalf("expected exactly 1 live goal after promotion, got %d", len(liveGoals))
	}
	if liveGoals[0].ID == g.ID {
		t.Fatal("expected the old live goal id to have been replaced, not reused")
	}
	if liveGoals[0].Code != originalCode {
		t.Fatalf("expected promoted goal's Code to survive unchanged, got %q want %q", liveGoals[0].Code, originalCode)
	}
}
