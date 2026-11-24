package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

var newStepFieldKeys = []string{
	"function", "bindings", "result_name", "on_failure", "amount_step_id",
	"debit_account", "credit_account", "reason", "hold", "held_step_id",
}

func loanBindings(applyStepID, recordID string) []models.StepBinding {
	return []models.StepBinding{
		{Input: "principal", SourceType: models.SourceAnswer, StepID: applyStepID, Question: "Amount requested"},
		{Input: "monthly_rate", SourceType: models.SourceFixed, Value: "0.01"},
		{Input: "months", SourceType: models.SourceRecordField, RecordID: recordID, Question: "Months"},
		{Input: "balance", SourceType: models.SourceLedgerBalance},
	}
}

func readBack(t *testing.T, w stepWorld, id string, draftID string) (models.WorkItem, models.WorkItem) {
	t.Helper()
	ctx := context.Background()
	got, err := w.mgr.GetWorkItem(ctx, id)
	if err != nil {
		t.Fatalf("GetWorkItem: %v", err)
	}
	var listed models.WorkItem
	all, err := w.mgr.ListWorkItems(ctx, draftID, mwanachamaagency.WorkItemFilter{})
	if err != nil {
		t.Fatalf("ListWorkItems: %v", err)
	}
	for _, wi := range all {
		if wi.ID == id {
			listed = wi
		}
	}
	return got, listed
}

func TestStepFields_ComputeStepRoundTrips(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Loan payment")
	apply, err := w.mgr.CreateWorkItem(ctx, w.item(wf, 1, stepSpec{
		title: "Apply for a loan", kind: models.StepKindFillForm, role: coordinator, form: "Loan request",
		questions: []models.StepQuestion{ask("Amount requested", models.AnswerCurrency), ask("Months", models.AnswerNumber)},
	}))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	terms, err := w.mgr.CreateRecord(ctx, models.Record{
		DraftID: w.draftID, WorkflowID: wf.ID, Name: "Loan terms",
		FormName: "Terms", Questions: []models.StepQuestion{ask("Months", models.AnswerNumber)},
	})
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	fallback, err := w.mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: w.draftID, WorkflowID: wf.ID, Title: "Ask the officer", Ordinality: 3})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}

	want := models.WorkItem{
		DraftID: w.draftID, WorkflowID: wf.ID, Title: "Work out the payment", Ordinality: 2,
		Kind:       models.StepKindCompute,
		Function:   "fin.pmt@2",
		Bindings:   loanBindings(apply.ID, terms.ID),
		ResultName: "Payment to post",
		OnFailure:  &models.StepFailure{Choice: models.FailureGoToStep, StepID: fallback.ID},
		Hold:       &models.StepHold{ExpiresAfterDays: 14},
	}
	created, err := w.mgr.CreateWorkItem(ctx, want)
	if err != nil {
		t.Fatalf("CreateWorkItem compute step: %v", err)
	}
	got, listed := readBack(t, w, created.ID, w.draftID)
	for name, step := range map[string]models.WorkItem{"created": created, "got": got, "listed": listed} {
		if step.Kind != models.StepKindCompute || step.Function != want.Function || step.ResultName != want.ResultName {
			t.Errorf("%s compute step = %q %q %q", name, step.Kind, step.Function, step.ResultName)
		}
		if !reflect.DeepEqual(step.Bindings, want.Bindings) {
			t.Errorf("%s bindings = %+v, want %+v", name, step.Bindings, want.Bindings)
		}
		if !reflect.DeepEqual(step.OnFailure, want.OnFailure) {
			t.Errorf("%s on_failure = %+v, want %+v", name, step.OnFailure, want.OnFailure)
		}
		if !reflect.DeepEqual(step.Hold, want.Hold) {
			t.Errorf("%s hold = %+v, want %+v", name, step.Hold, want.Hold)
		}
	}
}

func TestStepFields_StopFailureChoiceRoundTrips(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Loan payment")
	created, err := w.mgr.CreateWorkItem(ctx, models.WorkItem{
		DraftID: w.draftID, WorkflowID: wf.ID, Title: "Work out the payment", Ordinality: 1,
		Kind: models.StepKindCompute, Function: "fin.pmt@2", ResultName: "Payment",
		OnFailure: &models.StepFailure{Choice: models.FailureStop},
	})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	got, _ := readBack(t, w, created.ID, w.draftID)
	if got.OnFailure == nil || got.OnFailure.Choice != models.FailureStop || got.OnFailure.StepID != "" {
		t.Errorf("stop choice read back as %+v", got.OnFailure)
	}
	if got.Hold != nil {
		t.Errorf("a step with no hold read back with hold %+v", got.Hold)
	}
}

