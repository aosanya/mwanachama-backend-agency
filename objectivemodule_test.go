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

// TestDeleteObjective_LeavesModuleLinkDangling pins board row AG31: unlike
// DeleteObjective's cascade into RoleAssignment/WorkItem (AG27),
// DeleteObjective never cascades into ObjectiveModule. agency_attach_module's
// own documented contract is "a Module is enabled exactly while some
// Objective links it" (mcp/module.go) — deleting the only Objective that
// linked a Module leaves the ObjectiveModule row behind, so the Module
// stays permanently Enabled, and AG26's Issue scan (issue_impl.go,
// checkAttachments/checkRACI/checkWorkItems) has no check for this dangling
// reference the way it does for WorkItem/RoleAssignment/attachments, so
// PublishAgency's validation gate never catches it either.
//
// This test asserts the CURRENT (broken) behavior. Once fixed —
// DeleteObjective/DeleteGoal cascading into ObjectiveModule, and/or the
// Issue scan gaining a check for an ObjectiveModule whose ObjectiveID no
// longer resolves — this test should go red: `remaining` should be empty
// and `mod.Enabled` should be false after the Objective is deleted.
func TestDeleteObjective_LeavesModuleLinkDangling(t *testing.T) {
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

	before, err := mgr.GetModule(ctx, mod.ID)
	if err != nil {
		t.Fatalf("GetModule (before delete): %v", err)
	}
	if !before.Enabled {
		t.Fatalf("expected module enabled once linked, got %+v", before)
	}

	if err := mgr.DeleteObjective(ctx, o.ID); err != nil {
		t.Fatalf("DeleteObjective: %v", err)
	}
	if _, err := mgr.GetObjective(ctx, o.ID); !errors.Is(err, mwanachamaagency.ErrObjectiveNotFound) {
		t.Fatalf("expected the Objective to really be gone, got %v", err)
	}

	// BUG (AG31): the link row survives the Objective it pointed at.
	remaining, err := mgr.ListObjectiveModulesForObjective(ctx, o.ID, "", mwanachamaagency.ObjectiveModuleFilter{})
	if err != nil {
		t.Fatalf("ListObjectiveModulesForObjective: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("pinned bug regressed: expected the dangling link to still be reported (len=1), got %d", len(remaining))
	}

	// BUG (AG31): the Module stays derived-Enabled forever, since
	// linkedModuleIDs (module_impl.go) never checks the Objective it found
	// still exists.
	after, err := mgr.GetModule(ctx, mod.ID)
	if err != nil {
		t.Fatalf("GetModule (after delete): %v", err)
	}
	if !after.Enabled {
		t.Fatalf("pinned bug regressed: expected the module to still read Enabled=true, got %+v", after)
	}

	// BUG (AG31): the Issue scan does not flag this dangling reference at
	// all, so PublishAgency's validation gate never catches it either.
	issues, err := mgr.ListIssues(ctx, "", mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	for _, iss := range issues {
		if iss.EntityType == "objective_module" {
			t.Fatalf("pinned bug regressed: scan now reports an objective_module issue (%+v) — this test needs updating, not just deleting", iss)
		}
	}
}
