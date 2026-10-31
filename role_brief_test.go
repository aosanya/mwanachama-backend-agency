package mwanachamaagency_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type briefFixture struct {
	mgr                  mwanachamaagency.AgencyManager
	draftID              string
	seat, other          models.Role
	objectiveMine        models.Objective
	objectiveOther       models.Objective
	seatRule, otherRule  models.Rule
	orgValue, otherValue models.Value
}

func newBriefFixture(t *testing.T) briefFixture {
	t.Helper()
	ctx := context.Background()
	mgr := newTestManager(t)
	_, o, seat := seedPublishableAgency(t, mgr, ctx)
	f := briefFixture{mgr: mgr, draftID: o.DraftID, seat: seat, objectiveMine: o}
	var err error
	f.other, err = mgr.CreateRole(ctx, models.Role{DraftID: f.draftID, Name: "auditor", Description: "checks the books"})
	must(t, err)
	goal := goalOf(t, mgr, ctx, f.draftID, o.GoalID)
	f.objectiveOther, err = mgr.CreateObjective(ctx, models.Objective{DraftID: f.draftID, GoalID: goal.ID, Title: "Audit the books"})
	must(t, err)
	_, mineStep := flowWithStep(t, mgr, ctx, f.draftID, "Mine flow", "Mine", "PROMPT-MINE", o)
	assignOn(t, mgr, ctx, f.draftID, seat, mineStep, models.RACILabelAccountable)
	_, theirStep := flowWithStep(t, mgr, ctx, f.draftID, "Theirs flow", "Theirs", "PROMPT-THEIRS", f.objectiveOther)
	assignOn(t, mgr, ctx, f.draftID, f.other, theirStep, models.RACILabelAccountable)

	f.seatRule, err = mgr.CreateRule(ctx, models.Rule{DraftID: f.draftID, Text: "RULE-SEAT"})
	must(t, err)
	_, err = mgr.CreateRuleAttachment(ctx, f.seatRule.ID, models.TargetTypeRole, seat.ID)
	must(t, err)
	f.otherRule, err = mgr.CreateRule(ctx, models.Rule{DraftID: f.draftID, Text: "RULE-OTHER-SEAT"})
	must(t, err)
	_, err = mgr.CreateRuleAttachment(ctx, f.otherRule.ID, models.TargetTypeRole, f.other.ID)
	must(t, err)
	goalRule, err := mgr.CreateRule(ctx, models.Rule{DraftID: f.draftID, Text: "RULE-GOAL"})
	must(t, err)
	_, err = mgr.CreateRuleAttachment(ctx, goalRule.ID, models.TargetTypeGoal, goal.ID)
	must(t, err)
	objValue, err := mgr.CreateValue(ctx, models.Value{DraftID: f.draftID, Text: "VALUE-OBJECTIVE"})
	must(t, err)
	_, err = mgr.CreateValueAttachment(ctx, objValue.ID, models.TargetTypeObjective, o.ID)
	must(t, err)
	f.orgValue, err = mgr.CreateValue(ctx, models.Value{DraftID: f.draftID, Text: "VALUE-ORG"})
	must(t, err)
	f.otherValue, err = mgr.CreateValue(ctx, models.Value{DraftID: f.draftID, Text: "VALUE-OTHER-OBJECTIVE"})
	must(t, err)
	_, err = mgr.CreateValueAttachment(ctx, f.otherValue.ID, models.TargetTypeObjective, f.objectiveOther.ID)
	must(t, err)

	_, err = mgr.PublishAgency(ctx, f.draftID, "", "")
	must(t, err)
	return f
}

func flowWithStep(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID, flowName, stepTitle, prompt string, objectives ...models.Objective) (models.Workflow, models.WorkItem) {
	t.Helper()
	wf, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftID, Name: flowName})
	must(t, err)
	for _, o := range objectives {
		_, err = mgr.CreateWorkflowObjective(ctx, models.WorkflowObjective{DraftID: draftID, WorkflowID: wf.ID, ObjectiveID: o.ID})
		must(t, err)
	}
	step, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: draftID, WorkflowID: wf.ID, Title: stepTitle, Prompt: prompt})
	must(t, err)
	return wf, step
}

func assignOn(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID string, role models.Role, step models.WorkItem, label models.RACILabel) {
	t.Helper()
	_, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: draftID, RoleID: role.ID, WorkItemID: step.ID, Label: label})
	must(t, err)
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func goalOf(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID, id string) models.Goal {
	t.Helper()
	gs, err := mgr.ListGoals(ctx, draftID, mwanachamaagency.GoalFilter{})
	must(t, err)
	for _, g := range gs {
		if g.ID == id {
			return g
		}
	}
	t.Fatalf("goal %s missing", id)
	return models.Goal{}
}

