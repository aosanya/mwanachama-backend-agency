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

func seedPublishableAgency(t *testing.T, mgr mwanachamaagency.AgencyManager) string {
	t.Helper()
	ctx := t.Context()
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "SWELab"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	d, err := mgr.CreateDraft(ctx, "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	g, err := mgr.CreateGoal(ctx, models.Goal{DraftID: d.ID, Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	o, err := mgr.CreateObjective(ctx, models.Objective{DraftID: d.ID, GoalID: g.ID, Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	w, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: d.ID, Name: "Onboarding"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	if _, err := mgr.CreateWorkflowObjective(ctx, models.WorkflowObjective{DraftID: d.ID, WorkflowID: w.ID, ObjectiveID: o.ID}); err != nil {
		t.Fatalf("CreateWorkflowObjective: %v", err)
	}
	step, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: d.ID, WorkflowID: w.ID, Title: "Write charter"})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: d.ID, Name: "owner"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	step.AccountableRoleID = role.ID
	if _, err := mgr.UpdateWorkItem(ctx, step); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}
	return d.ID
}

func TestPublishAgency_ValidationFailureIsConflict(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.CreateAgency(t.Context(), models.Agency{Name: "SWELab"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	d, err := mgr.CreateDraft(t.Context(), "empty", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/publications", strings.NewReader(`{"draft_id":"`+d.ID+`"}`))
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "no Goals") {
		t.Errorf("expected the body to name the finding, got %s", rec.Body.String())
	}
}

func TestPublishAgency_Tag(t *testing.T) {
	mgr := newTestManager(t)
	draftID := seedPublishableAgency(t, mgr)

	req := httptest.NewRequest(http.MethodPost, "/publications", strings.NewReader(`{"draft_id":"`+draftID+`","tag":"launch"}`))
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("publish status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var pub models.AgencyPublication
	if err := json.Unmarshal(rec.Body.Bytes(), &pub); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if pub.Tag != "launch" {
		t.Fatalf("expected tag %q, got %q", "launch", pub.Tag)
	}

	if _, err := mgr.CreateValue(t.Context(), models.Value{DraftID: draftID, Text: "correctness over speed"}); err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	dup := httptest.NewRequest(http.MethodPost, "/publications", strings.NewReader(`{"draft_id":"`+draftID+`","tag":"launch"}`))
	dupRec := httptest.NewRecorder()
	serve(mgr, dupRec, dup)
	if dupRec.Code != http.StatusConflict {
		t.Fatalf("duplicate-tag status = %d, body = %s", dupRec.Code, dupRec.Body.String())
	}
}

func TestPublishAgencyAndGetPublication(t *testing.T) {
	mgr := newTestManager(t)
	draftID := seedPublishableAgency(t, mgr)

	req := httptest.NewRequest(http.MethodPost, "/publications", strings.NewReader(`{"draft_id":"`+draftID+`"}`))
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("publish status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var pub models.AgencyPublication
	if err := json.Unmarshal(rec.Body.Bytes(), &pub); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if pub.Version != 1 {
		t.Fatalf("expected version 1, got %d", pub.Version)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/publications", strings.NewReader(`{"draft_id":"`+draftID+`"}`))
	rec2 := httptest.NewRecorder()
	serve(mgr, rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("re-publish status = %d, body = %s", rec2.Code, rec2.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/publications/1", nil)
	getReq.SetPathValue("version", "1")
	getRec := httptest.NewRecorder()
	serve(mgr, getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", getRec.Code, getRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/publications", nil)
	listRec := httptest.NewRecorder()
	serve(mgr, listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}

	statusReq := httptest.NewRequest(http.MethodPatch, "/publications/1", strings.NewReader(`{"channel":""}`))
	statusReq.SetPathValue("version", "1")
	statusRec := httptest.NewRecorder()
	serve(mgr, statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", statusRec.Code, statusRec.Body.String())
	}
}

func TestGetPublication_BadVersion(t *testing.T) {
	mgr := newTestManager(t)
	req := httptest.NewRequest(http.MethodGet, "/publications/not-a-number", nil)
	req.SetPathValue("version", "not-a-number")
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestPublishAgency_WorkflowWithNoObjectiveIsConflict(t *testing.T) {
	mgr := newTestManager(t)
	draftID := seedPublishableAgency(t, mgr)
	w, err := mgr.CreateWorkflow(t.Context(), models.Workflow{DraftID: draftID, Name: "Orphan flow"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	if _, err := mgr.CreateWorkItem(t.Context(), models.WorkItem{DraftID: draftID, WorkflowID: w.ID, Title: "Orphan step"}); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/publications", strings.NewReader(`{"draft_id":"`+draftID+`"}`))
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "no Objective") {
		t.Errorf("expected the body to name the finding, got %s", rec.Body.String())
	}
}

func TestPublishAgency_DocumentCarriesTheJoinAndTheAccountableRole(t *testing.T) {
	mgr := newTestManager(t)
	draftID := seedPublishableAgency(t, mgr)

	req := httptest.NewRequest(http.MethodPost, "/publications", strings.NewReader(`{"draft_id":"`+draftID+`"}`))
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("publish status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Document struct {
			WorkflowObjectives []map[string]any `json:"workflow_objectives"`
			WorkItems          []map[string]any `json:"work_items"`
		} `json:"document"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body.Document.WorkflowObjectives) != 1 {
		t.Fatalf("expected one workflow objective in the document, got %s", rec.Body.String())
	}
	if len(body.Document.WorkItems) != 1 || body.Document.WorkItems[0]["accountable_role_id"] == nil {
		t.Fatalf("expected the step to carry its accountable role in the document, got %+v", body.Document.WorkItems)
	}
	if _, has := body.Document.WorkItems[0]["role_assignments"]; has {
		t.Errorf("a work item must not carry role assignments: %+v", body.Document.WorkItems[0])
	}
	if _, has := body.Document.WorkItems[0]["objective_id"]; has {
		t.Errorf("a work item must not carry an objective_id any more: %+v", body.Document.WorkItems[0])
	}
}
