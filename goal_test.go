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
