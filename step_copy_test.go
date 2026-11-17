package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

const (
	applyTitle    = "Apply for a loan"
	computeTitle  = "Work out the monthly payment"
	stopTitle     = "Check the balance"
	fallbackTitle = "Ask the officer"
	approveTitle  = "Approve the figure"
	postTitle     = "Post the payment"
	termsName     = "Loan terms"
)

func useStepCopyRules(t *testing.T) {
	t.Helper()
	mwanachamaagency.ClearStepKindRegistry()
	mwanachamaagency.RegisterStepCopyRules()
	t.Cleanup(mwanachamaagency.ClearStepKindRegistry)
}

type paymentFlow struct {
	world    stepWorld
	workflow models.Workflow
	apply    models.WorkItem
	record   models.Record
	fallback models.WorkItem
	compute  models.WorkItem
	stop     models.WorkItem
	approve  models.WorkItem
	post     models.WorkItem
}

func paymentBindings(applyID, recordID string) []models.StepBinding {
	return []models.StepBinding{
		{Input: "principal", SourceType: models.SourceAnswer, StepID: applyID, Question: "Amount requested"},
		{Input: "monthly_rate", SourceType: models.SourceFixed, Value: "0.01"},
		{Input: "months", SourceType: models.SourceRecordField, RecordID: recordID, Question: "Months"},
		{Input: "balance", SourceType: models.SourceLedgerBalance},
	}
}

func buildPaymentFlow(t *testing.T) paymentFlow {
	t.Helper()
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Loan payment")
	must := func(wi models.WorkItem) models.WorkItem {
		t.Helper()
		out, err := w.mgr.CreateWorkItem(ctx, wi)
		if err != nil {
			t.Fatalf("CreateWorkItem %s: %v", wi.Title, err)
		}
		return out
	}

	apply := must(w.item(wf, 1, stepSpec{
		title: applyTitle, kind: models.StepKindFillForm, role: coordinator, form: "Loan request",
		questions: []models.StepQuestion{ask("Amount requested", models.AnswerCurrency)},
	}))
	record, err := w.mgr.CreateRecord(ctx, models.Record{
		DraftID: w.draftID, WorkflowID: wf.ID, Name: termsName, RoleID: w.roles[coordinator].ID,
		FormName: "Terms", Questions: []models.StepQuestion{ask("Months", models.AnswerNumber)},
	})
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	fallback := must(models.WorkItem{DraftID: w.draftID, WorkflowID: wf.ID, Title: fallbackTitle, Ordinality: 6})
	compute := must(models.WorkItem{
		DraftID: w.draftID, WorkflowID: wf.ID, Title: computeTitle, Ordinality: 2, Kind: models.StepKindCompute,
		Function: "fin.pmt@2", Bindings: paymentBindings(apply.ID, record.ID), ResultName: "Payment to post",
		OnFailure: &models.StepFailure{Choice: models.FailureGoToStep, StepID: fallback.ID},
		Hold:      &models.StepHold{ExpiresAfterDays: 14},
	})
	stop := must(models.WorkItem{
		DraftID: w.draftID, WorkflowID: wf.ID, Title: stopTitle, Ordinality: 3, Kind: models.StepKindCompute,
		Function: "fin.balance@1", ResultName: "Balance",
		Bindings:  []models.StepBinding{{Input: "member", SourceType: models.SourceLedgerBalance}},
		OnFailure: &models.StepFailure{Choice: models.FailureStop},
	})
	approveIn := w.item(wf, 4, stepSpec{
		title: approveTitle, kind: models.StepKindYesNo, role: coordinator, form: "Approval",
		questions: []models.StepQuestion{ask("Approved", models.AnswerYesNo)},
	})
	approveIn.HeldStepID = compute.ID
	approve := must(approveIn)
	post := must(models.WorkItem{
		DraftID: w.draftID, WorkflowID: wf.ID, Title: postTitle, Ordinality: 5, Kind: models.StepKindPost,
		AmountStepID: compute.ID, Account: "Member loans", CounterAccount: "Cash",
		Direction: "credit", Reason: "Loan payment", HeldStepID: compute.ID,
	})
	return paymentFlow{
		world: w, workflow: wf, apply: apply, record: record, fallback: fallback,
		compute: compute, stop: stop, approve: approve, post: post,
	}
}

