package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// newDraftAgency creates an Agency and one empty open Draft, returning
// the Draft's id — every business-entity row belongs to a real Draft
// (AGD-006), so this is the minimum setup any Create<Type> call needs.
func newDraftAgency(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context) string {
	t.Helper()
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "SWELab"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	d, err := mgr.CreateDraft(ctx, "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	return d.ID
}

func TestGoalCRUD(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)

	g, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, Title: "Grow adoption", Ordinality: 1})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if g.ID == "" {
		t.Fatal("expected minted ID")
	}

	got, err := mgr.GetGoal(ctx, g.ID)
	if err != nil {
		t.Fatalf("GetGoal: %v", err)
	}
	if got.Title != "Grow adoption" {
		t.Fatalf("unexpected goal: %+v", got)
	}

	g.Title = "Grow adoption globally"
	g.Ordinality = 0 // exercise the zero-value update path
	updated, err := mgr.UpdateGoal(ctx, g)
	if err != nil {
		t.Fatalf("UpdateGoal: %v", err)
	}
	if updated.Title != "Grow adoption globally" || updated.Ordinality != 0 {
		t.Fatalf("update did not apply zero-value field: %+v", updated)
	}

	goals, err := mgr.ListGoals(ctx, draftID, mwanachamaagency.GoalFilter{})
	if err != nil {
		t.Fatalf("ListGoals: %v", err)
	}
	if len(goals) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(goals))
	}
}

func TestGetGoal_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.GetGoal(context.Background(), "unknown"); !errors.Is(err, mwanachamaagency.ErrGoalNotFound) {
		t.Fatalf("expected ErrGoalNotFound, got %v", err)
	}
}

func TestDeleteGoal_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if err := mgr.DeleteGoal(context.Background(), "unknown"); !errors.Is(err, mwanachamaagency.ErrGoalNotFound) {
		t.Fatalf("expected ErrGoalNotFound, got %v", err)
	}
}

// TestDeleteGoal_CascadesObjectivesAndRoleAssignments asserts DeleteGoal
// takes its Objectives (and those Objectives' RoleAssignments) down with
// it (see AgencyManager's DeleteGoal doc).
func TestDeleteGoal_CascadesObjectivesAndRoleAssignments(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, o, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := g.DraftID

	role, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	assignment, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		DraftID: draftID, RoleID: role.ID, ObjectiveID: o.ID, Label: models.RACILabelAccountable,
	})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	if err := mgr.DeleteGoal(ctx, g.ID); err != nil {
		t.Fatalf("DeleteGoal: %v", err)
	}

	if _, err := mgr.GetGoal(ctx, g.ID); !errors.Is(err, mwanachamaagency.ErrGoalNotFound) {
		t.Fatalf("expected the Goal itself to be gone, got %v", err)
	}
	if _, err := mgr.GetObjective(ctx, o.ID); !errors.Is(err, mwanachamaagency.ErrObjectiveNotFound) {
		t.Fatalf("expected the cascaded Objective to be gone, got %v", err)
	}
	if _, err := mgr.GetRoleAssignment(ctx, assignment.ID); !errors.Is(err, mwanachamaagency.ErrRoleAssignmentNotFound) {
		t.Fatalf("expected the cascaded RoleAssignment to be gone, got %v", err)
	}
	// The Role itself is a separate entity, only referenced by the
	// assignment — it must survive the Goal's deletion untouched.
	if _, err := mgr.GetRole(ctx, role.ID); err != nil {
		t.Fatalf("expected the Role to survive the Goal's deletion, got %v", err)
	}
}

// TestDeleteGoal_NoObjectives asserts deleting a Goal with nothing under
// it is just a plain delete — the cascade's "any objectives?" branch must
// not choke on the empty case.
func TestDeleteGoal_NoObjectives(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)
	g, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, Title: "Lonely goal"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if err := mgr.DeleteGoal(ctx, g.ID); err != nil {
		t.Fatalf("DeleteGoal: %v", err)
	}
	if _, err := mgr.GetGoal(ctx, g.ID); !errors.Is(err, mwanachamaagency.ErrGoalNotFound) {
		t.Fatalf("expected ErrGoalNotFound, got %v", err)
	}
}

// TestGoalCode_SequentialAndImmutable asserts Code is minted "G-1", "G-2",
// ... in creation order, and survives an Update untouched even when the
// caller's struct carries a stale/empty Code (see UpdateGoal's doc for why
// it preserves the persisted Code rather than trusting the input).
func TestGoalCode_SequentialAndImmutable(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)

	g1, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, Title: "First"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if g1.Code != "G-1" {
		t.Fatalf("expected Code %q, got %q", "G-1", g1.Code)
	}

	g2, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, Title: "Second"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if g2.Code != "G-2" {
		t.Fatalf("expected Code %q, got %q", "G-2", g2.Code)
	}

	g1.Title = "First, renamed"
	updated, err := mgr.UpdateGoal(ctx, g1)
	if err != nil {
		t.Fatalf("UpdateGoal: %v", err)
	}
	if updated.Code != "G-1" {
		t.Fatalf("expected Code to stay %q after update, got %q", "G-1", updated.Code)
	}

	got, err := mgr.GetGoal(ctx, g1.ID)
	if err != nil {
		t.Fatalf("GetGoal: %v", err)
	}
	if got.Code != "G-1" {
		t.Fatalf("expected persisted Code to stay %q after update, got %q", "G-1", got.Code)
	}
}

// TestGoalCode_SurvivesDraftFork asserts a Goal's Code carries unchanged
// from the source row, through its draft-scoped copy (which gets a
// brand-new ID at fork time, AG16), through an edit of that copy.
func TestGoalCode_SurvivesDraftFork(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, _, _, _ := seedLiveAgency(t, mgr, ctx)
	originalCode := g.Code
	if originalCode == "" {
		t.Fatal("expected seeded goal to have a non-empty Code")
	}

	d, err := mgr.CreateDraft(ctx, "rename the goal", g.DraftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftGoals, err := mgr.ListGoals(ctx, d.ID, mwanachamaagency.GoalFilter{})
	if err != nil {
		t.Fatalf("ListGoals(draft): %v", err)
	}
	edited := draftGoals[0]
	if edited.ID == g.ID {
		t.Fatal("expected the draft-scoped copy to have a new id")
	}
	if edited.Code != originalCode {
		t.Fatalf("expected draft-scoped copy to carry the same Code %q, got %q", originalCode, edited.Code)
	}
	edited.Title = "Grow adoption worldwide"
	updated, err := mgr.UpdateGoal(ctx, edited)
	if err != nil {
		t.Fatalf("UpdateGoal: %v", err)
	}
	if updated.Code != originalCode {
		t.Fatalf("expected edited copy's Code to survive unchanged, got %q want %q", updated.Code, originalCode)
	}
}
