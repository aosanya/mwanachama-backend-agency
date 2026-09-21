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
	draftID := newDraftAgency(t, mgr, ctx)
	mod, err := mgr.CreateModule(ctx, models.Module{DraftID: draftID, Code: "widgets", Name: "Widgets"})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}

	_, err = mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{DraftID: draftID, ObjectiveID: "unknown", ModuleID: mod.ID, Rationale: "needed for the objective"})
	if !errors.Is(err, mwanachamaagency.ErrObjectiveNotFound) {
		t.Fatalf("expected ErrObjectiveNotFound, got %v", err)
	}
}

func TestCreateObjectiveModule_ModuleNotFound(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	_, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{DraftID: o.DraftID, ObjectiveID: o.ID, ModuleID: "unknown", Rationale: "needed for the objective"})
	if !errors.Is(err, mwanachamaagency.ErrModuleNotFound) {
		t.Fatalf("expected ErrModuleNotFound, got %v", err)
	}
}

// TestCreateObjectiveModule_ScopeMismatch asserts a Module forked into one
// Draft cannot be linked to an Objective that only exists in a different
// Draft — the same same-scope rule AG6 established for RoleAssignment
// (AGD-002 decision #6).
func TestCreateObjectiveModule_ScopeMismatch(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	seedDraftID := o.DraftID

	d, err := mgr.CreateDraft(ctx, "second draft", seedDraftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftModule, err := mgr.CreateModule(ctx, models.Module{DraftID: d.ID, Code: "widgets", Name: "Widgets"})
	if err != nil {
		t.Fatalf("CreateModule(draft-scoped): %v", err)
	}

	_, err = mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{DraftID: d.ID, ObjectiveID: o.ID, ModuleID: draftModule.ID, Rationale: "needed for the objective"})
	if !errors.Is(err, mwanachamaagency.ErrObjectiveNotFound) {
		t.Fatalf("expected ErrObjectiveNotFound linking across two different drafts, got %v", err)
	}
}

func TestCreateObjectiveModule_RationaleRequired(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	mod, err := mgr.CreateModule(ctx, models.Module{DraftID: o.DraftID, Code: "widgets", Name: "Widgets"})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	for _, blank := range []string{"", "   \n"} {
		_, err = mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{DraftID: o.DraftID, ObjectiveID: o.ID, ModuleID: mod.ID, Rationale: blank})
		if !errors.Is(err, mwanachamaagency.ErrRationaleRequired) {
			t.Fatalf("rationale %q: expected ErrRationaleRequired, got %v", blank, err)
		}
	}
	link, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{DraftID: o.DraftID, ObjectiveID: o.ID, ModuleID: mod.ID, Rationale: "  keeps the ledger  "})
	if err != nil {
		t.Fatalf("CreateObjectiveModule: %v", err)
	}
	if link.Rationale != "keeps the ledger" {
		t.Fatalf("rationale not stored trimmed: %q", link.Rationale)
	}
}

func TestCreateObjectiveModule_Duplicate(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	mod, err := mgr.CreateModule(ctx, models.Module{DraftID: o.DraftID, Code: "widgets", Name: "Widgets"})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}

	if _, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{DraftID: o.DraftID, ObjectiveID: o.ID, ModuleID: mod.ID, Rationale: "needed for the objective"}); err != nil {
		t.Fatalf("CreateObjectiveModule: %v", err)
	}
	_, err = mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{DraftID: o.DraftID, ObjectiveID: o.ID, ModuleID: mod.ID, Rationale: "needed for the objective"})
	if !errors.Is(err, mwanachamaagency.ErrDuplicateObjectiveModule) {
		t.Fatalf("expected ErrDuplicateObjectiveModule, got %v", err)
	}
}