func copiedSteps(t *testing.T, mgr mwanachamaagency.AgencyManager, draftID string) map[string]models.WorkItem {
	t.Helper()
	all, err := mgr.ListWorkItems(context.Background(), draftID, mwanachamaagency.WorkItemFilter{})
	if err != nil {
		t.Fatalf("ListWorkItems: %v", err)
	}
	out := make(map[string]models.WorkItem, len(all))
	for _, wi := range all {
		out[wi.Title] = wi
	}
	return out
}

func recordNamed(t *testing.T, mgr mwanachamaagency.AgencyManager, draftID, name string) models.Record {
	t.Helper()
	all, err := mgr.ListRecords(context.Background(), draftID, mwanachamaagency.RecordFilter{})
	if err != nil {
		t.Fatalf("ListRecords: %v", err)
	}
	for _, r := range all {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("no record named %q in draft %s", name, draftID)
	return models.Record{}
}

func assertCopiedFlow(t *testing.T, mgr mwanachamaagency.AgencyManager, copyID string, original paymentFlow) {
	t.Helper()
	steps := copiedSteps(t, mgr, copyID)
	record := recordNamed(t, mgr, copyID, termsName)
	apply, fallback, compute := steps[applyTitle], steps[fallbackTitle], steps[computeTitle]

	if apply.ID == original.apply.ID || record.ID == original.record.ID || fallback.ID == original.fallback.ID || compute.ID == original.compute.ID {
		t.Fatal("the copy must have its own steps and its own record")
	}
	if compute.Function != "fin.pmt@2" || compute.ResultName != "Payment to post" {
		t.Errorf("function %q and result name %q must be copied as they are", compute.Function, compute.ResultName)
	}
	if len(compute.Bindings) != 4 {
		t.Fatalf("bindings = %+v", compute.Bindings)
	}
	if b := compute.Bindings[0]; b.StepID != apply.ID || b.Question != "Amount requested" || b.Input != "principal" {
		t.Errorf("the answer binding must point at the copied step %s with its wording, got %+v", apply.ID, b)
	}
	if b := compute.Bindings[1]; b.SourceType != models.SourceFixed || b.Value != "0.01" {
		t.Errorf("the fixed binding must keep its value, got %+v", b)
	}
	if b := compute.Bindings[2]; b.RecordID != record.ID || b.Question != "Months" {
		t.Errorf("the record binding must point at the copied record %s with its wording, got %+v", record.ID, b)
	}
	if b := compute.Bindings[3]; b.SourceType != models.SourceLedgerBalance || b.StepID != "" || b.RecordID != "" {
		t.Errorf("the ledger binding must stay bare, got %+v", b)
	}
	if compute.OnFailure == nil || compute.OnFailure.Choice != models.FailureGoToStep || compute.OnFailure.StepID != fallback.ID {
		t.Errorf("the failure choice must point at the copied step %s, got %+v", fallback.ID, compute.OnFailure)
	}
	if compute.Hold == nil || compute.Hold.ExpiresAfterDays != 14 {
		t.Errorf("hold = %+v, want 14 days", compute.Hold)
	}

	if stop := steps[stopTitle]; stop.OnFailure == nil || stop.OnFailure.Choice != models.FailureStop || stop.OnFailure.StepID != "" {
		t.Errorf("a stop has no id and must be copied as it is, got %+v", stop.OnFailure)
	}
	if approve := steps[approveTitle]; approve.HeldStepID != compute.ID {
		t.Errorf("the releasing step must name the copied calculating step %s, got %s", compute.ID, approve.HeldStepID)
	}
	post := steps[postTitle]
	if post.AmountStepID != compute.ID || post.HeldStepID != compute.ID {
		t.Errorf("the posting step must point at the copied step %s, got amount %s held %s", compute.ID, post.AmountStepID, post.HeldStepID)
	}
	if post.Account != "Member loans" || post.CounterAccount != "Cash" || post.Direction != "credit" || post.Reason != "Loan payment" {
		t.Errorf("accounts, direction and reason are copied as they are, got %+v", post)
	}
}

func TestStepCopy_ADraftForkRemapsBindingsRecordsFailureAndAmountOntoTheCopies(t *testing.T) {
	useStepCopyRules(t)
	flow := buildPaymentFlow(t)
	ctx := context.Background()

	fork, err := flow.world.mgr.CreateDraft(ctx, "fork", flow.world.draftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	assertCopiedFlow(t, flow.world.mgr, fork.ID, flow)
}

func TestStepCopy_APublicationCarriesTheFieldsAndAForkFromItRestoresThemWithTheSameRemap(t *testing.T) {
	useStepCopyRules(t)
	flow := buildPaymentFlow(t)
	ctx := context.Background()

	pub, err := flow.world.mgr.PublishAgency(ctx, flow.world.draftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	var docCompute, docApply, docPost models.DocumentWorkItem
	for _, wi := range pub.Document.WorkItems {
		switch wi.Title {
		case computeTitle:
			docCompute = wi
		case applyTitle:
			docApply = wi
		case postTitle:
			docPost = wi
		}
	}
	if docCompute.Function != "fin.pmt@2" || docCompute.ResultName != "Payment to post" || len(docCompute.Bindings) != 4 ||
		docCompute.OnFailure == nil || docCompute.Hold == nil || docCompute.Hold.ExpiresAfterDays != 14 {
		t.Fatalf("the document must carry the calculating fields: %+v", docCompute)
	}
	if docPost.AmountStepID != docCompute.ID || docPost.Account != "Member loans" || docPost.CounterAccount != "Cash" ||
		docPost.Direction != "credit" || docPost.Reason != "Loan payment" || docPost.HeldStepID != docCompute.ID {
		t.Fatalf("the document must carry the posting fields: %+v", docPost)
	}
	raw, err := json.Marshal(docApply)
	if err != nil {
		t.Fatal(err)
	}
	var keys map[string]any
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatal(err)
	}
	for _, key := range newStepFieldKeys {
		if _, present := keys[key]; present {
			t.Errorf("a step of an older kind must leave %q out of the document", key)
		}
	}

	restored, err := flow.world.mgr.CreateDraft(ctx, "restored", pub.Tag, "publication")
	if err != nil {
		t.Fatalf("CreateDraft from the publication: %v", err)
	}
	assertCopiedFlow(t, flow.world.mgr, restored.ID, flow)
}

func publishAfter(t *testing.T, mgr mwanachamaagency.AgencyManager, draftID string, change func()) error {
	t.Helper()
	change()
	_, err := mgr.PublishAgency(context.Background(), draftID, "", "")
	return err
}

func TestStepCopy_AnAgencyOfOlderKindsHashesAsItDidBeforeAnyRuleWasRegistered(t *testing.T) {
	mwanachamaagency.ClearStepKindRegistry()
	t.Cleanup(mwanachamaagency.ClearStepKindRegistry)
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Onboarding steps")
	w.build(t, wf, []stepSpec{
		{title: "Collect the form", kind: models.StepKindFillForm, role: coordinator, form: "Intake", questions: []models.StepQuestion{ask("Name", models.AnswerFreeText)}},
		{title: "Approve the form", kind: models.StepKindYesNo, role: coordinator, form: "Approval", questions: []models.StepQuestion{ask("Approved", models.AnswerYesNo)}},
	})
	for _, kind := range []models.StepKind{models.StepKindCompute, models.StepKindPost} {
		if _, err := w.mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: w.draftID, WorkflowID: wf.ID, Title: "Empty " + string(kind), Kind: kind}); err != nil {
			t.Fatalf("CreateWorkItem empty %s: %v", kind, err)
		}
	}
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "", ""); err != nil {
		t.Fatalf("PublishAgency before: %v", err)
	}

	mwanachamaagency.RegisterStepCopyRules()
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "", ""); !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("an agency with none of the new fields must hash as before, got %v", err)
	}
}

