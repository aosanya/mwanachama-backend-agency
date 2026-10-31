package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type workflowObjectiveRoutesSeed struct {
	mgr        mwanachamaagency.AgencyManager
	draftID    string
	workflowA  models.Workflow
	workflowB  models.Workflow
	objectiveA models.Objective
	objectiveB models.Objective
}

func seedWorkflowObjectiveRoutes(t *testing.T) workflowObjectiveRoutesSeed {
	t.Helper()
	mgr, draftID := newTestManagerWithDraft(t)
	ctx := t.Context()
	g, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	s := workflowObjectiveRoutesSeed{mgr: mgr, draftID: draftID}
	if s.objectiveA, err = mgr.CreateObjective(ctx, models.Objective{DraftID: draftID, GoalID: g.ID, Title: "Objective A"}); err != nil {
		t.Fatalf("CreateObjective(A): %v", err)
	}
	if s.objectiveB, err = mgr.CreateObjective(ctx, models.Objective{DraftID: draftID, GoalID: g.ID, Title: "Objective B"}); err != nil {
		t.Fatalf("CreateObjective(B): %v", err)
	}
	if s.workflowA, err = mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftID, Name: "Onboarding"}); err != nil {
		t.Fatalf("CreateWorkflow(A): %v", err)
	}
	if s.workflowB, err = mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftID, Name: "Offboarding"}); err != nil {
		t.Fatalf("CreateWorkflow(B): %v", err)
	}
	return s
}

