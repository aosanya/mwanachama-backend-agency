package mwanachamaagency_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func storedActorsOf(s models.Scenario) []models.ScenarioActor {
	var out []models.ScenarioActor
	for _, a := range s.Actors {
		if !a.BuiltIn {
			out = append(out, a)
		}
	}
	return out
}

func actorNamed(t *testing.T, s models.Scenario, name string) models.ScenarioActor {
	t.Helper()
	for _, a := range storedActorsOf(s) {
		if a.Name == name {
			return a
		}
	}
	t.Fatalf("scenario has no stored actor %q in %+v", name, s.Actors)
	return models.ScenarioActor{}
}

func (w scenarioWorld) setOf(t *testing.T, id string, patch func(s *models.Scenario)) (models.Scenario, error) {
	t.Helper()
	body := models.Scenario{ID: id, Name: w.get(t, id).Name}
	patch(&body)
	return w.mgr.UpdateScenario(context.Background(), body)
}

func TestUpdateScenario_AppliesActorsAndInputsAsOneSet(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow,
		w.actor("Farmer 1", farmer), w.actor("Shop", supplier), w.actor("Helper", coordinator))
	ask := s.Blocks[0].StepID
	if _, err := w.mgr.SetScenarioInput(ctx, models.ScenarioInput{ScenarioID: s.ID, StepID: ask, Field: "Enter the product", Value: "urea"}); err != nil {
		t.Fatalf("SetScenarioInput: %v", err)
	}
	w.pass(t, s.ID)
	before := w.get(t, s.ID)
	farmerActor, shopActor := actorNamed(t, before, "Farmer 1"), actorNamed(t, before, "Shop")
	runsBefore, err := w.mgr.ListTestRuns(ctx, s.ID)
	if err != nil {
		t.Fatalf("ListTestRuns: %v", err)
	}

	out, err := w.setOf(t, s.ID, func(u *models.Scenario) {
		u.Actors = []models.ScenarioActor{
			{ID: farmerActor.ID, Name: "Mama Akinyi"},
			{Name: "Shop", RoleID: w.roles[coordinator].ID},
			w.actor("Buyer", farmer),
		}
		u.Inputs = []models.ScenarioInput{{StepID: ask, Field: "Enter the product", Value: "lime"}}
	})
	if err != nil {
		t.Fatalf("UpdateScenario: %v", err)
	}

	got := w.get(t, s.ID)
	if !reflect.DeepEqual(out.Actors, got.Actors) || !reflect.DeepEqual(out.Inputs, got.Inputs) {
		t.Errorf("the answer differs from a fresh read: %+v vs %+v", out, got)
	}
	stored := storedActorsOf(got)
	if len(stored) != 3 {
		t.Fatalf("stored actors = %+v, want the renamed, the changed and the new one", stored)
	}
	renamed := actorNamed(t, got, "Mama Akinyi")
	if renamed.ID != farmerActor.ID || renamed.RoleID != w.roles[farmer].ID || renamed.Ordinality != 1 {
		t.Errorf("renamed = %+v, want the same row, the same role, first", renamed)
	}
	changed := actorNamed(t, got, "Shop")
	if changed.ID != shopActor.ID || changed.RoleID != w.roles[coordinator].ID || changed.RoleName != coordinator || changed.Ordinality != 2 {
		t.Errorf("role change = %+v, want the same row now holding %s", changed, coordinator)
	}
	added := actorNamed(t, got, "Buyer")
	if added.ID == "" || added.RoleID != w.roles[farmer].ID || added.Ordinality != 3 {
		t.Errorf("added = %+v, want a new row numbered 3", added)
	}
	for _, a := range stored {
		if a.Name == "Helper" || a.Name == "Farmer 1" {
			t.Errorf("actor %q is still there, want it renamed away or removed", a.Name)
		}
	}
	if len(got.Inputs) != 1 || got.Inputs[0].Value != "lime" || got.Inputs[0].ID != before.Inputs[0].ID {
		t.Errorf("inputs = %+v, want the one fixed answer changed to lime in place", got.Inputs)
	}
	if got.Blocks[0].Actor != "Mama Akinyi" {
		t.Errorf("first block actor = %q, want the new name", got.Blocks[0].Actor)
	}
	if got.Result != models.ScenarioNotRun || got.LastRunID != before.LastRunID {
		t.Errorf("result %q, last run %q, want not_run and the old last run", got.Result, got.LastRunID)
	}
	runsAfter, err := w.mgr.ListTestRuns(ctx, s.ID)
	if err != nil || !reflect.DeepEqual(runsBefore, runsAfter) {
		t.Errorf("runs after = %+v %v, want them untouched", runsAfter, err)
	}
}

