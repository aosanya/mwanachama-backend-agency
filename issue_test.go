package mwanachamaagency_test

import (
	"context"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// TestListIssues_ScopedToDraft asserts a scan is recorded against the
// Draft it ran on, not some implicit shared scope (AGD-006 — there is no
// live row-scope).
func TestListIssues_ScopedToDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := o.DraftID

	issues, err := mgr.ListIssues(ctx, draftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected exactly the unassigned-Objective issue, got %d: %+v", len(issues), issues)
	}
	if issues[0].EntityID != o.ID || issues[0].DraftID != draftID {
		t.Fatalf("expected an issue scoped to draft %s for objective %s, got %+v", draftID, o.ID, issues[0])
	}
}

// TestListIssues_ScopesAreIndependent asserts a scan of one Draft does not
// delete or return another Draft's findings — both live in one table,
// separated only by the draft_id scopeQuery filters on.
func TestListIssues_ScopesAreIndependent(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	draftA := o.DraftID

	draftB, err := mgr.CreateDraft(ctx, "second draft", draftA, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	objectivesB, err := mgr.ListObjectives(ctx, draftB.ID, mwanachamaagency.ObjectiveFilter{})
	if err != nil || len(objectivesB) != 1 {
		t.Fatalf("ListObjectives(draftB): %v, %+v", err, objectivesB)
	}
	roleB, err := mgr.CreateRole(ctx, models.Role{DraftID: draftB.ID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		DraftID: draftB.ID, RoleID: roleB.ID, ObjectiveID: objectivesB[0].ID, Label: models.RACILabelAccountable,
	}); err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	issuesA, err := mgr.ListIssues(ctx, draftA, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues(draftA): %v", err)
	}
	if len(issuesA) != 1 {
		t.Fatalf("expected draftA's unassigned-Objective issue to still be there, got %+v", issuesA)
	}
	for _, i := range issuesA {
		if i.DraftID != draftA {
			t.Fatalf("draftA scan returned an out-of-scope issue: %+v", i)
		}
	}

	issuesB, err := mgr.ListIssues(ctx, draftB.ID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues(draftB): %v", err)
	}
	if len(issuesB) != 0 {
		t.Fatalf("expected draftB's Objective (now Accountable) to have no issue, got %+v", issuesB)
	}

	// Re-scan draftA again — draftB's scan must not have wiped it.
	again, err := mgr.ListIssues(ctx, draftA, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues(draftA, again): %v", err)
	}
	if len(again) != 1 {
		t.Fatalf("draftB's scan affected draftA's issues: %+v", again)
	}
}

// seedDraftForIssues opens a Draft forked from a fresh seed and returns it
// alongside the Draft-scoped Goal it forked in — seedLiveAgency
// (draft_test.go) already gives that Goal one Objective, so tests here
// start from "one Objective, no RoleAssignment" and adjust from there.
func seedDraftForIssues(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context) (models.Draft, models.Goal, models.Objective) {
	t.Helper()
	_, g, _, _, _ := seedLiveAgency(t, mgr, ctx)
	d, err := mgr.CreateDraft(ctx, "review pass", g.DraftID, "draft")
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
// (nothing assigned to it yet) surfaces as an error pointing at it.
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
			if i.Severity != models.IssueSeverityError {
				t.Errorf("expected error severity, got %q", i.Severity)
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
// surfaces as an error-level issue.
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
			if i.Severity != models.IssueSeverityError {
				t.Errorf("expected error severity, got %q", i.Severity)
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

	errs, err := mgr.ListIssues(ctx, d.ID, mwanachamaagency.IssueFilter{Severity: "error"})
	if err != nil {
		t.Fatalf("ListIssues(error): %v", err)
	}
	for _, i := range errs {
		if i.Severity != models.IssueSeverityError {
			t.Errorf("filter leaked a %q severity issue", i.Severity)
		}
	}
	if len(errs) == 0 {
		t.Fatal("expected at least one error (the unassigned Objective)")
	}

	warnings, err := mgr.ListIssues(ctx, d.ID, mwanachamaagency.IssueFilter{Severity: "warning"})
	if err != nil {
		t.Fatalf("ListIssues(warning): %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("no check emits a warning yet, got %+v", warnings)
	}
}
