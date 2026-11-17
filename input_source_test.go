package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type loanWorld struct {
	world   stepWorld
	apply   models.WorkItem
	compute models.WorkItem
	record  models.Record
}

func buildLoanWorld(t *testing.T) loanWorld {
	t.Helper()
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Loan application")
	apply, err := w.mgr.CreateWorkItem(ctx, w.item(wf, 1, stepSpec{
		title: "Apply for a loan", kind: models.StepKindFillForm, role: coordinator, form: "Loan request",
		questions: []models.StepQuestion{
			ask("Amount requested", models.AnswerCurrency),
			ask("Reason for the loan", models.AnswerFreeText),
			ask("Photo of the id", models.AnswerPhoto),
		},
	}))
	if err != nil {
		t.Fatalf("CreateWorkItem apply: %v", err)
	}
	compute, err := w.mgr.CreateWorkItem(ctx, models.WorkItem{
		DraftID: w.draftID, WorkflowID: wf.ID, Title: "Work out the monthly payment",
		Ordinality: 2, Kind: models.StepKindCompute, Function: "fin.pmt@2", ResultName: "Monthly payment",
	})
	if err != nil {
		t.Fatalf("CreateWorkItem compute: %v", err)
	}
	if _, err := w.mgr.CreateWorkItem(ctx, w.item(wf, 3, stepSpec{
		title: "Confirm the loan", kind: models.StepKindFillForm, role: coordinator, form: "Loan confirmation",
		questions: []models.StepQuestion{ask("Confirmed on", models.AnswerDate)},
	})); err != nil {
		t.Fatalf("CreateWorkItem confirm: %v", err)
	}
	record, err := w.mgr.CreateRecord(ctx, models.Record{
		DraftID: w.draftID, WorkflowID: wf.ID, Name: "Loan terms", RoleID: w.roles[coordinator].ID,
		FormName: "Terms", Questions: []models.StepQuestion{
			ask("Months", models.AnswerNumber),
			{Prompt: "Attachments", AnswerType: models.AnswerMultiSelect, Required: true, Options: []string{"Identity card", "Pay slip"}},
		},
	})
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	return loanWorld{world: w, apply: apply, compute: compute, record: record}
}

func listSources(t *testing.T, l loanWorld, filter mwanachamaagency.InputSourceFilter) mwanachamaagency.InputSources {
	t.Helper()
	got, err := l.world.mgr.ListInputSources(context.Background(), l.world.draftID, l.compute.ID, filter)
	if err != nil {
		t.Fatalf("ListInputSources: %v", err)
	}
	return got
}

func promptsOf(sources []mwanachamaagency.InputSource) []string {
	out := make([]string, len(sources))
	for i, s := range sources {
		out[i] = s.Question
	}
	return out
}

func TestInputSources_ACurrencyInputIsOfferedTheAmountAndTheBalanceAndNotTheMonths(t *testing.T) {
	l := buildLoanWorld(t)
	got := listSources(t, l, mwanachamaagency.InputSourceFilter{InputType: "currency"})

	if p := promptsOf(got.Answer); len(p) != 1 || p[0] != "Amount requested" {
		t.Fatalf("answer sources = %v, want [Amount requested]", p)
	}
	a := got.Answer[0]
	if a.StepID != l.apply.ID || a.StepTitle != "Apply for a loan" || a.AnswerType != models.AnswerCurrency || a.Type != "currency" {
		t.Errorf("answer source = %+v, want the step, its title, the answer type currency and the type currency", a)
	}
	if len(got.RecordField) != 0 {
		t.Errorf("record fields = %v, want none: Months is a number", promptsOf(got.RecordField))
	}
	if len(got.LedgerBalance) != 1 || got.LedgerBalance[0].Type != "currency" {
		t.Errorf("ledger balance = %+v, want the one balance, a currency", got.LedgerBalance)
	}
}

func TestInputSources_WithoutAFilterTheListIsGroupedByTheFourSourceTypes(t *testing.T) {
	l := buildLoanWorld(t)
	got := listSources(t, l, mwanachamaagency.InputSourceFilter{})

	if p := promptsOf(got.Answer); len(p) != 2 || p[0] != "Amount requested" || p[1] != "Reason for the loan" {
		t.Errorf("answer sources = %v, want [Amount requested Reason for the loan]: the photo and the later step are not offered", p)
	}
	if got.Answer[1].Type != "text" {
		t.Errorf("a free-text question has type %q, want text", got.Answer[1].Type)
	}
	if len(got.RecordField) != 1 {
		t.Fatalf("record fields = %v, want [Months]: a multi select cannot be a source", promptsOf(got.RecordField))
	}
	r := got.RecordField[0]
	if r.RecordID != l.record.ID || r.RecordName != "Loan terms" || r.Question != "Months" || r.AnswerType != models.AnswerNumber || r.Type != "number" {
		t.Errorf("record field = %+v, want the record, its name, the question Months and the type number", r)
	}
	if len(got.LedgerBalance) != 1 {
		t.Errorf("ledger balance = %+v, want the one balance", got.LedgerBalance)
	}
	if len(got.Fixed) != 1 {
		t.Errorf("fixed = %+v, want the one typed-in value", got.Fixed)
	}
}

func TestInputSources_ANumberInputIsOfferedTheMonthsAndNotTheAmount(t *testing.T) {
	l := buildLoanWorld(t)
	got := listSources(t, l, mwanachamaagency.InputSourceFilter{InputType: "number"})

	if len(got.Answer) != 0 {
		t.Errorf("answer sources = %v, want none", promptsOf(got.Answer))
	}
	if p := promptsOf(got.RecordField); len(p) != 1 || p[0] != "Months" {
		t.Errorf("record fields = %v, want [Months]", p)
	}
	if len(got.LedgerBalance) != 0 {
		t.Errorf("ledger balance = %+v, want none: it is a currency", got.LedgerBalance)
	}
}

func TestInputSources_TheFirstStepHasNoEarlierAnswers(t *testing.T) {
	l := buildLoanWorld(t)
	got, err := l.world.mgr.ListInputSources(context.Background(), l.world.draftID, l.apply.ID, mwanachamaagency.InputSourceFilter{})
	if err != nil {
		t.Fatalf("ListInputSources: %v", err)
	}
	if len(got.Answer) != 0 {
		t.Errorf("answer sources = %v, want none", promptsOf(got.Answer))
	}
	if len(got.RecordField) != 1 || len(got.LedgerBalance) != 1 || len(got.Fixed) != 1 {
		t.Errorf("the other three groups still list: %+v", got)
	}
}

func TestInputSources_AnUnknownStepOrTypeIsRefused(t *testing.T) {
	l := buildLoanWorld(t)
	ctx := context.Background()
	if _, err := l.world.mgr.ListInputSources(ctx, l.world.draftID, "no-such-step", mwanachamaagency.InputSourceFilter{}); !errors.Is(err, mwanachamaagency.ErrWorkItemNotFound) {
		t.Errorf("an unknown step: want ErrWorkItemNotFound, got %v", err)
	}
	if _, err := l.world.mgr.ListInputSources(ctx, l.world.draftID, l.compute.ID, mwanachamaagency.InputSourceFilter{InputType: "colour"}); !errors.Is(err, mwanachamaagency.ErrInvalidInputType) {
		t.Errorf("an unknown type: want ErrInvalidInputType, got %v", err)
	}
}
