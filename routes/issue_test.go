package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/aosanya/mwanachama-backend-agency/routes"
)

// TestListIssues_ScopedToDraft covers GET {draftID}/issues — every scan
// names a real Draft (AGD-006), there is no top-level "live" route.
func TestListIssues_ScopedToDraft(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	ctx := t.Context()
	g, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if _, err := mgr.CreateObjective(ctx, models.Objective{DraftID: draftID, GoalID: g.ID, Title: "Grow adoption"}); err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/drafts/"+draftID+"/issues", nil)
	req.SetPathValue("draftID", draftID)
	rec := httptest.NewRecorder()
	routes.ListIssues(mgr)(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out []models.Issue
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != 1 || out[0].EntityType != "objective" || out[0].DraftID != draftID {
		t.Fatalf("expected one draft-scoped objective issue, got %+v", out)
	}
}

func TestListIssues_ObjectiveWithNoAccountable(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	ctx := t.Context()
	g, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if _, err := mgr.CreateObjective(ctx, models.Objective{DraftID: draftID, GoalID: g.ID, Title: "Grow adoption"}); err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/drafts/"+draftID+"/issues", nil)
	req.SetPathValue("draftID", draftID)
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
	mgr, draftID := newTestManagerWithDraft(t)
	ctx := t.Context()
	if _, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, Title: "Lonely goal"}); err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/drafts/"+draftID+"/issues?severity=error", nil)
	req.SetPathValue("draftID", draftID)
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
