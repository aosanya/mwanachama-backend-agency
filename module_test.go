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
	// Enabled is derived from the Objectives linking this Module, so a
	// caller-supplied true on a module nothing links yet is ignored.
	if mod.Enabled {
		t.Fatalf("expected a freshly created, unlinked module to be disabled, got %+v", mod)
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
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	widgets, err := mgr.CreateModule(ctx, models.Module{Code: "widgets", Name: "Widgets"})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	if _, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{ObjectiveID: o.ID, ModuleID: widgets.ID}); err != nil {
		t.Fatalf("CreateObjectiveModule: %v", err)
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
	// "auth" is enabled because it is locked; "widgets" because the seeded
	// Objective links it. Nothing else in the catalog is linked at all.
	if len(enabled) != 2 {
		t.Fatalf("expected auth plus widgets to be enabled, got %d: %+v", len(enabled), enabled)
	}
}

// TestModuleEnabled_DerivedFromObjectiveLinks is AGD-002's selection rule
// as the operator meets it: a Module is selected by linking it to an
// Objective that needs it, and unselected by removing the last such link.
func TestModuleEnabled_DerivedFromObjectiveLinks(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	git := moduleByCode(t, mgr, ctx, "", "git")
	if git.Enabled {
		t.Fatalf("expected git to start disabled, got %+v", git)
	}

	link, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{ObjectiveID: o.ID, ModuleID: git.ID})
	if err != nil {
		t.Fatalf("CreateObjectiveModule: %v", err)
	}
	if got := moduleByCode(t, mgr, ctx, "", "git"); !got.Enabled {
		t.Fatalf("expected git to be enabled once an objective links it, got %+v", got)
	}
	if got, err := mgr.GetModule(ctx, git.ID); err != nil || !got.Enabled {
		t.Fatalf("expected GetModule to agree git is enabled, got %+v (%v)", got, err)
	}

	if err := mgr.DeleteObjectiveModule(ctx, link.ID); err != nil {
		t.Fatalf("DeleteObjectiveModule: %v", err)
	}
	if got := moduleByCode(t, mgr, ctx, "", "git"); got.Enabled {
		t.Fatalf("expected git to be disabled once its last link is gone, got %+v", got)
	}
}

// TestModuleEnabled_LockedStaysEnabled asserts the one exception: a locked
// code is mounted whether or not any Objective names it.
func TestModuleEnabled_LockedStaysEnabled(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	auth := moduleByCode(t, mgr, ctx, "", "auth")
	if !auth.Locked || !auth.Enabled {
		t.Fatalf("expected auth to be locked and enabled with no links at all, got %+v", auth)
	}
}

// TestModuleEnabled_ScopedToItsOwnDraft asserts a link made inside a Draft
// enables that Draft's own Module copy and leaves the working set's alone —
// selecting a module is a draft edit like any other, taking effect live
// only on promotion.
func TestModuleEnabled_ScopedToItsOwnDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, _, _, _, _ := seedLiveAgency(t, mgr, ctx)

	d, err := mgr.CreateDraft(ctx, "d", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftObjectives, err := mgr.ListObjectives(ctx, d.ID, mwanachamaagency.ObjectiveFilter{})
	if err != nil || len(draftObjectives) == 0 {
		t.Fatalf("ListObjectives(draft): %+v (%v)", draftObjectives, err)
	}
	draftGit := moduleByCode(t, mgr, ctx, d.ID, "git")

	if _, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{
		DraftID: d.ID, ObjectiveID: draftObjectives[0].ID, ModuleID: draftGit.ID,
	}); err != nil {
		t.Fatalf("CreateObjectiveModule(draft): %v", err)
	}

	if got := moduleByCode(t, mgr, ctx, d.ID, "git"); !got.Enabled {
		t.Fatalf("expected the draft's git copy to be enabled, got %+v", got)
	}
	if got := moduleByCode(t, mgr, ctx, "", "git"); got.Enabled {
		t.Fatalf("expected the working set's git to stay disabled until promotion, got %+v", got)
	}

	if _, err := mgr.PromoteDraft(ctx, d.ID); err != nil {
		t.Fatalf("PromoteDraft: %v", err)
	}
	if got := moduleByCode(t, mgr, ctx, "", "git"); !got.Enabled {
		t.Fatalf("expected git to be enabled live after promotion, got %+v", got)
	}
}

func moduleByCode(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID, code string) models.Module {
	t.Helper()
	mods, err := mgr.ListModules(ctx, draftID, mwanachamaagency.ModuleFilter{Code: code})
	if err != nil {
		t.Fatalf("ListModules(%q): %v", code, err)
	}
	for _, m := range mods {
		if m.Code == code {
			return m
		}
	}
	t.Fatalf("no module %q in scope %q: %+v", code, draftID, mods)
	return models.Module{}
}
