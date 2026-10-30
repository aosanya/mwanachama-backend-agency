package mwanachamaagency_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"gorm.io/gorm"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type gatedRunner struct {
	mu       sync.Mutex
	gates    map[string]chan struct{}
	started  chan mwanachamaagency.RunRequest
	reported chan string
}

func newGatedRunner() *gatedRunner {
	return &gatedRunner{
		gates:    map[string]chan struct{}{},
		started:  make(chan mwanachamaagency.RunRequest, 64),
		reported: make(chan string, 64),
	}
}

func (g *gatedRunner) gate(scenarioID string) chan struct{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.gates[scenarioID] == nil {
		g.gates[scenarioID] = make(chan struct{}, 64)
	}
	return g.gates[scenarioID]
}

func (g *gatedRunner) letStepsFinish(scenarioID string, steps int) {
	for i := 0; i < steps; i++ {
		g.gate(scenarioID) <- struct{}{}
	}
}

func (g *gatedRunner) Run(ctx context.Context, req mwanachamaagency.RunRequest) (mwanachamaagency.RunOutcome, error) {
	return g.RunReporting(ctx, req, func(models.RunStep) bool { return true })
}

func (g *gatedRunner) RunReporting(_ context.Context, req mwanachamaagency.RunRequest, finished func(models.RunStep) bool) (mwanachamaagency.RunOutcome, error) {
	g.started <- req
	gate := g.gate(req.Scenario.ID)
	var fresh []models.RunStep
	tokens := 0
	for _, b := range req.Scenario.Blocks[len(req.Done):] {
		<-gate
		step := models.RunStep{StepID: b.StepID, Step: b.Title, Actor: b.Actor, Did: b.Action, Tokens: 100}
		fresh = append(fresh, step)
		tokens += step.Tokens
		carryOn := finished(step)
		g.reported <- req.RunID
		if !carryOn {
			return mwanachamaagency.RunOutcome{Result: models.TestRunPassed, Steps: fresh, Tokens: tokens}, nil
		}
	}
	return mwanachamaagency.RunOutcome{Result: models.TestRunPassed, Steps: fresh, Tokens: tokens}, nil
}

func newGatedWorld(t *testing.T, limit int) (scenarioWorld, *gatedRunner) {
	t.Helper()
	g := newGatedRunner()
	w := newScenarioWorld(t,
		mwanachamaagency.WithScenarioRunner(g),
		mwanachamaagency.WithScenariosAtOnce(func(context.Context) int { return limit }),
	)
	return w, g
}

func (w scenarioWorld) press(t *testing.T, id string) models.TestRun {
	t.Helper()
	run, err := w.mgr.RunScenario(context.Background(), id)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	return run
}

func (w scenarioWorld) read(t *testing.T, runID string) models.TestRun {
	t.Helper()
	run, err := w.mgr.GetTestRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("GetTestRun: %v", err)
	}
	return run
}

func (w scenarioWorld) scenarios(t *testing.T, n int) []models.Scenario {
	t.Helper()
	out := make([]models.Scenario, n)
	for i := range out {
		out[i] = w.scenario(t, "Scenario "+string(rune('A'+i)), w.buyNow)
	}
	return out
}

func TestRunScenario_AnswersAtOnceWithTheRunRunning(t *testing.T) {
	w, g := newGatedWorld(t, 0)
	s := w.scenario(t, "Shop says yes", w.buyNow)

	run := w.press(t, s.ID)
	if run.Result != models.TestRunRunning || run.FinishedAt != "" || run.StartedAt == "" || run.QueuedAt == "" {
		t.Fatalf("run = %+v, want Running, started and queued, not finished", run)
	}
	<-g.started
	if got := w.get(t, s.ID).Result; got != models.ScenarioRunning {
		t.Errorf("scenario = %q while the runner has not finished, want running", got)
	}
	if got := w.read(t, run.ID).Result; got != models.TestRunRunning {
		t.Errorf("stored run = %q while the runner has not finished, want running", got)
	}

	g.letStepsFinish(s.ID, 3)
	ended, err := w.mgr.AwaitRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("AwaitRun: %v", err)
	}
	if ended.Result != models.TestRunPassed || len(ended.Steps) != 3 || ended.Tokens != 300 || ended.FinishedAt == "" {
		t.Errorf("ended run = %+v, want passed with three steps and 300 tokens", ended)
	}
	if got := w.get(t, s.ID).Result; got != models.ScenarioPassed {
		t.Errorf("scenario = %q, want passed", got)
	}
}

