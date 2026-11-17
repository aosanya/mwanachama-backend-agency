package mcp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

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

type lakesideTools struct {
	mgr        mwanachamaagency.AgencyManager
	session    *gosdkmcp.ClientSession
	draftID    string
	workflowID string
}

func newLakesideTools(t *testing.T) lakesideTools {
	t.Helper()
	mgr := newTestManager(t, mwanachamaagency.WithFunctionCatalog(lakesideLibrary{"fin.pmt": nil}))
	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	cs := mcpConnect(t, srv.URL)
	if _, err := mgr.CreateAgency(ctxT(t), models.Agency{Name: "Lakeside Savings Circle"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draft, err := mgr.CreateDraft(ctxT(t), "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	wf, err := mgr.CreateWorkflow(ctxT(t), models.Workflow{DraftID: draft.ID, Name: "Loan payment"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	return lakesideTools{mgr: mgr, session: cs, draftID: draft.ID, workflowID: wf.ID}
}

func (l lakesideTools) createWorkItem(t *testing.T, args map[string]any) *gosdkmcp.CallToolResult {
	t.Helper()
	args["instance_id"] = "any"
	args["draft_id"] = l.draftID
	args["workflow_id"] = l.workflowID
	result, err := l.session.CallTool(ctxT(t), &gosdkmcp.CallToolParams{Name: "agency_create_work_item", Arguments: args})
	if err != nil {
		t.Fatalf("agency_create_work_item: %v", err)
	}
	return result
}

func TestComputeStepTools_AFunctionTheLibraryDoesNotHaveIsNotFound(t *testing.T) {
	world := newLakesideTools(t)
	result := world.createWorkItem(t, map[string]any{
		"title": "Work out the payment", "ordinality": 1, "kind": "compute",
		"function": "fin.nothing@1", "result_name": "Payment to post",
	})
	if !result.IsError || !strings.Contains(textOf(result), "function not found") {
		t.Fatalf("result = IsError %v, %q, want a refusal saying the function was not found", result.IsError, textOf(result))
	}
}

func TestComputeStepTools_AnAccountTheAgencyDoesNotHaveIsNotFound(t *testing.T) {
	world := newLakesideTools(t)
	amount := world.createWorkItem(t, map[string]any{
		"title": "Work out the payment", "ordinality": 1, "kind": "compute",
		"function": "fin.pmt@2", "result_name": "Payment to post",
	})
	if amount.IsError {
		t.Fatalf("the calculating step: %q", textOf(amount))
	}
	steps, err := world.mgr.ListWorkItems(ctxT(t), world.draftID, mwanachamaagency.WorkItemFilter{})
	if err != nil || len(steps) != 1 {
		t.Fatalf("ListWorkItems = %v, %v", steps, err)
	}
	mwanachamaagency.SetAccountDirectory(lakesideAccounts{"Cash": true})
	t.Cleanup(func() { mwanachamaagency.SetAccountDirectory(nil) })

	result := world.createWorkItem(t, map[string]any{
		"title": "Post the payment", "ordinality": 2, "kind": "post", "amount_step_id": steps[0].ID,
		"account": "Cash", "counter_account": "Member loans", "direction": "debit", "reason": "Loan payment",
	})
	if !result.IsError || !strings.Contains(textOf(result), "account not found") {
		t.Fatalf("result = IsError %v, %q, want a refusal saying the account was not found", result.IsError, textOf(result))
	}
}