func TestObjectiveModule_CreateListDelete(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	modA, err := mgr.CreateModule(ctx, models.Module{DraftID: o.DraftID, Code: "widgets", Name: "Widgets"})
	if err != nil {
		t.Fatalf("CreateModule(A): %v", err)
	}
	modB, err := mgr.CreateModule(ctx, models.Module{DraftID: o.DraftID, Code: "gadgets", Name: "Gadgets"})
	if err != nil {
		t.Fatalf("CreateModule(B): %v", err)
	}

	linkA, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{DraftID: o.DraftID, ObjectiveID: o.ID, ModuleID: modA.ID, Rationale: "needed for the objective"})
	if err != nil {
		t.Fatalf("CreateObjectiveModule(A): %v", err)
	}
	if _, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{DraftID: o.DraftID, ObjectiveID: o.ID, ModuleID: modB.ID, Rationale: "needed for the objective"}); err != nil {
		t.Fatalf("CreateObjectiveModule(B): %v", err)
	}

	links, err := mgr.ListObjectiveModulesForObjective(ctx, o.ID, o.DraftID, mwanachamaagency.ObjectiveModuleFilter{})
	if err != nil {
		t.Fatalf("ListObjectiveModulesForObjective: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("expected 2 links for the objective, got %d", len(links))
	}

	all, err := mgr.ListObjectiveModules(ctx, o.DraftID, mwanachamaagency.ObjectiveModuleFilter{})
	if err != nil {
		t.Fatalf("ListObjectiveModules: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 links overall, got %d", len(all))
	}

	if err := mgr.DeleteObjectiveModule(ctx, linkA.ID); err != nil {
		t.Fatalf("DeleteObjectiveModule: %v", err)
	}
	remaining, err := mgr.ListObjectiveModulesForObjective(ctx, o.ID, o.DraftID, mwanachamaagency.ObjectiveModuleFilter{})
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

// TestDeleteObjective_CascadesIntoObjectiveModule pins the AG31 fix:
// deleting the only Objective that linked a Module removes the link row too,
// so the Module reads disabled again ("enabled exactly while some Objective
// links it", mcp/module.go).
func TestDeleteObjective_CascadesIntoObjectiveModule(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	mod, err := mgr.CreateModule(ctx, models.Module{DraftID: o.DraftID, Code: "widgets", Name: "Widgets"})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	if _, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{DraftID: o.DraftID, ObjectiveID: o.ID, ModuleID: mod.ID, Rationale: "needed for the objective"}); err != nil {
		t.Fatalf("CreateObjectiveModule: %v", err)
	}
	if before, err := mgr.GetModule(ctx, mod.ID); err != nil || !before.Enabled {
		t.Fatalf("expected module enabled once linked, got %+v (err %v)", before, err)
	}

	if err := mgr.DeleteObjective(ctx, o.ID); err != nil {
		t.Fatalf("DeleteObjective: %v", err)
	}
	if _, err := mgr.GetObjective(ctx, o.ID); !errors.Is(err, mwanachamaagency.ErrObjectiveNotFound) {
		t.Fatalf("expected the Objective to really be gone, got %v", err)
	}

	remaining, err := mgr.ListObjectiveModulesForObjective(ctx, o.ID, o.DraftID, mwanachamaagency.ObjectiveModuleFilter{})
	if err != nil {
		t.Fatalf("ListObjectiveModulesForObjective: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected the link to be deleted with its Objective, got %d", len(remaining))
	}
	after, err := mgr.GetModule(ctx, mod.ID)
	if err != nil {
		t.Fatalf("GetModule (after delete): %v", err)
	}
	if after.Enabled {
		t.Fatalf("expected the module disabled once its only Objective is gone, got %+v", after)
	}
}

func TestDeleteGoal_CascadesIntoObjectiveModule(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, o, _, _ := seedLiveAgency(t, mgr, ctx)

	mod, err := mgr.CreateModule(ctx, models.Module{DraftID: o.DraftID, Code: "widgets", Name: "Widgets"})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	if _, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{DraftID: o.DraftID, ObjectiveID: o.ID, ModuleID: mod.ID, Rationale: "needed for the objective"}); err != nil {
		t.Fatalf("CreateObjectiveModule: %v", err)
	}
	if err := mgr.DeleteGoal(ctx, g.ID); err != nil {
		t.Fatalf("DeleteGoal: %v", err)
	}
	after, err := mgr.GetModule(ctx, mod.ID)
	if err != nil {
		t.Fatalf("GetModule: %v", err)
	}
	if after.Enabled {
		t.Fatalf("expected the module disabled once its Goal's Objectives are gone, got %+v", after)
	}
}

func TestUpdateObjectiveModule(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	mod, err := mgr.CreateModule(ctx, models.Module{DraftID: o.DraftID, Code: "widgets", Name: "Widgets"})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	link, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{DraftID: o.DraftID, ObjectiveID: o.ID, ModuleID: mod.ID, Rationale: "first reason"})
	if err != nil {
		t.Fatalf("CreateObjectiveModule: %v", err)
	}

	if _, err := mgr.UpdateObjectiveModule(ctx, link.ID, "  "); !errors.Is(err, mwanachamaagency.ErrRationaleRequired) {
		t.Fatalf("blank: expected ErrRationaleRequired, got %v", err)
	}
	if _, err := mgr.UpdateObjectiveModule(ctx, "unknown", "x"); !errors.Is(err, mwanachamaagency.ErrObjectiveModuleNotFound) {
		t.Fatalf("unknown id: expected ErrObjectiveModuleNotFound, got %v", err)
	}
	got, err := mgr.UpdateObjectiveModule(ctx, link.ID, " second reason ")
	if err != nil {
		t.Fatalf("UpdateObjectiveModule: %v", err)
	}
	if got.Rationale != "second reason" || got.ObjectiveID != o.ID || got.ModuleID != mod.ID {
		t.Fatalf("unexpected link after update: %+v", got)
	}
	links, err := mgr.ListObjectiveModulesForObjective(ctx, o.ID, o.DraftID, mwanachamaagency.ObjectiveModuleFilter{})
	if err != nil || len(links) != 1 || links[0].Rationale != "second reason" {
		t.Fatalf("update not persisted: %+v err=%v", links, err)
	}
}
