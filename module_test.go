package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// bootstrapDraftID returns the single Draft [gormstore.EnsureWorkingDraft]
// creates during Migrate — the home for gormstore.BackfillModules' seeded
// catalog rows before any test has forked or created a Draft of its own.
func bootstrapDraftID(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context) string {
	t.Helper()
	drafts, err := mgr.ListDrafts(ctx, mwanachamaagency.DraftFilter{})
	if err != nil {
		t.Fatalf("ListDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("expected exactly 1 bootstrap draft right after Migrate, got %d: %+v", len(drafts), drafts)
	}
	return drafts[0].ID
}

func TestCreateAndGetModule(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)

	mod, err := mgr.CreateModule(ctx, models.Module{DraftID: draftID, Code: "custom-module", Name: "Custom", Description: "a custom capability", Enabled: true})
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

func TestCreateModule_EmptyDraftID(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.CreateModule(context.Background(), models.Module{Name: "no draft"}); !errors.Is(err, mwanachamaagency.ErrDraftRequired) {
		t.Fatalf("expected ErrDraftRequired, got %v", err)
	}
}

func TestCreateModule_EmptyCode(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)
	if _, err := mgr.CreateModule(ctx, models.Module{DraftID: draftID, Name: "no code"}); !errors.Is(err, mwanachamaagency.ErrInvalidModule) {
		t.Fatalf("expected ErrInvalidModule, got %v", err)
	}
}

// TestCreateModule_DuplicateCode relies on gormstore.BackfillModules having
// already seeded every models.ModuleCatalog entry into the bootstrap Draft
// at Migrate time (see gormstore/module.go) — so re-creating one of those
// codes in that same Draft must be rejected outright.
func TestCreateModule_DuplicateCode(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := bootstrapDraftID(t, mgr, ctx)

	if _, err := mgr.CreateModule(ctx, models.Module{DraftID: draftID, Code: "git", Name: "Git again"}); !errors.Is(err, mwanachamaagency.ErrDuplicateModule) {
		t.Fatalf("expected ErrDuplicateModule for an already-backfilled code, got %v", err)
	}

	if _, err := mgr.CreateModule(ctx, models.Module{DraftID: draftID, Code: "brand-new", Name: "Brand New"}); err != nil {
		t.Fatalf("CreateModule(brand-new): %v", err)
	}
	if _, err := mgr.CreateModule(ctx, models.Module{DraftID: draftID, Code: "brand-new", Name: "Brand New Again"}); !errors.Is(err, mwanachamaagency.ErrDuplicateModule) {
		t.Fatalf("expected ErrDuplicateModule for a repeat of a freshly created code, got %v", err)
	}
}

func TestBackfillModules_SeedsCatalogDefaults(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := bootstrapDraftID(t, mgr, ctx)

	mods, err := mgr.ListModules(ctx, draftID, mwanachamaagency.ModuleFilter{})
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}
	if len(mods) != len(models.ModuleCatalog) {
		t.Fatalf("expected %d backfilled modules, got %d: %+v", len(models.ModuleCatalog), len(mods), mods)
	}
	byCode := make(map[string]models.Module, len(mods))
	for _, m := range mods {
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

	neverOff := map[string]bool{"auth": true, "actor": true}
	for code := range neverOff {
		mod, ok := byCode[code]
		if !ok {
			t.Fatalf("expected a backfilled %s module", code)
		}
		if !mod.Enabled || !mod.Locked {
			t.Fatalf("expected %s to backfill Enabled: true, Locked: true, got %+v", code, mod)
		}
	}

	for _, entry := range models.ModuleCatalog {
		if neverOff[entry.Code] {
			continue
		}
		mod := byCode[entry.Code]
		if mod.Enabled || mod.Locked {
			t.Fatalf("expected module %q to backfill Enabled: false, Locked: false, got %+v", entry.Code, mod)
		}
	}
}

