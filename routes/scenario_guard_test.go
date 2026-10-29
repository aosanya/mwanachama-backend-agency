package routes_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestScenarioRoutes_DeletingWhatAScenarioUsesIsRefusedWith409UntilTheScenarioGoes(t *testing.T) {
	r := newScenarioRoutes(t)
	var sc models.Scenario
	r.expect(r.call(http.MethodPost, "/scenarios", `{"draft_id":"`+r.draftID+`","name":"Shop says yes","workflow_id":"`+r.flow.ID+`","actors":[{"name":"Farmer 1","role_id":"`+r.farmer.ID+`"}]}`), http.StatusCreated, &sc)

	for _, path := range []string{"/workflows/" + r.flow.ID, "/roles/" + r.farmer.ID} {
		rec := r.call(http.MethodDelete, path, "")
		r.expect(rec, http.StatusConflict, nil)
		if !strings.Contains(rec.Body.String(), "Shop says yes") {
			t.Errorf("DELETE %s body = %s, want the scenario named", path, rec.Body.String())
		}
	}

	r.expect(r.call(http.MethodDelete, "/scenarios/"+sc.ID, ""), http.StatusNoContent, nil)
	r.expect(r.call(http.MethodDelete, "/roles/"+r.farmer.ID, ""), http.StatusNoContent, nil)
	r.expect(r.call(http.MethodDelete, "/workflows/"+r.flow.ID, ""), http.StatusNoContent, nil)
}
