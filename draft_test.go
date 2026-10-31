package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func seedLiveAgency(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context) (models.Agency, models.Goal, models.Objective, models.Workflow, models.WorkItem) {
	t.Helper()
	ag, err := mgr.CreateAgency(ctx, models.Agency{Name: "SWELab"})
	if err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	d, err := mgr.CreateDraft(ctx, "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft (seed): %v", err)
	}
	g, err := mgr.CreateGoal(ctx, models.Goal{DraftID: d.ID, Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	o, err := mgr.CreateObjective(ctx, models.Objective{DraftID: d.ID, GoalID: g.ID, Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	w, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: d.ID, Name: "Onboarding"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	if _, err := mgr.CreateWorkflowObjective(ctx, models.WorkflowObjective{DraftID: d.ID, WorkflowID: w.ID, ObjectiveID: o.ID}); err != nil {
		t.Fatalf("CreateWorkflowObjective: %v", err)
	}
	wi, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: d.ID, WorkflowID: w.ID, Title: "Write charter"})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	return ag, g, o, w, wi
}

func TestCreateDraft_Empty(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "SWELab"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}

	d, err := mgr.CreateDraft(ctx, "brand new", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if d.Status != models.DraftStatusOpen {
		t.Fatalf("expected new draft to be open, got %q", d.Status)
	}

	goals, err := mgr.ListGoals(ctx, d.ID, mwanachamaagency.GoalFilter{})
	if err != nil {
		t.Fatalf("ListGoals: %v", err)
	}
	if len(goals) != 0 {
		t.Fatalf("expected an empty new draft to have no goals, got %+v", goals)
	}
}

func TestCreateDraft_InvalidForkSource(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "SWELab"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}

	if _, err := mgr.CreateDraft(ctx, "bogus", "whatever", "live"); !errors.Is(err, mwanachamaagency.ErrInvalidForkSource) {
		t.Fatalf("expected ErrInvalidForkSource for a \"live\" fork type, got %v", err)
	}
}

