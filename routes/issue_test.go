package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/aosanya/mwanachama-backend-agency/routes"
)

func TestListIssues_EmptyDraftID(t *testing.T) {
	mgr := newTestManager(t)
	req := httptest.NewRequest(http.MethodGet, "/drafts//issues", nil)
	req.SetPathValue("draftID", "")
	rec := httptest.NewRecorder()
	routes.ListIssues(mgr)(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestListIssues_ObjectiveWithNoAccountable(t *testing.T) {
	mgr := newTestManager(t)
	ctx := t.Context()
	ag, err := mgr.CreateAgency(ctx, models.Agency{Name: "NeoLabHQ"})
	if err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	g, err := mgr.CreateGoal(ctx, models.Goal{Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if _, err := mgr.CreateObjective(ctx, models.Objective{GoalID: g.ID, Title: "Grow adoption"}); err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	d, err := mgr.CreateDraft(ctx, "review pass", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/drafts/"+d.ID+"/issues", nil)
	req.SetPathValue("draftID", d.ID)
	rec := httptest.NewRecorder()
	routes.ListIssues(mgr)(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out []models.Issue
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	found := false
	for _, i := range out {
		if i.EntityType == "objective" && i.Severity == models.IssueSeverityWarning {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a warning for the unassigned objective, got %+v", out)
	}
}

func TestListIssues_FilterBySeverityQueryParam(t *testing.T) {
	mgr := newTestManager(t)
	ctx := t.Context()
	ag, _ := mgr.CreateAgency(ctx, models.Agency{Name: "NeoLabHQ"})
	if _, err := mgr.CreateGoal(ctx, models.Goal{Title: "Lonely goal"}); err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	d, err := mgr.CreateDraft(ctx, "review pass", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/drafts/"+d.ID+"/issues?severity=info", nil)
	req.SetPathValue("draftID", d.ID)
	rec := httptest.NewRecorder()
	routes.ListIssues(mgr)(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out []models.Issue
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != 1 || out[0].Severity != models.IssueSeverityInfo {
		t.Fatalf("expected exactly 1 info issue (the goal with no objectives), got %+v", out)
	}
}