func TestRunScenario_StepsAreWrittenIntoTheRunAsTheyFinish(t *testing.T) {
	w, g := newGatedWorld(t, 0)
	s := w.scenario(t, "Shop says yes", w.buyNow)
	run := w.press(t, s.ID)
	<-g.started

	g.letStepsFinish(s.ID, 1)
	<-g.reported
	partial := w.read(t, run.ID)
	if partial.Result != models.TestRunRunning || len(partial.Steps) != 1 || partial.Tokens != 100 || partial.Steps[0].StepID != s.Blocks[0].StepID {
		t.Fatalf("run after one step = %+v, want Running with the first step and 100 tokens", partial)
	}

	g.letStepsFinish(s.ID, 1)
	<-g.reported
	if got := len(w.read(t, run.ID).Steps); got != 2 {
		t.Errorf("run after two steps holds %d steps, want 2", got)
	}

	g.letStepsFinish(s.ID, 1)
	ended, _ := w.mgr.AwaitRun(context.Background(), run.ID)
	if len(ended.Steps) != 3 || ended.Result != models.TestRunPassed {
		t.Errorf("ended run = %+v, want three steps, passed", ended)
	}
}

func TestRunScenario_TheCallersContextCannotCancelTheRun(t *testing.T) {
	w, g := newGatedWorld(t, 0)
	s := w.scenario(t, "Shop says yes", w.buyNow)
	ctx, cancel := context.WithCancel(context.Background())
	run, err := w.mgr.RunScenario(ctx, s.ID)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	cancel()
	req := <-g.started
	if req.RunID != run.ID {
		t.Fatalf("runner started %s, want %s", req.RunID, run.ID)
	}
	g.letStepsFinish(s.ID, 3)
	ended, err := w.mgr.AwaitRun(context.Background(), run.ID)
	if err != nil || ended.Result != models.TestRunPassed {
		t.Errorf("run after the caller gave up = %+v, %v, want passed", ended, err)
	}
}

func TestRunScenarios_WithNoLimitEveryRunStartsAtOnce(t *testing.T) {
	w, g := newGatedWorld(t, 0)
	scenarios := w.scenarios(t, 6)
	runs, err := w.mgr.RunScenarios(context.Background(), w.draftID, nil)
	if err != nil {
		t.Fatalf("RunScenarios: %v", err)
	}
	for _, run := range runs {
		if run.Result != models.TestRunRunning {
			t.Errorf("run of %s = %q, want running: no limit is set", run.ScenarioID, run.Result)
		}
	}
	for range runs {
		<-g.started
	}
	for _, s := range scenarios {
		g.letStepsFinish(s.ID, 3)
	}
	w.awaitAll(t, runs)
}

