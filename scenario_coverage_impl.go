package mwanachamaagency

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (s *issueScan) checkScenarioCoverage() error {
	steps, err := list[models.WorkItem](s.m, s.scoped(roleWorkItem).Order("created_at"), roleWorkItem)
	if err != nil {
		return err
	}
	needed := stepsNeedingCoverage(steps)
	if len(needed) == 0 {
		return nil
	}
	played, err := s.stepsPlayedByCurrentRuns()
	if err != nil {
		return err
	}
	for _, step := range needed {
		if !played[step.ID] {
			s.add(models.IssueSeverityWarning, "work_item", step.ID,
				"Work item %s %q is not passed through by any scenario", step.Code, step.Title)
		}
	}
	return nil
}

func stepsNeedingCoverage(steps []models.WorkItem) []models.WorkItem {
	var needed []models.WorkItem
	for _, step := range steps {
		if !step.Deleted && (step.Kind == models.StepKindCompute || step.Kind == models.StepKindPost) {
			needed = append(needed, step)
		}
	}
	return needed
}

func (s *issueScan) stepsPlayedByCurrentRuns() (map[string]bool, error) {
	scenarios, err := list[models.Scenario](s.m, s.scoped(roleScenario), roleScenario)
	if err != nil {
		return nil, err
	}
	runs, err := list[models.TestRun](s.m, s.scoped(roleTestRun), roleTestRun)
	if err != nil {
		return nil, err
	}
	version, err := s.m.contentHashFor(s.ctx, s.tx, s.draftID)
	if err != nil {
		return nil, err
	}
	runByID := make(map[string]models.TestRun, len(runs))
	for _, run := range runs {
		runByID[run.ID] = run
	}
	played := map[string]bool{}
	for _, sc := range scenarios {
		run, ok := runByID[sc.LastRunID]
		if !ok || run.DraftVersion != version {
			continue
		}
		for _, step := range run.Steps {
			played[step.StepID] = true
		}
	}
	return played, nil
}
