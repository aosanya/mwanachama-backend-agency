package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestWorkItemStepRoutes(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	ctx := t.Context()
	wf, _ := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftID, Name: "Buy now"})
	g, _ := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, Title: "Sell"})
	o, _ := mgr.CreateObjective(ctx, models.Objective{DraftID: draftID, GoalID: g.ID, Title: "Sell"})
	if _, err := mgr.CreateWorkflowObjective(ctx, models.WorkflowObjective{DraftID: draftID, WorkflowID: wf.ID, ObjectiveID: o.ID}); err != nil {
		t.Fatalf("CreateWorkflowObjective: %v", err)
	}
	farmer, _ := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "Farmer"})
	supplier, _ := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "Supplier"})

	body := func(role string) string {
		return `{"workflow_id":"` + wf.ID + `","draft_id":"` + draftID + `","title":"Ask",` +
			`"kind":"fill_form","role_id":` + role + `,"form_name":"Buy now form",` +
			`"questions":[{"prompt":"Enter the number of bags","answer_type":"number","required":true}]}`
	}
	post := func(payload string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		serve(mgr, rec, httptest.NewRequest(http.MethodPost, "/work-items", strings.NewReader(payload)))
		return rec
	}

	created := post(body(`"` + farmer.ID + `"`))
	if created.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", created.Code, created.Body.String())
	}
	var out models.WorkItem
	if err := json.Unmarshal(created.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Kind != models.StepKindFillForm || out.RoleID != farmer.ID || out.Action != "Fills the Buy now form" {
		t.Fatalf("created step = %+v", out)
	}

	read := httptest.NewRecorder()
	serve(mgr, read, httptest.NewRequest(http.MethodGet, "/work-items/"+out.ID, nil))
	var got models.WorkItem
	if err := json.Unmarshal(read.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Kind != models.StepKindFillForm || got.RoleID != farmer.ID || len(got.Questions) != 1 || got.Action != "Fills the Buy now form" {
		t.Fatalf("read step = %+v", got)
	}

	for name, role := range map[string]string{
		"a list of two roles": `["` + farmer.ID + `","` + supplier.ID + `"]`,
		"two roles in one id": `"` + farmer.ID + `,` + supplier.ID + `"`,
	} {
		if rec := post(body(role)); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400, body = %s", name, rec.Code, rec.Body.String())
		}
	}

	missing := strings.Replace(body(`""`), `"kind":"fill_form"`, `"kind":"run_workflow","run_workflow_id":"no-such-workflow"`, 1)
	missing = strings.Replace(missing, `"form_name":"Buy now form",`, ``, 1)
	missing = strings.Replace(missing, `,"questions":[{"prompt":"Enter the number of bags","answer_type":"number","required":true}]`, ``, 1)
	if rec := post(missing); rec.Code != http.StatusNotFound {
		t.Errorf("a workflow that does not exist: status = %d, want 404, body = %s", rec.Code, rec.Body.String())
	}
}
