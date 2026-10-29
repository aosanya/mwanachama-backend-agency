package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

type scenarioRoutes struct {
	t       *testing.T
	call    func(method, path, body string) *httptest.ResponseRecorder
	draftID string
	flow    models.Workflow
	askID   string
	farmer  models.Role
}

func newScenarioRoutes(t *testing.T) scenarioRoutes {
	t.Helper()
	mgr, draftID := newTestManagerWithDraft(t)
	ctx := t.Context()
	g, _ := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, Title: "Sell"})
	o, _ := mgr.CreateObjective(ctx, models.Objective{DraftID: draftID, GoalID: g.ID, Title: "Sell"})
	flow, _ := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftID, Name: "Buy now"})
	farmer, _ := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "Farmer"})
	ask, err := mgr.CreateWorkItem(ctx, models.WorkItem{
		DraftID: draftID, WorkflowID: flow.ID, ObjectiveID: o.ID, Title: "Ask", Kind: models.StepKindFillForm,
		RoleID: farmer.ID, FormName: "Buy now form",
		Questions: []models.StepQuestion{{Prompt: "Enter the product", AnswerType: models.AnswerFreeText, Required: true}},
	})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	call := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		serve(mgr, rec, httptest.NewRequest(method, path, strings.NewReader(body)))
		return rec
	}
	return scenarioRoutes{t: t, call: call, draftID: draftID, flow: flow, askID: ask.ID, farmer: farmer}
}

func (s scenarioRoutes) expect(rec *httptest.ResponseRecorder, status int, into any) {
	s.t.Helper()
	if rec.Code != status {
		s.t.Fatalf("status = %d, want %d, body = %s", rec.Code, status, rec.Body.String())
	}
	if into != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), into); err != nil {
			s.t.Fatalf("unmarshal %s: %v", rec.Body.String(), err)
		}
	}
}