func TestRoleBrief_EveryPartAppears(t *testing.T) {
	f := newBriefFixture(t)
	b, err := f.mgr.GetRoleBrief(context.Background(), f.seat.ID)
	must(t, err)
	for _, want := range []string{f.seat.Name, f.objectiveMine.Title, "Grow adoption", "RULE-SEAT", "RULE-GOAL", "VALUE-OBJECTIVE", "VALUE-ORG", "PROMPT-MINE"} {
		if !strings.Contains(b.Text, want) {
			t.Errorf("brief text lacks %q:\n%s", want, b.Text)
		}
	}
	if len(b.Objectives) != 1 || b.Objectives[0].GoalTitle != "Grow adoption" || b.Objectives[0].Label != string(models.RACILabelAccountable) {
		t.Errorf("objectives part wrong: %+v", b.Objectives)
	}
	if len(b.Rules) != 2 || len(b.Values) != 2 || len(b.WorkItems) != 1 {
		t.Errorf("parts wrong: rules=%d values=%d work_items=%d", len(b.Rules), len(b.Values), len(b.WorkItems))
	}
}

func TestRoleBrief_LeavesOutWhatBelongsToAnotherSeat(t *testing.T) {
	f := newBriefFixture(t)
	b, err := f.mgr.GetRoleBrief(context.Background(), f.seat.ID)
	must(t, err)
	for _, unwanted := range []string{"RULE-OTHER-SEAT", "VALUE-OTHER-OBJECTIVE", "PROMPT-THEIRS", "Audit the books"} {
		if strings.Contains(b.Text, unwanted) {
			t.Errorf("brief text carries %q:\n%s", unwanted, b.Text)
		}
	}
}

func TestRoleBrief_OrganizationWideValueReachesEverySeat(t *testing.T) {
	f := newBriefFixture(t)
	for _, id := range []string{f.seat.ID, f.other.ID} {
		b, err := f.mgr.GetRoleBrief(context.Background(), id)
		must(t, err)
		if !strings.Contains(b.Text, "VALUE-ORG") {
			t.Errorf("seat %s lacks the organization-wide value:\n%s", id, b.Text)
		}
	}
}

func TestRoleBrief_ChangedRuleShowsInNextBrief(t *testing.T) {
	f := newBriefFixture(t)
	ctx := context.Background()
	pub, err := f.mgr.GetPublicationForChannel(ctx, "")
	must(t, err)
	d, err := f.mgr.CreateDraft(ctx, "edit", pub.Tag, "publication")
	must(t, err)
	rules, err := f.mgr.ListRules(ctx, d.ID, mwanachamaagency.RuleFilter{})
	must(t, err)
	changed := false
	for _, r := range rules {
		if r.Text == "RULE-SEAT" {
			r.Text = "RULE-SEAT-CHANGED"
			_, err := f.mgr.UpdateRule(ctx, r)
			must(t, err)
			changed = true
		}
	}
	if !changed {
		t.Fatal("rule not found in forked draft")
	}
	_, err = f.mgr.PublishAgency(ctx, d.ID, "", "")
	must(t, err)
	b, err := f.mgr.GetRoleBrief(ctx, f.seat.Code)
	must(t, err)
	if !strings.Contains(b.Text, "RULE-SEAT-CHANGED") || strings.Contains(b.Text, "RULE-SEAT\n") {
		t.Errorf("brief did not follow the new publication:\n%s", b.Text)
	}
}

func TestRoleBrief_UnknownSeatAndNothingPublished(t *testing.T) {
	f := newBriefFixture(t)
	if _, err := f.mgr.GetRoleBrief(context.Background(), "nope"); !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Errorf("unknown seat: %v", err)
	}
	empty := newTestManager(t)
	if _, err := empty.GetRoleBrief(context.Background(), "x"); !errors.Is(err, mwanachamaagency.ErrPublicationNotFound) {
		t.Errorf("nothing published: %v", err)
	}
}

func TestRoleBrief_ObjectivesComeThroughTheWorkflowsOfMySteps(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	draftID := o.DraftID
	goal := goalOf(t, mgr, ctx, draftID, o.GoalID)
	second, err := mgr.CreateObjective(ctx, models.Objective{DraftID: draftID, GoalID: goal.ID, Title: "Second aim"})
	must(t, err)
	scribe, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "scribe"})
	must(t, err)

	_, drafting := flowWithStep(t, mgr, ctx, draftID, "Drafting", "Draft it", "PROMPT-DRAFT", o, second)
	_, filing := flowWithStep(t, mgr, ctx, draftID, "Filing", "File it", "PROMPT-FILE", second)
	assignOn(t, mgr, ctx, draftID, scribe, drafting, models.RACILabelResponsible)
	assignOn(t, mgr, ctx, draftID, scribe, drafting, models.RACILabelAccountable)
	assignOn(t, mgr, ctx, draftID, scribe, filing, models.RACILabelConsulted)
	_, err = mgr.PublishAgency(ctx, draftID, "", "")
	must(t, err)

	b, err := mgr.GetRoleBrief(ctx, scribe.ID)
	must(t, err)
	labels := map[string]string{}
	for _, bo := range b.Objectives {
		labels[bo.Title] = bo.Label
	}
	if len(b.Objectives) != 2 {
		t.Fatalf("expected both objectives reached through the two workflows, got %+v", b.Objectives)
	}
	if labels["Grow adoption"] != "responsible, accountable" {
		t.Errorf("the objective reached through one workflow should carry both labels held on its step, got %q", labels["Grow adoption"])
	}
	if labels["Second aim"] != "responsible, accountable, consulted" {
		t.Errorf("the objective linked to both workflows should merge the labels of both, got %q", labels["Second aim"])
	}
	if len(b.WorkItems) != 2 {
		t.Errorf("expected a work item per workflow the seat has a step in, got %+v", b.WorkItems)
	}
	for _, want := range []string{"PROMPT-DRAFT", "PROMPT-FILE"} {
		if !strings.Contains(b.Text, want) {
			t.Errorf("brief text lacks %q:\n%s", want, b.Text)
		}
	}
}

