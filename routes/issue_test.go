package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/aosanya/mwanachama-backend-agency/routes"
)

// TestListIssues_LiveScope covers the top-level GET /issues route, where
// {draftID} is absent from the pattern entirely and so reads back "".
func TestListIssues_LiveScope(t *testing.T) {
	mgr := newTestManager(t)
	ctx := t.Context()
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "SWELab"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	g, err := mgr.CreateGoal(ctx, models.Goal{Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if _, err := mgr.CreateObjective(ctx, models.Objective{GoalID: g.ID, Title: "Grow adoption"}); err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/issues", nil)
	rec := httptest.NewRecorder()
	routes.ListIssues(mgr)(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out []models.Issue
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != 1 || out[0].EntityType != "objective" || out[0].DraftID != "" {
		t.Fatalf("expected one live-scoped objective issue, got %+v", out)
	}
}

func TestListIssues_ObjectiveWithNoAccountable(t *testing.T) {
	mgr := newTestManager(t)
	ctx := t.Context()
	ag, err := mgr.CreateAgency(ctx, models.Agency{Name: "SWELab"})
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
		if i.EntityType == "objective" && i.Severity == models.IssueSeverityError {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an error for the unassigned objective, got %+v", out)
	}
}

func TestListIssues_FilterBySeverityQueryParam(t *testing.T) {
	mgr := newTestManager(t)
	ctx := t.Context()
	ag, _ := mgr.CreateAgency(ctx, models.Agency{Name: "SWELab"})
	if _, err := mgr.CreateGoal(ctx, models.Goal{Title: "Lonely goal"}); err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	d, err := mgr.CreateDraft(ctx, "review pass", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/drafts/"+d.ID+"/issues?severity=error", nil)
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
	if len(out) != 1 || out[0].Severity != models.IssueSeverityError {
		t.Fatalf("expected exactly 1 error issue (the goal with no objectives), got %+v", out)
	}
}