func TestUpdateScenario_InputsLeftOutAreFreedAndAnEmptyListFreesAll(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow, w.actor("Farmer 1", farmer))
	ask := s.Blocks[0].StepID
	for field, value := range map[string]string{"Enter the product": "urea", "Enter the number of bags": "5"} {
		if _, err := w.mgr.SetScenarioInput(ctx, models.ScenarioInput{ScenarioID: s.ID, StepID: ask, Field: field, Value: value}); err != nil {
			t.Fatalf("SetScenarioInput: %v", err)
		}
	}

	one, err := w.setOf(t, s.ID, func(u *models.Scenario) {
		u.Inputs = []models.ScenarioInput{{StepID: ask, Field: "Enter the number of bags", Value: "7"}}
	})
	if err != nil || len(one.Inputs) != 1 || one.Inputs[0].Field != "Enter the number of bags" || one.Inputs[0].Value != "7" {
		t.Fatalf("one input: %v %+v, want only the bags answer, changed", err, one.Inputs)
	}
	if len(storedActorsOf(one)) != 1 {
		t.Errorf("actors = %+v, want them kept: actors was omitted", one.Actors)
	}

	none, err := w.setOf(t, s.ID, func(u *models.Scenario) { u.Inputs = []models.ScenarioInput{} })
	if err != nil || len(none.Inputs) != 0 {
		t.Errorf("an empty list: %v %+v, want every field Free", err, none.Inputs)
	}
	if !none.Blocks[0].Fields[1].Free {
		t.Errorf("block field = %+v, want it Free", none.Blocks[0].Fields[1])
	}
}

func TestUpdateScenario_ARefusedSetChangesNothing(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow, w.actor("Farmer 1", farmer), w.actor("Shop", supplier))
	ask := s.Blocks[0].StepID
	if _, err := w.mgr.SetScenarioInput(ctx, models.ScenarioInput{ScenarioID: s.ID, StepID: ask, Field: "Enter the product", Value: "urea"}); err != nil {
		t.Fatalf("SetScenarioInput: %v", err)
	}
	w.pass(t, s.ID)
	before := w.get(t, s.ID)
	farmerActor := actorNamed(t, before, "Farmer 1")
	other := w.stepsOf(t, w.draftID, w.groupOrd)[0]

	refusals := map[string]struct {
		want  error
		patch func(u *models.Scenario)
	}{
		"two actors with one name": {mwanachamaagency.ErrDuplicateScenarioActor, func(u *models.Scenario) {
			u.Name = "Renamed"
			u.Actors = []models.ScenarioActor{{ID: farmerActor.ID, Name: "Pat"}, w.actor(" pat ", supplier)}
		}},
		"a name an agent has": {mwanachamaagency.ErrDuplicateScenarioActor, func(u *models.Scenario) {
			u.Actors = []models.ScenarioActor{{ID: farmerActor.ID, Name: "farmers' agent"}}
		}},
		"the System's name": {mwanachamaagency.ErrDuplicateScenarioActor, func(u *models.Scenario) {
			u.Actors = []models.ScenarioActor{w.actor("system", farmer)}
		}},
		"a nameless actor": {mwanachamaagency.ErrInvalidScenario, func(u *models.Scenario) {
			u.Actors = []models.ScenarioActor{{ID: farmerActor.ID, Name: "  "}}
		}},
		"a new actor with no role": {mwanachamaagency.ErrInvalidScenario, func(u *models.Scenario) {
			u.Actors = []models.ScenarioActor{{Name: "Nobody"}}
		}},
		"a role that is not in the draft": {mwanachamaagency.ErrRoleNotFound, func(u *models.Scenario) {
			u.Actors = []models.ScenarioActor{{ID: farmerActor.ID, Name: "Farmer 1", RoleID: "no-such-role"}}
		}},
		"an actor id that is not this scenario's": {mwanachamaagency.ErrScenarioActorNotFound, func(u *models.Scenario) {
			u.Actors = []models.ScenarioActor{{ID: "no-such-actor", Name: "X"}}
		}},
		"one actor named twice by id": {mwanachamaagency.ErrInvalidScenario, func(u *models.Scenario) {
			u.Actors = []models.ScenarioActor{{ID: farmerActor.ID, Name: "A"}, {ID: farmerActor.ID, Name: "B"}}
		}},
		"a good actor change with a bad input": {mwanachamaagency.ErrInvalidScenario, func(u *models.Scenario) {
			u.Actors = []models.ScenarioActor{{ID: farmerActor.ID, Name: "Mama Akinyi"}}
			u.Inputs = []models.ScenarioInput{{StepID: ask, Field: "No such question", Value: "x"}}
		}},
		"an input on a step of another workflow": {mwanachamaagency.ErrInvalidScenario, func(u *models.Scenario) {
			u.Inputs = []models.ScenarioInput{{StepID: other.ID, Field: "Select the target group", Value: "x"}}
		}},
		"one field fixed twice": {mwanachamaagency.ErrInvalidScenario, func(u *models.Scenario) {
			u.Inputs = []models.ScenarioInput{
				{StepID: ask, Field: "Enter the product", Value: "a"}, {StepID: ask, Field: "Enter the product", Value: "b"},
			}
		}},
	}
	for name, c := range refusals {
		t.Run(name, func(t *testing.T) {
			if _, err := w.setOf(t, s.ID, c.patch); !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
			if after := w.get(t, s.ID); !reflect.DeepEqual(after, before) {
				t.Errorf("the scenario changed although the update was refused:\n before %+v\n after  %+v", before, after)
			}
		})
	}
}

