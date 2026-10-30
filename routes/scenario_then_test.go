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
