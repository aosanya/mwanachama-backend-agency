package mwanachamaagency_test

import (
	"context"
	"errors"
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

func TestDeleteLevel(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	lvl, err := mgr.CreateLevel(ctx, models.Level{Name: "Division"})
	if err != nil {
		t.Fatalf("CreateLevel: %v", err)
	}
	if _, err := mgr.CreateDepartment(ctx, models.Department{Name: "Software", LevelID: lvl.ID}); err != nil {
		t.Fatalf("CreateDepartment: %v", err)
	}

	if err := mgr.DeleteLevel(ctx, lvl.ID); !errors.Is(err, mwanachamaagency.ErrLevelWornByDepartments) {
		t.Fatalf("expected ErrLevelWornByDepartments, got %v", err)
	}

	empty, err := mgr.CreateLevel(ctx, models.Level{Name: "Unused"})
	if err != nil {
		t.Fatalf("CreateLevel(empty): %v", err)
	}
	if err := mgr.DeleteLevel(ctx, empty.ID); err != nil {
		t.Fatalf("DeleteLevel(empty): %v", err)
	}
	if _, err := mgr.GetLevel(ctx, empty.ID); !errors.Is(err, mwanachamaagency.ErrLevelNotFound) {
		t.Fatalf("expected level gone after delete, got %v", err)
	}

	if err := mgr.DeleteLevel(ctx, "unknown"); !errors.Is(err, mwanachamaagency.ErrLevelNotFound) {
		t.Fatalf("expected ErrLevelNotFound deleting an unknown level, got %v", err)
	}
}
