package mwanachamaagency_test

import (
	"context"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestLevelCRUDAndOrdering(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	team, err := mgr.CreateLevel(ctx, models.Level{Name: "Team", Depth: 1})
	if err != nil {
		t.Fatalf("CreateLevel(Team): %v", err)
	}
	division, err := mgr.CreateLevel(ctx, models.Level{Name: "Division", Depth: 0})
	if err != nil {
		t.Fatalf("CreateLevel(Division): %v", err)
	}

	all, err := mgr.ListLevels(ctx, mwanachamaagency.LevelFilter{})
	if err != nil {
		t.Fatalf("ListLevels: %v", err)
	}
	if len(all) != 2 || all[0].ID != division.ID || all[1].ID != team.ID {
		t.Fatalf("expected levels ordered by Depth (Division, Team), got %+v", all)
	}

	renamed, err := mgr.RenameLevel(ctx, team.ID, "Squad")
	if err != nil {
		t.Fatalf("RenameLevel: %v", err)
	}
	if renamed.Name != "Squad" || renamed.Depth != 1 {
		t.Fatalf("expected rename to leave Depth untouched, got %+v", renamed)
	}
}

// TestDeleteLevel covers the manager-level delete. Its old companion —
// "refused while a Department wears it" — went away with the Department
// type itself (AGD-017): nothing inside this package references a Level
// any more, so the only guard left is that the Level exists.
func TestDeleteLevel(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	lvl, err := mgr.CreateLevel(ctx, models.Level{Name: "Division"})
	if err != nil {
		t.Fatalf("CreateLevel: %v", err)
	}
	if err := mgr.DeleteLevel(ctx, lvl.ID); err != nil {
		t.Fatalf("DeleteLevel: %v", err)
	}
	if _, err := mgr.GetLevel(ctx, lvl.ID); err == nil {
		t.Fatal("expected the deleted Level to be gone")
	}
	if err := mgr.DeleteLevel(ctx, lvl.ID); err == nil {
		t.Fatal("expected deleting an unknown Level to fail")
	}
}
