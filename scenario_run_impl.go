package mwanachamaagency

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

const noExecutorReason = "No test executor is installed"

type RunRequest struct {
	RunID        string
	Scenario     models.Scenario
	Workflow     models.Workflow
	Rules        []models.Rule
	DraftVersion string
}

type RunOutcome struct {
	Result     models.TestRunResult
	StopReason string
	Steps      []models.RunStep
	Tokens     int
}

type ScenarioRunner interface {
	Run(ctx context.Context, req RunRequest) (RunOutcome, error)
}

type stoppedRunner struct{}

func (stoppedRunner) Run(context.Context, RunRequest) (RunOutcome, error) {
	return RunOutcome{Result: models.TestRunStopped, StopReason: noExecutorReason}, nil
}

func WithScenarioRunner(r ScenarioRunner) Option {
	return func(m *agencyManager) { m.runner = r }
}

func (m *agencyManager) scenarioRunner() ScenarioRunner {
	if m.runner == nil {
		return stoppedRunner{}
	}
	return m.runner
}

func settled(o RunOutcome, err error) RunOutcome {
	switch {
	case err != nil:
		return RunOutcome{Result: models.TestRunStopped, StopReason: fmt.Sprintf("The executor failed: %v", err), Steps: o.Steps, Tokens: o.Tokens}
	case !o.Result.Finished():
		o.Result, o.StopReason = models.TestRunStopped, "The executor ended without a result"
	}
	return o
}

type startedRun struct {
	run      models.TestRun
	scenario models.Scenario
	workflow models.Workflow
	rules    []models.Rule
}

func (m *agencyManager) startRun(ctx context.Context, scenarioID string) (startedRun, error) {
	var started startedRun
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		s, err := m.findScenario(ctx, tx, scenarioID)
		if err != nil {
			return err
		}
		if err := m.requireOpenDraft(ctx, tx, s.DraftID); err != nil {
			return err
		}
		if s.Result == models.ScenarioRunning {
			return ErrScenarioRunning
		}
		var wf models.Workflow
		wq := scopeQuery(tx.WithContext(ctx).Table(m.table(roleWorkflow)), s.DraftID).Where("id = ?", s.WorkflowID)
		if err := m.take(wq, roleWorkflow, &wf, ErrWorkflowNotFound); err != nil {
			return err
		}
		rules, err := m.workflowRules(ctx, tx, s.DraftID, wf)
		if err != nil {
			return err
		}
		version, err := m.contentHashFor(ctx, tx, s.DraftID)
		if err != nil {
			return err
		}

		now := models.NowRFC3339()
		run := models.TestRun{
			ID: newID(), ScenarioID: s.ID, DraftID: s.DraftID, DraftVersion: version,
			Result: models.TestRunRunning, StartedAt: now,
		}
		if err := m.insertRow(ctx, tx, roleTestRun, run); err != nil {
			return err
		}
		s.Result, s.LastRunID, s.LastUpdated = models.ScenarioRunning, run.ID, now
		if err := m.writeUpdate(ctx, tx, roleScenario, s.ID, s, ErrScenarioNotFound); err != nil {
			return err
		}
		view, err := m.scenarioView(ctx, tx, s)
		if err != nil {
			return err
		}
		started = startedRun{run: run, scenario: view, workflow: wf, rules: rules}
		return nil
	})
	return started, err
}

func (m *agencyManager) finishRun(ctx context.Context, started startedRun, outcome RunOutcome) (models.TestRun, error) {
	run := started.run
	run.Result, run.StopReason, run.Steps, run.Tokens = outcome.Result, outcome.StopReason, outcome.Steps, outcome.Tokens
	run.FinishedAt = models.NowRFC3339()
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := m.writeUpdate(ctx, tx, roleTestRun, run.ID, run, ErrTestRunNotFound); err != nil {
			return err
		}
		return tx.WithContext(ctx).Table(m.table(roleScenario)).
			Where("id = ? AND result = ? AND last_run_id = ?", run.ScenarioID, string(models.ScenarioRunning), run.ID).
			Updates(map[string]any{"result": string(run.Result.ScenarioResult()), "last_updated": run.FinishedAt}).Error
	})
	return run, err
}

func (m *agencyManager) RunScenario(ctx context.Context, id string) (models.TestRun, error) {
	started, err := m.startRun(ctx, id)
	if err != nil {
		return models.TestRun{}, fmt.Errorf("RunScenario: %w", err)
	}
	outcome, runErr := m.scenarioRunner().Run(ctx, RunRequest{
		RunID: started.run.ID, Scenario: started.scenario, Workflow: started.workflow,
		Rules: started.rules, DraftVersion: started.run.DraftVersion,
	})
	run, err := m.finishRun(ctx, started, settled(outcome, runErr))
	if err != nil {
		return models.TestRun{}, fmt.Errorf("RunScenario: %w", err)
	}
	return run, nil
}

func (m *agencyManager) RunScenarios(ctx context.Context, draftID string, scenarioIDs []string) ([]models.TestRun, error) {
	ids := scenarioIDs
	inDraft, err := list[models.Scenario](m, scopeQuery(m.q(ctx, roleScenario).Order("created_at"), draftID), roleScenario)
	if err != nil {
		return nil, fmt.Errorf("RunScenarios: %w", err)
	}
	known := make(map[string]bool, len(inDraft))
	for _, s := range inDraft {
		known[s.ID] = true
		if len(scenarioIDs) == 0 {
			ids = append(ids, s.ID)
		}
	}
	for _, id := range ids {
		if !known[id] {
			return nil, fmt.Errorf("RunScenarios: %w: %s is not a scenario of this draft", ErrScenarioNotFound, id)
		}
	}
	runs := make([]models.TestRun, 0, len(ids))
	for _, id := range ids {
		run, err := m.RunScenario(ctx, id)
		if err != nil {
			return runs, err
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func (m *agencyManager) GetTestRun(ctx context.Context, id string) (models.TestRun, error) {
	var run models.TestRun
	if err := m.find(ctx, m.db, roleTestRun, id, &run, ErrTestRunNotFound); err != nil {
		return models.TestRun{}, err
	}
	return run, nil
}

func (m *agencyManager) ListTestRuns(ctx context.Context, scenarioID string) ([]models.TestRun, error) {
	if _, err := m.findScenario(ctx, m.db, scenarioID); err != nil {
		return nil, err
	}
	q := m.q(ctx, roleTestRun).Where("scenario_id = ?", scenarioID).Order("started_at DESC")
	out, err := list[models.TestRun](m, q, roleTestRun)
	if err != nil {
		return nil, fmt.Errorf("ListTestRuns: %w", err)
	}
	return out, nil
}
