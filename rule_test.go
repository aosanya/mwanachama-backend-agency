package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCreateRule_RequiresText(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.CreateRule(context.Background(), models.Rule{}); !errors.Is(err, mwanachamaagency.ErrInvalidRule) {
		t.Fatalf("expected ErrInvalidRule, got %v", err)
	}
}

// TestRule_AgencyWideNeverForkable mirrors TestValue_AgencyWideNeverForkable.
func TestRule_AgencyWideNeverForkable(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, _, _, _, _ := seedLiveAgency(t, mgr, ctx)

	r, err := mgr.CreateRule(ctx, models.Rule{Text: "never touch production data"})
	if err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	if err := mgr.DeleteRule(ctx, r.ID); !errors.Is(err, mwanachamaagency.ErrCannotEditLive) {
		t.Fatalf("expected ErrCannotEditLive, got %v", err)
	}

	d, err := mgr.CreateDraft(ctx, "try to edit the rule", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftRules, err := mgr.ListRules(ctx, d.ID, mwanachamaagency.RuleFilter{})
	if err != nil {
		t.Fatalf("ListRules(draft): %v", err)
	}
	if len(draftRules) != 0 {
		t.Fatalf("expected an Agency-wide Rule to never fork into a Draft, got %+v", draftRules)
	}
}

// TestRuleAttachment_MultipleTargets covers a single Rule carrying more
// than one attachment at once (AGD-003's core requirement), plus that
// attaching to two different target types works.
func TestRuleAttachment_MultipleTargets(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, goal, _, _, _ := seedLiveAgency(t, mgr, ctx)

	r, err := mgr.CreateRule(ctx, models.Rule{Text: "never touch production data"})
	if err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	role1, err := mgr.CreateRole(ctx, models.Role{Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	role2, err := mgr.CreateRole(ctx, models.Role{Name: "reviewer"})
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

	foundByRole1, err := mgr.ListRules(ctx, "", mwanachamaagency.RuleFilter{RoleID: role1.ID})
	if err != nil {
		t.Fatalf("ListRules(role1 filter): %v", err)
	}
	foundByGoal, err := mgr.ListRules(ctx, "", mwanachamaagency.RuleFilter{GoalID: goal.ID})
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

// TestRule_AttachedForksAndSurvivesPromote mirrors
// TestValue_AttachedForksAndSurvivesPromote, using a Goal target instead of
// a WorkItem to exercise a second attachable type through the fork/promote
// cycle.
func TestRule_AttachedForksAndSurvivesPromote(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, goal, _, _, _ := seedLiveAgency(t, mgr, ctx)

	r, err := mgr.CreateRule(ctx, models.Rule{Text: "review before merging"})
	if err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	if _, err := mgr.CreateRuleAttachment(ctx, r.ID, models.TargetTypeGoal, goal.ID); err != nil {
		t.Fatalf("CreateRuleAttachment: %v", err)
	}

	d, err := mgr.CreateDraft(ctx, "reword the rule", ag.ID, "live")
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
	draftR.Text = "review before merging, always"
	if _, err := mgr.UpdateRule(ctx, draftR); err != nil {
		t.Fatalf("UpdateRule(draft copy): %v", err)
	}

	if _, err := mgr.PromoteDraft(ctx, d.ID); err != nil {
		t.Fatalf("PromoteDraft: %v", err)
	}

	live, err := mgr.ListRules(ctx, "", mwanachamaagency.RuleFilter{})
	if err != nil {
		t.Fatalf("ListRules(live): %v", err)
	}
	if len(live) != 1 || live[0].ID == r.ID || live[0].Code != r.Code {
		t.Fatalf("expected a new live row carrying the same Code, got %+v (original id %s, code %s)", live, r.ID, r.Code)
	}

	atts, err := mgr.ListRuleAttachments(ctx, live[0].ID)
	if err != nil {
		t.Fatalf("ListRuleAttachments(new live id): %v", err)
	}
	if len(atts) != 1 || atts[0].TargetType != models.TargetTypeGoal {
		t.Fatalf("expected the attachment to survive promotion, got %+v", atts)
	}
}
