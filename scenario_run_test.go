package mwanachamaagency_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestRunScenario_APassingRunIsKeptAndPassesTheScenario(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow, w.actor("Farmer 1", farmer))

	run, err := w.runToEnd(ctx, s.ID)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	if run.ID == "" || run.ScenarioID != s.ID || run.DraftID != w.draftID || run.Result != models.TestRunPassed {
		t.Fatalf("run = %+v, want a passed run of the scenario on its draft", run)
	}
	if run.StartedAt == "" || run.FinishedAt == "" || run.DraftVersion == "" || run.Tokens != 300 || len(run.Steps) != 3 {
		t.Errorf("run = %+v, want times, a draft version, 300 tokens and three steps", run)
	}
	after := w.get(t, s.ID)
	if after.Result != models.ScenarioPassed || after.LastRunID != run.ID {
		t.Errorf("scenario = %q/%q, want passed by %s", after.Result, after.LastRunID, run.ID)
	}

	stored, err := w.mgr.GetTestRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetTestRun: %v", err)
	}
	if !reflect.DeepEqual(stored, run) {
		t.Errorf("stored run differs from the returned one:\n%+v\n%+v", stored, run)
	}
}

func TestRunScenario_TheRunnerIsHandedTheWholeScenario(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	rule, _ := w.mgr.CreateRule(ctx, models.Rule{DraftID: w.draftID, Text: "Ask the shop once"})
	if _, err := w.mgr.CreateRuleAttachment(ctx, rule.ID, models.TargetTypeWorkflow, w.buyNow.ID); err != nil {
		t.Fatal(err)
	}
	s := w.scenario(t, "Shop says yes", w.buyNow, w.actor("Farmer 1", farmer))
	w.pass(t, s.ID)

	req := w.runner.seen[0]
	if req.Scenario.ID != s.ID || len(req.Scenario.Blocks) != 3 || len(req.Scenario.Actors) < 2 {
		t.Errorf("scenario handed over = %+v", req.Scenario)
	}
	if req.Workflow.ID != w.buyNow.ID || len(req.Rules) != 1 || req.Rules[0].ID != rule.ID {
		t.Errorf("workflow %q rules %v handed over, want Buy now and its rule", req.Workflow.Name, req.Rules)
	}
	if req.Scenario.Result != models.ScenarioRunning {
		t.Errorf("scenario was %q while it ran, want running", req.Scenario.Result)
	}
	if req.DraftVersion == "" || req.RunID == "" {
		t.Errorf("request = %+v, want a draft version and the run's id", req)
	}
}

func TestRunScenario_AFailedRunKeepsWhatWasExpectedAndFound(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow)
	w.runner.outcome = func(req mwanachamaagency.RunRequest) (mwanachamaagency.RunOutcome, error) {
		return mwanachamaagency.RunOutcome{
			Result: models.TestRunFailed, Tokens: 3900,
			Steps: []models.RunStep{
				{StepID: req.Scenario.Blocks[0].StepID, Step: "Ask", Actor: "Farmer 1", Tokens: 1800},
				{StepID: req.Scenario.Blocks[1].StepID, Step: "Answered", Failed: true, Expected: "One message", Found: "Two messages",
					Rules: []models.RuleCheck{{Code: "RU-7", Held: false}}, Tokens: 600},
			},
		}, nil
	}
	run, err := w.runToEnd(ctx, s.ID)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	stored, _ := w.mgr.GetTestRun(ctx, run.ID)
	failing := stored.Steps[1]
	if stored.Result != models.TestRunFailed || !failing.Failed || failing.Expected != "One message" || failing.Found != "Two messages" || failing.Rules[0].Held {
		t.Errorf("stored run = %+v, want the failing step with expected, found and its rule not held", stored)
	}
	if got := w.get(t, s.ID).Result; got != models.ScenarioFailed {
		t.Errorf("scenario = %q, want failed", got)
	}
}

func TestRunScenario_AStoppedRunKeepsTheStepsDone(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow)
	w.runner.outcome = func(req mwanachamaagency.RunRequest) (mwanachamaagency.RunOutcome, error) {
		return mwanachamaagency.RunOutcome{
			Result: models.TestRunStopped, StopReason: "Budget reached", Tokens: 30000,
			Steps: []models.RunStep{{StepID: req.Scenario.Blocks[0].StepID, Step: "Ask", Tokens: 30000}},
		}, nil
	}
	run, err := w.runToEnd(ctx, s.ID)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	if run.Result != models.TestRunStopped || run.StopReason != "Budget reached" || len(run.Steps) != 1 || run.Tokens != 30000 {
		t.Errorf("run = %+v", run)
	}
	if got := w.get(t, s.ID).Result; got != models.ScenarioStopped {
		t.Errorf("scenario = %q, want stopped", got)
	}
}

