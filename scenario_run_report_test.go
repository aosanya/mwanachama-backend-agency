package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestGetTestRun_CarriesWhatTheRunLeftInTheTestCopy(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow)
	left := []models.StoredRecord{{Record: "Orders", Holds: "Each order placed", Rows: 3}, {Record: "Farmers", Holds: "Who sells", Rows: 1}}
	w.runner.outcome = func(req mwanachamaagency.RunRequest) (mwanachamaagency.RunOutcome, error) {
		out, err := passing(req)
		out.Stored = left
		return out, err
	}

	run, err := w.runToEnd(ctx, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	got := w.read(t, run.ID).Stored
	if len(got) != 2 || got[0] != left[0] || got[1] != left[1] {
		t.Errorf("stored = %+v, want %+v", got, left)
	}
}

func TestGetTestRun_AnEmptyTestCopyIsAnEmptyListNotNull(t *testing.T) {
	w := newScenarioWorld(t)
	s := w.scenario(t, "Shop says yes", w.buyNow)
	run, err := w.runToEnd(context.Background(), s.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(w.read(t, run.ID))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"stored":[]`) {
		t.Errorf("run = %s, want stored as an empty list", raw)
	}
}

func TestGetTestRun_KeepsTheTokenBudgetItWasStartedUnder(t *testing.T) {
	limit := 5000
	w := newScenarioWorld(t, mwanachamaagency.WithTokensPerTest(func(context.Context) int { return limit }))
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow)

	run, err := w.runToEnd(ctx, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	limit = 9000
	if got := w.read(t, run.ID).TokenBudget; got != 5000 {
		t.Errorf("token_budget = %d, want the 5000 it started under", got)
	}
	if got := w.runner.seen[0].TokenBudget; got != 5000 {
		t.Errorf("runner was handed a budget of %d, want 5000", got)
	}
}

func TestGetTestRun_NoTokenLimitIsZero(t *testing.T) {
	w := newScenarioWorld(t)
	s := w.scenario(t, "Shop says yes", w.buyNow)
	run, err := w.runToEnd(context.Background(), s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := w.read(t, run.ID).TokenBudget; got != 0 {
		t.Errorf("token_budget = %d, want 0", got)
	}
}

func TestGetTestRun_NamesTheOpenDraftLatestEdit(t *testing.T) {
	w := newScenarioWorld(t)
	s := w.scenario(t, "Shop says yes", w.buyNow)
	run, err := w.runToEnd(context.Background(), s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := w.read(t, run.ID).DraftLabel; got != "Latest edit" {
		t.Errorf("draft_label = %q, want Latest edit", got)
	}
}

func TestGetTestRun_NamesAnArchivedPublishedDraftByItsVersion(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow)
	run, err := w.runToEnd(ctx, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "spring", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if _, err := w.mgr.ArchiveDraft(ctx, w.draftID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}
	if got := w.read(t, run.ID).DraftLabel; got != "spring" {
		t.Errorf("draft_label = %q, want the published label spring", got)
	}
}

func TestGetTestRun_NamesAnArchivedUnpublishedDraftByItsVersion(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow)
	run, err := w.runToEnd(ctx, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.mgr.ArchiveDraft(ctx, w.draftID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}
	got := w.read(t, run.ID)
	if got.DraftVersion == "" || got.DraftLabel != got.DraftVersion {
		t.Errorf("draft_label = %q, want the draft version %q", got.DraftLabel, got.DraftVersion)
	}
}

func TestGetTestRun_KeepsEachStepWaitedFlag(t *testing.T) {
	w := newScenarioWorld(t)
	s := w.scenario(t, "Shop says yes", w.buyNow)
	w.runner.outcome = func(req mwanachamaagency.RunRequest) (mwanachamaagency.RunOutcome, error) {
		out, err := passing(req)
		out.Steps[1].Waited = true
		return out, err
	}
	run, err := w.runToEnd(context.Background(), s.ID)
	if err != nil {
		t.Fatal(err)
	}
	steps := w.read(t, run.ID).Steps
	if len(steps) != 3 || steps[0].Waited || !steps[1].Waited || steps[2].Waited {
		t.Errorf("waited = %v %v %v, want false true false", steps[0].Waited, steps[1].Waited, steps[2].Waited)
	}
	raw, _ := json.Marshal(steps[0])
	if !strings.Contains(string(raw), `"waited":false`) {
		t.Errorf("step = %s, want waited always present", raw)
	}
}

type markingRunner struct {
	gatedRunner
	marks   chan struct{}
	proceed chan struct{}
}

func (r *markingRunner) RunReporting(_ context.Context, req mwanachamaagency.RunRequest, finished func(models.RunStep) bool) (mwanachamaagency.RunOutcome, error) {
	var fresh []models.RunStep
	for _, b := range req.Scenario.Blocks[len(req.Done):] {
		req.Starting(b.StepID)
		r.marks <- struct{}{}
		<-r.gate(req.Scenario.ID)
		step := models.RunStep{StepID: b.StepID, Step: b.Title, Actor: b.Actor, Did: b.Action, Tokens: 100}
		fresh = append(fresh, step)
		finished(step)
		r.marks <- struct{}{}
		<-r.proceed
	}
	return mwanachamaagency.RunOutcome{Result: models.TestRunPassed, Steps: fresh, Tokens: 100 * len(fresh)}, nil
}

func (r *markingRunner) Run(ctx context.Context, req mwanachamaagency.RunRequest) (mwanachamaagency.RunOutcome, error) {
	return r.RunReporting(ctx, req, func(models.RunStep) bool { return true })
}

func TestGetTestRun_ShowsTheStepInProgressUntilTheRunEnds(t *testing.T) {
	r := &markingRunner{gatedRunner: *newGatedRunner(), marks: make(chan struct{}, 64), proceed: make(chan struct{}, 64)}
	w := newScenarioWorld(t, mwanachamaagency.WithScenarioRunner(r))
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow)

	run := w.press(t, s.ID)
	<-r.marks
	first := w.scenarioBlocks(t, s.ID)[0].StepID
	if got := w.read(t, run.ID).RunningStep; got != first {
		t.Errorf("running_step = %q while the first step runs, want %q", got, first)
	}
	r.letStepsFinish(s.ID, 1)
	<-r.marks
	if got := w.read(t, run.ID).RunningStep; got != "" {
		t.Errorf("running_step = %q between steps, want empty", got)
	}
	r.proceed <- struct{}{}
	<-r.marks
	second := w.scenarioBlocks(t, s.ID)[1].StepID
	if got := w.read(t, run.ID).RunningStep; got != second {
		t.Errorf("running_step = %q while the second step runs, want %q", got, second)
	}
	r.letStepsFinish(s.ID, 2)
	r.proceed <- struct{}{}
	r.proceed <- struct{}{}
	ended, err := w.mgr.AwaitRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ended.RunningStep != "" {
		t.Errorf("running_step = %q after the run ended, want empty", ended.RunningStep)
	}
}

func (w scenarioWorld) scenarioBlocks(t *testing.T, id string) []models.ScenarioBlock {
	t.Helper()
	return w.get(t, id).Blocks
}