func (s workflowObjectiveRoutesSeed) call(method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	serve(s.mgr, rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func (s workflowObjectiveRoutesSeed) linkBody(workflowID, objectiveID string) string {
	return `{"workflow_id":"` + workflowID + `","objective_id":"` + objectiveID + `","draft_id":"` + s.draftID + `"}`
}

func (s workflowObjectiveRoutesSeed) link(t *testing.T, workflowID, objectiveID string) models.WorkflowObjective {
	t.Helper()
	rec := s.call(http.MethodPost, "/workflow-objectives", s.linkBody(workflowID, objectiveID))
	if rec.Code != http.StatusCreated {
		t.Fatalf("precondition: POST /workflow-objectives -> %d %s", rec.Code, rec.Body.String())
	}
	var out models.WorkflowObjective
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}

func (s workflowObjectiveRoutesSeed) listed(t *testing.T, query string) []models.WorkflowObjective {
	t.Helper()
	rec := s.call(http.MethodGet, "/workflow-objectives?draft_id="+s.draftID+query, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list -> %d %s", rec.Code, rec.Body.String())
	}
	var out []models.WorkflowObjective
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}

func TestCreateWorkflowObjective(t *testing.T) {
	s := seedWorkflowObjectiveRoutes(t)

	rec := s.call(http.MethodPost, "/workflow-objectives", s.linkBody(s.workflowA.ID, s.objectiveA.ID))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.WorkflowObjective
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID == "" || out.WorkflowID != s.workflowA.ID || out.ObjectiveID != s.objectiveA.ID || out.DraftID != s.draftID {
		t.Fatalf("unexpected link: %+v", out)
	}
}

func TestCreateWorkflowObjective_DuplicateIsConflict(t *testing.T) {
	s := seedWorkflowObjectiveRoutes(t)
	s.link(t, s.workflowA.ID, s.objectiveA.ID)

	rec := s.call(http.MethodPost, "/workflow-objectives", s.linkBody(s.workflowA.ID, s.objectiveA.ID))

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := s.listed(t, ""); len(got) != 1 {
		t.Fatalf("the refused duplicate left %d links, want 1", len(got))
	}
	if rec := s.call(http.MethodPost, "/workflow-objectives", s.linkBody(s.workflowB.ID, s.objectiveA.ID)); rec.Code != http.StatusCreated {
		t.Fatalf("another workflow for the same objective: status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestCreateWorkflowObjective_UnknownEndsAreNotFound(t *testing.T) {
	s := seedWorkflowObjectiveRoutes(t)

	if rec := s.call(http.MethodPost, "/workflow-objectives", s.linkBody("unknown", s.objectiveA.ID)); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown workflow: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec := s.call(http.MethodPost, "/workflow-objectives", s.linkBody(s.workflowA.ID, "unknown")); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown objective: status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestCreateWorkflowObjective_AnEndFromAnotherDraftIsNotFound(t *testing.T) {
	s := seedWorkflowObjectiveRoutes(t)
	other, err := s.mgr.CreateDraft(t.Context(), "other", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	g, _ := s.mgr.CreateGoal(t.Context(), models.Goal{DraftID: other.ID, Title: "Elsewhere"})
	foreignObjective, err := s.mgr.CreateObjective(t.Context(), models.Objective{DraftID: other.ID, GoalID: g.ID, Title: "Elsewhere"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	foreignWorkflow, err := s.mgr.CreateWorkflow(t.Context(), models.Workflow{DraftID: other.ID, Name: "Elsewhere"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	if rec := s.call(http.MethodPost, "/workflow-objectives", s.linkBody(s.workflowA.ID, foreignObjective.ID)); rec.Code != http.StatusNotFound {
		t.Fatalf("objective from another draft: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec := s.call(http.MethodPost, "/workflow-objectives", s.linkBody(foreignWorkflow.ID, s.objectiveA.ID)); rec.Code != http.StatusNotFound {
		t.Fatalf("workflow from another draft: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := s.listed(t, ""); len(got) != 0 {
		t.Fatalf("a refused create stored %d links", len(got))
	}
}

func TestGetListAndDeleteWorkflowObjectives(t *testing.T) {
	s := seedWorkflowObjectiveRoutes(t)
	first := s.link(t, s.workflowA.ID, s.objectiveA.ID)
	s.link(t, s.workflowA.ID, s.objectiveB.ID)
	s.link(t, s.workflowB.ID, s.objectiveA.ID)

	getRec := s.call(http.MethodGet, "/workflow-objectives/"+first.ID, "")
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", getRec.Code, getRec.Body.String())
	}
	var got models.WorkflowObjective
	if err := json.Unmarshal(getRec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != first.ID || got.WorkflowID != s.workflowA.ID || got.ObjectiveID != s.objectiveA.ID {
		t.Fatalf("unexpected link: %+v", got)
	}

	if all := s.listed(t, ""); len(all) != 3 {
		t.Fatalf("list returned %d links, want 3", len(all))
	}
	if byWorkflow := s.listed(t, "&workflow_id="+s.workflowA.ID); len(byWorkflow) != 2 {
		t.Fatalf("list by workflow returned %d links, want 2", len(byWorkflow))
	}
	if byObjective := s.listed(t, "&objective_id="+s.objectiveA.ID); len(byObjective) != 2 {
		t.Fatalf("list by objective returned %d links, want 2", len(byObjective))
	}

	if rec := s.call(http.MethodDelete, "/workflow-objectives/"+first.ID, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec := s.call(http.MethodGet, "/workflow-objectives/"+first.ID, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("get after delete status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec := s.call(http.MethodGet, "/workflows/"+s.workflowA.ID, ""); rec.Code != http.StatusOK {
		t.Fatalf("the workflow must survive being unlinked: status = %d", rec.Code)
	}
}

func TestWorkflowObjective_UnknownIdIsNotFound(t *testing.T) {
	s := seedWorkflowObjectiveRoutes(t)

	if rec := s.call(http.MethodGet, "/workflow-objectives/unknown", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("get status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec := s.call(http.MethodDelete, "/workflow-objectives/unknown", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteWorkflowTakesItsObjectiveLinks(t *testing.T) {
	s := seedWorkflowObjectiveRoutes(t)
	s.link(t, s.workflowA.ID, s.objectiveA.ID)
	s.link(t, s.workflowA.ID, s.objectiveB.ID)
	kept := s.link(t, s.workflowB.ID, s.objectiveA.ID)

	if rec := s.call(http.MethodDelete, "/workflows/"+s.workflowA.ID, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete workflow status = %d, body = %s", rec.Code, rec.Body.String())
	}

	left := s.listed(t, "")
	if len(left) != 1 || left[0].ID != kept.ID {
		t.Fatalf("expected only the other workflow's link to remain, got %+v", left)
	}
}

func TestDeleteObjectiveTakesItsWorkflowLinks(t *testing.T) {
	s := seedWorkflowObjectiveRoutes(t)
	s.link(t, s.workflowA.ID, s.objectiveA.ID)
	s.link(t, s.workflowB.ID, s.objectiveA.ID)
	kept := s.link(t, s.workflowA.ID, s.objectiveB.ID)

	if rec := s.call(http.MethodDelete, "/objectives/"+s.objectiveA.ID, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete objective status = %d, body = %s", rec.Code, rec.Body.String())
	}

	left := s.listed(t, "")
	if len(left) != 1 || left[0].ID != kept.ID {
		t.Fatalf("expected only the other objective's link to remain, got %+v", left)
	}
}
