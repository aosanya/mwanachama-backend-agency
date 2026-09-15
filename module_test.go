package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCreateAndGetModule(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	mod, err := mgr.CreateModule(ctx, models.Module{Code: "custom-module", Name: "Custom", Description: "a custom capability", Enabled: true})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	if mod.ID == "" {
		t.Fatal("expected minted ID")
	}
	if !mod.Enabled {
		t.Fatal("expected Enabled to round-trip true")
	}

	got, err := mgr.GetModule(ctx, mod.ID)
	if err != nil {
		t.Fatalf("GetModule: %v", err)
	}
	if got.Code != "custom-module" || got.Name != "Custom" {
		t.Fatalf("unexpected module: %+v", got)
	}
}

func TestGetModule_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.GetModule(context.Background(), "unknown"); !errors.Is(err, mwanachamaagency.ErrModuleNotFound) {
		t.Fatalf("expected ErrModuleNotFound, got %v", err)
	}
}

func TestCreateModule_EmptyCode(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.CreateModule(context.Background(), models.Module{Name: "no code"}); !errors.Is(err, mwanachamaagency.ErrInvalidModule) {
		t.Fatalf("expected ErrInvalidModule, got %v", err)
	}
}

// TestCreateModule_DuplicateCode relies on gormstore.BackfillModules having
// already seeded every models.ModuleCatalog entry as a live Module at
// Migrate time (see gormstore/module.go) — so re-creating one of those
// codes must be rejected outright.
func TestCreateModule_DuplicateCode(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.CreateModule(ctx, models.Module{Code: "git", Name: "Git again"}); !errors.Is(err, mwanachamaagency.ErrDuplicateModule) {
		t.Fatalf("expected ErrDuplicateModule for an already-backfilled code, got %v", err)
	}

	if _, err := mgr.CreateModule(ctx, models.Module{Code: "brand-new", Name: "Brand New"}); err != nil {
		t.Fatalf("CreateModule(brand-new): %v", err)
	}
	if _, err := mgr.CreateModule(ctx, models.Module{Code: "brand-new", Name: "Brand New Again"}); !errors.Is(err, mwanachamaagency.ErrDuplicateModule) {
		t.Fatalf("expected ErrDuplicateModule for a repeat of a freshly created code, got %v", err)
	}
}

// TestBackfillModules_SeedsCatalogDefaults asserts every Migrate call
// (newTestManager's included) leaves every models.ModuleCatalog entry
// present and live, with "auth" backfilled Enabled+Locked and every other
// known code backfilled disabled and unlocked — the correctness property
// AGD-002's reversal exists for: a fresh instance starts with only auth
// enabled, but an already-migrated instance's existing rows (not exercised
// by this fresh-table test) are never touched by a later Migrate.
func TestBackfillModules_SeedsCatalogDefaults(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	live, err := mgr.ListModules(ctx, "", mwanachamaagency.ModuleFilter{})
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}
	if len(live) != len(models.ModuleCatalog) {
		t.Fatalf("expected %d backfilled modules, got %d: %+v", len(models.ModuleCatalog), len(live), live)
	}
	byCode := make(map[string]models.Module, len(live))
	for _, m := range live {
		byCode[m.Code] = m
	}
	for _, entry := range models.ModuleCatalog {
		mod, ok := byCode[entry.Code]
		if !ok {
			t.Fatalf("expected a backfilled module for code %q", entry.Code)
		}
		if mod.Enabled != entry.Enabled {
			t.Fatalf("expected backfilled module %q Enabled=%v, got %+v", entry.Code, entry.Enabled, mod)
		}
		if mod.Locked != entry.Locked {
			t.Fatalf("expected backfilled module %q Locked=%v, got %+v", entry.Code, entry.Locked, mod)
		}
	}

	authMod, ok := byCode["auth"]
	if !ok {
		t.Fatal("expected a backfilled auth module")
	}
	if !authMod.Enabled || !authMod.Locked {
		t.Fatalf("expected auth to backfill Enabled: true, Locked: true, got %+v", authMod)
	}

	for _, entry := range models.ModuleCatalog {
		if entry.Code == "auth" {
			continue
		}
		mod := byCode[entry.Code]
		if mod.Enabled || mod.Locked {
			t.Fatalf("expected non-auth module %q to backfill Enabled: false, Locked: false, got %+v", entry.Code, mod)
		}
	}
}

func TestListModules_Filter(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	if _, err := mgr.CreateModule(ctx, models.Module{Code: "widgets", Name: "Widgets", Enabled: true}); err != nil {
		t.Fatalf("CreateModule: %v", err)
	}

	byCode, err := mgr.ListModules(ctx, "", mwanachamaagency.ModuleFilter{Code: "widg"})
	if err != nil {
		t.Fatalf("ListModules(code filter): %v", err)
	}
	if len(byCode) != 1 || byCode[0].Code != "widgets" {
		t.Fatalf("expected exactly the widgets module, got %+v", byCode)
	}

	trueVal := true
	enabled, err := mgr.ListModules(ctx, "", mwanachamaagency.ModuleFilter{Enabled: &trueVal})
	if err != nil {
		t.Fatalf("ListModules(enabled filter): %v", err)
	}
	// Only "auth" backfills enabled; "widgets" was created enabled too.
	if len(enabled) != 2 {
		t.Fatalf("expected auth plus widgets to be enabled, got %d: %+v", len(enabled), enabled)
	}
}
