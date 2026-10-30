package routes_test

import (
	"net/http"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestScenarioRoutes_AThenLineIsWrittenWithTheScenarioAndReadBackOnItsBlock(t *testing.T) {
	r := newScenarioRoutes(t)

	var sc models.Scenario
	r.expect(r.call(http.MethodPost, "/scenarios", `{"draft_id":"`+r.draftID+`","name":"Shop says yes","workflow_id":"`+r.flow.ID+`","thens":[{"step_id":"`+r.askID+`","then":"The order shows urea"}]}`), http.StatusCreated, &sc)
	if len(sc.Blocks) != 1 || sc.Blocks[0].Then != "The order shows urea" {
		t.Fatalf("created blocks = %+v, want the Then line on the block", sc.Blocks)
	}

	var changed models.Scenario
	r.expect(r.call(http.MethodPatch, "/scenarios/"+sc.ID, `{"name":"Shop says yes","thens":[{"step_id":"`+r.askID+`","then":"The order shows maize"}]}`), http.StatusOK, &changed)
	if changed.Blocks[0].Then != "The order shows maize" {
		t.Errorf("updated Then = %q, want the changed line", changed.Blocks[0].Then)
	}

	var read models.Scenario
	r.expect(r.call(http.MethodGet, "/scenarios/"+sc.ID, ""), http.StatusOK, &read)
	if read.Blocks[0].Then != "The order shows maize" {
		t.Errorf("read Then = %q, want it returned when read", read.Blocks[0].Then)
	}

	r.expect(r.call(http.MethodPatch, "/scenarios/"+sc.ID, `{"name":"Shop says yes","thens":[{"step_id":"nope","then":"x"}]}`), http.StatusBadRequest, nil)
}

func TestScenarioRoutes_APatchSetsTheActorsAndInputsAsOneSet(t *testing.T) {
	r := newScenarioRoutes(t)

	var sc models.Scenario
	r.expect(r.call(http.MethodPost, "/scenarios", `{"draft_id":"`+r.draftID+`","name":"Shop says yes","workflow_id":"`+r.flow.ID+`","actors":[{"name":"Farmer 1","role_id":"`+r.farmer.ID+`"}]}`), http.StatusCreated, &sc)
	first := sc.Actors[0].ID

	var set models.Scenario
	body := `{"name":"Shop says yes","actors":[{"id":"` + first + `","name":"Mama Akinyi"},{"name":"Buyer","role_id":"` + r.farmer.ID + `"}],"inputs":[{"step_id":"` + r.askID + `","field":"Enter the product","value":"urea"}]}`
	r.expect(r.call(http.MethodPatch, "/scenarios/"+sc.ID, body), http.StatusOK, &set)
	if len(set.Inputs) != 1 || set.Inputs[0].Value != "urea" || set.Blocks[0].Actor != "Mama Akinyi" {
		t.Fatalf("set = %+v, want the renamed actor and the fixed answer", set)
	}

	clash := `{"name":"Shop says yes","actors":[{"id":"` + first + `","name":"Buyer"},{"name":" buyer ","role_id":"` + r.farmer.ID + `"}]}`
	r.expect(r.call(http.MethodPatch, "/scenarios/"+sc.ID, clash), http.StatusConflict, nil)

	var read models.Scenario
	r.expect(r.call(http.MethodGet, "/scenarios/"+sc.ID, ""), http.StatusOK, &read)
	if read.Blocks[0].Actor != "Mama Akinyi" || len(read.Inputs) != 1 {
		t.Errorf("after the refused patch: %+v, want the earlier set kept", read)
	}
}