func TestRoleBrief_NoAssignmentMeansNoObjectives(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	idle, err := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "idle"})
	must(t, err)
	_, err = mgr.PublishAgency(ctx, o.DraftID, "", "")
	must(t, err)

	b, err := mgr.GetRoleBrief(ctx, idle.ID)
	must(t, err)
	if len(b.Objectives) != 0 || len(b.WorkItems) != 0 {
		t.Errorf("a seat with no step assignment answers for nothing, got objectives=%+v work_items=%+v", b.Objectives, b.WorkItems)
	}
}

func TestRoleBrief_FollowsAStepAssignmentMovedInANewPublication(t *testing.T) {
	f := newBriefFixture(t)
	ctx := context.Background()
	pub, err := f.mgr.GetPublicationForChannel(ctx, "")
	must(t, err)
	d, err := f.mgr.CreateDraft(ctx, "move the work", pub.Tag, "publication")
	must(t, err)

	roles, err := f.mgr.ListRoles(ctx, d.ID, mwanachamaagency.RoleFilter{})
	must(t, err)
	var otherInDraft models.Role
	for _, r := range roles {
		if r.Code == f.other.Code {
			otherInDraft = r
		}
	}
	items, err := f.mgr.ListWorkItems(ctx, d.ID, mwanachamaagency.WorkItemFilter{})
	must(t, err)
	var mineInDraft models.WorkItem
	for _, it := range items {
		if it.Title == "Mine" {
			mineInDraft = it
		}
	}
	if otherInDraft.ID == "" || mineInDraft.ID == "" {
		t.Fatalf("forked draft lacks the auditor or the Mine step: %+v %+v", roles, items)
	}
	assignOn(t, f.mgr, ctx, d.ID, otherInDraft, mineInDraft, models.RACILabelInformed)
	_, err = f.mgr.PublishAgency(ctx, d.ID, "", "")
	must(t, err)

	b, err := f.mgr.GetRoleBrief(ctx, f.other.Code)
	must(t, err)
	if !strings.Contains(b.Text, "PROMPT-MINE") || !strings.Contains(b.Text, "Grow adoption") {
		t.Errorf("auditor, now informed on Mine, should see its prompt and objective:\n%s", b.Text)
	}
}

func TestRoleBrief_LeavesOutTheStepsOfAWorkflowTheSeatIsNotOn(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	draftID := o.DraftID
	scribe, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "scribe"})
	must(t, err)
	lead, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "lead"})
	must(t, err)

	wf, first := flowWithStep(t, mgr, ctx, draftID, "Charter", "Write", "PROMPT-WRITE", o)
	second, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: draftID, WorkflowID: wf.ID, Title: "Sign", Prompt: "PROMPT-SIGN-LEAD-ONLY"})
	must(t, err)
	assignOn(t, mgr, ctx, draftID, scribe, first, models.RACILabelResponsible)
	assignOn(t, mgr, ctx, draftID, scribe, first, models.RACILabelConsulted)
	assignOn(t, mgr, ctx, draftID, lead, second, models.RACILabelAccountable)
	_, err = mgr.PublishAgency(ctx, draftID, "", "")
	must(t, err)

	b, err := mgr.GetRoleBrief(ctx, scribe.ID)
	must(t, err)
	if strings.Contains(b.Text, "PROMPT-SIGN-LEAD-ONLY") {
		t.Errorf("the brief carries the prompt of a step only another seat is on:\n%s", b.Text)
	}
	if len(b.WorkItems) != 1 || b.WorkItems[0].Title != "Write" {
		t.Fatalf("expected only the Write step, got %+v", b.WorkItems)
	}
	want := []string{"responsible", "consulted"}
	if got := b.WorkItems[0].Labels; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("labels = %v, want %v", got, want)
	}
	if !strings.Contains(b.Text, "[responsible, consulted]") {
		t.Errorf("brief text does not show the labels held on the step:\n%s", b.Text)
	}

	lb, err := mgr.GetRoleBrief(ctx, lead.ID)
	must(t, err)
	if len(lb.WorkItems) != 1 || lb.WorkItems[0].Title != "Sign" || strings.Join(lb.WorkItems[0].Labels, ",") != "accountable" {
		t.Errorf("lead's brief wrong: %+v", lb.WorkItems)
	}
}
