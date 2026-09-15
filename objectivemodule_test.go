package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCreateObjectiveModule_ObjectiveNotFound(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	mod, err := mgr.CreateModule(ctx, models.Module{Code: "widgets", Name: "Widgets"})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}

	_, err = mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{ObjectiveID: "unknown", ModuleID: mod.ID})
	if !errors.Is(err, mwanachamaagency.ErrObjectiveNotFound) {
		t.Fatalf("expected ErrObjectiveNotFound, got %v", err)
	}
}

func TestCreateObjectiveModule_ModuleNotFound(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	_, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{ObjectiveID: o.ID, ModuleID: "unknown"})
	if !errors.Is(err, mwanachamaagency.ErrModuleNotFound) {
		t.Fatalf("expected ErrModuleNotFound, got %v", err)
	}
}

// TestCreateObjectiveModule_ScopeMismatch asserts a draft-scoped Module
// cannot be linked to a live Objective — the same same-scope rule AG6
// established for RoleAssignment (AGD-002 decision #6).
func TestCreateObjectiveModule_ScopeMismatch(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	d, err := mgr.CreateDraft(ctx, "d", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftModule, err := mgr.CreateModule(ctx, models.Module{DraftID: d.ID, Code: "widgets", Name: "Widgets"})
	if err != nil {
		t.Fatalf("CreateModule(draft-scoped): %v", err)
	}

	_, err = mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{DraftID: d.ID, ObjectiveID: o.ID, ModuleID: draftModule.ID})
	if !errors.Is(err, mwanachamaagency.ErrObjectiveNotFound) {
		t.Fatalf("expected ErrObjectiveNotFound linking a draft-scoped module to a live objective, got %v", err)
	}
}

func TestCreateObjectiveModule_Duplicate(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	mod, err := mgr.CreateModule(ctx, models.Module{Code: "widgets", Name: "Widgets"})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}

	if _, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{ObjectiveID: o.ID, ModuleID: mod.ID}); err != nil {
		t.Fatalf("CreateObjectiveModule: %v", err)
	}
	_, err = mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{ObjectiveID: o.ID, ModuleID: mod.ID})
	if !errors.Is(err, mwanachamaagency.ErrDuplicateObjectiveModule) {
		t.Fatalf("expected ErrDuplicateObjectiveModule, got %v", err)
	}
}

func TestObjectiveModule_CreateListDelete(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	modA, err := mgr.CreateModule(ctx, models.Module{Code: "widgets", Name: "Widgets"})
	if err != nil {
		t.Fatalf("CreateModule(A): %v", err)
	}
	modB, err := mgr.CreateModule(ctx, models.Module{Code: "gadgets", Name: "Gadgets"})
	if err != nil {
		t.Fatalf("CreateModule(B): %v", err)
	}

	linkA, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{ObjectiveID: o.ID, ModuleID: modA.ID})
	if err != nil {
		t.Fatalf("CreateObjectiveModule(A): %v", err)
	}
	if _, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{ObjectiveID: o.ID, ModuleID: modB.ID}); err != nil {
		t.Fatalf("CreateObjectiveModule(B): %v", err)
	}

	links, err := mgr.ListObjectiveModulesForObjective(ctx, o.ID, "", mwanachamaagency.ObjectiveModuleFilter{})
	if err != nil {
		t.Fatalf("ListObjectiveModulesForObjective: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("expected 2 links for the objective, got %d", len(links))
	}

	all, err := mgr.ListObjectiveModules(ctx, "", mwanachamaagency.ObjectiveModuleFilter{})
	if err != nil {
		t.Fatalf("ListObjectiveModules: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 links overall, got %d", len(all))
	}

	if err := mgr.DeleteObjectiveModule(ctx, linkA.ID); err != nil {
		t.Fatalf("DeleteObjectiveModule: %v", err)
	}
	remaining, err := mgr.ListObjectiveModulesForObjective(ctx, o.ID, "", mwanachamaagency.ObjectiveModuleFilter{})
	if err != nil {
		t.Fatalf("ListObjectiveModulesForObjective(after delete): %v", err)
	}
	if len(remaining) != 1 || remaining[0].ModuleID != modB.ID {
		t.Fatalf("expected only the B link to remain, got %+v", remaining)
	}
}

func TestDeleteObjectiveModule_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if err := mgr.DeleteObjectiveModule(context.Background(), "unknown"); !errors.Is(err, mwanachamaagency.ErrObjectiveModuleNotFound) {
		t.Fatalf("expected ErrObjectiveModuleNotFound, got %v", err)
	}
}

// TestDeleteModule_CascadesObjectiveModules asserts DeleteModule takes its
// ObjectiveModule links down with it, matching DeleteGoal's cascade shape.
func TestDeleteModule_CascadesObjectiveModules(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	mod, err := mgr.CreateModule(ctx, models.Module{Code: "widgets", Name: "Widgets"})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	link, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{ObjectiveID: o.ID, ModuleID: mod.ID})
	if err != nil {
		t.Fatalf("CreateObjectiveModule: %v", err)
	}

	if err := mgr.DeleteModule(ctx, mod.ID); err != nil {
		t.Fatalf("DeleteModule: %v", err)
	}

	links, err := mgr.ListObjectiveModulesForObjective(ctx, o.ID, "", mwanachamaagency.ObjectiveModuleFilter{})
	if err != nil {
		t.Fatalf("ListObjectiveModulesForObjective: %v", err)
	}
	for _, l := range links {
		if l.ID == link.ID {
			t.Fatalf("expected the cascaded link to be gone, still found: %+v", l)
		}
	}
}
