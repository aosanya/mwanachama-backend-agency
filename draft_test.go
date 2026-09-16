package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// seedLiveAgency creates a live Agency with one Goal, one Objective for
// that Goal, one Workflow, and one WorkItem belonging to that Objective
// and owned by that Workflow.
func seedLiveAgency(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context) (models.Agency, models.Goal, models.Objective, models.Workflow, models.WorkItem) {
	t.Helper()
	ag, err := mgr.CreateAgency(ctx, models.Agency{Name: "SWELab"})
	if err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	g, err := mgr.CreateGoal(ctx, models.Goal{Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	o, err := mgr.CreateObjective(ctx, models.Objective{GoalID: g.ID, Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	w, err := mgr.CreateWorkflow(ctx, models.Workflow{Name: "Onboarding"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	wi, err := mgr.CreateWorkItem(ctx, models.WorkItem{WorkflowID: w.ID, ObjectiveID: o.ID, Title: "Write charter"})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	return ag, g, o, w, wi
}

func TestCreateDraft_ForkFromLive(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, g, o, w, wi := seedLiveAgency(t, mgr, ctx)

	d, err := mgr.CreateDraft(ctx, "Q3 restructuring", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if d.Status != models.DraftStatusOpen {
		t.Fatalf("expected new draft to be open, got %q", d.Status)
	}

	liveGoals, _ := mgr.ListGoals(ctx, "", mwanachamaagency.GoalFilter{})
	if len(liveGoals) != 1 || liveGoals[0].ID != g.ID {
		t.Fatalf("live goals should be unchanged, got %+v", liveGoals)
	}

	draftGoals, err := mgr.ListGoals(ctx, d.ID, mwanachamaagency.GoalFilter{})
	if err != nil {
		t.Fatalf("ListGoals(draft): %v", err)
	}
	if len(draftGoals) != 1 || draftGoals[0].ID == g.ID {
		t.Fatalf("expected one draft-scoped goal copy with a new id, got %+v", draftGoals)
	}

	draftObjectives, err := mgr.ListObjectives(ctx, d.ID, mwanachamaagency.ObjectiveFilter{})
	if err != nil {
		t.Fatalf("ListObjectives(draft): %v", err)
	}
	if len(draftObjectives) != 1 || draftObjectives[0].ID == o.ID {
		t.Fatalf("expected one draft-scoped objective copy with a new id, got %+v", draftObjectives)
	}
	if draftObjectives[0].GoalID != draftGoals[0].ID {
		t.Fatalf("expected Objective.GoalID remapped to the draft-scoped goal copy, got %q want %q",
			draftObjectives[0].GoalID, draftGoals[0].ID)
	}

	draftWorkflows, _ := mgr.ListWorkflows(ctx, d.ID, mwanachamaagency.WorkflowFilter{})
	if len(draftWorkflows) != 1 || draftWorkflows[0].ID == w.ID {
		t.Fatalf("expected one draft-scoped workflow copy with a new id, got %+v", draftWorkflows)
	}

	draftWorkItems, err := mgr.ListWorkItems(ctx, d.ID, mwanachamaagency.WorkItemFilter{})
	if err != nil {
		t.Fatalf("ListWorkItems(draft): %v", err)
	}
	if len(draftWorkItems) != 1 || draftWorkItems[0].ID == wi.ID {
		t.Fatalf("expected one draft-scoped work item copy with a new id, got %+v", draftWorkItems)
	}
	if draftWorkItems[0].WorkflowID != draftWorkflows[0].ID {
		t.Fatalf("expected WorkItem.WorkflowID remapped to the draft-scoped workflow copy, got %q want %q",
			draftWorkItems[0].WorkflowID, draftWorkflows[0].ID)
	}
	if draftWorkItems[0].ObjectiveID != draftObjectives[0].ID {
		t.Fatalf("expected WorkItem.ObjectiveID remapped to the draft-scoped objective copy, got %q want %q",
			draftWorkItems[0].ObjectiveID, draftObjectives[0].ID)
	}
}

func TestCreateDraft_ForksRoleAssignments(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	role, err := mgr.CreateRole(ctx, models.Role{Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	assignment, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		RoleID: role.ID, ObjectiveID: o.ID, Label: models.RACILabelAccountable,
	})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	d, err := mgr.CreateDraft(ctx, "restaff the objective", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	draftRoles, err := mgr.ListRoles(ctx, d.ID, mwanachamaagency.RoleFilter{})
	if err != nil {
		t.Fatalf("ListRoles(draft): %v", err)
	}
	if len(draftRoles) != 1 || draftRoles[0].ID == role.ID {
		t.Fatalf("expected one draft-scoped role copy with a new id, got %+v", draftRoles)
	}

	draftObjectives, _ := mgr.ListObjectives(ctx, d.ID, mwanachamaagency.ObjectiveFilter{})
	draftAssignments, err := mgr.ListRoleAssignments(ctx, d.ID, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil {
		t.Fatalf("ListRoleAssignments(draft): %v", err)
	}
	if len(draftAssignments) != 1 || draftAssignments[0].ID == assignment.ID {
		t.Fatalf("expected one draft-scoped assignment copy with a new id, got %+v", draftAssignments)
	}
	if draftAssignments[0].RoleID != draftRoles[0].ID {
		t.Fatalf("expected RoleAssignment.RoleID remapped to the draft-scoped role copy, got %q want %q",
			draftAssignments[0].RoleID, draftRoles[0].ID)
	}
	if draftAssignments[0].ObjectiveID != draftObjectives[0].ID {
		t.Fatalf("expected RoleAssignment.ObjectiveID remapped to the draft-scoped objective copy, got %q want %q",
			draftAssignments[0].ObjectiveID, draftObjectives[0].ID)
	}
}

// TestCreateDraft_ForksModulesAndObjectiveModules asserts Module and
// ObjectiveModule survive a fork with new ids, and that ObjectiveModule's
// ObjectiveID/ModuleID get remapped to the draft-scoped copies — the same
// shape as TestCreateDraft_ForksRoleAssignments.
func TestCreateDraft_ForksModulesAndObjectiveModules(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	mod, err := mgr.CreateModule(ctx, models.Module{Code: "widgets", Name: "Widgets", Enabled: true})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	link, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{ObjectiveID: o.ID, ModuleID: mod.ID})
	if err != nil {
		t.Fatalf("CreateObjectiveModule: %v", err)
	}

	d, err := mgr.CreateDraft(ctx, "restaff the modules", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	// Module has no "only fork if referenced" rule like Value/Rule — every
	// live Module (including gormstore.BackfillModules' models.ModuleCatalog
	// rows) forks, so narrow to the one this test cares about by Code.
	draftModules, err := mgr.ListModules(ctx, d.ID, mwanachamaagency.ModuleFilter{Code: "widgets"})
	if err != nil {
		t.Fatalf("ListModules(draft): %v", err)
	}
	if len(draftModules) != 1 || draftModules[0].ID == mod.ID {
		t.Fatalf("expected one draft-scoped module copy with a new id, got %+v", draftModules)
	}
	if draftModules[0].Code != mod.Code {
		t.Fatalf("expected draft-scoped module copy to carry the live Code %q, got %q", mod.Code, draftModules[0].Code)
	}

	draftObjectives, _ := mgr.ListObjectives(ctx, d.ID, mwanachamaagency.ObjectiveFilter{})
	draftLinks, err := mgr.ListObjectiveModules(ctx, d.ID, mwanachamaagency.ObjectiveModuleFilter{})
	if err != nil {
		t.Fatalf("ListObjectiveModules(draft): %v", err)
	}
	if len(draftLinks) != 1 || draftLinks[0].ID == link.ID {
		t.Fatalf("expected one draft-scoped link copy with a new id, got %+v", draftLinks)
	}
	if draftLinks[0].ModuleID != draftModules[0].ID {
		t.Fatalf("expected ObjectiveModule.ModuleID remapped to the draft-scoped module copy, got %q want %q",
			draftLinks[0].ModuleID, draftModules[0].ID)
	}
	if draftLinks[0].ObjectiveID != draftObjectives[0].ID {
		t.Fatalf("expected ObjectiveModule.ObjectiveID remapped to the draft-scoped objective copy, got %q want %q",
			draftLinks[0].ObjectiveID, draftObjectives[0].ID)
	}
}

// TestPromoteDraft_ModulesAndObjectiveModules asserts a forked Module and
// its ObjectiveModule link both survive PromoteDraft with a new live id —
// same shape as TestPromoteDraft's Role/RoleAssignment coverage. Module has
// no Update path any more (AGD-002 reversed to fully read-only), so unlike
// this test's earlier form there is no draft-side edit step: the fork's
// values (including Enabled) carry through unchanged, only the id changes.
func TestPromoteDraft_ModulesAndObjectiveModules(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	mod, err := mgr.CreateModule(ctx, models.Module{Code: "widgets", Name: "Widgets", Enabled: true})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	if _, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{ObjectiveID: o.ID, ModuleID: mod.ID}); err != nil {
		t.Fatalf("CreateObjectiveModule: %v", err)
	}

	d, err := mgr.CreateDraft(ctx, "fork widgets", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftModules, err := mgr.ListModules(ctx, d.ID, mwanachamaagency.ModuleFilter{Code: "widgets"})
	if err != nil || len(draftModules) != 1 {
		t.Fatalf("ListModules(draft, code=widgets): rows=%+v err=%v", draftModules, err)
	}

	if _, err := mgr.PromoteDraft(ctx, d.ID); err != nil {
		t.Fatalf("PromoteDraft: %v", err)
	}

	liveModules, err := mgr.ListModules(ctx, "", mwanachamaagency.ModuleFilter{Code: "widgets"})
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}
	if len(liveModules) != 1 {
		t.Fatalf("expected exactly 1 live widgets module after promotion, got %d", len(liveModules))
	}
	if !liveModules[0].Enabled {
		t.Fatal("expected the forked module's Enabled value to carry through promotion unchanged")
	}
	if liveModules[0].ID == mod.ID {
		t.Fatal("expected the old live module id to have been replaced, not reused")
	}

	liveLinks, err := mgr.ListObjectiveModules(ctx, "", mwanachamaagency.ObjectiveModuleFilter{})
	if err != nil {
		t.Fatalf("ListObjectiveModules: %v", err)
	}
	if len(liveLinks) != 1 || liveLinks[0].ModuleID != liveModules[0].ID {
		t.Fatalf("expected the promoted link to reference the promoted module, got %+v", liveLinks)
	}
}

func TestCreateDraft_ForkFromOpenDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, _, _, _, _ := seedLiveAgency(t, mgr, ctx)

	d1, err := mgr.CreateDraft(ctx, "first draft", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft d1: %v", err)
	}

	d2, err := mgr.CreateDraft(ctx, "forked from first draft", d1.ID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft d2: %v", err)
	}

	d2Goals, err := mgr.ListGoals(ctx, d2.ID, mwanachamaagency.GoalFilter{})
	if err != nil {
		t.Fatalf("ListGoals(d2): %v", err)
	}
	if len(d2Goals) != 1 {
		t.Fatalf("expected d2 to inherit d1's one goal, got %d", len(d2Goals))
	}
}

func TestCreateDraft_ForkFromNonOpenDraftRefused(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, _, _, _, _ := seedLiveAgency(t, mgr, ctx)

	d1, err := mgr.CreateDraft(ctx, "will be archived", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if _, err := mgr.ArchiveDraft(ctx, d1.ID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}

	if _, err := mgr.CreateDraft(ctx, "forked from archived", d1.ID, "draft"); !errors.Is(err, mwanachamaagency.ErrDraftNotOpen) {
		t.Fatalf("expected ErrDraftNotOpen forking from an archived draft, got %v", err)
	}
}

func TestPromoteDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, g, o, _, _ := seedLiveAgency(t, mgr, ctx)

	role, err := mgr.CreateRole(ctx, models.Role{Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		RoleID: role.ID, ObjectiveID: o.ID, Label: models.RACILabelResponsible,
	}); err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	d, err := mgr.CreateDraft(ctx, "rename the goal", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftGoals, _ := mgr.ListGoals(ctx, d.ID, mwanachamaagency.GoalFilter{})
	edited := draftGoals[0]
	edited.Title = "Grow adoption worldwide"
	if _, err := mgr.UpdateGoal(ctx, edited); err != nil {
		t.Fatalf("UpdateGoal: %v", err)
	}

	draftAssignments, err := mgr.ListRoleAssignments(ctx, d.ID, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil {
		t.Fatalf("ListRoleAssignments(draft): %v", err)
	}
	editedAssignment := draftAssignments[0]
	editedAssignment.Label = models.RACILabelAccountable
	if _, err := mgr.UpdateRoleAssignment(ctx, editedAssignment); err != nil {
		t.Fatalf("UpdateRoleAssignment: %v", err)
	}

	promoted, err := mgr.PromoteDraft(ctx, d.ID)
	if err != nil {
		t.Fatalf("PromoteDraft: %v", err)
	}
	if !promoted.Enabled {
		t.Fatal("expected Agency.Enabled to be true after promotion")
	}

	liveGoals, err := mgr.ListGoals(ctx, "", mwanachamaagency.GoalFilter{})
	if err != nil {
		t.Fatalf("ListGoals: %v", err)
	}
	if len(liveGoals) != 1 {
		t.Fatalf("expected exactly 1 live goal after promotion, got %d", len(liveGoals))
	}
	if liveGoals[0].Title != "Grow adoption worldwide" {
		t.Fatalf("expected promoted goal's edited title to be live, got %q", liveGoals[0].Title)
	}
	if liveGoals[0].ID == g.ID {
		t.Fatal("expected the old live goal to have been replaced, not reused")
	}

	liveObjectives, err := mgr.ListObjectives(ctx, "", mwanachamaagency.ObjectiveFilter{})
	if err != nil {
		t.Fatalf("ListObjectives: %v", err)
	}
	if len(liveObjectives) != 1 {
		t.Fatalf("expected exactly 1 live objective after promotion, got %d", len(liveObjectives))
	}
	if liveObjectives[0].ID == o.ID {
		t.Fatal("expected the old live objective to have been replaced, not reused")
	}

	liveAssignments, err := mgr.ListRoleAssignments(ctx, "", mwanachamaagency.RoleAssignmentFilter{})
	if err != nil {
		t.Fatalf("ListRoleAssignments: %v", err)
	}
	if len(liveAssignments) != 1 || liveAssignments[0].Label != models.RACILabelAccountable {
		t.Fatalf("expected the draft's edited Accountable label to be live, got %+v", liveAssignments)
	}
	if liveAssignments[0].ID != editedAssignment.ID {
		t.Fatalf("expected the draft-scoped assignment to be promoted in place (same id, DraftID flipped to live), got %q want %q",
			liveAssignments[0].ID, editedAssignment.ID)
	}

	gotDraft, err := mgr.GetDraft(ctx, d.ID)
	if err != nil {
		t.Fatalf("GetDraft: %v", err)
	}
	if gotDraft.Status != models.DraftStatusPromoted {
		t.Fatalf("expected draft status promoted, got %q", gotDraft.Status)
	}

	if _, err := mgr.PromoteDraft(ctx, d.ID); !errors.Is(err, mwanachamaagency.ErrDraftNotOpen) {
		t.Fatalf("expected ErrDraftNotOpen re-promoting an already-promoted draft, got %v", err)
	}
}

func TestArchiveDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, _, _, _, _ := seedLiveAgency(t, mgr, ctx)

	d, err := mgr.CreateDraft(ctx, "abandoned idea", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	archived, err := mgr.ArchiveDraft(ctx, d.ID)
	if err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}
	if archived.Status != models.DraftStatusArchived {
		t.Fatalf("expected archived status, got %q", archived.Status)
	}

	// Archived draft's copies are left in place, matching source.
	draftGoals, err := mgr.ListGoals(ctx, d.ID, mwanachamaagency.GoalFilter{})
	if err != nil {
		t.Fatalf("ListGoals(archived draft): %v", err)
	}
	if len(draftGoals) != 1 {
		t.Fatalf("expected the archived draft's goal copy to remain, got %d", len(draftGoals))
	}

	if _, err := mgr.ArchiveDraft(ctx, d.ID); !errors.Is(err, mwanachamaagency.ErrDraftNotOpen) {
		t.Fatalf("expected ErrDraftNotOpen re-archiving, got %v", err)
	}
}