func TestRunScenario_PastTheLimitRunsWaitInTheOrderPressed(t *testing.T) {
	w, g := newGatedWorld(t, 2)
	scenarios := w.scenarios(t, 6)

	runs := make([]models.TestRun, len(scenarios))
	for i, s := range scenarios {
		runs[i] = w.press(t, s.ID)
	}
	for i, run := range runs {
		want := models.TestRunWaiting
		if i < 2 {
			want = models.TestRunRunning
		}
		if run.Result != want {
			t.Errorf("run %d answered %q, want %q", i, run.Result, want)
		}
		if i > 0 && run.QueuedAt <= runs[i-1].QueuedAt {
			t.Errorf("run %d queued at %s, not after run %d at %s", i, run.QueuedAt, i-1, runs[i-1].QueuedAt)
		}
		if i >= 2 && (run.StartedAt != "" || run.Result.Finished()) {
			t.Errorf("waiting run %d = %+v, want not started", i, run)
		}
		scenarioWant := models.ScenarioWaiting
		if i < 2 {
			scenarioWant = models.ScenarioRunning
		}
		if got := w.get(t, scenarios[i].ID).Result; got != scenarioWant {
			t.Errorf("scenario %d = %q, want %q", i, got, scenarioWant)
		}
	}
	if _, err := w.mgr.RunScenario(context.Background(), scenarios[3].ID); !errors.Is(err, mwanachamaagency.ErrScenarioRunning) {
		t.Errorf("pressing a waiting scenario again: %v, want ErrScenarioRunning", err)
	}

	first, second := <-g.started, <-g.started
	if first.RunID != runs[0].ID || second.RunID != runs[1].ID {
		t.Fatalf("started %s and %s, want the first two pressed", first.RunID, second.RunID)
	}
	if got := w.read(t, runs[2].ID).Result; got != models.TestRunWaiting {
		t.Fatalf("third run = %q, want waiting while two run", got)
	}

	g.letStepsFinish(scenarios[1].ID, 3)
	if _, err := w.mgr.AwaitRun(context.Background(), runs[1].ID); err != nil {
		t.Fatal(err)
	}
	third := <-g.started
	if third.RunID != runs[2].ID {
		t.Errorf("started %s when one ended, want the third pressed (%s)", third.RunID, runs[2].ID)
	}
	if got := w.read(t, runs[2].ID); got.Result != models.TestRunRunning || got.StartedAt == "" {
		t.Errorf("third run = %+v, want running and started", got)
	}
	for _, i := range []int{3, 4, 5} {
		if got := w.read(t, runs[i].ID).Result; got != models.TestRunWaiting {
			t.Errorf("run %d = %q, want still waiting", i, got)
		}
	}
	if got := w.get(t, scenarios[2].ID).Result; got != models.ScenarioRunning {
		t.Errorf("third scenario = %q, want running", got)
	}

	for _, s := range scenarios {
		g.letStepsFinish(s.ID, 3)
	}
	w.awaitAll(t, runs)
	for i, run := range runs {
		if got := w.read(t, run.ID).Result; got != models.TestRunPassed {
			t.Errorf("run %d ended %q, want passed", i, got)
		}
	}
}

func TestRunScenario_ALimitRaisedWhileRunsWaitStartsThemAsOthersEnd(t *testing.T) {
	var limitMu sync.Mutex
	limit := 1
	g := newGatedRunner()
	w := newScenarioWorld(t,
		mwanachamaagency.WithScenarioRunner(g),
		mwanachamaagency.WithScenariosAtOnce(func(context.Context) int {
			limitMu.Lock()
			defer limitMu.Unlock()
			return limit
		}),
	)
	scenarios := w.scenarios(t, 4)
	runs := make([]models.TestRun, len(scenarios))
	for i, s := range scenarios {
		runs[i] = w.press(t, s.ID)
	}
	<-g.started
	limitMu.Lock()
	limit = 3
	limitMu.Unlock()

	g.letStepsFinish(scenarios[0].ID, 3)
	if _, err := w.mgr.AwaitRun(context.Background(), runs[0].ID); err != nil {
		t.Fatal(err)
	}
	for _, i := range []int{1, 2, 3} {
		if got := w.read(t, runs[i].ID).Result; got != models.TestRunRunning {
			t.Errorf("run %d = %q, want running now the limit allows three", i, got)
		}
	}
	for _, s := range scenarios {
		g.letStepsFinish(s.ID, 3)
	}
	w.awaitAll(t, runs)
}