func TestUpdateScenario_TwoActorsCanSwapNamesInOneCall(t *testing.T) {
	w := newScenarioWorld(t)
	s := w.scenario(t, "Shop says yes", w.buyNow, w.actor("Ann", farmer), w.actor("Bob", supplier))
	ann, bob := actorNamed(t, s, "Ann"), actorNamed(t, s, "Bob")

	out, err := w.setOf(t, s.ID, func(u *models.Scenario) {
		u.Actors = []models.ScenarioActor{{ID: ann.ID, Name: "Bob"}, {ID: bob.ID, Name: "Ann"}}
	})
	if err != nil {
		t.Fatalf("a swap: %v", err)
	}
	if a, b := actorNamed(t, out, "Bob"), actorNamed(t, out, "Ann"); a.ID != ann.ID || b.ID != bob.ID {
		t.Errorf("after the swap %+v and %+v, want each row to keep its id and take the other's name", a, b)
	}

	byName, err := w.setOf(t, s.ID, func(u *models.Scenario) {
		u.Actors = []models.ScenarioActor{
			{Name: "Ann", RoleID: w.roles[supplier].ID}, {Name: "Bob", RoleID: w.roles[farmer].ID},
		}
	})
	if err != nil {
		t.Fatalf("a set named without ids: %v", err)
	}
	if a := actorNamed(t, byName, "Ann"); a.ID != bob.ID || a.RoleID != w.roles[supplier].ID {
		t.Errorf("Ann = %+v, want the existing Ann row (id %s) with the role changed", a, bob.ID)
	}
	if n := len(storedActorsOf(byName)); n != 2 {
		t.Errorf("stored actors = %d, want 2: a name that exists is the same actor", n)
	}
}

func TestUpdateScenario_OmittingActorsAndInputsLeavesThemAsTheyWere(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow, w.actor("Farmer 1", farmer))
	if _, err := w.mgr.SetScenarioInput(ctx, models.ScenarioInput{ScenarioID: s.ID, StepID: s.Blocks[0].StepID, Field: "Enter the product", Value: "urea"}); err != nil {
		t.Fatalf("SetScenarioInput: %v", err)
	}
	before := w.get(t, s.ID)

	out, err := w.setOf(t, s.ID, func(u *models.Scenario) { u.Name, u.Summary = "Shop agrees", "at once" })
	if err != nil {
		t.Fatalf("UpdateScenario: %v", err)
	}
	if !reflect.DeepEqual(out.Actors, before.Actors) || !reflect.DeepEqual(out.Inputs, before.Inputs) {
		t.Errorf("actors %+v inputs %+v, want both as they were", out.Actors, out.Inputs)
	}

	echoed, err := w.mgr.UpdateScenario(ctx, w.get(t, s.ID))
	if err != nil {
		t.Fatalf("UpdateScenario with the read-back scenario: %v", err)
	}
	if !reflect.DeepEqual(echoed.Actors, before.Actors) || !reflect.DeepEqual(echoed.Inputs, before.Inputs) {
		t.Errorf("echoing a read back changed the set: %+v %+v", echoed.Actors, echoed.Inputs)
	}
}