func TestRunScenario_ARunnerThatCannotFinishStopsTheRun(t *testing.T) {
	cases := map[string]func(mwanachamaagency.RunRequest) (mwanachamaagency.RunOutcome, error){
		"an error": func(mwanachamaagency.RunRequest) (mwanachamaagency.RunOutcome, error) {
			return mwanachamaagency.RunOutcome{}, errors.New("provider unreachable")
		},
		"a result that is not an ending": func(mwanachamaagency.RunRequest) (mwanachamaagency.RunOutcome, error) {
			return mwanachamaagency.RunOutcome{Result: models.TestRunRunning}, nil
		},
		"no result at all": func(mwanachamaagency.RunRequest) (mwanachamaagency.RunOutcome, error) {
			return mwanachamaagency.RunOutcome{}, nil
		},
	}
	for name, outcome := range cases {
		t.Run(name, func(t *testing.T) {
			w := newScenarioWorld(t)
			w.runner.outcome = outcome
			s := w.scenario(t, "Shop says yes", w.buyNow)
			run, err := w.runToEnd(context.Background(), s.ID)
			if err != nil {
				t.Fatalf("RunScenario: %v", err)
			}
			if run.Result != models.TestRunStopped || run.StopReason == "" || run.FinishedAt == "" {
				t.Errorf("run = %+v, want a stopped run with a reason", run)
			}
			if got := w.get(t, s.ID).Result; got != models.ScenarioStopped {
				t.Errorf("scenario = %q, want stopped, never left running", got)
			}
		})
	}
}

func TestRunScenario_WithoutAnExecutorTheRunIsStoppedAndSaysSo(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sp, _ := mwanachamaagency.SpecFor("test")
	if err := mwanachamaagency.Provision(db, sp); err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	mgr, err := mwanachamaagency.NewAgencyManager(db, sp)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_, _, o, wf, _ := seedLiveAgency(t, mgr, ctx)
	s, err := mgr.CreateScenario(ctx, models.Scenario{DraftID: o.DraftID, Name: "n", WorkflowID: wf.ID})
	if err != nil {
		t.Fatal(err)
	}
	pressed, err := mgr.RunScenario(ctx, s.ID)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	run, err := mgr.AwaitRun(ctx, pressed.ID)
	if err != nil {
		t.Fatalf("AwaitRun: %v", err)
	}
	if run.Result != models.TestRunStopped || !strings.Contains(run.StopReason, "executor") {
		t.Errorf("run = %+v, want stopped because no executor is installed", run)
	}
}

func TestRunScenario_AScenarioAlreadyRunningRefusesASecondRun(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow)
	var inner error
	w.runner.outcome = func(req mwanachamaagency.RunRequest) (mwanachamaagency.RunOutcome, error) {
		_, inner = w.runToEnd(ctx, s.ID)
		return passing(req)
	}
	w.pass(t, s.ID)
	if !errors.Is(inner, mwanachamaagency.ErrScenarioRunning) {
		t.Errorf("second run while running: %v, want ErrScenarioRunning", inner)
	}
	if got := len(w.runner.seen); got != 1 {
		t.Errorf("runner called %d times, want once", got)
	}
}

func TestRunScenario_AnEditDuringTheRunKeepsTheScenarioNotRun(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow)
	w.runner.outcome = func(req mwanachamaagency.RunRequest) (mwanachamaagency.RunOutcome, error) {
		cur := w.get(t, s.ID)
		cur.Name = "Edited while running"
		if _, err := w.mgr.UpdateScenario(ctx, cur); err != nil {
			t.Errorf("UpdateScenario while running: %v", err)
		}
		return passing(req)
	}
	run, err := w.runToEnd(ctx, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Result != models.TestRunPassed {
		t.Errorf("the run itself = %q, want it kept as it ended", run.Result)
	}
	if got := w.get(t, s.ID).Result; got != models.ScenarioNotRun {
		t.Errorf("scenario = %q, want not_run: the pass was for the scenario as it was before the edit", got)
	}
}