func TestStepCopy_ACalculatingStepChangesTheHashWhenItsFieldsChangeAndNotWhenOnlyTheBindingsAreReordered(t *testing.T) {
	useStepCopyRules(t)
	flow := buildPaymentFlow(t)
	mgr, draftID := flow.world.mgr, flow.world.draftID
	ctx := context.Background()
	if _, err := mgr.PublishAgency(ctx, draftID, "", ""); err != nil {
		t.Fatalf("PublishAgency (March): %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, draftID, "", ""); !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("nothing changed, got %v", err)
	}

	current := flow.compute
	edit := func(label string, change func(wi *models.WorkItem)) {
		t.Helper()
		err := publishAfter(t, mgr, draftID, func() {
			change(&current)
			updated, err := mgr.UpdateWorkItem(ctx, current)
			if err != nil {
				t.Fatalf("UpdateWorkItem (%s): %v", label, err)
			}
			current = updated
		})
		if err != nil {
			t.Errorf("%s must be a change that can be published, got %v", label, err)
		}
	}

	edit("a new function version", func(wi *models.WorkItem) { wi.Function = "fin.pmt@3" })

	reordered := current
	for i, j := 0, len(reordered.Bindings)-1; i < j; i, j = i+1, j-1 {
		reordered.Bindings[i], reordered.Bindings[j] = reordered.Bindings[j], reordered.Bindings[i]
	}
	err := publishAfter(t, mgr, draftID, func() {
		updated, err := mgr.UpdateWorkItem(ctx, reordered)
		if err != nil {
			t.Fatalf("UpdateWorkItem (reorder): %v", err)
		}
		current = updated
	})
	if !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("reordering the bindings alone must be no change, got %v", err)
	}

	edit("a changed binding", func(wi *models.WorkItem) {
		for i := range wi.Bindings {
			if wi.Bindings[i].Input == "monthly_rate" {
				wi.Bindings[i].Value = "0.02"
			}
		}
	})
	edit("a changed result name", func(wi *models.WorkItem) { wi.ResultName = "Monthly payment" })
	edit("a failure choice going from a step to a stop", func(wi *models.WorkItem) {
		wi.OnFailure = &models.StepFailure{Choice: models.FailureStop}
	})
	edit("a changed hold", func(wi *models.WorkItem) { wi.Hold = &models.StepHold{ExpiresAfterDays: 30} })
	edit("a removed hold", func(wi *models.WorkItem) { wi.Hold = nil })
}