func TestCreateDraft_ForkFromDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, o, w, wi := seedLiveAgency(t, mgr, ctx)
	seedDraftID := g.DraftID

	d, err := mgr.CreateDraft(ctx, "Q3 restructuring", seedDraftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if d.Status != models.DraftStatusOpen {
		t.Fatalf("expected new draft to be open, got %q", d.Status)
	}

	seedGoals, _ := mgr.ListGoals(ctx, seedDraftID, mwanachamaagency.GoalFilter{})
	if len(seedGoals) != 1 || seedGoals[0].ID != g.ID {
		t.Fatalf("seed draft's goals should be unchanged, got %+v", seedGoals)
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

	links, err := mgr.ListWorkflowObjectives(ctx, d.ID, mwanachamaagency.WorkflowObjectiveFilter{})
	if err != nil {
		t.Fatalf("ListWorkflowObjectives(draft): %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected one workflow objective copy, got %+v", links)
	}
	if links[0].WorkflowID != draftWorkflows[0].ID || links[0].ObjectiveID != draftObjectives[0].ID {
		t.Fatalf("expected the workflow objective remapped onto both draft copies, got %+v want workflow %q objective %q",
			links[0], draftWorkflows[0].ID, draftObjectives[0].ID)
	}
	seedLinks, err := mgr.ListWorkflowObjectives(ctx, seedDraftID, mwanachamaagency.WorkflowObjectiveFilter{})
	if err != nil || len(seedLinks) != 1 || seedLinks[0].WorkflowID != w.ID || seedLinks[0].ObjectiveID != o.ID {
		t.Fatalf("seed draft's workflow objective should be unchanged, got %v / %+v", err, seedLinks)
	}
	if seedLinks[0].ID == links[0].ID {
		t.Fatalf("expected the fork to mint a new workflow objective id, got %q twice", links[0].ID)
	}
}

func TestCreateDraft_ForksRoleAssignments(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, wi := seedLiveAgency(t, mgr, ctx)
	seedDraftID := o.DraftID

	role, err := mgr.CreateRole(ctx, models.Role{DraftID: seedDraftID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	assignment, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		DraftID: seedDraftID, RoleID: role.ID, WorkItemID: wi.ID, Label: models.RACILabelAccountable,
	})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	d, err := mgr.CreateDraft(ctx, "restaff the objective", seedDraftID, "draft")
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

	draftItems, _ := mgr.ListWorkItems(ctx, d.ID, mwanachamaagency.WorkItemFilter{})
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
	if len(draftItems) != 1 || draftAssignments[0].WorkItemID != draftItems[0].ID {
		t.Fatalf("expected RoleAssignment.WorkItemID remapped to the draft-scoped step copy, got %q want %+v",
			draftAssignments[0].WorkItemID, draftItems)
	}
	if draftAssignments[0].Label != models.RACILabelAccountable {
		t.Fatalf("expected the label carried across the fork, got %q", draftAssignments[0].Label)
	}
}

func TestCreateDraft_ForkCopiesTheStoredModuleFlag(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	seedDraftID := o.DraftID

	mod, err := mgr.CreateModule(ctx, models.Module{DraftID: seedDraftID, Code: "widgets", Name: "Widgets", Enabled: true})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	if !mod.Enabled || !mod.Explicit {
		t.Fatalf("a module created switched on reads %+v", mod)
	}
	git := moduleByCode(t, mgr, ctx, seedDraftID, "git")
	if _, err := mgr.SetModuleEnabled(ctx, seedDraftID, git.ID, true); err != nil {
		t.Fatalf("SetModuleEnabled: %v", err)
	}

	d, err := mgr.CreateDraft(ctx, "carry the switches", seedDraftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	widgets := moduleByCode(t, mgr, ctx, d.ID, "widgets")
	if widgets.ID == mod.ID || !widgets.Explicit || !widgets.Enabled {
		t.Fatalf("expected a new widgets row carrying the switch, got %+v", widgets)
	}
	forkedGit := moduleByCode(t, mgr, ctx, d.ID, "git")
	if forkedGit.ID == git.ID || !forkedGit.Explicit || !forkedGit.Enabled {
		t.Fatalf("expected a new git row carrying the switch, got %+v", forkedGit)
	}
	if moduleByCode(t, mgr, ctx, d.ID, "taskmanager").Explicit {
		t.Fatal("a module nobody switched on was switched on by the fork")
	}
}

func TestCreateDraft_ForkFromOpenDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, _, _, _ := seedLiveAgency(t, mgr, ctx)
	seedDraftID := g.DraftID

	d2, err := mgr.CreateDraft(ctx, "forked from seed", seedDraftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft d2: %v", err)
	}

	d2Goals, err := mgr.ListGoals(ctx, d2.ID, mwanachamaagency.GoalFilter{})
	if err != nil {
		t.Fatalf("ListGoals(d2): %v", err)
	}
	if len(d2Goals) != 1 {
		t.Fatalf("expected d2 to inherit the seed draft's one goal, got %d", len(d2Goals))
	}
}

func TestCreateDraft_ForkFromNonOpenDraftRefused(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, _, _, _ := seedLiveAgency(t, mgr, ctx)
	seedDraftID := g.DraftID

	if _, err := mgr.ArchiveDraft(ctx, seedDraftID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}

	if _, err := mgr.CreateDraft(ctx, "forked from archived", seedDraftID, "draft"); !errors.Is(err, mwanachamaagency.ErrDraftNotOpen) {
		t.Fatalf("expected ErrDraftNotOpen forking from an archived draft, got %v", err)
	}
}

func TestArchiveDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, _, _, _ := seedLiveAgency(t, mgr, ctx)
	seedDraftID := g.DraftID

	d, err := mgr.CreateDraft(ctx, "abandoned idea", seedDraftID, "draft")
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

func TestCreateDraft_ForkOfAForkKeepsBothJoinsAndCodes(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, w, wi := seedLiveAgency(t, mgr, ctx)
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		DraftID: o.DraftID, RoleID: role.ID, WorkItemID: wi.ID, Label: models.RACILabelResponsible,
	}); err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	first, err := mgr.CreateDraft(ctx, "first", o.DraftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft(first): %v", err)
	}
	second, err := mgr.CreateDraft(ctx, "second", first.ID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft(second): %v", err)
	}

	workflows, _ := mgr.ListWorkflows(ctx, second.ID, mwanachamaagency.WorkflowFilter{})
	objectives, _ := mgr.ListObjectives(ctx, second.ID, mwanachamaagency.ObjectiveFilter{})
	items, _ := mgr.ListWorkItems(ctx, second.ID, mwanachamaagency.WorkItemFilter{})
	if len(workflows) != 1 || len(objectives) != 1 || len(items) != 1 {
		t.Fatalf("second fork lost rows: %+v %+v %+v", workflows, objectives, items)
	}
	if workflows[0].Code != w.Code || objectives[0].Code != o.Code || items[0].Code != wi.Code {
		t.Fatalf("codes moved across two forks: workflow %q/%q objective %q/%q step %q/%q",
			workflows[0].Code, w.Code, objectives[0].Code, o.Code, items[0].Code, wi.Code)
	}

	links, err := mgr.ListWorkflowObjectives(ctx, second.ID, mwanachamaagency.WorkflowObjectiveFilter{})
	if err != nil || len(links) != 1 {
		t.Fatalf("ListWorkflowObjectives(second): %v, %+v", err, links)
	}
	if links[0].WorkflowID != workflows[0].ID || links[0].ObjectiveID != objectives[0].ID {
		t.Fatalf("workflow objective not remapped into the second fork: %+v", links[0])
	}
	assignments, err := mgr.ListRoleAssignmentsForWorkItem(ctx, items[0].ID, second.ID, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil || len(assignments) != 1 {
		t.Fatalf("ListRoleAssignmentsForWorkItem(second): %v, %+v", err, assignments)
	}
}
