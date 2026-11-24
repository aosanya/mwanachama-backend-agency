package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCreateObjective(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	g, err := mgr.CreateGoal(t.Context(), models.Goal{DraftID: draftID, Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}

	body := `{"goal_id":"` + g.ID + `","title":"Grow adoption","draft_id":"` + draftID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/objectives", strings.NewReader(body))
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.Objective
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID == "" || out.GoalID != g.ID {
		t.Fatalf("unexpected objective: %+v", out)
	}
}

func TestCreateObjective_MultiplePerGoal(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	g, _ := mgr.CreateGoal(t.Context(), models.Goal{DraftID: draftID, Title: "Grow adoption"})
	if _, err := mgr.CreateObjective(t.Context(), models.Objective{DraftID: draftID, GoalID: g.ID, Title: "First"}); err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}

	body := `{"goal_id":"` + g.ID + `","title":"Second","draft_id":"` + draftID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/objectives", strings.NewReader(body))
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/objectives?draft_id="+draftID, nil)
	listRec := httptest.NewRecorder()
	serve(mgr, listRec, listReq)
	var out []models.Objective
	if err := json.Unmarshal(listRec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 objectives for goal %q, got %d: %+v", g.ID, len(out), out)
	}
}

func TestLinkWorkflowToObjective(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	g, _ := mgr.CreateGoal(t.Context(), models.Goal{DraftID: draftID, Title: "Grow adoption"})
	o, _ := mgr.CreateObjective(t.Context(), models.Objective{DraftID: draftID, GoalID: g.ID, Title: "Grow adoption"})
	wf, _ := mgr.CreateWorkflow(t.Context(), models.Workflow{DraftID: draftID, Name: "Onboarding"})

	body := `{"workflow_id":"` + wf.ID + `","objective_id":"` + o.ID + `","draft_id":"` + draftID + `"}`
	rec := httptest.NewRecorder()
	serve(mgr, rec, httptest.NewRequest(http.MethodPost, "/workflow-objectives", strings.NewReader(body)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var link models.WorkflowObjective
	if err := json.Unmarshal(rec.Body.Bytes(), &link); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if link.ID == "" || link.WorkflowID != wf.ID || link.ObjectiveID != o.ID {
		t.Fatalf("unexpected link: %+v", link)
	}

	again := httptest.NewRecorder()
	serve(mgr, again, httptest.NewRequest(http.MethodPost, "/workflow-objectives", strings.NewReader(body)))
	if again.Code != http.StatusConflict {
		t.Fatalf("linking the same pair twice: status = %d, body = %s, want %d", again.Code, again.Body.String(), http.StatusConflict)
	}

	listRec := httptest.NewRecorder()
	serve(mgr, listRec, httptest.NewRequest(http.MethodGet, "/workflow-objectives?draft_id="+draftID+"&objective_id="+o.ID, nil))
	var links []models.WorkflowObjective
	if err := json.Unmarshal(listRec.Body.Bytes(), &links); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(links) != 1 || links[0].ID != link.ID {
		t.Fatalf("unexpected links: %+v", links)
	}
}
