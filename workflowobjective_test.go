package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type workflowObjectiveSeed struct {
	draftID    string
	workflowA  models.Workflow
	workflowB  models.Workflow
	objectiveA models.Objective
	objectiveB models.Objective
}

func seedWorkflowObjectives(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context) workflowObjectiveSeed {
	t.Helper()
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "SWELab"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	d, err := mgr.CreateDraft(ctx, "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	g, err := mgr.CreateGoal(ctx, models.Goal{DraftID: d.ID, Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	s := workflowObjectiveSeed{draftID: d.ID}
	if s.objectiveA, err = mgr.CreateObjective(ctx, models.Objective{DraftID: d.ID, GoalID: g.ID, Title: "Objective A"}); err != nil {
		t.Fatalf("CreateObjective(A): %v", err)
	}
	if s.objectiveB, err = mgr.CreateObjective(ctx, models.Objective{DraftID: d.ID, GoalID: g.ID, Title: "Objective B"}); err != nil {
		t.Fatalf("CreateObjective(B): %v", err)
	}
	if s.workflowA, err = mgr.CreateWorkflow(ctx, models.Workflow{DraftID: d.ID, Name: "Onboarding"}); err != nil {
		t.Fatalf("CreateWorkflow(A): %v", err)
	}
	if s.workflowB, err = mgr.CreateWorkflow(ctx, models.Workflow{DraftID: d.ID, Name: "Offboarding"}); err != nil {
		t.Fatalf("CreateWorkflow(B): %v", err)
	}
	return s
}

func (s workflowObjectiveSeed) link(w models.Workflow, o models.Objective) models.WorkflowObjective {
	return models.WorkflowObjective{DraftID: s.draftID, WorkflowID: w.ID, ObjectiveID: o.ID}
}

func TestWorkflowObjective_CreateGetListDelete(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedWorkflowObjectives(t, mgr, ctx)

	created, err := mgr.CreateWorkflowObjective(ctx, s.link(s.workflowA, s.objectiveA))
	if err != nil {
		t.Fatalf("CreateWorkflowObjective: %v", err)
	}
	if created.ID == "" || created.CreatedAt == "" || created.WorkflowID != s.workflowA.ID || created.ObjectiveID != s.objectiveA.ID || created.DraftID != s.draftID {
		t.Fatalf("unexpected link: %+v", created)
	}

	got, err := mgr.GetWorkflowObjective(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetWorkflowObjective: %v", err)
	}
	if got.ID != created.ID || got.WorkflowID != s.workflowA.ID || got.ObjectiveID != s.objectiveA.ID {
		t.Fatalf("unexpected link read back: %+v", got)
	}

	listed, err := mgr.ListWorkflowObjectives(ctx, s.draftID, mwanachamaagency.WorkflowObjectiveFilter{})
	if err != nil {
		t.Fatalf("ListWorkflowObjectives: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("expected the one link, got %+v", listed)
	}

	if err := mgr.DeleteWorkflowObjective(ctx, created.ID); err != nil {
		t.Fatalf("DeleteWorkflowObjective: %v", err)
	}
	if _, err := mgr.GetWorkflowObjective(ctx, created.ID); !errors.Is(err, mwanachamaagency.ErrWorkflowObjectiveNotFound) {
		t.Fatalf("expected ErrWorkflowObjectiveNotFound after delete, got %v", err)
	}
	if _, err := mgr.GetWorkflow(ctx, s.workflowA.ID); err != nil {
		t.Fatalf("unlinking must leave the workflow in place, got %v", err)
	}
	if _, err := mgr.GetObjective(ctx, s.objectiveA.ID); err != nil {
		t.Fatalf("unlinking must leave the objective in place, got %v", err)
	}
}

func TestWorkflowObjective_ManyToMany(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedWorkflowObjectives(t, mgr, ctx)

	for _, pair := range [][2]any{{s.workflowA, s.objectiveA}, {s.workflowA, s.objectiveB}, {s.workflowB, s.objectiveA}} {
		if _, err := mgr.CreateWorkflowObjective(ctx, s.link(pair[0].(models.Workflow), pair[1].(models.Objective))); err != nil {
			t.Fatalf("CreateWorkflowObjective: %v", err)
		}
	}

	byWorkflow, err := mgr.ListWorkflowObjectives(ctx, s.draftID, mwanachamaagency.WorkflowObjectiveFilter{WorkflowID: s.workflowA.ID})
	if err != nil {
		t.Fatalf("ListWorkflowObjectives(workflow): %v", err)
	}
	if len(byWorkflow) != 2 {
		t.Fatalf("one workflow serving two objectives: got %d links, want 2", len(byWorkflow))
	}

	byObjective, err := mgr.ListWorkflowObjectives(ctx, s.draftID, mwanachamaagency.WorkflowObjectiveFilter{ObjectiveID: s.objectiveA.ID})
	if err != nil {
		t.Fatalf("ListWorkflowObjectives(objective): %v", err)
	}
	if len(byObjective) != 2 {
		t.Fatalf("one objective served by two workflows: got %d links, want 2", len(byObjective))
	}

	both, err := mgr.ListWorkflowObjectives(ctx, s.draftID, mwanachamaagency.WorkflowObjectiveFilter{WorkflowID: s.workflowB.ID, ObjectiveID: s.objectiveB.ID})
	if err != nil {
		t.Fatalf("ListWorkflowObjectives(both): %v", err)
	}
	if len(both) != 0 {
		t.Fatalf("a pair never linked listed %d links, want 0", len(both))
	}
}

func TestWorkflowObjective_SamePairTwiceIsRefused(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedWorkflowObjectives(t, mgr, ctx)

	if _, err := mgr.CreateWorkflowObjective(ctx, s.link(s.workflowA, s.objectiveA)); err != nil {
		t.Fatalf("CreateWorkflowObjective: %v", err)
	}
	_, err := mgr.CreateWorkflowObjective(ctx, s.link(s.workflowA, s.objectiveA))
	if !errors.Is(err, mwanachamaagency.ErrDuplicateWorkflowObjective) {
		t.Fatalf("expected ErrDuplicateWorkflowObjective, got %v", err)
	}

	listed, err := mgr.ListWorkflowObjectives(ctx, s.draftID, mwanachamaagency.WorkflowObjectiveFilter{})
	if err != nil {
		t.Fatalf("ListWorkflowObjectives: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("the refused duplicate left %d links, want 1", len(listed))
	}

	if _, err := mgr.CreateWorkflowObjective(ctx, s.link(s.workflowB, s.objectiveA)); err != nil {
		t.Fatalf("another workflow for the same objective must be allowed, got %v", err)
	}
}

func TestWorkflowObjective_UnknownEndsAreNotFound(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedWorkflowObjectives(t, mgr, ctx)

	_, err := mgr.CreateWorkflowObjective(ctx, models.WorkflowObjective{DraftID: s.draftID, WorkflowID: "unknown", ObjectiveID: s.objectiveA.ID})
	if !errors.Is(err, mwanachamaagency.ErrWorkflowNotFound) {
		t.Fatalf("expected ErrWorkflowNotFound, got %v", err)
	}
	_, err = mgr.CreateWorkflowObjective(ctx, models.WorkflowObjective{DraftID: s.draftID, WorkflowID: s.workflowA.ID, ObjectiveID: "unknown"})
	if !errors.Is(err, mwanachamaagency.ErrObjectiveNotFound) {
		t.Fatalf("expected ErrObjectiveNotFound, got %v", err)
	}
	if _, err := mgr.GetWorkflowObjective(ctx, "unknown"); !errors.Is(err, mwanachamaagency.ErrWorkflowObjectiveNotFound) {
		t.Fatalf("expected ErrWorkflowObjectiveNotFound on get, got %v", err)
	}
	if err := mgr.DeleteWorkflowObjective(ctx, "unknown"); !errors.Is(err, mwanachamaagency.ErrWorkflowObjectiveNotFound) {
		t.Fatalf("expected ErrWorkflowObjectiveNotFound on delete, got %v", err)
	}
}

func TestWorkflowObjective_BothEndsMustShareTheDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedWorkflowObjectives(t, mgr, ctx)

	other, err := mgr.CreateDraft(ctx, "other", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	g, err := mgr.CreateGoal(ctx, models.Goal{DraftID: other.ID, Title: "Elsewhere"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	foreignObjective, err := mgr.CreateObjective(ctx, models.Objective{DraftID: other.ID, GoalID: g.ID, Title: "Elsewhere"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	foreignWorkflow, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: other.ID, Name: "Elsewhere"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	_, err = mgr.CreateWorkflowObjective(ctx, models.WorkflowObjective{DraftID: s.draftID, WorkflowID: s.workflowA.ID, ObjectiveID: foreignObjective.ID})
	if !errors.Is(err, mwanachamaagency.ErrObjectiveNotFound) {
		t.Fatalf("an objective from another draft: expected ErrObjectiveNotFound, got %v", err)
	}
	_, err = mgr.CreateWorkflowObjective(ctx, models.WorkflowObjective{DraftID: s.draftID, WorkflowID: foreignWorkflow.ID, ObjectiveID: s.objectiveA.ID})
	if !errors.Is(err, mwanachamaagency.ErrWorkflowNotFound) {
		t.Fatalf("a workflow from another draft: expected ErrWorkflowNotFound, got %v", err)
	}

	for _, d := range []string{s.draftID, other.ID} {
		listed, err := mgr.ListWorkflowObjectives(ctx, d, mwanachamaagency.WorkflowObjectiveFilter{})
		if err != nil {
			t.Fatalf("ListWorkflowObjectives: %v", err)
		}
		if len(listed) != 0 {
			t.Fatalf("a refused create stored %d links in draft %s", len(listed), d)
		}
	}
}

func TestWorkflowObjective_LinksAreScopedToTheirDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedWorkflowObjectives(t, mgr, ctx)

	if _, err := mgr.CreateWorkflowObjective(ctx, s.link(s.workflowA, s.objectiveA)); err != nil {
		t.Fatalf("CreateWorkflowObjective: %v", err)
	}
	other, err := mgr.CreateDraft(ctx, "other", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	listed, err := mgr.ListWorkflowObjectives(ctx, other.ID, mwanachamaagency.WorkflowObjectiveFilter{})
	if err != nil {
		t.Fatalf("ListWorkflowObjectives: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("another draft listed %d links, want 0", len(listed))
	}
}

func TestWorkflowObjective_DeletingTheWorkflowTakesItsLinksOnly(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedWorkflowObjectives(t, mgr, ctx)

	gone, err := mgr.CreateWorkflowObjective(ctx, s.link(s.workflowA, s.objectiveA))
	if err != nil {
		t.Fatalf("CreateWorkflowObjective: %v", err)
	}
	goneToo, err := mgr.CreateWorkflowObjective(ctx, s.link(s.workflowA, s.objectiveB))
	if err != nil {
		t.Fatalf("CreateWorkflowObjective: %v", err)
	}
	kept, err := mgr.CreateWorkflowObjective(ctx, s.link(s.workflowB, s.objectiveA))
	if err != nil {
		t.Fatalf("CreateWorkflowObjective: %v", err)
	}

	if err := mgr.DeleteWorkflow(ctx, s.workflowA.ID); err != nil {
		t.Fatalf("DeleteWorkflow: %v", err)
	}
	for _, id := range []string{gone.ID, goneToo.ID} {
		if _, err := mgr.GetWorkflowObjective(ctx, id); !errors.Is(err, mwanachamaagency.ErrWorkflowObjectiveNotFound) {
			t.Fatalf("link %s outlived its workflow: %v", id, err)
		}
	}
	if _, err := mgr.GetWorkflowObjective(ctx, kept.ID); err != nil {
		t.Fatalf("another workflow's link was taken with it: %v", err)
	}
	if _, err := mgr.GetObjective(ctx, s.objectiveA.ID); err != nil {
		t.Fatalf("deleting the workflow must leave the objective, got %v", err)
	}
}

func TestWorkflowObjective_DeletingTheObjectiveTakesItsLinksOnly(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedWorkflowObjectives(t, mgr, ctx)

	gone, err := mgr.CreateWorkflowObjective(ctx, s.link(s.workflowA, s.objectiveA))
	if err != nil {
		t.Fatalf("CreateWorkflowObjective: %v", err)
	}
	goneToo, err := mgr.CreateWorkflowObjective(ctx, s.link(s.workflowB, s.objectiveA))
	if err != nil {
		t.Fatalf("CreateWorkflowObjective: %v", err)
	}
	kept, err := mgr.CreateWorkflowObjective(ctx, s.link(s.workflowA, s.objectiveB))
	if err != nil {
		t.Fatalf("CreateWorkflowObjective: %v", err)
	}

	if err := mgr.DeleteObjective(ctx, s.objectiveA.ID); err != nil {
		t.Fatalf("DeleteObjective: %v", err)
	}
	for _, id := range []string{gone.ID, goneToo.ID} {
		if _, err := mgr.GetWorkflowObjective(ctx, id); !errors.Is(err, mwanachamaagency.ErrWorkflowObjectiveNotFound) {
			t.Fatalf("link %s outlived its objective: %v", id, err)
		}
	}
	if _, err := mgr.GetWorkflowObjective(ctx, kept.ID); err != nil {
		t.Fatalf("another objective's link was taken with it: %v", err)
	}
	if _, err := mgr.GetWorkflow(ctx, s.workflowA.ID); err != nil {
		t.Fatalf("deleting the objective must leave the workflow, got %v", err)
	}
}

func TestConcurrentCreateWorkflowObjectiveWithOnePairStoresOneAndClassifiesTheRest(t *testing.T) {
	mgr := newSharedCacheManager(t, "rw")
	ctx := context.Background()
	s := seedWorkflowObjectives(t, mgr, ctx)

	created, refused := raceWriters(6, func() error {
		_, err := mgr.CreateWorkflowObjective(ctx, s.link(s.workflowA, s.objectiveA))
		return err
	})
	if created != 1 {
		t.Fatalf("%d links stored for one workflow and objective pair, want exactly 1", created)
	}
	for _, err := range refused {
		if !errors.Is(err, mwanachamaagency.ErrDuplicateWorkflowObjective) && !errors.Is(err, mwanachamaagency.ErrConflict) {
			t.Errorf("the losing writer saw %v, want ErrDuplicateWorkflowObjective or ErrConflict", err)
		}
	}
	listed, err := mgr.ListWorkflowObjectives(ctx, s.draftID, mwanachamaagency.WorkflowObjectiveFilter{})
	if err != nil {
		t.Fatalf("ListWorkflowObjectives: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("the draft holds %d links for one pair, want 1", len(listed))
	}
}
