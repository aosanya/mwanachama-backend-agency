package routes_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type lakesideLibrary map[string][]mwanachamaagency.FunctionInput

func (l lakesideLibrary) InputsOf(ctx context.Context, name string, version int) ([]mwanachamaagency.FunctionInput, bool, error) {
	inputs, ok := l[name]
	return inputs, ok, nil
}

type lakesideAccounts map[string]bool

func (a lakesideAccounts) HasAccount(ctx context.Context, draftID, account string) (bool, error) {
	return a[account], nil
}

type lakesideRoutes struct {
	mgr        mwanachamaagency.AgencyManager
	draftID    string
	workflowID string
}

func newLakesideRoutes(t *testing.T) lakesideRoutes {
	t.Helper()
	mgr := newTestManager(t, mwanachamaagency.WithFunctionCatalog(lakesideLibrary{"fin.pmt": nil}))
	if _, err := mgr.CreateAgency(t.Context(), models.Agency{Name: "Lakeside Savings Circle"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draft, err := mgr.CreateDraft(t.Context(), "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	wf, err := mgr.CreateWorkflow(t.Context(), models.Workflow{DraftID: draft.ID, Name: "Loan payment"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	return lakesideRoutes{mgr: mgr, draftID: draft.ID, workflowID: wf.ID}
}

func (l lakesideRoutes) post(payload string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	serve(l.mgr, rec, httptest.NewRequest(http.MethodPost, "/work-items", strings.NewReader(payload)))
	return rec
}

func (l lakesideRoutes) computeBody(function string) string {
	return `{"workflow_id":"` + l.workflowID + `","draft_id":"` + l.draftID + `","title":"Work out the payment",` +
		`"ordinality":1,"kind":"compute","function":"` + function + `","result_name":"Payment to post"}`
}

func TestComputeStepRoutes_AFunctionTheLibraryDoesNotHaveIsNotFound(t *testing.T) {
	world := newLakesideRoutes(t)
	rec := world.post(world.computeBody("fin.nothing@1"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "function not found") {
		t.Errorf("body = %s, want the sentence saying the function was not found", rec.Body.String())
	}
}

func TestComputeStepRoutes_AnAccountTheAgencyDoesNotHaveIsNotFound(t *testing.T) {
	world := newLakesideRoutes(t)
	created := world.post(world.computeBody("fin.pmt@2"))
	if created.Code != http.StatusCreated {
		t.Fatalf("the calculating step: status = %d, body = %s", created.Code, created.Body.String())
	}
	amount, err := world.mgr.ListWorkItems(t.Context(), world.draftID, mwanachamaagency.WorkItemFilter{})
	if err != nil || len(amount) != 1 {
		t.Fatalf("ListWorkItems = %v, %v", amount, err)
	}
	mwanachamaagency.SetAccountDirectory(lakesideAccounts{"Cash": true})
	t.Cleanup(func() { mwanachamaagency.SetAccountDirectory(nil) })

	rec := world.post(`{"workflow_id":"` + world.workflowID + `","draft_id":"` + world.draftID + `","title":"Post the payment",` +
		`"ordinality":2,"kind":"post","amount_step_id":"` + amount[0].ID + `","debit_account":"Cash","credit_account":"Member loans",` +
		`"reason":"Loan payment"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "account not found") {
		t.Errorf("body = %s, want the sentence saying the account was not found", rec.Body.String())
	}
}