func TestStepFields_PostStepRoundTrips(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Loan payment")
	work, err := w.mgr.CreateWorkItem(ctx, models.WorkItem{
		DraftID: w.draftID, WorkflowID: wf.ID, Title: "Work out the payment", Ordinality: 1,
		Kind: models.StepKindCompute, Function: "fin.pmt@2", ResultName: "Payment to post",
	})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	want := models.WorkItem{
		DraftID: w.draftID, WorkflowID: wf.ID, Title: "Post the payment", Ordinality: 2,
		Kind:          models.StepKindPost,
		AmountStepID:  work.ID,
		DebitAccount:  "Cash",
		CreditAccount: "Member loans",
		Reason:        "Loan payment",
	}
	created, err := w.mgr.CreateWorkItem(ctx, want)
	if err != nil {
		t.Fatalf("CreateWorkItem post step: %v", err)
	}
	got, listed := readBack(t, w, created.ID, w.draftID)
	for name, step := range map[string]models.WorkItem{"created": created, "got": got, "listed": listed} {
		if step.Kind != models.StepKindPost || step.AmountStepID != work.ID || step.DebitAccount != "Cash" ||
			step.CreditAccount != "Member loans" || step.Reason != "Loan payment" {
			t.Errorf("%s post step = %+v", name, step)
		}
		if step.Function != "" || len(step.Bindings) != 0 || step.OnFailure != nil || step.Hold != nil {
			t.Errorf("%s post step carries compute fields: %+v", name, step)
		}
	}
}

func TestStepFields_HeldStepIDRoundTripsOnYesNoAndPost(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Quick loan interest")
	work, err := w.mgr.CreateWorkItem(ctx, models.WorkItem{
		DraftID: w.draftID, WorkflowID: wf.ID, Title: "Work out the interest", Ordinality: 1,
		Kind: models.StepKindCompute, Function: "fin.interest@1", ResultName: "Interest",
		Hold: &models.StepHold{ExpiresAfterDays: 14},
	})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	approve, err := w.mgr.CreateWorkItem(ctx, models.WorkItem{
		DraftID: w.draftID, WorkflowID: wf.ID, Title: "Approve the interest", Ordinality: 2,
		Kind: models.StepKindYesNo, RoleID: w.roles[coordinator].ID, FormName: "Interest approval",
		Questions:  []models.StepQuestion{ask("Approve this interest?", models.AnswerYesNo)},
		HeldStepID: work.ID,
	})
	if err != nil {
		t.Fatalf("CreateWorkItem yes_no step: %v", err)
	}
	post, err := w.mgr.CreateWorkItem(ctx, models.WorkItem{
		DraftID: w.draftID, WorkflowID: wf.ID, Title: "Post the interest", Ordinality: 3,
		Kind: models.StepKindPost, AmountStepID: work.ID, HeldStepID: work.ID,
		DebitAccount: "Member loans", CreditAccount: "Interest income", Reason: "Interest",
	})
	if err != nil {
		t.Fatalf("CreateWorkItem post step: %v", err)
	}
	for _, step := range []models.WorkItem{approve, post} {
		got, listed := readBack(t, w, step.ID, w.draftID)
		if got.HeldStepID != work.ID || listed.HeldStepID != work.ID {
			t.Errorf("%s held_step_id: got %q, listed %q, want %q", step.Title, got.HeldStepID, listed.HeldStepID, work.ID)
		}
	}
	held, _ := readBack(t, w, work.ID, w.draftID)
	if held.Hold == nil || held.Hold.ExpiresAfterDays != 14 {
		t.Errorf("held compute step read back with hold %+v", held.Hold)
	}
}

func TestStepFields_OlderKindsCarryNone(t *testing.T) {
	w := newStepWorld(t)
	wf := w.workflow(t, "Group order")
	w.build(t, wf, groupOrderSteps)
	for _, step := range w.stepsOf(t, w.draftID, wf) {
		if step.Function != "" || len(step.Bindings) != 0 || step.ResultName != "" || step.OnFailure != nil ||
			step.AmountStepID != "" || step.DebitAccount != "" || step.CreditAccount != "" ||
			step.Reason != "" || step.Hold != nil || step.HeldStepID != "" {
			t.Errorf("step %q of kind %q carries a new field: %+v", step.Title, step.Kind, step)
		}
	}
}

func TestStepFields_DocumentLeavesOutEmptyFields(t *testing.T) {
	bare, err := json.Marshal(models.DocumentWorkItem{ID: "wi-1", Kind: models.StepKindNotify, FormName: "Order"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, key := range newStepFieldKeys {
		if strings.Contains(string(bare), `"`+key+`"`) {
			t.Errorf("a document step of an older kind has the key %q: %s", key, bare)
		}
	}

	full, err := json.Marshal(models.DocumentWorkItem{
		ID: "wi-2", Kind: models.StepKindCompute, Function: "fin.pmt@2", ResultName: "Payment",
		Bindings:  []models.StepBinding{{Input: "months", SourceType: models.SourceFixed, Value: "12"}},
		OnFailure: &models.StepFailure{Choice: models.FailureStop},
		Hold:      &models.StepHold{ExpiresAfterDays: 14},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, key := range []string{"function", "bindings", "result_name", "on_failure", "hold"} {
		if !strings.Contains(string(full), `"`+key+`"`) {
			t.Errorf("a document compute step is missing the key %q: %s", key, full)
		}
	}
}
