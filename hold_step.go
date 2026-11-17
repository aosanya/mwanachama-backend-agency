package mwanachamaagency

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

const (
	holdLeastDays = 1
	holdMostDays  = 90
)

type stepsHolding interface {
	NamingHeldStep(ctx context.Context, draftID, heldStepID string) ([]models.WorkItem, error)
}

func init() {
	registerHoldChecks()
}

func registerHoldChecks() {
	hooks := StepKindHooks{Shape: holdShape, References: holdReferences, Scan: holdScan}
	for _, kind := range append(stepKindsOtherThanPost(), models.StepKindPost) {
		RegisterStepKind(kind, hooks)
	}
	RegisterStepKind(models.StepKindPost, StepKindHooks{Scan: postReadsHeldFigureAfterItIsReleased})
}

func releasesAHeldFigure(kind models.StepKind) bool {
	return kind == models.StepKindYesNo || kind == models.StepKindPost
}

func holdShape(wi models.WorkItem) error {
	if wi.Hold != nil {
		if wi.Kind != models.StepKindCompute {
			return invalidStep("only a calculating step can hold its result, not a %s step", kindWord(wi.Kind))
		}
		if wi.RoleID != "" {
			return invalidStep("a calculating step waits for a person either through the role that confirms it or through a hold, not both")
		}
		if wi.Hold.ExpiresAfterDays < holdLeastDays || wi.Hold.ExpiresAfterDays > holdMostDays {
			return invalidStep("a hold expires after a whole number of days from %d to %d, not %d", holdLeastDays, holdMostDays, wi.Hold.ExpiresAfterDays)
		}
	}
	if wi.HeldStepID == "" {
		return nil
	}
	if !releasesAHeldFigure(wi.Kind) {
		return invalidStep("only a yes_no or post step names the held figure it decides, not a %s step", kindWord(wi.Kind))
	}
	if wi.Kind == models.StepKindPost && wi.AmountStepID != "" && wi.AmountStepID != wi.HeldStepID {
		return invalidStep("a post step that releases a held figure posts that same figure: its held step and its amount step are one step")
	}
	return nil
}

func holdReferences(ctx context.Context, lookup StepLookup, wi models.WorkItem) error {
	if wi.HeldStepID == "" {
		return nil
	}
	if wi.ID != "" && wi.HeldStepID == wi.ID {
		return invalidStep("a step cannot release its own figure")
	}
	held, found, err := lookup.StepOf(ctx, wi.DraftID, wi.HeldStepID)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("%w: the held step %s is not a step of this draft", ErrWorkItemNotFound, wi.HeldStepID)
	}
	switch {
	case held.WorkflowID != wi.WorkflowID:
		return invalidStep("the held step %q belongs to another workflow", held.Title)
	case held.Ordinality >= wi.Ordinality:
		return invalidStep("the held step %q is not earlier than this step", held.Title)
	case held.Kind != models.StepKindCompute:
		return invalidStep("the held step %q is not a calculating step", held.Title)
	case held.Hold == nil:
		return invalidStep("the calculating step %q has no hold, so it has no held figure to release", held.Title)
	}
	return onlyStepNamingHeldStep(ctx, lookup, wi)
}

func onlyStepNamingHeldStep(ctx context.Context, lookup StepLookup, wi models.WorkItem) error {
	finder, ok := lookup.(stepsHolding)
	if !ok {
		return nil
	}
	others, err := finder.NamingHeldStep(ctx, wi.DraftID, wi.HeldStepID)
	if err != nil {
		return err
	}
	for _, other := range others {
		if other.ID != wi.ID {
			return invalidStep("the held figure of that calculating step is already released by step %s %q", other.Code, other.Title)
		}
	}
	return nil
}

func (l managerStepLookup) NamingHeldStep(ctx context.Context, draftID, heldStepID string) ([]models.WorkItem, error) {
	q := scopeQuery(l.tx.WithContext(ctx).Table(l.m.table(roleWorkItem)), draftID).Where("held_step_id = ?", heldStepID)
	return list[models.WorkItem](l.m, q, roleWorkItem)
}

func holdScan(report StepScanReport, wi models.WorkItem, steps []models.WorkItem) {
	if wi.Hold != nil {
		scanHold(report, wi, steps)
	}
	if wi.HeldStepID != "" {
		scanHeldStep(report, wi, steps)
	}
}

