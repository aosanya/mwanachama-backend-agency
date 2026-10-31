package mwanachamaagency_test

import (
	"context"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

const catalogueAction = "forms.results.publish"

func (w scenarioWorld) enforced(t *testing.T, text string) models.Control {
	t.Helper()
	c, err := w.mgr.CreateControl(context.Background(), models.Control{
		DraftID:  w.draftID,
		Text:     text,
		Action:   catalogueAction,
		OnBreach: models.OnBreachBlock,
		Severity: models.SeverityHigh,
	})
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	return c
}

func (w scenarioWorld) attachToWorkflow(t *testing.T, c models.Control, wf models.Workflow) {
	t.Helper()
	if _, err := w.mgr.CreateControlAttachment(context.Background(), c.ID, models.TargetTypeWorkflow, wf.ID); err != nil {
		t.Fatalf("CreateControlAttachment: %v", err)
	}
}

func TestControlStrengthOf_AnActionlessControlIsAdvisory(t *testing.T) {
	w := newScenarioWorld(t)
	c, err := w.mgr.CreateControl(context.Background(), models.Control{DraftID: w.draftID, Text: "Correctness over speed"})
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	got, err := mwanachamaagency.ControlStrengthOf(context.Background(), w.mgr, w.draftID, c)
	if err != nil || got != models.StrengthAdvisory {
		t.Fatalf("strength = %q, %v, want advisory", got, err)
	}
}

func TestControlStrengthOf_AnEnforcedControlNoScenarioReachesIsEnforced(t *testing.T) {
	w := newScenarioWorld(t)
	c := w.enforced(t, "Only the owner publishes results")
	got, err := mwanachamaagency.ControlStrengthOf(context.Background(), w.mgr, w.draftID, c)
	if err != nil || got != models.StrengthEnforced {
		t.Fatalf("strength = %q, %v, want enforced", got, err)
	}
}

func TestControlStrengthOf_ANotYetRunScenarioDoesNotMakeItTested(t *testing.T) {
	w := newScenarioWorld(t)
	c := w.enforced(t, "Only the owner publishes results")
	w.attachToWorkflow(t, c, w.buyNow)
	w.scenario(t, "Shop says yes", w.buyNow, w.actor("Farmer 1", farmer))

	got, err := mwanachamaagency.ControlStrengthOf(context.Background(), w.mgr, w.draftID, c)
	if err != nil || got != models.StrengthEnforced {
		t.Fatalf("strength = %q, %v, want enforced", got, err)
	}
}

func TestControlStrengthOf_APassedScenarioOnItsWorkflowMakesItTested(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	c := w.enforced(t, "Only the owner publishes results")
	w.attachToWorkflow(t, c, w.buyNow)
	s := w.scenario(t, "Shop says yes", w.buyNow, w.actor("Farmer 1", farmer))
	if _, err := w.runToEnd(ctx, s.ID); err != nil {
		t.Fatalf("RunScenario: %v", err)
	}

	got, err := mwanachamaagency.ControlStrengthOf(ctx, w.mgr, w.draftID, c)
	if err != nil || got != models.StrengthTested {
		t.Fatalf("strength = %q, %v, want tested", got, err)
	}
}

func TestControlStrengthOf_APassedScenarioOnAnotherWorkflowDoesNotCountForIt(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	c := w.enforced(t, "Only the owner publishes results")
	w.attachToWorkflow(t, c, w.groupOrd)
	s := w.scenario(t, "Shop says yes", w.buyNow, w.actor("Farmer 1", farmer))
	if _, err := w.runToEnd(ctx, s.ID); err != nil {
		t.Fatalf("RunScenario: %v", err)
	}

	got, err := mwanachamaagency.ControlStrengthOf(ctx, w.mgr, w.draftID, c)
	if err != nil || got != models.StrengthEnforced {
		t.Fatalf("strength = %q, %v, want enforced", got, err)
	}
}