func TestStopRun_ARunningRunEndsAfterItsCurrentStep(t *testing.T) {
	w, g := newGatedWorld(t, 0)
	s := w.scenario(t, "Shop says yes", w.buyNow)
	run := w.press(t, s.ID)
	<-g.started
	g.letStepsFinish(s.ID, 1)
	<-g.reported

	answer, err := w.mgr.StopRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("StopRun: %v", err)
	}
	if answer.Result != models.TestRunRunning {
		t.Errorf("answer = %q, want the run still running until its step ends", answer.Result)
	}

	g.letStepsFinish(s.ID, 1)
	ended, err := w.mgr.AwaitRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("AwaitRun: %v", err)
	}
	if ended.Result != models.TestRunStopped || ended.StopReason != "Stopped by the designer" {
		t.Errorf("ended = %q %q, want Stopped, Stopped by the designer", ended.Result, ended.StopReason)
	}
	if len(ended.Steps) != 2 || ended.Tokens != 200 || ended.FinishedAt == "" {
		t.Errorf("ended = %+v, want the two steps done, 200 tokens", ended)
	}
	if got := w.get(t, s.ID); got.Result != models.ScenarioNotRun || got.Result.Label() != "Not run yet" {
		t.Errorf("scenario = %q, want Not run yet", got.Result)
	}
}

func TestStopRun_AWaitingRunEndsAtOnceAndNeverStarts(t *testing.T) {
	w, g := newGatedWorld(t, 1)
	scenarios := w.scenarios(t, 3)
	runs := []models.TestRun{w.press(t, scenarios[0].ID), w.press(t, scenarios[1].ID), w.press(t, scenarios[2].ID)}
	<-g.started

	stopped, err := w.mgr.StopRun(context.Background(), runs[1].ID)
	if err != nil {
		t.Fatalf("StopRun: %v", err)
	}
	if stopped.Result != models.TestRunStopped || stopped.StopReason != "Stopped by the designer" || stopped.FinishedAt == "" {
		t.Errorf("stopped = %+v, want Stopped by the designer, finished", stopped)
	}
	if got := w.get(t, scenarios[1].ID).Result; got != models.ScenarioNotRun {
		t.Errorf("scenario = %q, want not_run", got)
	}

	g.letStepsFinish(scenarios[0].ID, 3)
	if _, err := w.mgr.AwaitRun(context.Background(), runs[0].ID); err != nil {
		t.Fatal(err)
	}
	next := <-g.started
	if next.RunID != runs[2].ID {
		t.Errorf("started %s after the first ended, want the third (%s): the stopped one is skipped", next.RunID, runs[2].ID)
	}
	if got := w.read(t, runs[1].ID); got.Result != models.TestRunStopped {
		t.Errorf("stopped run = %q afterwards, want it left Stopped", got.Result)
	}
	g.letStepsFinish(scenarios[2].ID, 3)
	w.awaitAll(t, runs)
}

