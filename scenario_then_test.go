package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func blockThen(s models.Scenario, stepID string) string {
	for _, b := range s.Blocks {
		if b.StepID == stepID {
			return b.Then
		}
	}
	return "<no such block>"
}

func TestCreateScenario_ABlockCarriesTheThenLineItWasWrittenWith(t *testing.T) {
	w := newScenarioWorld(t)
	probe := w.scenario(t, "Probe", w.buyNow)
	first, second := probe.Blocks[0].StepID, probe.Blocks[1].StepID

	s, err := w.mgr.CreateScenario(context.Background(), models.Scenario{
		DraftID: w.draftID, Name: "Shop says yes", WorkflowID: w.buyNow.ID,
		Thens: []models.ScenarioThen{{StepID: first, Then: "The order shows urea"}},
	})
	if err != nil {
		t.Fatalf("CreateScenario: %v", err)
	}
	if got := blockThen(s, s.Blocks[0].StepID); got != "The order shows urea" {
		t.Errorf("first block Then = %q, want the line it was written with", got)
	}
	if got := s.Blocks[1].Then; got != "" {
		t.Errorf("second block Then = %q, want none: no line was written for it", got)
	}
	if first == s.Blocks[0].StepID && second != s.Blocks[1].StepID {
		t.Fatalf("steps changed between scenarios of one workflow")
	}

	read := w.get(t, s.ID)
	if got := read.Blocks[0].Then; got != "The order shows urea" {
		t.Errorf("read first block Then = %q, want it returned when read", got)
	}
	listed, err := w.mgr.ListScenarios(context.Background(), w.draftID, mwanachamaagency.ScenarioFilter{Name: "Shop says yes"})
	if err != nil || len(listed) != 1 || listed[0].Blocks[0].Then != "The order shows urea" {
		t.Errorf("listed = %+v %v, want the Then line in the list too", listed, err)
	}
}

func TestUpdateScenario_SetsChangesAndClearsAThenLine(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow)
	a, b := s.Blocks[0].StepID, s.Blocks[1].StepID
	w.pass(t, s.ID)

	cur := w.get(t, s.ID)
	cur.Thens = []models.ScenarioThen{{StepID: a, Then: "First"}, {StepID: b, Then: "Second"}}
	out, err := w.mgr.UpdateScenario(ctx, cur)
	if err != nil {
		t.Fatalf("UpdateScenario: %v", err)
	}
	if out.Blocks[0].Then != "First" || out.Blocks[1].Then != "Second" || out.Result != models.ScenarioNotRun {
		t.Errorf("after setting = %+v, want both lines and the scenario back to not run", out.Blocks)
	}

	change := w.get(t, s.ID)
	change.Thens = []models.ScenarioThen{{StepID: a, Then: "First, changed"}, {StepID: b}}
	out, err = w.mgr.UpdateScenario(ctx, change)
	if err != nil {
		t.Fatalf("UpdateScenario change: %v", err)
	}
	if out.Blocks[0].Then != "First, changed" || out.Blocks[1].Then != "" {
		t.Errorf("after changing = %q / %q, want the first changed and the second cleared", out.Blocks[0].Then, out.Blocks[1].Then)
	}

	untouched := w.get(t, s.ID)
	untouched.Thens = nil
	untouched.Summary = "only the words"
	out, err = w.mgr.UpdateScenario(ctx, untouched)
	if err != nil || out.Blocks[0].Then != "First, changed" {
		t.Errorf("an update that names no lines: %v %q, want the lines it already has", err, out.Blocks[0].Then)
	}
}

func TestScenarioThen_Refusals(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	other := w.scenario(t, "Other flow", w.groupOrd)
	s := w.scenario(t, "Shop says yes", w.buyNow)

	_, err := w.mgr.CreateScenario(ctx, models.Scenario{
		DraftID: w.draftID, Name: "Wrong step", WorkflowID: w.buyNow.ID,
		Thens: []models.ScenarioThen{{StepID: other.Blocks[0].StepID, Then: "x"}},
	})
	if !errors.Is(err, mwanachamaagency.ErrInvalidScenario) {
		t.Errorf("a Then on a step of another workflow: %v, want ErrInvalidScenario", err)
	}
	_, err = w.mgr.CreateScenario(ctx, models.Scenario{
		DraftID: w.draftID, Name: "No step", WorkflowID: w.buyNow.ID,
		Thens: []models.ScenarioThen{{Then: "x"}},
	})
	if !errors.Is(err, mwanachamaagency.ErrInvalidScenario) {
		t.Errorf("a Then naming no step: %v, want ErrInvalidScenario", err)
	}
	cur := w.get(t, s.ID)
	cur.Thens = []models.ScenarioThen{{StepID: "no-such-step", Then: "x"}}
	if _, err := w.mgr.UpdateScenario(ctx, cur); !errors.Is(err, mwanachamaagency.ErrInvalidScenario) {
		t.Errorf("updating with an unknown step: %v, want ErrInvalidScenario", err)
	}
	if got := w.get(t, s.ID); got.Blocks[0].Then != "" {
		t.Errorf("a refused update left Then = %q, want nothing written", got.Blocks[0].Then)
	}
}

func TestScenarioThen_TravelsWithTheScenarioIntoForksAndPublications(t *testing.T) {
	w := newTravelWorld(t)
	ctx := context.Background()
	cur := w.get(t, w.base.ID)
	cur.Thens = []models.ScenarioThen{{StepID: cur.Blocks[0].StepID, Then: "The order shows urea"}}
	if _, err := w.mgr.UpdateScenario(ctx, cur); err != nil {
		t.Fatalf("UpdateScenario: %v", err)
	}
	w.pass(t, cur.ID)

	fork, err := w.mgr.CreateDraft(ctx, "forked", w.draftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if got := w.onlyScenario(t, fork.ID).Blocks[0].Then; got != "The order shows urea" {
		t.Errorf("forked block Then = %q, want the line copied onto the new draft's step", got)
	}

	p, err := w.mgr.PublishAgency(ctx, w.draftID, "v-then", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	imported, err := w.mgr.ImportDocument(ctx, "imported", p.Document)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	if got := w.onlyScenario(t, imported.Draft.ID).Blocks[0].Then; got != "The order shows urea" {
		t.Errorf("imported block Then = %q, want the line carried by the document", got)
	}
}

func TestRunScenario_AStepKeepsWhoPlayedIt(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow, w.actor("Farmer 1", farmer))
	w.runner.outcome = func(req mwanachamaagency.RunRequest) (mwanachamaagency.RunOutcome, error) {
		out, err := passing(req)
		for i := range out.Steps {
			out.Steps[i].PlayedBy = "Test agent · Fill a form"
		}
		return out, err
	}

	run, err := w.mgr.RunScenario(ctx, s.ID)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	for i, step := range run.Steps {
		if step.PlayedBy != "Test agent · Fill a form" {
			t.Errorf("returned step %d PlayedBy = %q, want who played it", i, step.PlayedBy)
		}
	}
	stored, err := w.mgr.GetTestRun(ctx, run.ID)
	if err != nil || len(stored.Steps) != 3 {
		t.Fatalf("GetTestRun: %v %+v", err, stored)
	}
	for i, step := range stored.Steps {
		if step.PlayedBy != "Test agent · Fill a form" {
			t.Errorf("stored step %d PlayedBy = %q, want it kept with the run", i, step.PlayedBy)
		}
	}
}