func TestRunScenario_Refusals(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	if _, err := w.runToEnd(ctx, "no-such-scenario"); !errors.Is(err, mwanachamaagency.ErrScenarioNotFound) {
		t.Errorf("an unknown scenario: %v", err)
	}
	s := w.scenario(t, "Shop says yes", w.buyNow)
	if _, err := w.mgr.ArchiveDraft(ctx, w.draftID); err != nil {
		t.Fatal(err)
	}
	if _, err := w.runToEnd(ctx, s.ID); !errors.Is(err, mwanachamaagency.ErrDraftNotOpen) {
		t.Errorf("a scenario of an archived draft: %v, want ErrDraftNotOpen", err)
	}
	if len(w.runner.seen) != 0 {
		t.Errorf("the runner was called %d times for refused runs", len(w.runner.seen))
	}
}

func TestRunScenario_AWorkflowThatIsGoneStopsTheRunWithoutCallingTheRunner(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow)
	w.removeWorkflowRow(t, w.buyNow.ID)
	if _, err := w.runToEnd(ctx, s.ID); !errors.Is(err, mwanachamaagency.ErrWorkflowNotFound) {
		t.Errorf("run of a scenario whose workflow is gone: %v, want ErrWorkflowNotFound", err)
	}
	if len(w.runner.seen) != 0 {
		t.Errorf("the runner was called for a scenario with no workflow")
	}
}

func TestRunScenarios_RunsTheNamedOnesOrEveryOneInTheDraft(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	a := w.scenario(t, "A", w.buyNow)
	b := w.scenario(t, "B", w.groupOrd)
	c := w.scenario(t, "C", w.buyNow)

	runs, err := w.mgr.RunScenarios(ctx, w.draftID, []string{c.ID, a.ID})
	if err != nil {
		t.Fatalf("RunScenarios: %v", err)
	}
	if len(runs) != 2 || runs[0].ScenarioID != c.ID || runs[1].ScenarioID != a.ID {
		t.Errorf("runs = %+v, want C then A", runs)
	}
	w.awaitAll(t, runs)
	if got := w.results(t); got["A"] != models.ScenarioPassed || got["C"] != models.ScenarioPassed || got["B"] != models.ScenarioNotRun {
		t.Errorf("results = %v, want only A and C passed", got)
	}

	all, err := w.mgr.RunScenarios(ctx, w.draftID, nil)
	if err != nil || len(all) != 3 {
		t.Fatalf("run all: %v %d runs, want 3", err, len(all))
	}
	w.awaitAll(t, all)
	_ = b

	before := len(w.runner.seen)
	other, _ := w.mgr.CreateDraft(ctx, "other", "", "")
	if _, err := w.mgr.RunScenarios(ctx, other.ID, []string{a.ID}); !errors.Is(err, mwanachamaagency.ErrScenarioNotFound) {
		t.Errorf("a scenario of another draft: %v, want ErrScenarioNotFound", err)
	}
	if _, err := w.mgr.RunScenarios(ctx, w.draftID, []string{a.ID, "no-such-scenario"}); !errors.Is(err, mwanachamaagency.ErrScenarioNotFound) {
		t.Errorf("an unknown scenario: %v, want ErrScenarioNotFound", err)
	}
	if len(w.runner.seen) != before {
		t.Errorf("a refused batch still ran %d scenarios", len(w.runner.seen)-before)
	}
}

func TestListTestRuns_NewestFirstAndOnlyThisScenarios(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	a := w.scenario(t, "A", w.buyNow)
	b := w.scenario(t, "B", w.groupOrd)
	first, _ := w.runToEnd(ctx, a.ID)
	second, _ := w.runToEnd(ctx, a.ID)
	w.runToEnd(ctx, b.ID)

	runs, err := w.mgr.ListTestRuns(ctx, a.ID)
	if err != nil {
		t.Fatalf("ListTestRuns: %v", err)
	}
	if len(runs) != 2 || runs[0].ID != second.ID || runs[1].ID != first.ID {
		t.Errorf("runs = %v, want the second run then the first", ids(runs))
	}
	if _, err := w.mgr.ListTestRuns(ctx, "no-such-scenario"); !errors.Is(err, mwanachamaagency.ErrScenarioNotFound) {
		t.Errorf("an unknown scenario: %v", err)
	}
	if _, err := w.mgr.GetTestRun(ctx, "no-such-run"); !errors.Is(err, mwanachamaagency.ErrTestRunNotFound) {
		t.Errorf("an unknown run: %v", err)
	}
}

