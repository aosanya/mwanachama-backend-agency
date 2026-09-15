package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestModuleCRUD(t *testing.T) {
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

	mod.Name = "Custom Renamed"
	mod.Description = ""
	mod.Enabled = false
	updated, err := mgr.UpdateModule(ctx, mod)
	if err != nil {
		t.Fatalf("UpdateModule: %v", err)
	}
	if updated.Name != "Custom Renamed" || updated.Description != "" || updated.Enabled {
		t.Fatalf("update did not apply zero-value fields: %+v", updated)
	}
	if updated.Code != "custom-module" {
		t.Fatalf("expected Code to survive update untouched, got %q", updated.Code)
	}

	if err := mgr.DeleteModule(ctx, mod.ID); err != nil {
		t.Fatalf("DeleteModule: %v", err)
	}
	if _, err := mgr.GetModule(ctx, mod.ID); !errors.Is(err, mwanachamaagency.ErrModuleNotFound) {
		t.Fatalf("expected ErrModuleNotFound after delete, got %v", err)
	}
}

func TestGetModule_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.GetModule(context.Background(), "unknown"); !errors.Is(err, mwanachamaagency.ErrModuleNotFound) {
		t.Fatalf("expected ErrModuleNotFound, got %v", err)
	}
}

func TestDeleteModule_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if err := mgr.DeleteModule(context.Background(), "unknown"); !errors.Is(err, mwanachamaagency.ErrModuleNotFound) {
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
// already seeded every models.KnownModuleCodes entry as a live, enabled
// Module at Migrate time (see gormstore/module.go) — so re-creating one of
// those codes must be rejected outright.
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

// TestBackfillModules_SeedsKnownCodes asserts every Migrate call
// (newTestManager's included) leaves every models.KnownModuleCodes entry
// present, live, and enabled — the correctness property AGD-002's backfill
// exists for: an existing deployment's gateway must never find its
// Module catalog empty and take every route dark.
func TestBackfillModules_SeedsKnownCodes(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	live, err := mgr.ListModules(ctx, "", mwanachamaagency.ModuleFilter{})
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}
	if len(live) != len(models.KnownModuleCodes) {
		t.Fatalf("expected %d backfilled modules, got %d: %+v", len(models.KnownModuleCodes), len(live), live)
	}
	byCode := make(map[string]models.Module, len(live))
	for _, m := range live {
		byCode[m.Code] = m
	}
	for _, code := range models.KnownModuleCodes {
		mod, ok := byCode[code]
		if !ok {
			t.Fatalf("expected a backfilled module for code %q", code)
		}
		if !mod.Enabled {
			t.Fatalf("expected backfilled module %q to default Enabled, got %+v", code, mod)
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
	if len(enabled) != len(models.KnownModuleCodes)+1 {
		t.Fatalf("expected every backfilled module plus widgets to be enabled, got %d", len(enabled))
	}
}
