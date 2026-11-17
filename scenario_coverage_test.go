package mwanachamaagency_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type coverageWorld struct {
	scenarioWorld
	loan    models.Workflow
	compute models.WorkItem
	post    models.WorkItem
}

func newCoverageWorld(t *testing.T) coverageWorld {
	t.Helper()
	w := newScenarioWorld(t)
	ctx := context.Background()
	loan := w.workflow(t, "Loan payment")
	compute, err := w.mgr.CreateWorkItem(ctx, models.WorkItem{
		DraftID: w.draftID, WorkflowID: loan.ID, Title: "Work out the payment", Ordinality: 1, Kind: models.StepKindCompute,
	})
	if err != nil {
		t.Fatalf("CreateWorkItem compute: %v", err)
	}
	post, err := w.mgr.CreateWorkItem(ctx, models.WorkItem{
		DraftID: w.draftID, WorkflowID: loan.ID, Title: "Post the payment", Ordinality: 2, Kind: models.StepKindPost,
	})
	if err != nil {
		t.Fatalf("CreateWorkItem post: %v", err)
	}
	return coverageWorld{scenarioWorld: w, loan: loan, compute: compute, post: post}
}

func (w coverageWorld) playing(titles ...string) {
	played := map[string]bool{}
	for _, title := range titles {
		played[title] = true
	}
	w.runner.outcome = func(req mwanachamaagency.RunRequest) (mwanachamaagency.RunOutcome, error) {
		var steps []models.RunStep
		for _, b := range req.Scenario.Blocks {
			if played[b.Title] {
				steps = append(steps, models.RunStep{StepID: b.StepID, Step: b.Title, Actor: b.Actor, Did: b.Action, Tokens: 100})
			}
		}
		return mwanachamaagency.RunOutcome{Result: models.TestRunPassed, Steps: steps, Tokens: 100 * len(steps)}, nil
	}
}

func (w coverageWorld) warnings(t *testing.T) []string {
	t.Helper()
	issues, err := w.mgr.ListIssues(context.Background(), w.draftID, mwanachamaagency.IssueFilter{Severity: string(models.IssueSeverityWarning), EntityType: "work_item"})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	var out []string
	for _, is := range issues {
		out = append(out, is.Message)
	}
	return out
}

func notPassedThrough(step models.WorkItem) string {
	return fmt.Sprintf("Work item %s %q is not passed through by any scenario", step.Code, step.Title)
}

func sameMessages(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := map[string]int{}
	for _, m := range got {
		seen[m]++
	}
	for _, m := range want {
		seen[m]--
	}
	for _, n := range seen {
		if n != 0 {
			return false
		}
	}
	return true
}

func TestScenarioCoverage_NoScenarioWarnsOnTheCalculatingAndThePostingStep(t *testing.T) {
	w := newCoverageWorld(t)

	got := w.warnings(t)

	want := []string{notPassedThrough(w.compute), notPassedThrough(w.post)}
	if !sameMessages(got, want) {
		t.Fatalf("warnings = %q, want %q", got, want)
	}
}

func TestScenarioCoverage_AScenarioPlayingOnlyTheCalculatingStepLeavesTheWarningOnThePostingStep(t *testing.T) {
	w := newCoverageWorld(t)
	w.playing("Work out the payment")
	s := w.scenario(t, "Payment is worked out", w.loan, w.actor("Wanjiru", farmer))
	w.pass(t, s.ID)

	got := w.warnings(t)

	want := []string{notPassedThrough(w.post)}
	if !sameMessages(got, want) {
		t.Fatalf("warnings = %q, want %q", got, want)
	}
}

func TestScenarioCoverage_ScenariosThatPlayBothStepsLeaveNoWarning(t *testing.T) {
	w := newCoverageWorld(t)
	w.playing("Work out the payment", "Post the payment")
	s := w.scenario(t, "Payment is posted", w.loan, w.actor("Wanjiru", farmer))
	w.pass(t, s.ID)

	if got := w.warnings(t); len(got) != 0 {
		t.Fatalf("warnings = %q, want none", got)
	}
}