func ids(runs []models.TestRun) []string {
	out := make([]string, len(runs))
	for i, r := range runs {
		out[i] = r.ID
	}
	return out
}

func TestPublish_RefusedUntilEveryScenarioHasPassed(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	a := w.scenario(t, "Shop says yes", w.buyNow)
	b := w.scenario(t, "Group joins", w.groupOrd)

	_, err := w.mgr.PublishAgency(ctx, w.draftID, "", "")
	if !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) {
		t.Fatalf("publish with scenarios not run: %v, want ErrPublishValidationFailed", err)
	}
	for _, name := range []string{"Shop says yes", "Group joins", "Not run yet"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("refusal %q does not name %q", err.Error(), name)
		}
	}

	issues, err := w.mgr.ListIssues(ctx, w.draftID, mwanachamaagency.IssueFilter{EntityType: "scenario"})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]models.IssueSeverity{}
	for _, i := range issues {
		got[i.EntityID] = i.Severity
	}
	if len(issues) != 2 || got[a.ID] != models.IssueSeverityError || got[b.ID] != models.IssueSeverityError {
		t.Errorf("issues = %+v, want one error for each scenario", issues)
	}

	w.pass(t, a.ID)
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "", ""); err == nil || strings.Contains(err.Error(), "Shop says yes") || !strings.Contains(err.Error(), "Group joins") {
		t.Errorf("publish with one scenario left: %v, want it to name only Group joins", err)
	}
	w.pass(t, b.ID)
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "", ""); err != nil {
		t.Fatalf("publish with every scenario passed: %v", err)
	}
}

func TestPublish_FailedStoppedAndRunningScenariosBlockToo(t *testing.T) {
	for _, result := range []models.TestRunResult{models.TestRunFailed, models.TestRunStopped} {
		t.Run(string(result), func(t *testing.T) {
			w := newScenarioWorld(t)
			w.runner.outcome = func(mwanachamaagency.RunRequest) (mwanachamaagency.RunOutcome, error) {
				return mwanachamaagency.RunOutcome{Result: result, StopReason: "x"}, nil
			}
			s := w.scenario(t, "Shop says yes", w.buyNow)
			if _, err := w.runToEnd(context.Background(), s.ID); err != nil {
				t.Fatal(err)
			}
			_, err := w.mgr.PublishAgency(context.Background(), w.draftID, "", "")
			want := models.ScenarioResult(result).Label()
			if !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) || !strings.Contains(err.Error(), want) {
				t.Errorf("publish after a %s run: %v, want a refusal saying %q", result, err, want)
			}
		})
	}

	w := newScenarioWorld(t)
	var during error
	s := w.scenario(t, "Shop says yes", w.buyNow)
	w.runner.outcome = func(req mwanachamaagency.RunRequest) (mwanachamaagency.RunOutcome, error) {
		_, during = w.mgr.PublishAgency(context.Background(), w.draftID, "", "")
		return passing(req)
	}
	w.pass(t, s.ID)
	if !errors.Is(during, mwanachamaagency.ErrPublishValidationFailed) || !strings.Contains(during.Error(), "Running") {
		t.Errorf("publish while a scenario runs: %v, want a refusal saying Running", during)
	}
}

func TestPublish_AnEditAfterAPassBlocksAgain(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow)
	w.pass(t, s.ID)
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "", ""); err != nil {
		t.Fatalf("publish: %v", err)
	}
	cur := w.get(t, s.ID)
	cur.Summary = "changed"
	if _, err := w.mgr.UpdateScenario(ctx, cur); err != nil {
		t.Fatal(err)
	}
	if _, err := w.mgr.CreateValue(ctx, models.Value{DraftID: w.draftID, Text: "so that the content changes"}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "", ""); !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) {
		t.Errorf("publish after editing the passed scenario: %v, want ErrPublishValidationFailed", err)
	}
}

func TestPublish_AScenarioWhoseWorkflowIsGoneBlocksOnce(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow)
	w.pass(t, s.ID)
	w.removeWorkflowRow(t, w.buyNow.ID)
	issues, err := w.mgr.ListIssues(ctx, w.draftID, mwanachamaagency.IssueFilter{EntityType: "scenario"})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 || issues[0].EntityID != s.ID || issues[0].Severity != models.IssueSeverityError {
		t.Fatalf("issues = %+v, want exactly one error for the scenario", issues)
	}
}