func TestStepCopy_APostingStepAndAReleasingStepChangeTheHashWhenAnyOfTheirFieldsChange(t *testing.T) {
	useStepCopyRules(t)
	flow := buildPaymentFlow(t)
	mgr, draftID := flow.world.mgr, flow.world.draftID
	ctx := context.Background()
	if _, err := mgr.PublishAgency(ctx, draftID, "", ""); err != nil {
		t.Fatalf("PublishAgency (March): %v", err)
	}

	post, approve := flow.post, flow.approve
	changePost := func(label string, change func(wi *models.WorkItem)) {
		t.Helper()
		err := publishAfter(t, mgr, draftID, func() {
			change(&post)
			updated, err := mgr.UpdateWorkItem(ctx, post)
			if err != nil {
				t.Fatalf("UpdateWorkItem (%s): %v", label, err)
			}
			post = updated
		})
		if err != nil {
			t.Errorf("%s must be a change that can be published, got %v", label, err)
		}
	}
	changePost("a reason in more words", func(wi *models.WorkItem) { wi.Reason = "Loan payment for the month" })
	changePost("another account", func(wi *models.WorkItem) { wi.Account = "Member savings" })
	changePost("another counter account", func(wi *models.WorkItem) { wi.CounterAccount = "Bank" })
	changePost("another direction", func(wi *models.WorkItem) { wi.Direction = "debit" })
	changePost("another amount step", func(wi *models.WorkItem) { wi.AmountStepID = flow.stop.ID })
	changePost("another held step", func(wi *models.WorkItem) { wi.HeldStepID = flow.stop.ID })

	err := publishAfter(t, mgr, draftID, func() {
		approve.HeldStepID = flow.stop.ID
		updated, err := mgr.UpdateWorkItem(ctx, approve)
		if err != nil {
			t.Fatalf("UpdateWorkItem (approval): %v", err)
		}
		approve = updated
	})
	if err != nil {
		t.Errorf("another releasing target must be a change that can be published, got %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, draftID, "", ""); !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("nothing changed, got %v", err)
	}
}

func olderStepsDocument() models.AgencyDocument {
	steps := []models.DocumentWorkItem{
		{ID: "s1", WorkflowID: "w1", Title: "Write the charter", Ordinality: 1},
		{ID: "s2", WorkflowID: "w1", Title: "Review the charter", Ordinality: 2},
		{ID: "s3", WorkflowID: "w1", Title: "Sign the charter", Ordinality: 3},
		{ID: "s4", WorkflowID: "w1", Title: "File the charter", Ordinality: 4},
		{ID: "s5", WorkflowID: "w1", Title: "Collect the form", Ordinality: 5, Kind: models.StepKindFillForm, RoleID: "r1", FormName: "Intake",
			Questions: []models.StepQuestion{{Prompt: "Name", AnswerType: models.AnswerFreeText, Required: true}}},
		{ID: "s6", WorkflowID: "w1", Title: "Approve the form", Ordinality: 6, Kind: models.StepKindYesNo, RoleID: "r1", FormName: "Approval",
			Questions: []models.StepQuestion{{Prompt: "Approved", AnswerType: models.AnswerYesNo, Required: true}}},
	}
	return lakesideDocument(steps, nil)
}

func lakesideDocument(steps []models.DocumentWorkItem, records []models.DocumentRecord) models.AgencyDocument {
	return models.AgencyDocument{
		Agency:             models.DocumentAgency{Name: "Lakeside"},
		Goals:              []models.DocumentGoal{{ID: "g1", Title: "Lend well"}},
		Objectives:         []models.DocumentObjective{{ID: "o1", GoalID: "g1", Title: "Lend well"}},
		Workflows:          []models.DocumentWorkflow{{ID: "w1", Name: "Loan payment"}},
		WorkflowObjectives: []models.DocumentWorkflowObjective{{ID: "wo1", WorkflowID: "w1", ObjectiveID: "o1"}},
		Roles:              []models.DocumentRole{{ID: "r1", Name: "Treasurer"}},
		WorkItems:          steps,
		Records:            records,
	}
}

func hasNoStepFields(wi models.WorkItem) bool {
	return wi.Function == "" && len(wi.Bindings) == 0 && wi.ResultName == "" && wi.OnFailure == nil &&
		wi.AmountStepID == "" && wi.Account == "" && wi.CounterAccount == "" && wi.Direction == "" &&
		wi.Reason == "" && wi.Hold == nil && wi.HeldStepID == ""
}

func TestStepCopy_AnAgencyWrittenBeforeTheFieldsImportsAsSixStepsOfItsOldKinds(t *testing.T) {
	useStepCopyRules(t)
	mgr := newTestManager(t)
	ctx := context.Background()
	out, err := mgr.ImportDocument(ctx, "older", olderStepsDocument())
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	steps, err := mgr.ListWorkItems(ctx, out.Draft.ID, mwanachamaagency.WorkItemFilter{})
	if err != nil {
		t.Fatalf("ListWorkItems: %v", err)
	}
	if len(steps) != 6 {
		t.Fatalf("steps = %d, want 6", len(steps))
	}
	kinds := map[string]models.StepKind{}
	for _, wi := range steps {
		kinds[wi.Title] = wi.Kind
		if !hasNoStepFields(wi) {
			t.Errorf("nothing is invented for %q: %+v", wi.Title, wi)
		}
	}
	if kinds["Collect the form"] != models.StepKindFillForm || kinds["Approve the form"] != models.StepKindYesNo || kinds["Write the charter"] != "" {
		t.Errorf("the old kinds must be kept, got %v", kinds)
	}
}

func TestStepCopy_AnImportKeepsFewBindingsNoAmountSourceAndAHoldWithNoReleasingStepAndRemapsEveryId(t *testing.T) {
	useStepCopyRules(t)
	mgr := newTestManager(t)
	ctx := context.Background()
	steps := []models.DocumentWorkItem{
		{ID: "s1", WorkflowID: "w1", Title: applyTitle, Ordinality: 1, Kind: models.StepKindFillForm, RoleID: "r1", FormName: "Loan request",
			Questions: []models.StepQuestion{{Prompt: "Amount requested", AnswerType: models.AnswerCurrency, Required: true}}},
		{ID: "s2", WorkflowID: "w1", Title: computeTitle, Ordinality: 2, Kind: models.StepKindCompute, Function: "fin.pmt@2", ResultName: "Payment to post",
			Bindings: []models.StepBinding{
				{Input: "principal", SourceType: models.SourceAnswer, StepID: "s1", Question: "Amount requested"},
				{Input: "months", SourceType: models.SourceRecordField, RecordID: "rc1", Question: "Months"},
			},
			OnFailure: &models.StepFailure{Choice: models.FailureGoToStep, StepID: "s3"},
			Hold:      &models.StepHold{ExpiresAfterDays: 14}},
		{ID: "s3", WorkflowID: "w1", Title: fallbackTitle, Ordinality: 3},
		{ID: "s4", WorkflowID: "w1", Title: approveTitle, Ordinality: 4, Kind: models.StepKindYesNo, RoleID: "r1", FormName: "Approval",
			Questions: []models.StepQuestion{{Prompt: "Approved", AnswerType: models.AnswerYesNo, Required: true}}, HeldStepID: "s2"},
		{ID: "s5", WorkflowID: "w1", Title: postTitle, Ordinality: 5, Kind: models.StepKindPost, Account: "Member loans", Direction: "credit"},
	}
	records := []models.DocumentRecord{{ID: "rc1", WorkflowID: "w1", Name: termsName, RoleID: "r1", FormName: "Terms",
		Questions: []models.StepQuestion{{Prompt: "Months", AnswerType: models.AnswerNumber, Required: true}}}}
	out, err := mgr.ImportDocument(ctx, "lakeside", lakesideDocument(steps, records))
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}

	imported := copiedSteps(t, mgr, out.Draft.ID)
	record := recordNamed(t, mgr, out.Draft.ID, termsName)
	compute := imported[computeTitle]
	if len(compute.Bindings) != 2 || compute.Function != "fin.pmt@2" || compute.ResultName != "Payment to post" {
		t.Fatalf("a calculating step with fewer bindings than inputs must import as it is: %+v", compute)
	}
	if compute.Bindings[0].StepID != imported[applyTitle].ID || compute.Bindings[0].StepID == "s1" {
		t.Errorf("the answer binding must be remapped onto the imported step, got %q", compute.Bindings[0].StepID)
	}
	if compute.Bindings[1].RecordID != record.ID || compute.Bindings[1].RecordID == "rc1" {
		t.Errorf("the record binding must be remapped onto the imported record, got %q", compute.Bindings[1].RecordID)
	}
	if compute.OnFailure == nil || compute.OnFailure.StepID != imported[fallbackTitle].ID {
		t.Errorf("the failure choice must keep its step, remapped, got %+v", compute.OnFailure)
	}
	if compute.Hold == nil || compute.Hold.ExpiresAfterDays != 14 {
		t.Errorf("hold = %+v", compute.Hold)
	}
	if imported[approveTitle].HeldStepID != compute.ID {
		t.Errorf("the releasing step must name the imported calculating step, got %q", imported[approveTitle].HeldStepID)
	}
	post := imported[postTitle]
	if post.AmountStepID != "" || post.Account != "Member loans" || post.Direction != "credit" {
		t.Errorf("a posting step with no amount source must import as it is, got %+v", post)
	}
}

func TestStepCopy_AHoldWithNoReleasingStepIsAcceptedOnImport(t *testing.T) {
	useStepCopyRules(t)
	mgr := newTestManager(t)
	steps := []models.DocumentWorkItem{
		{ID: "s1", WorkflowID: "w1", Title: computeTitle, Ordinality: 1, Kind: models.StepKindCompute, Function: "fin.pmt@2",
			Hold: &models.StepHold{ExpiresAfterDays: 14}},
	}
	out, err := mgr.ImportDocument(context.Background(), "hold", lakesideDocument(steps, nil))
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	got := copiedSteps(t, mgr, out.Draft.ID)[computeTitle]
	if !reflect.DeepEqual(got.Hold, &models.StepHold{ExpiresAfterDays: 14}) {
		t.Errorf("hold = %+v", got.Hold)
	}
}