func TestStopRun_AFinishedRunIsRefused(t *testing.T) {
	w := newScenarioWorld(t)
	s := w.scenario(t, "Shop says yes", w.buyNow)
	run, err := w.runToEnd(context.Background(), s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.mgr.StopRun(context.Background(), run.ID); !errors.Is(err, mwanachamaagency.ErrTestRunFinished) {
		t.Errorf("stopping a finished run: %v, want ErrTestRunFinished", err)
	}
	if got := w.read(t, run.ID); got.Result != models.TestRunPassed || got.StopReason != "" {
		t.Errorf("run = %+v, want it left passed", got)
	}
	if got := w.get(t, s.ID).Result; got != models.ScenarioPassed {
		t.Errorf("scenario = %q, want it left passed", got)
	}
	if _, err := w.mgr.StopRun(context.Background(), "no-such-run"); !errors.Is(err, mwanachamaagency.ErrTestRunNotFound) {
		t.Errorf("stopping an unknown run: %v, want ErrTestRunNotFound", err)
	}
}

func TestStopRun_ARunnerThatReportsNoStepsStillEndsStopped(t *testing.T) {
	w := newScenarioWorld(t)
	s := w.scenario(t, "Shop says yes", w.buyNow)
	var answered error
	w.runner.outcome = func(req mwanachamaagency.RunRequest) (mwanachamaagency.RunOutcome, error) {
		_, answered = w.mgr.StopRun(context.Background(), req.RunID)
		return passing(req)
	}
	run, err := w.runToEnd(context.Background(), s.ID)
	if err != nil || answered != nil {
		t.Fatalf("run: %v, stop: %v", err, answered)
	}
	if run.Result != models.TestRunStopped || run.StopReason != "Stopped by the designer" || len(run.Steps) != 3 {
		t.Errorf("run = %+v, want Stopped by the designer with the steps it had", run)
	}
	if got := w.get(t, s.ID).Result; got != models.ScenarioNotRun {
		t.Errorf("scenario = %q, want not_run", got)
	}
}

func restartedManager(t *testing.T, db *gorm.DB, opts ...mwanachamaagency.Option) mwanachamaagency.AgencyManager {
	t.Helper()
	s, err := mwanachamaagency.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	mgr, err := mwanachamaagency.NewAgencyManager(db, s, opts...)
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	return mgr
}

func TestResumeRuns_ARunLeftRunningContinuesAtItsNextStep(t *testing.T) {
	w, g := newGatedWorld(t, 0)
	s := w.scenario(t, "Shop says yes", w.buyNow)
	run := w.press(t, s.ID)
	<-g.started
	g.letStepsFinish(s.ID, 1)
	<-g.reported

	g2 := newGatedRunner()
	after := restartedManager(t, w.db, mwanachamaagency.WithScenarioRunner(g2))
	resumed, err := after.ResumeRuns(context.Background())
	if err != nil {
		t.Fatalf("ResumeRuns: %v", err)
	}
	if len(resumed) != 1 || resumed[0].ID != run.ID || resumed[0].Result != models.TestRunRunning {
		t.Fatalf("resumed = %+v, want the one running run", resumed)
	}
	req := <-g2.started
	if req.RunID != run.ID || len(req.Done) != 1 || req.Done[0].StepID != s.Blocks[0].StepID {
		t.Fatalf("resumed request = run %s done %+v, want the first step handed back as done", req.RunID, req.Done)
	}

	g2.letStepsFinish(s.ID, 2)
	ended, err := after.AwaitRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("AwaitRun: %v", err)
	}
	if ended.Result != models.TestRunPassed || len(ended.Steps) != 3 || ended.Tokens != 300 {
		t.Fatalf("ended = %+v, want passed with three steps and 300 tokens", ended)
	}
	for i, step := range ended.Steps {
		if step.StepID != s.Blocks[i].StepID {
			t.Errorf("step %d = %s, want %s: a stored step must not repeat", i, step.StepID, s.Blocks[i].StepID)
		}
	}
	if got := len(g2.reported); got != 2 {
		t.Errorf("the resumed runner played %d steps, want 2", got)
	}
}

func TestResumeRuns_ARunnerThatIgnoresDoneStillLeavesNoRepeatedStep(t *testing.T) {
	w, g := newGatedWorld(t, 0)
	s := w.scenario(t, "Shop says yes", w.buyNow)
	run := w.press(t, s.ID)
	<-g.started
	g.letStepsFinish(s.ID, 2)
	<-g.reported
	<-g.reported

	forgetful := &scriptedRunner{outcome: passing}
	after := restartedManager(t, w.db, mwanachamaagency.WithScenarioRunner(forgetful))
	if _, err := after.ResumeRuns(context.Background()); err != nil {
		t.Fatalf("ResumeRuns: %v", err)
	}
	ended, err := after.AwaitRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("AwaitRun: %v", err)
	}
	if len(ended.Steps) != 3 {
		t.Errorf("steps = %d, want 3 with the two stored ones not repeated", len(ended.Steps))
	}
}