func TestListModules_Filter(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := o.DraftID
	widgets, err := mgr.CreateModule(ctx, models.Module{DraftID: draftID, Code: "widgets", Name: "Widgets"})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	if _, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{DraftID: draftID, ObjectiveID: o.ID, ModuleID: widgets.ID, Rationale: "needed for the objective"}); err != nil {
		t.Fatalf("CreateObjectiveModule: %v", err)
	}

	byCode, err := mgr.ListModules(ctx, draftID, mwanachamaagency.ModuleFilter{Code: "widg"})
	if err != nil {
		t.Fatalf("ListModules(code filter): %v", err)
	}
	if len(byCode) != 1 || byCode[0].Code != "widgets" {
		t.Fatalf("expected exactly the widgets module, got %+v", byCode)
	}

	trueVal := true
	enabled, err := mgr.ListModules(ctx, draftID, mwanachamaagency.ModuleFilter{Enabled: &trueVal})
	if err != nil {
		t.Fatalf("ListModules(enabled filter): %v", err)
	}
	want := map[string]bool{"widgets": true}
	for _, entry := range models.ModuleCatalog {
		if entry.Locked {
			want[entry.Code] = true
		}
	}
	got := map[string]bool{}
	for _, mod := range enabled {
		if !want[mod.Code] {
			t.Fatalf("did not expect %q to be enabled: %+v", mod.Code, enabled)
		}
		got[mod.Code] = true
	}
	if len(got) != len(want) || len(enabled) != len(want) {
		t.Fatalf("expected widgets and the locked catalog modules to be enabled, got %d: %+v", len(enabled), enabled)
	}
}

// TestModuleEnabled_DerivedFromObjectiveLinks is AGD-002's selection rule
// as the operator meets it: a Module is selected by linking it to an
// Objective that needs it, and unselected by removing the last such link.
func TestModuleEnabled_DerivedFromObjectiveLinks(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := o.DraftID

	gitMod, err := seededModule(mgr, ctx, draftID, "git")
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	if gitMod.Enabled {
		t.Fatalf("expected git to start disabled, got %+v", gitMod)
	}

	link, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{DraftID: draftID, ObjectiveID: o.ID, ModuleID: gitMod.ID, Rationale: "needed for the objective"})
	if err != nil {
		t.Fatalf("CreateObjectiveModule: %v", err)
	}
	if got := moduleByCode(t, mgr, ctx, draftID, "git"); !got.Enabled {
		t.Fatalf("expected git to be enabled once an objective links it, got %+v", got)
	}
	if got, err := mgr.GetModule(ctx, gitMod.ID); err != nil || !got.Enabled {
		t.Fatalf("expected GetModule to agree git is enabled, got %+v (%v)", got, err)
	}

	if err := mgr.DeleteObjectiveModule(ctx, link.ID); err != nil {
		t.Fatalf("DeleteObjectiveModule: %v", err)
	}
	if got := moduleByCode(t, mgr, ctx, draftID, "git"); got.Enabled {
		t.Fatalf("expected git to be disabled once its last link is gone, got %+v", got)
	}
}

// TestModuleEnabled_LockedStaysEnabled asserts the one exception: a locked
// code is mounted whether or not any Objective names it — exercised
// against the bootstrap Draft, since that's the one carrying the
// models.ModuleCatalog backfill "auth" comes from.
func TestModuleEnabled_LockedStaysEnabled(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := bootstrapDraftID(t, mgr, ctx)

	auth := moduleByCode(t, mgr, ctx, draftID, "auth")
	if !auth.Locked || !auth.Enabled {
		t.Fatalf("expected auth to be locked and enabled with no links at all, got %+v", auth)
	}
}

// TestModuleEnabled_ScopedToItsOwnDraft asserts a link made inside a Draft
// enables that Draft's own Module copy and leaves the Draft it forked from
// alone — selecting a module is a draft edit like any other, taking effect
// only where it was made.
func TestModuleEnabled_ScopedToItsOwnDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, seedGoal, _, _, _ := seedLiveAgency(t, mgr, ctx)
	seedDraftID := seedGoal.DraftID

	seedGit, err := seededModule(mgr, ctx, seedDraftID, "git")
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}

	d, err := mgr.CreateDraft(ctx, "d", seedDraftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftObjectives, err := mgr.ListObjectives(ctx, d.ID, mwanachamaagency.ObjectiveFilter{})
	if err != nil || len(draftObjectives) == 0 {
		t.Fatalf("ListObjectives(draft): %+v (%v)", draftObjectives, err)
	}
	draftGit := moduleByCode(t, mgr, ctx, d.ID, "git")
	if draftGit.ID == seedGit.ID {
		t.Fatal("expected the fork to mint a new Module id")
	}

	if _, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{
		DraftID: d.ID, ObjectiveID: draftObjectives[0].ID, ModuleID: draftGit.ID, Rationale: "needed for the objective",
	}); err != nil {
		t.Fatalf("CreateObjectiveModule(draft): %v", err)
	}

	if got := moduleByCode(t, mgr, ctx, d.ID, "git"); !got.Enabled {
		t.Fatalf("expected the new draft's git copy to be enabled, got %+v", got)
	}
	if got := moduleByCode(t, mgr, ctx, seedDraftID, "git"); got.Enabled {
		t.Fatalf("expected the seed draft's git to stay disabled — the link only exists in the new draft, got %+v", got)
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
