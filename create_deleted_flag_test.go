package mwanachamaagency_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAG57_EveryCreateMintsItsOwnDeletedFlag(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)

	goal, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, Title: "G", Deleted: true})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	objective, err := mgr.CreateObjective(ctx, models.Objective{DraftID: draftID, GoalID: goal.ID, Title: "O", Deleted: true})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "R", Deleted: true})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	workflow, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftID, Name: "W", Deleted: true})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	workItem, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: draftID, WorkflowID: workflow.ID, Title: "WI", Deleted: true})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	link, err := mgr.CreateWorkflowObjective(ctx, models.WorkflowObjective{DraftID: draftID, WorkflowID: workflow.ID, ObjectiveID: objective.ID, Deleted: true})
	if err != nil {
		t.Fatalf("CreateWorkflowObjective: %v", err)
	}
	assignment, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: draftID, WorkItemID: workItem.ID, RoleID: role.ID, Label: models.RACILabelResponsible, Deleted: true})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}
	value, err := mgr.CreateValue(ctx, models.Value{DraftID: draftID, Text: "V", Deleted: true})
	if err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	rule, err := mgr.CreateRule(ctx, models.Rule{DraftID: draftID, Text: "RU", Deleted: true})
	if err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	module, err := mgr.CreateModule(ctx, models.Module{DraftID: draftID, Code: "widgets", Name: "Widgets", Deleted: true})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}

	got := map[string]bool{}
	read := func(name string, deleted bool, err error) {
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		got[name] = deleted
	}
	g, err := mgr.GetGoal(ctx, goal.ID)
	read("goal", g.Deleted, err)
	o, err := mgr.GetObjective(ctx, objective.ID)
	read("objective", o.Deleted, err)
	r, err := mgr.GetRole(ctx, role.ID)
	read("role", r.Deleted, err)
	w, err := mgr.GetWorkflow(ctx, workflow.ID)
	read("workflow", w.Deleted, err)
	wi, err := mgr.GetWorkItem(ctx, workItem.ID)
	read("workitem", wi.Deleted, err)
	wo, err := mgr.GetWorkflowObjective(ctx, link.ID)
	read("workflowobjective", wo.Deleted, err)
	ra, err := mgr.GetRoleAssignment(ctx, assignment.ID)
	read("roleassignment", ra.Deleted, err)
	v, err := mgr.GetValue(ctx, value.ID)
	read("value", v.Deleted, err)
	ru, err := mgr.GetRule(ctx, rule.ID)
	read("rule", ru.Deleted, err)
	mo, err := mgr.GetModule(ctx, module.ID)
	read("module", mo.Deleted, err)

	for name, deleted := range got {
		if deleted {
			t.Errorf("AG57: Create%s stored a body-set deleted=true, want false", name)
		}
	}
	if goal.Deleted || objective.Deleted || role.Deleted || workflow.Deleted || workItem.Deleted || link.Deleted || assignment.Deleted || value.Deleted || rule.Deleted || module.Deleted {
		t.Errorf("AG57: a Create answered a row already marked deleted")
	}
}