func TestResumeRuns_TheWaitingOrderSurvivesARestart(t *testing.T) {
	w, g := newGatedWorld(t, 1)
	scenarios := w.scenarios(t, 4)
	runs := make([]models.TestRun, len(scenarios))
	for i, s := range scenarios {
		runs[i] = w.press(t, s.ID)
	}
	<-g.started
	g.letStepsFinish(scenarios[0].ID, 1)
	<-g.reported

	g2 := newGatedRunner()
	atOnce := mwanachamaagency.WithScenariosAtOnce(func(context.Context) int { return 1 })
	after := restartedManager(t, w.db, mwanachamaagency.WithScenarioRunner(g2), atOnce)
	resumed, err := after.ResumeRuns(context.Background())
	if err != nil {
		t.Fatalf("ResumeRuns: %v", err)
	}
	if got := ids(resumed); len(got) != 4 || got[0] != runs[0].ID || got[1] != runs[1].ID || got[2] != runs[2].ID || got[3] != runs[3].ID {
		t.Fatalf("resumed = %v, want the four in the order pressed", got)
	}
	for i, run := range resumed {
		want := models.TestRunWaiting
		if i == 0 {
			want = models.TestRunRunning
		}
		if run.Result != want {
			t.Errorf("resumed run %d = %q, want %q", i, run.Result, want)
		}
	}

	for i, s := range scenarios {
		req := <-g2.started
		if req.RunID != runs[i].ID {
			t.Fatalf("restart started run %d as %s, want %s: order lost", i, req.RunID, runs[i].ID)
		}
		if i == 0 && len(req.Done) != 1 {
			t.Errorf("first resumed run was handed %d done steps, want 1", len(req.Done))
		}
		g2.letStepsFinish(s.ID, 3)
		if _, err := after.AwaitRun(context.Background(), runs[i].ID); err != nil {
			t.Fatal(err)
		}
	}
	for _, run := range runs {
		got, _ := after.GetTestRun(context.Background(), run.ID)
		if got.Result != models.TestRunPassed {
			t.Errorf("run %s = %q, want passed", run.ID, got.Result)
		}
	}
}

func TestResumeRuns_ARestartDemotesRunsBeyondTheLimit(t *testing.T) {
	w, g := newGatedWorld(t, 0)
	scenarios := w.scenarios(t, 3)
	runs := make([]models.TestRun, len(scenarios))
	for i, s := range scenarios {
		runs[i] = w.press(t, s.ID)
		<-g.started
	}

	g2 := newGatedRunner()
	after := restartedManager(t, w.db, mwanachamaagency.WithScenarioRunner(g2),
		mwanachamaagency.WithScenariosAtOnce(func(context.Context) int { return 1 }))
	if _, err := after.ResumeRuns(context.Background()); err != nil {
		t.Fatalf("ResumeRuns: %v", err)
	}
	<-g2.started
	for i, want := range []models.TestRunResult{models.TestRunRunning, models.TestRunWaiting, models.TestRunWaiting} {
		got, _ := after.GetTestRun(context.Background(), runs[i].ID)
		if got.Result != want {
			t.Errorf("run %d = %q, want %q", i, got.Result, want)
		}
	}
	if got := w.get(t, scenarios[2].ID).Result; got != models.ScenarioWaiting {
		t.Errorf("scenario 2 = %q, want waiting", got)
	}
	for i, s := range scenarios {
		if i > 0 {
			<-g2.started
		}
		g2.letStepsFinish(s.ID, 3)
		if _, err := after.AwaitRun(context.Background(), runs[i].ID); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResumeRuns_FinishedRunsAreLeftAlone(t *testing.T) {
	w := newScenarioWorld(t)
	s := w.scenario(t, "Shop says yes", w.buyNow)
	run, err := w.runToEnd(context.Background(), s.ID)
	if err != nil {
		t.Fatal(err)
	}
	after := restartedManager(t, w.db)
	resumed, err := after.ResumeRuns(context.Background())
	if err != nil || len(resumed) != 0 {
		t.Errorf("resumed = %v, %v, want nothing", resumed, err)
	}
	if got := w.read(t, run.ID); got.Result != models.TestRunPassed {
		t.Errorf("run = %q, want passed", got.Result)
	}
}