func TestScenarioCoverage_TwoScenariosTogetherCoverBothSteps(t *testing.T) {
	w := newCoverageWorld(t)
	w.playing("Work out the payment")
	first := w.scenario(t, "Payment is worked out", w.loan, w.actor("Wanjiru", farmer))
	w.pass(t, first.ID)
	w.playing("Post the payment")
	second := w.scenario(t, "Payment is posted", w.loan, w.actor("Wanjiru", farmer))
	w.pass(t, second.ID)

	if got := w.warnings(t); len(got) != 0 {
		t.Fatalf("warnings = %q, want none", got)
	}
}

func TestScenarioCoverage_AScenarioThatHasNotRunExercisedNothing(t *testing.T) {
	w := newCoverageWorld(t)
	w.scenario(t, "Payment is posted", w.loan, w.actor("Wanjiru", farmer))

	got := w.warnings(t)

	want := []string{notPassedThrough(w.compute), notPassedThrough(w.post)}
	if !sameMessages(got, want) {
		t.Fatalf("warnings = %q, want %q", got, want)
	}
}

func TestScenarioCoverage_OnlyTheLastRunOfAScenarioCounts(t *testing.T) {
	w := newCoverageWorld(t)
	w.playing("Work out the payment", "Post the payment")
	s := w.scenario(t, "Payment is posted", w.loan, w.actor("Wanjiru", farmer))
	w.pass(t, s.ID)
	w.playing("Work out the payment")
	w.pass(t, s.ID)

	got := w.warnings(t)

	want := []string{notPassedThrough(w.post)}
	if !sameMessages(got, want) {
		t.Fatalf("warnings = %q, want %q", got, want)
	}
}

func TestScenarioCoverage_ARunOfAnOlderVersionOfTheDraftCoversNothing(t *testing.T) {
	w := newCoverageWorld(t)
	w.playing("Work out the payment", "Post the payment")
	s := w.scenario(t, "Payment is posted", w.loan, w.actor("Wanjiru", farmer))
	w.pass(t, s.ID)
	if got := w.warnings(t); len(got) != 0 {
		t.Fatalf("warnings before the edit = %q, want none", got)
	}
	renamed := w.post
	renamed.Title = "Post the monthly payment"
	renamed, err := w.mgr.UpdateWorkItem(context.Background(), renamed)
	if err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}

	got := w.warnings(t)

	want := []string{notPassedThrough(w.compute), notPassedThrough(renamed)}
	if !sameMessages(got, want) {
		t.Fatalf("warnings = %q, want %q", got, want)
	}
}

func TestScenarioCoverage_OtherKindsOfStepNeverWarn(t *testing.T) {
	w := newCoverageWorld(t)

	got := w.warnings(t)

	want := []string{notPassedThrough(w.compute), notPassedThrough(w.post)}
	if !sameMessages(got, want) {
		t.Fatalf("warnings = %q, want only the two of Loan payment although the draft holds Buy now and Group order", got)
	}
}

func TestScenarioCoverage_TheWarningNeverBlocksAPublishAndNeverFailsAScenario(t *testing.T) {
	w := newCoverageWorld(t)
	w.playing("Work out the payment")
	s := w.scenario(t, "Payment is worked out", w.loan, w.actor("Wanjiru", farmer))
	w.pass(t, s.ID)
	want := notPassedThrough(w.post)
	if got := w.warnings(t); !sameMessages(got, []string{want}) {
		t.Fatalf("warnings = %q, want the one for the posting step", got)
	}
	errors, err := w.mgr.ListIssues(context.Background(), w.draftID, mwanachamaagency.IssueFilter{Severity: string(models.IssueSeverityError), Message: "is not passed through"})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(errors) != 0 {
		t.Fatalf("coverage findings of severity error = %+v, want none", errors)
	}

	_, err = w.mgr.PublishAgency(context.Background(), w.draftID, "spring", "")

	if err != nil && strings.Contains(err.Error(), "is not passed through") {
		t.Fatalf("PublishAgency was refused for the coverage warning: %v", err)
	}
	if after := w.get(t, s.ID); after.Result != models.ScenarioPassed {
		t.Fatalf("scenario = %q after the scan, want passed", after.Result)
	}
}