func scanHold(report StepScanReport, wi models.WorkItem, steps []models.WorkItem) {
	if wi.Kind != models.StepKindCompute {
		report.Add(models.IssueSeverityError, "work_item", wi.ID,
			"Work item %s %q is a %s step with a hold, and only a calculating step can hold its result", wi.Code, wi.Title, kindWord(wi.Kind))
	}
	if wi.Hold.ExpiresAfterDays < holdLeastDays || wi.Hold.ExpiresAfterDays > holdMostDays {
		report.Add(models.IssueSeverityError, "work_item", wi.ID,
			"Work item %s %q holds its result for %d days, and a hold lasts %d to %d days", wi.Code, wi.Title, wi.Hold.ExpiresAfterDays, holdLeastDays, holdMostDays)
	}
	if wi.Kind == models.StepKindCompute && len(releasingSteps(steps, wi.ID)) == 0 {
		report.Add(models.IssueSeverityError, "work_item", wi.ID,
			"Work item %s %q holds its result, and no step releases it", wi.Code, wi.Title)
	}
}

func releasingSteps(steps []models.WorkItem, heldStepID string) []models.WorkItem {
	var releasing []models.WorkItem
	for _, step := range steps {
		if step.HeldStepID == heldStepID {
			releasing = append(releasing, step)
		}
	}
	return releasing
}

func scanHeldStep(report StepScanReport, wi models.WorkItem, steps []models.WorkItem) {
	if !releasesAHeldFigure(wi.Kind) {
		report.Add(models.IssueSeverityError, "work_item", wi.ID,
			"Work item %s %q is a %s step that names a held figure, and only a yes_no or post step can release one", wi.Code, wi.Title, kindWord(wi.Kind))
		return
	}
	held, found := stepByID(steps, wi.HeldStepID)
	switch {
	case !found:
		report.Add(models.IssueSeverityError, "work_item", wi.ID,
			"Work item %s %q releases the figure of step %s, which does not exist", wi.Code, wi.Title, QuoteIfUnsafe(wi.HeldStepID))
	case held.WorkflowID != wi.WorkflowID:
		report.Add(models.IssueSeverityError, "work_item", wi.ID,
			"Work item %s %q releases the figure of step %s %q, which belongs to another workflow", wi.Code, wi.Title, held.Code, held.Title)
	case held.Ordinality >= wi.Ordinality:
		report.Add(models.IssueSeverityError, "work_item", wi.ID,
			"Work item %s %q releases the figure of step %s %q, which is not earlier than it", wi.Code, wi.Title, held.Code, held.Title)
	case held.Kind != models.StepKindCompute:
		report.Add(models.IssueSeverityError, "work_item", wi.ID,
			"Work item %s %q releases the figure of step %s %q, which is no longer a calculating step", wi.Code, wi.Title, held.Code, held.Title)
	case held.Hold == nil:
		report.Add(models.IssueSeverityError, "work_item", wi.ID,
			"Work item %s %q releases the figure of step %s %q, which has no hold", wi.Code, wi.Title, held.Code, held.Title)
	}
	if wi.RoleID == "" {
		report.Add(models.IssueSeverityError, "work_item", wi.ID,
			"Work item %s %q releases a held figure and names no role: the agency cannot approve its own figure", wi.Code, wi.Title)
	}
}

func postReadsHeldFigureAfterItIsReleased(report StepScanReport, wi models.WorkItem, steps []models.WorkItem) {
	if wi.AmountStepID == "" || wi.HeldStepID != "" {
		return
	}
	amount, found := stepByID(steps, wi.AmountStepID)
	if !found || amount.Hold == nil {
		return
	}
	releasing := releasingSteps(steps, amount.ID)
	if len(releasing) == 0 {
		return
	}
	for _, step := range releasing {
		if step.WorkflowID == wi.WorkflowID && step.Ordinality < wi.Ordinality {
			return
		}
	}
	report.Add(models.IssueSeverityError, "work_item", wi.ID,
		"Work item %s %q posts the held figure of step %s %q before the step that releases it", wi.Code, wi.Title, amount.Code, amount.Title)
}
