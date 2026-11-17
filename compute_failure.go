package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func init() {
	registerComputeFailure()
}

func registerComputeFailure() {
	RegisterStepKind(models.StepKindCompute, StepKindHooks{
		Shape:      computeFailureShape,
		References: computeFailureReferences,
	})
	for _, kind := range append(models.StepKinds(), "") {
		if kind != models.StepKindCompute {
			RegisterStepKind(kind, StepKindHooks{Shape: otherKindFailureShape})
		}
	}
	RegisterIssueCheck(checkComputeFailures)
}

func otherKindFailureShape(wi models.WorkItem) error {
	if wi.OnFailure == nil {
		return nil
	}
	subject := "step with no kind"
	if wi.Kind != "" {
		subject = fmt.Sprintf("%s step", wi.Kind)
	}
	return fmt.Errorf("a %s has no failure choice; only a compute step can say what happens when it cannot work out a result", subject)
}

func computeFailureShape(wi models.WorkItem) error {
	choice := wi.OnFailure
	if choice == nil {
		return nil
	}
	switch choice.Choice {
	case models.FailureStop:
		if choice.StepID != "" {
			return fmt.Errorf("stop names no step, got %s", choice.StepID)
		}
	case models.FailureGoToStep:
		if choice.StepID == "" {
			return errors.New("go_to_step names the step to go to")
		}
		if wi.ID != "" && choice.StepID == wi.ID {
			return errors.New("a step cannot go to itself on failure")
		}
	default:
		return fmt.Errorf("the failure choice %q is neither stop nor go_to_step", choice.Choice)
	}
	return nil
}

func computeFailureReferences(ctx context.Context, lookup StepLookup, wi models.WorkItem) error {
	choice := wi.OnFailure
	if choice == nil || choice.Choice != models.FailureGoToStep || choice.StepID == "" {
		return nil
	}
	target, found, err := lookup.StepOf(ctx, wi.DraftID, choice.StepID)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("%w: the step to go to on failure, %s, is not a step of this draft", ErrWorkItemNotFound, choice.StepID)
	}
	if target.WorkflowID != wi.WorkflowID {
		return invalidStep("the step to go to on failure, %s, belongs to another workflow", choice.StepID)
	}
	return nil
}

func checkComputeFailures(report StepScanReport, steps []models.WorkItem) error {
	byID := make(map[string]models.WorkItem, len(steps))
	for _, step := range steps {
		byID[step.ID] = step
	}
	next := map[string]string{}
	for _, step := range steps {
		if step.Kind != models.StepKindCompute {
			continue
		}
		if target, ok := scanFailureChoice(report, step, byID); ok {
			next[step.ID] = target
		}
	}
	reportFailureLoops(report, byID, next)
	return nil
}

func scanFailureChoice(report StepScanReport, step models.WorkItem, byID map[string]models.WorkItem) (string, bool) {
	choice := step.OnFailure
	if choice == nil {
		report.Add(models.IssueSeverityError, "work_item", step.ID,
			"Work item %s %q has no answer for a failure; choose stop or a step to go to", step.Code, step.Title)
		return "", false
	}
	if choice.Choice != models.FailureGoToStep || choice.StepID == "" {
		return "", false
	}
	if choice.StepID == step.ID {
		report.Add(models.IssueSeverityError, "work_item", step.ID,
			"Work item %s %q goes on failure to itself", step.Code, step.Title)
		return "", false
	}
	target, exists := byID[choice.StepID]
	if !exists {
		report.Add(models.IssueSeverityError, "work_item", step.ID,
			"Work item %s %q goes on failure to Step %s, which does not exist", step.Code, step.Title, choice.StepID)
		return "", false
	}
	if target.WorkflowID != step.WorkflowID {
		report.Add(models.IssueSeverityError, "work_item", step.ID,
			"Work item %s %q goes on failure to Step %s, which belongs to another workflow", step.Code, step.Title, choice.StepID)
		return "", false
	}
	if target.Kind != models.StepKindCompute {
		return "", false
	}
	return target.ID, true
}

func reportFailureLoops(report StepScanReport, byID map[string]models.WorkItem, next map[string]string) {
	starts := make([]string, 0, len(next))
	for id := range next {
		starts = append(starts, id)
	}
	sort.Strings(starts)
	const (
		onPath = 1
		walked = 2
	)
	state := map[string]int{}
	for _, start := range starts {
		var path []string
		node := start
		for {
			if state[node] == walked {
				break
			}
			if state[node] == onPath {
				reportFailureLoop(report, byID, loopFrom(path, node))
				break
			}
			target, ok := next[node]
			state[node] = onPath
			path = append(path, node)
			if !ok {
				break
			}
			node = target
		}
		for _, id := range path {
			state[id] = walked
		}
	}
}

func loopFrom(path []string, entry string) []string {
	for i, id := range path {
		if id == entry {
			return path[i:]
		}
	}
	return nil
}

func reportFailureLoop(report StepScanReport, byID map[string]models.WorkItem, members []string) {
	first := 0
	for i, id := range members {
		if id < members[first] {
			first = i
		}
	}
	ordered := append(append([]string(nil), members[first:]...), members[:first]...)
	named := make([]string, 0, len(ordered))
	for _, id := range ordered {
		step := byID[id]
		named = append(named, fmt.Sprintf("%s %q", step.Code, step.Title))
	}
	report.Add(models.IssueSeverityError, "work_item", ordered[0],
		"Work items %s fail onto each other with no way out; one of them must stop or go to a step outside the loop",
		strings.Join(named, ", "))
}
