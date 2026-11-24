package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type stepSeed struct {
	draftID string
	step    models.WorkItem
	other   models.WorkItem
	roleA   models.Role
	roleB   models.Role
}

func seedStepsAndRoles(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context) stepSeed {
	t.Helper()
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "SWELab"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	d, err := mgr.CreateDraft(ctx, "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	w, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: d.ID, Name: "Onboarding"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	step, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: d.ID, WorkflowID: w.ID, Title: "Write charter"})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	other, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: d.ID, WorkflowID: w.ID, Title: "Review charter"})
	if err != nil {
		t.Fatalf("CreateWorkItem(other): %v", err)
	}
	roleA, err := mgr.CreateRole(ctx, models.Role{DraftID: d.ID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole(A): %v", err)
	}
	roleB, err := mgr.CreateRole(ctx, models.Role{DraftID: d.ID, Name: "tech-lead"})
	if err != nil {
		t.Fatalf("CreateRole(B): %v", err)
	}
	return stepSeed{draftID: d.ID, step: step, other: other, roleA: roleA, roleB: roleB}
}

func TestRV23_CreateWorkItemInForeignOrMissingWorkflow(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedStepsAndRoles(t, mgr, ctx)
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
	s := seedStepsAndRoles(t, mgr, ctx)
	d2, _ := mgr.CreateDraft(ctx, "other", "", "")
	w2, _ := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: d2.ID, Name: "W2"})
	r2, _ := mgr.CreateRole(ctx, models.Role{DraftID: d2.ID, Name: "r2"})

	foreignRole := s.step
	foreignRole.AccountableRoleID = r2.ID
	if _, err := mgr.UpdateWorkItem(ctx, foreignRole); err == nil {
		t.Errorf("update named a foreign role as accountable")
	}
	wi := s.step
	wi.WorkflowID = w2.ID
	if _, err := mgr.UpdateWorkItem(ctx, wi); err == nil {
		t.Errorf("step moved to a workflow in another draft")
	}
	steps, _ := mgr.ListWorkItems(ctx, d2.ID, mwanachamaagency.WorkItemFilter{})
	for _, st := range steps {
		if st.AccountableRoleID != "" {
			t.Errorf("foreign draft holds an accountable role: %+v", st)
		}
	}
}

func TestRV24_FullPathThroughForeignWorkflowMove(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedStepsAndRoles(t, mgr, ctx)
	g, _ := mgr.CreateGoal(ctx, models.Goal{DraftID: s.draftID, Title: "G"})
	o, _ := mgr.CreateObjective(ctx, models.Objective{DraftID: s.draftID, GoalID: g.ID, Title: "O"})
	mgr.CreateWorkflowObjective(ctx, models.WorkflowObjective{DraftID: s.draftID, WorkflowID: s.step.WorkflowID, ObjectiveID: o.ID})
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
