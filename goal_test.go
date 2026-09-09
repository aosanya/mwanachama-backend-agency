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
