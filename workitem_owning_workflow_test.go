package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestRV23_CreateWorkItemInForeignOrMissingWorkflow(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedRaci(t, mgr, ctx)
	d2, _ := mgr.CreateDraft(ctx, "o", "", "")
	w2, _ := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: d2.ID, Name: "W2"})
	if _, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: s.draftID, WorkflowID: w2.ID, Title: "x"}); !errors.Is(err, mwanachamaagency.ErrWorkflowNotFound) {
		t.Errorf("created a step in draft A under a workflow of draft B: %v", err)
	}
	if _, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: s.draftID, WorkflowID: "nope", Title: "x"}); !errors.Is(err, mwanachamaagency.ErrWorkflowNotFound) {
		t.Errorf("created a step under a workflow that does not exist: %v", err)
	}
	if _, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: s.draftID, Title: "x"}); !errors.Is(err, mwanachamaagency.ErrWorkflowNotFound) {
		t.Errorf("created a step that names no workflow: %v", err)
	}
	wi := s.step
	wi.WorkflowID = w2.ID
	if _, err := mgr.UpdateWorkItem(ctx, wi); !errors.Is(err, mwanachamaagency.ErrWorkflowNotFound) {
		t.Errorf("moved a step to another draft's workflow: %v", err)
	}
}

func TestRV11_CrossDraftRepointing(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedRaci(t, mgr, ctx)
	d2, _ := mgr.CreateDraft(ctx, "other", "", "")
	w2, _ := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: d2.ID, Name: "W2"})
	st2, _ := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: d2.ID, WorkflowID: w2.ID, Title: "x"})
	r2, _ := mgr.CreateRole(ctx, models.Role{DraftID: d2.ID, Name: "r2"})
	a, _ := mgr.CreateRoleAssignment(ctx, s.assignment(s.roleA.ID, models.RACILabelResponsible))

	b := a
	b.WorkItemID = st2.ID
	if _, err := mgr.UpdateRoleAssignment(ctx, b); err == nil {
		t.Errorf("update repointed to foreign step")
	}
	b = a
	b.RoleID = r2.ID
	if _, err := mgr.UpdateRoleAssignment(ctx, b); err == nil {
		t.Errorf("update repointed to foreign role")
	}
	wi := s.step
	wi.WorkflowID = w2.ID
	if _, err := mgr.UpdateWorkItem(ctx, wi); err == nil {
		t.Errorf("step moved to a workflow in another draft")
	}
	ras, _ := mgr.ListRoleAssignments(ctx, d2.ID, mwanachamaagency.RoleAssignmentFilter{})
	if len(ras) != 0 {
		t.Errorf("foreign draft holds assignments")
	}
	ras, _ = mgr.ListRoleAssignmentsForWorkItem(ctx, s.step.ID, d2.ID, mwanachamaagency.RoleAssignmentFilter{})
	if len(ras) != 0 {
		t.Errorf("list for work item leaks across draft")
	}
}

func TestRV24_FullPathSeesAssignmentsThroughForeignWorkflowMove(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedRaci(t, mgr, ctx)
	g, _ := mgr.CreateGoal(ctx, models.Goal{DraftID: s.draftID, Title: "G"})
	o, _ := mgr.CreateObjective(ctx, models.Objective{DraftID: s.draftID, GoalID: g.ID, Title: "O"})
	mgr.CreateWorkflowObjective(ctx, models.WorkflowObjective{DraftID: s.draftID, WorkflowID: s.step.WorkflowID, ObjectiveID: o.ID})
	mgr.CreateRoleAssignment(ctx, s.assignment(s.roleA.ID, models.RACILabelResponsible))
	d2, _ := mgr.CreateDraft(ctx, "o", "", "")
	w2, _ := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: d2.ID, Name: "W2"})
	wi := s.step
	wi.WorkflowID = w2.ID
	mgr.UpdateWorkItem(ctx, wi)
	if err := mgr.DeleteWorkflow(ctx, w2.ID); err != nil {
		t.Fatal(err)
	}
	steps, _ := mgr.ListWorkItems(ctx, s.draftID, mwanachamaagency.WorkItemFilter{})
	for _, st := range steps {
		if st.ID == s.step.ID && st.WorkflowID == w2.ID {
			t.Errorf("draft A's step still points at a workflow of draft B that no longer exists")
		}
	}
	d3, err := mgr.CreateDraft(ctx, "fork", s.draftID, "draft")
	if err != nil {
		t.Fatal(err)
	}
	fs, _ := mgr.ListWorkItems(ctx, d3.ID, mwanachamaagency.WorkItemFilter{})
	for _, st := range fs {
		if st.WorkflowID == w2.ID {
			t.Errorf("fork inherited the foreign pointer")
		}
	}
}
