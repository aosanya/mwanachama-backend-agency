package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestListIssues_EmptyDraftID(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.ListIssues(context.Background(), "", mwanachamaagency.IssueFilter{}); !errors.Is(err, mwanachamaagency.ErrDraftIDRequired) {
		t.Fatalf("expected ErrDraftIDRequired, got %v", err)
	}
}

// seedDraftForIssues opens a Draft forked from a fresh live Agency and
// returns it alongside the Draft-scoped Goal it forked in — seedLiveAgency
// (draft_test.go) already gives that Goal one Objective, so tests here
// start from "one Objective, no RoleAssignment" and adjust from there.
func seedDraftForIssues(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context) (models.Draft, models.Goal, models.Objective) {
	t.Helper()
	ag, _, _, _, _ := seedLiveAgency(t, mgr, ctx)
	d, err := mgr.CreateDraft(ctx, "review pass", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	goals, err := mgr.ListGoals(ctx, d.ID, mwanachamaagency.GoalFilter{})
	if err != nil || len(goals) != 1 {
		t.Fatalf("ListGoals(draft): %v, %+v", err, goals)
	}
	objectives, err := mgr.ListObjectives(ctx, d.ID, mwanachamaagency.ObjectiveFilter{})
	if err != nil || len(objectives) != 1 {
		t.Fatalf("ListObjectives(draft): %v, %+v", err, objectives)
	}
	return d, goals[0], objectives[0]
}

// TestListIssues_ObjectiveWithNoAccountable asserts the forked Objective
// (nothing assigned to it yet) surfaces as a warning pointing at it.
func TestListIssues_ObjectiveWithNoAccountable(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	d, _, o := seedDraftForIssues(t, mgr, ctx)

	issues, err := mgr.ListIssues(ctx, d.ID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	found := false
	for _, i := range issues {
		if i.EntityType == "objective" && i.EntityID == o.ID {
			found = true
			if i.Severity != models.IssueSeverityWarning {
				t.Errorf("expected warning severity, got %q", i.Severity)
			}
		}
	}
	if !found {
		t.Fatalf("expected an issue for objective %s, got %+v", o.ID, issues)
	}
}

// TestListIssues_ObjectiveWithAccountable_NoIssue asserts assigning an
// Accountable Role to the draft-scoped Objective clears its warning on the
// next scan.
func TestListIssues_ObjectiveWithAccountable_NoIssue(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	d, _, o := seedDraftForIssues(t, mgr, ctx)

	role, err := mgr.CreateRole(ctx, models.Role{Name: "bug-hunter", DraftID: d.ID})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		RoleID: role.ID, ObjectiveID: o.ID, Label: models.RACILabelAccountable, DraftID: d.ID,
	}); err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	issues, err := mgr.ListIssues(ctx, d.ID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	for _, i := range issues {
		if i.EntityType == "objective" && i.EntityID == o.ID {
			t.Fatalf("expected no issue for objective %s once it has an Accountable, got %+v", o.ID, i)
		}
	}
}

// TestListIssues_GoalWithNoObjectives asserts a Goal with no Objectives
// surfaces as an info-level issue.
func TestListIssues_GoalWithNoObjectives(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	d, _, _ := seedDraftForIssues(t, mgr, ctx)

	lonely, err := mgr.CreateGoal(ctx, models.Goal{Title: "Lonely goal", DraftID: d.ID})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}

	issues, err := mgr.ListIssues(ctx, d.ID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	found := false
	for _, i := range issues {
		if i.EntityType == "goal" && i.EntityID == lonely.ID {
			found = true
			if i.Severity != models.IssueSeverityInfo {
				t.Errorf("expected info severity, got %q", i.Severity)
			}
		}
	}
	if !found {
		t.Fatalf("expected an issue for goal %s, got %+v", lonely.ID, issues)
	}
}

// TestListIssues_Rescans asserts a second call reflects state changed since
// the first — proof the list is regenerated, not merely appended to.
func TestListIssues_Rescans(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	d, _, o := seedDraftForIssues(t, mgr, ctx)

	first, err := mgr.ListIssues(ctx, d.ID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues (1st): %v", err)
	}
	if len(first) != 1 {
		t.Fatalf("expected exactly 1 issue before assigning an Accountable, got %d: %+v", len(first), first)
	}

	role, err := mgr.CreateRole(ctx, models.Role{Name: "bug-hunter", DraftID: d.ID})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		RoleID: role.ID, ObjectiveID: o.ID, Label: models.RACILabelAccountable, DraftID: d.ID,
	}); err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	second, err := mgr.ListIssues(ctx, d.ID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues (2nd): %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("expected the issue to be gone on rescan, got %d: %+v", len(second), second)
	}
}

func TestListIssues_FilterBySeverity(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	d, _, _ := seedDraftForIssues(t, mgr, ctx)
	if _, err := mgr.CreateGoal(ctx, models.Goal{Title: "Lonely goal", DraftID: d.ID}); err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}

	warnings, err := mgr.ListIssues(ctx, d.ID, mwanachamaagency.IssueFilter{Severity: "warning"})
	if err != nil {
		t.Fatalf("ListIssues(warning): %v", err)
	}
	for _, i := range warnings {
		if i.Severity != models.IssueSeverityWarning {
			t.Errorf("filter leaked a %q severity issue", i.Severity)
		}
	}
	if len(warnings) == 0 {
		t.Fatal("expected at least one warning (the unassigned Objective)")
	}
}
