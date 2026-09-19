package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCreateRule_RequiresDraftID(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.CreateRule(context.Background(), models.Rule{Text: "x"}); !errors.Is(err, mwanachamaagency.ErrDraftRequired) {
		t.Fatalf("expected ErrDraftRequired, got %v", err)
	}
}

func TestCreateRule_RequiresText(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)
	if _, err := mgr.CreateRule(ctx, models.Rule{DraftID: draftID}); !errors.Is(err, mwanachamaagency.ErrInvalidRule) {
		t.Fatalf("expected ErrInvalidRule, got %v", err)
	}
}

// TestRule_AgencyWideNeverForkable mirrors TestValue_AgencyWideNeverForkable.
// A Rule with no attachment is permanently immutable and never forked into
// another Draft — it just sits in whichever Draft it was created in.
func TestRule_AgencyWideNeverForkable(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, _, _, _ := seedLiveAgency(t, mgr, ctx)
	seedDraftID := g.DraftID

	r, err := mgr.CreateRule(ctx, models.Rule{DraftID: seedDraftID, Text: "never touch production data"})
	if err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	if err := mgr.DeleteRule(ctx, r.ID); !errors.Is(err, mwanachamaagency.ErrCannotEditLive) {
		t.Fatalf("expected ErrCannotEditLive, got %v", err)
	}

	d, err := mgr.CreateDraft(ctx, "try to edit the rule", seedDraftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftRules, err := mgr.ListRules(ctx, d.ID, mwanachamaagency.RuleFilter{})
	if err != nil {
		t.Fatalf("ListRules(draft): %v", err)
	}
	if len(draftRules) != 0 {
		t.Fatalf("expected an Agency-wide Rule to never fork into another Draft, got %+v", draftRules)
	}
}

// TestRuleAttachment_MultipleTargets covers a single Rule carrying more
// than one attachment at once (AGD-003's core requirement), plus that
// attaching to two different target types works.
func TestRuleAttachment_MultipleTargets(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, goal, _, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := goal.DraftID

	r, err := mgr.CreateRule(ctx, models.Rule{DraftID: draftID, Text: "never touch production data"})
	if err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	role1, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	role2, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "reviewer"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	if _, err := mgr.CreateRuleAttachment(ctx, r.ID, models.TargetTypeRole, role1.ID); err != nil {
		t.Fatalf("CreateRuleAttachment(role1): %v", err)
	}
	if _, err := mgr.CreateRuleAttachment(ctx, r.ID, models.TargetTypeRole, role2.ID); err != nil {
		t.Fatalf("CreateRuleAttachment(role2): %v", err)
	}
	if _, err := mgr.CreateRuleAttachment(ctx, r.ID, models.TargetTypeGoal, goal.ID); err != nil {
		t.Fatalf("CreateRuleAttachment(goal): %v", err)
	}

	atts, err := mgr.ListRuleAttachments(ctx, r.ID)
	if err != nil {
		t.Fatalf("ListRuleAttachments: %v", err)
	}
	if len(atts) != 3 {
		t.Fatalf("expected 3 attachments on the one Rule, got %+v", atts)
	}

	foundByRole1, err := mgr.ListRules(ctx, draftID, mwanachamaagency.RuleFilter{RoleID: role1.ID})
	if err != nil {
		t.Fatalf("ListRules(role1 filter): %v", err)
	}
	foundByGoal, err := mgr.ListRules(ctx, draftID, mwanachamaagency.RuleFilter{GoalID: goal.ID})
	if err != nil {
		t.Fatalf("ListRules(goal filter): %v", err)
	}
	if len(foundByRole1) != 1 || foundByRole1[0].ID != r.ID {
		t.Fatalf("expected role1 filter to find the rule, got %+v", foundByRole1)
	}
	if len(foundByGoal) != 1 || foundByGoal[0].ID != r.ID {
		t.Fatalf("expected goal filter to also find the same rule, got %+v", foundByGoal)
	}
}

// TestRule_AttachedForksAndCanBeEdited mirrors
// TestValue_AttachedForksAndCanBeEdited, using a Goal target instead of a
// WorkItem to exercise a second attachable type through the fork cycle —
// an attached Rule's Text stays editable in a draft-scoped copy, and its
// attachment (addressed by Code, AGD-003) survives the fork's new row id.
func TestRule_AttachedForksAndCanBeEdited(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, goal, _, _, _ := seedLiveAgency(t, mgr, ctx)
	seedDraftID := goal.DraftID

	r, err := mgr.CreateRule(ctx, models.Rule{DraftID: seedDraftID, Text: "review before merging"})
	if err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	if _, err := mgr.CreateRuleAttachment(ctx, r.ID, models.TargetTypeGoal, goal.ID); err != nil {
		t.Fatalf("CreateRuleAttachment: %v", err)
	}

	d, err := mgr.CreateDraft(ctx, "reword the rule", seedDraftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftRules, err := mgr.ListRules(ctx, d.ID, mwanachamaagency.RuleFilter{})
	if err != nil {
		t.Fatalf("ListRules(draft): %v", err)
	}
	if len(draftRules) != 1 {
		t.Fatalf("expected the attached Rule to fork into the draft, got %+v", draftRules)
	}
	draftR := draftRules[0]
	if draftR.ID == r.ID {
		t.Fatal("expected the fork to mint a new row id")
	}
	if draftR.Code != r.Code {
		t.Fatalf("expected the fork to carry the same Code, got %q want %q", draftR.Code, r.Code)
	}
	draftR.Text = "review before merging, always"
	updated, err := mgr.UpdateRule(ctx, draftR)
	if err != nil {
		t.Fatalf("UpdateRule(draft copy): %v", err)
	}
	if updated.Text != "review before merging, always" {
		t.Fatalf("unexpected updated rule: %+v", updated)
	}

	atts, err := mgr.ListRuleAttachments(ctx, draftR.ID)
	if err != nil {
		t.Fatalf("ListRuleAttachments(fork): %v", err)
	}
	if len(atts) != 1 || atts[0].TargetType != models.TargetTypeGoal {
		t.Fatalf("expected the attachment to resolve against the fork by Code, got %+v", atts)
	}
}
