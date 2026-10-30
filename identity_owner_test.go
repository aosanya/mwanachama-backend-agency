package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/identity"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

var _ identity.Owner = mwanachamaagency.AgencyManager(nil)

type ownedThings struct {
	draftID  string
	byObject map[string]string
}

func seedOwnedThings(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context) ownedThings {
	t.Helper()
	draftID := newDraftAgency(t, mgr, ctx)
	goal, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, Title: "Grow"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	objective, err := mgr.CreateObjective(ctx, models.Objective{DraftID: draftID, GoalID: goal.ID, Title: "Reach"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	workflow, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftID, Name: "Onboarding"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	workItem, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: draftID, WorkflowID: workflow.ID, ObjectiveID: objective.ID, Title: "Charter"})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "Chair"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	value, err := mgr.CreateValue(ctx, models.Value{DraftID: draftID, Text: "Be honest"})
	if err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	rule, err := mgr.CreateRule(ctx, models.Rule{DraftID: draftID, Text: "Publish minutes"})
	if err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	return ownedThings{draftID: draftID, byObject: map[string]string{
		"goal": goal.ID, "objective": objective.ID, "workflow": workflow.ID,
		"work_item": workItem.ID, "role": role.ID, "value": value.ID, "rule": rule.ID,
	}}
}

func TestStableIdentityIsTheRawAgencyInstanceCodeOfEachObject(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	things := seedOwnedThings(t, mgr, ctx)

	want := map[string]string{
		"goal": "agency/test/G-1", "objective": "agency/test/O-1", "workflow": "agency/test/W-1",
		"work_item": "agency/test/WI-1", "role": "agency/test/R-1", "value": "agency/test/V-1", "rule": "agency/test/RU-1",
	}
	for object, id := range things.byObject {
		got, err := mgr.StableIdentity(ctx, object, id)
		if err != nil {
			t.Fatalf("StableIdentity(%s): %v", object, err)
		}
		if got != want[object] {
			t.Errorf("StableIdentity(%s) = %q, want %q", object, got, want[object])
		}
	}
}

func TestResolveGivesBackTheObjectAndIdStableIdentityWasAskedFor(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	things := seedOwnedThings(t, mgr, ctx)

	for object, id := range things.byObject {
		raw, err := mgr.StableIdentity(ctx, object, id)
		if err != nil {
			t.Fatalf("StableIdentity(%s): %v", object, err)
		}
		gotObject, gotID, err := mgr.Resolve(ctx, raw)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", raw, err)
		}
		if gotObject != object || gotID != id {
			t.Errorf("Resolve(%q) = (%q, %q), want (%q, %q)", raw, gotObject, gotID, object, id)
		}
	}
}

func TestAStableIdentitySurvivesAForkWhileTheRowIdDoesNot(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	things := seedOwnedThings(t, mgr, ctx)
	oldRaw, err := mgr.StableIdentity(ctx, "role", things.byObject["role"])
	if err != nil {
		t.Fatalf("StableIdentity: %v", err)
	}

	fork, err := mgr.CreateDraft(ctx, "next", things.draftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	roles, err := mgr.ListRoles(ctx, fork.ID, mwanachamaagency.RoleFilter{})
	if err != nil || len(roles) != 1 {
		t.Fatalf("ListRoles of the fork = %v, %v", roles, err)
	}
	if roles[0].ID == things.byObject["role"] {
		t.Fatalf("precondition: the fork re-mints row ids")
	}

	forkRaw, err := mgr.StableIdentity(ctx, "role", roles[0].ID)
	if err != nil {
		t.Fatalf("StableIdentity of the fork's role: %v", err)
	}
	if forkRaw != oldRaw {
		t.Errorf("the forked role is %q, the original %q; a promotion must keep the identity", forkRaw, oldRaw)
	}

	object, id, err := mgr.Resolve(ctx, oldRaw)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if object != "role" || id != roles[0].ID {
		t.Errorf("Resolve(%q) = (%q, %q), want the newest open draft's row (role, %q)", oldRaw, object, id, roles[0].ID)
	}
}

func TestResolveSkipsAnArchivedDraftAndAnswersNotFoundWhenNoDraftIsOpen(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	things := seedOwnedThings(t, mgr, ctx)
	raw, err := mgr.StableIdentity(ctx, "goal", things.byObject["goal"])
	if err != nil {
		t.Fatalf("StableIdentity: %v", err)
	}
	if _, err := mgr.ArchiveDraft(ctx, things.draftID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}
	if _, _, err := mgr.Resolve(ctx, raw); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("Resolve with no open draft: want ErrNotFound, got %v", err)
	}
}

func TestResolveNamesAMissAndARawItDoesNotOwnApart(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedOwnedThings(t, mgr, ctx)

	cases := []struct {
		raw  string
		want error
	}{
		{"agency/test/G-99", identity.ErrNotFound},
		{"agency/test/ZZ-1", identity.ErrNotFound},
		{"git/test/G-1", identity.ErrNotOwned},
		{"agency/elsewhere/G-1", identity.ErrNotOwned},
		{"agency/test", identity.ErrInvalidRaw},
		{"", identity.ErrInvalidRaw},
	}
	for _, c := range cases {
		if _, _, err := mgr.Resolve(ctx, c.raw); !errors.Is(err, c.want) {
			t.Errorf("Resolve(%q): want %v, got %v", c.raw, c.want, err)
		}
	}
}

func TestStableIdentityNamesAMissAndAnObjectWithNoIdentityApart(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	things := seedOwnedThings(t, mgr, ctx)

	if _, err := mgr.StableIdentity(ctx, "goal", "no-such-row"); !errors.Is(err, identity.ErrNotFound) {
		t.Errorf("a missing row: want ErrNotFound, got %v", err)
	}
	if _, err := mgr.StableIdentity(ctx, "draft", things.draftID); !errors.Is(err, mwanachamaagency.ErrNoStableIdentity) {
		t.Errorf("a draft has no code of its own: want ErrNoStableIdentity, got %v", err)
	}
}