func TestScenarioRoutes_WriteAScenarioAndPlayIt(t *testing.T) {
	r := newScenarioRoutes(t)

	var sc models.Scenario
	r.expect(r.call(http.MethodPost, "/scenarios", `{"draft_id":"`+r.draftID+`","name":"Shop says yes","summary":"tries it","workflow_id":"`+r.flow.ID+`"}`), http.StatusCreated, &sc)
	if sc.ID == "" || sc.Result != models.ScenarioNotRun || len(sc.Blocks) != 1 || sc.Blocks[0].Title != "Ask" {
		t.Fatalf("created = %+v", sc)
	}

	var actor models.ScenarioActor
	r.expect(r.call(http.MethodPost, "/scenarios/"+sc.ID+"/actors", `{"name":"Farmer 1","role_id":"`+r.farmer.ID+`"}`), http.StatusCreated, &actor)
	if actor.ScenarioID != sc.ID || actor.RoleName != "Farmer" {
		t.Errorf("actor = %+v, want it in the scenario of the address", actor)
	}
	r.expect(r.call(http.MethodPost, "/scenarios/"+sc.ID+"/actors", `{"name":"farmer 1","role_id":"`+r.farmer.ID+`"}`), http.StatusConflict, nil)
	r.expect(r.call(http.MethodPatch, "/scenarios/"+sc.ID+"/actors/"+actor.ID, `{"name":"Mama Akinyi"}`), http.StatusOK, &actor)
	if actor.Name != "Mama Akinyi" {
		t.Errorf("renamed = %+v", actor)
	}

	var in models.ScenarioInput
	r.expect(r.call(http.MethodPut, "/scenarios/"+sc.ID+"/inputs", `{"step_id":"`+r.askID+`","field":"Enter the product","value":"urea"}`), http.StatusOK, &in)
	if in.ScenarioID != sc.ID || in.Value != "urea" {
		t.Errorf("input = %+v", in)
	}
	r.expect(r.call(http.MethodPut, "/scenarios/"+sc.ID+"/inputs", `{"step_id":"`+r.askID+`","field":"Nothing asked","value":"x"}`), http.StatusBadRequest, nil)

	var read models.Scenario
	r.expect(r.call(http.MethodGet, "/scenarios/"+sc.ID, ""), http.StatusOK, &read)
	if read.Blocks[0].Actor != "Mama Akinyi" || read.Blocks[0].Fields[0].Value != "urea" {
		t.Errorf("read = %+v", read.Blocks)
	}

	var listed []models.Scenario
	r.expect(r.call(http.MethodGet, "/scenarios?draft_id="+r.draftID+"&result=not_run", ""), http.StatusOK, &listed)
	if len(listed) != 1 {
		t.Errorf("listed = %d scenarios, want 1", len(listed))
	}
	r.expect(r.call(http.MethodGet, "/scenarios?draft_id="+r.draftID+"&result=passed", ""), http.StatusOK, &listed)
	if len(listed) != 0 {
		t.Errorf("listed passed = %d, want none", len(listed))
	}

	var renamed models.Scenario
	r.expect(r.call(http.MethodPatch, "/scenarios/"+sc.ID, `{"name":"Shop agrees","summary":"s","workflow_id":"`+r.flow.ID+`"}`), http.StatusOK, &renamed)
	if renamed.Name != "Shop agrees" || renamed.ID != sc.ID {
		t.Errorf("updated = %+v", renamed)
	}

	var run models.TestRun
	r.expect(r.call(http.MethodPost, "/scenarios/"+sc.ID+"/runs", ""), http.StatusCreated, &run)
	if run.Result != models.TestRunStopped || run.ScenarioID != sc.ID {
		t.Errorf("run = %+v, want a stopped run: no executor is installed", run)
	}
	var got models.TestRun
	r.expect(r.call(http.MethodGet, "/test-runs/"+run.ID, ""), http.StatusOK, &got)
	if got.ID != run.ID {
		t.Errorf("read run = %+v", got)
	}
	var runs []models.TestRun
	r.expect(r.call(http.MethodGet, "/scenarios/"+sc.ID+"/runs", ""), http.StatusOK, &runs)
	if len(runs) != 1 {
		t.Errorf("runs = %d, want 1", len(runs))
	}

	var batch []models.TestRun
	r.expect(r.call(http.MethodPost, "/scenario-runs", `{"draft_id":"`+r.draftID+`"}`), http.StatusCreated, &batch)
	if len(batch) != 1 {
		t.Errorf("batch = %d runs, want the draft's one scenario", len(batch))
	}
	r.expect(r.call(http.MethodPost, "/scenario-runs", `{"draft_id":"`+r.draftID+`","scenario_ids":["no-such-scenario"]}`), http.StatusNotFound, nil)

	r.expect(r.call(http.MethodDelete, "/scenarios/"+sc.ID+"/actors/"+actor.ID, ""), http.StatusNoContent, nil)
	r.expect(r.call(http.MethodDelete, "/scenarios/"+sc.ID+"/actors/"+actor.ID, ""), http.StatusNotFound, nil)
	r.expect(r.call(http.MethodDelete, "/scenarios/"+sc.ID, ""), http.StatusNoContent, nil)
	r.expect(r.call(http.MethodGet, "/scenarios/"+sc.ID, ""), http.StatusNotFound, nil)
	r.expect(r.call(http.MethodGet, "/test-runs/"+run.ID, ""), http.StatusNotFound, nil)
}

func TestScenarioRoutes_Refusals(t *testing.T) {
	r := newScenarioRoutes(t)
	r.expect(r.call(http.MethodPost, "/scenarios", `{"draft_id":"`+r.draftID+`","name":"","workflow_id":"`+r.flow.ID+`"}`), http.StatusBadRequest, nil)
	r.expect(r.call(http.MethodPost, "/scenarios", `{"draft_id":"`+r.draftID+`","name":"n","workflow_id":"no-such-workflow"}`), http.StatusNotFound, nil)
	r.expect(r.call(http.MethodPost, "/scenarios", `{"draft_id":"`+r.draftID+`","name":"n","workflow_id":"`+r.flow.ID+`","made_up":1}`), http.StatusBadRequest, nil)
}

func TestScenarioRoutes_PublishAnswers409NamingTheScenario(t *testing.T) {
	r := newScenarioRoutes(t)
	var sc models.Scenario
	r.expect(r.call(http.MethodPost, "/scenarios", `{"draft_id":"`+r.draftID+`","name":"Shop says yes","workflow_id":"`+r.flow.ID+`"}`), http.StatusCreated, &sc)
	rec := r.call(http.MethodPost, "/publications", `{"draft_id":"`+r.draftID+`"}`)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "Shop says yes") {
		t.Errorf("publish = %d %s, want 409 naming the scenario", rec.Code, rec.Body.String())
	}
}
