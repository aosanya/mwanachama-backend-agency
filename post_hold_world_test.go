package mwanachamaagency_test

import (
	"context"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type lakesideAccounts map[string]bool

func (a lakesideAccounts) HasAccount(ctx context.Context, draftID, account string) (bool, error) {
	return a[account], nil
}

type postHoldWorld struct {
	stepWorld
	wf models.Workflow
}

func newPostHoldWorld(t *testing.T) postHoldWorld {
	t.Helper()
	mwanachamaagency.ClearStepKindRegistry()
	mwanachamaagency.RegisterPostAndHoldChecks()
	mwanachamaagency.SetAccountDirectory(nil)
	t.Cleanup(func() {
		mwanachamaagency.ClearStepKindRegistry()
		mwanachamaagency.SetAccountDirectory(nil)
	})
	w := newStepWorld(t)
	return postHoldWorld{stepWorld: w, wf: w.workflow(t, "Loan payment")}
}

func (w postHoldWorld) step(title string, ordinality int, kind models.StepKind, change func(*models.WorkItem)) (models.WorkItem, error) {
	wi := models.WorkItem{DraftID: w.draftID, WorkflowID: w.wf.ID, Title: title, Ordinality: ordinality, Kind: kind}
	if change != nil {
		change(&wi)
	}
	return w.mgr.CreateWorkItem(context.Background(), wi)
}

func (w postHoldWorld) mustStep(t *testing.T, title string, ordinality int, kind models.StepKind, change func(*models.WorkItem)) models.WorkItem {
	t.Helper()
	wi, err := w.step(title, ordinality, kind, change)
	if err != nil {
		t.Fatalf("CreateWorkItem %q: %v", title, err)
	}
	return wi
}

func (w postHoldWorld) workOutThePayment(t *testing.T, ordinality int, change func(*models.WorkItem)) models.WorkItem {
	return w.mustStep(t, "Work out the payment", ordinality, models.StepKindCompute, func(wi *models.WorkItem) {
		wi.Function = "loan.cap@3"
		wi.ResultName = "Payment to post"
		if change != nil {
			change(wi)
		}
	})
}

func (w postHoldWorld) postThePayment(t *testing.T, ordinality int, amountStepID string, change func(*models.WorkItem)) models.WorkItem {
	return w.mustStep(t, "Post the payment", ordinality, models.StepKindPost, func(wi *models.WorkItem) {
		wi.AmountStepID = amountStepID
		wi.Account = "Member loans"
		wi.CounterAccount = "Cash"
		wi.Direction = "credit"
		wi.Reason = "Loan payment"
		if change != nil {
			change(wi)
		}
	})
}

func (w postHoldWorld) errorsAbout(t *testing.T, stepID string) []string {
	t.Helper()
	issues, err := w.mgr.ListIssues(context.Background(), w.draftID, mwanachamaagency.IssueFilter{EntityType: "work_item"})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	var messages []string
	for _, issue := range issues {
		if issue.EntityID == stepID && issue.Severity == models.IssueSeverityError {
			messages = append(messages, issue.Message)
		}
	}
	return messages
}

func requireOneError(t *testing.T, messages []string, wants ...string) {
	t.Helper()
	if len(messages) != 1 {
		t.Fatalf("want one error, got %d: %q", len(messages), messages)
	}
	for _, want := range wants {
		if !strings.Contains(messages[0], want) {
			t.Fatalf("error %q does not say %q", messages[0], want)
		}
	}
}

func requireNoErrors(t *testing.T, messages []string) {
	t.Helper()
	if len(messages) != 0 {
		t.Fatalf("want no errors, got %q", messages)
	}
}

func withForm(wi *models.WorkItem) {
	wi.FormName = "Loan form"
	wi.Questions = []models.StepQuestion{{Prompt: "Amount requested", AnswerType: models.AnswerCurrency}}
}
