package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestMeasureRoutes_SetThenListByObjective(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	ctx := t.Context()
	g, _ := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, Title: "Sell"})
	o, _ := mgr.CreateObjective(ctx, models.Objective{DraftID: draftID, GoalID: g.ID, Title: "Sell"})
	other, _ := mgr.CreateObjective(ctx, models.Objective{DraftID: draftID, GoalID: g.ID, Title: "Buy"})
	call := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		serve(mgr, rec, httptest.NewRequest(method, path, strings.NewReader(body)))
		return rec
	}

	rec := call(http.MethodPut, "/measures", `{"objective_id":"`+o.ID+`","name":"Shops joined","target":"12","unit":"shops"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /measures = %d %s", rec.Code, rec.Body.String())
	}
	var set models.Measure
	if err := json.Unmarshal(rec.Body.Bytes(), &set); err != nil || set.ID == "" || set.ObjectiveID != o.ID || set.Unit != "shops" {
		t.Fatalf("set = %+v, %v", set, err)
	}
	call(http.MethodPut, "/measures", `{"objective_id":"`+other.ID+`","name":"Buyers joined","target":"5"}`)
	call(http.MethodPut, "/measures", `{"objective_id":"`+o.ID+`","name":"Shops joined","target":"15"}`)

	rec = call(http.MethodGet, "/measures?draft_id="+draftID+"&objective_id="+o.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /measures = %d %s", rec.Code, rec.Body.String())
	}
	var listed []models.Measure
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("unmarshal %s: %v", rec.Body.String(), err)
	}
	if len(listed) != 1 || listed[0].Target != "15" || listed[0].ObjectiveID != o.ID {
		t.Fatalf("listed = %+v, want the one measure with the second target", listed)
	}

	rec = call(http.MethodGet, "/measures?draft_id="+draftID, "")
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil || len(listed) != 2 {
		t.Fatalf("whole draft = %s, want both objectives' measures", rec.Body.String())
	}
}

func TestMeasureRoutes_Refusals(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	ctx := t.Context()
	g, _ := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, Title: "Sell"})
	o, _ := mgr.CreateObjective(ctx, models.Objective{DraftID: draftID, GoalID: g.ID, Title: "Sell"})

	cases := []struct {
		name   string
		body   string
		status int
	}{
		{"unknown objective", `{"objective_id":"nope","name":"Shops","target":"1"}`, http.StatusNotFound},
		{"no name", `{"objective_id":"` + o.ID + `","target":"1"}`, http.StatusBadRequest},
		{"no target", `{"objective_id":"` + o.ID + `","name":"Shops"}`, http.StatusBadRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			serve(mgr, rec, httptest.NewRequest(http.MethodPut, "/measures", strings.NewReader(c.body)))
			if rec.Code != c.status {
				t.Fatalf("PUT /measures = %d %s, want %d", rec.Code, rec.Body.String(), c.status)
			}
		})
	}
}
