package mwanachamaagency_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type fakeFunctions map[string][]mwanachamaagency.FunctionInput

func (f fakeFunctions) InputsOf(_ context.Context, name string, version int) ([]mwanachamaagency.FunctionInput, bool, error) {
	inputs, ok := f[fmt.Sprintf("%s@%d", name, version)]
	return inputs, ok, nil
}

func lakesideFunctions() fakeFunctions {
	return fakeFunctions{
		"fin.pmt@2": {
			{Name: "principal", Type: "currency"},
			{Name: "monthly_rate", Type: "number"},
			{Name: "months", Type: "number"},
		},
		"fin.pmt@1": {
			{Name: "principal", Type: "currency"},
			{Name: "monthly_rate", Type: "number"},
			{Name: "months", Type: "number"},
		},
	}
}

type computeWorld struct {
	stepWorld
	wf        models.Workflow
	other     models.Workflow
	greet     models.WorkItem
	apply     models.WorkItem
	confirm   models.WorkItem
	elsewhere models.WorkItem
	record    models.Record
	treasurer models.Role
}

func newComputeWorld(t *testing.T, opts ...mwanachamaagency.Option) computeWorld {
	t.Helper()
	mwanachamaagency.ClearStepKindRegistry()
	t.Cleanup(mwanachamaagency.ClearStepKindRegistry)
	mwanachamaagency.RegisterComputeStepChecks()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := mwanachamaagency.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	mgr, err := mwanachamaagency.NewAgencyManager(db, s, opts...)
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	w := stepWorld{mgr: mgr, draftID: o.DraftID, objective: o, roles: map[string]models.Role{}}
	r, err := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: coordinator})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	w.roles[coordinator] = r
	treasurer, err := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "Treasurer"})
	if err != nil {
		t.Fatalf("CreateRole Treasurer: %v", err)
	}
	c := computeWorld{stepWorld: w, treasurer: treasurer}
	c.wf = w.workflow(t, "Loan request")
	c.other = w.workflow(t, "Savings")
	c.greet = c.mustCreate(t, models.WorkItem{DraftID: w.draftID, WorkflowID: c.wf.ID, Title: "Greet the member", Ordinality: 1})
	c.apply = c.mustCreate(t, w.item(c.wf, 2, stepSpec{
		title: "Apply for a loan", kind: models.StepKindFillForm, role: coordinator, form: "Loan request",
		questions: []models.StepQuestion{
			ask("Amount requested", models.AnswerCurrency),
			ask("Reason for the loan", models.AnswerFreeText),
			ask("Photo of the id", models.AnswerPhoto),
		},
	}))
	c.confirm = c.mustCreate(t, w.item(c.wf, 5, stepSpec{
		title: "Confirm the loan", kind: models.StepKindFillForm, role: coordinator, form: "Loan confirmation",
		questions: []models.StepQuestion{ask("Confirmed on", models.AnswerDate)},
	}))
	c.elsewhere = c.mustCreate(t, w.item(c.other, 1, stepSpec{
		title: "Open an account", kind: models.StepKindFillForm, role: coordinator, form: "Account request",
		questions: []models.StepQuestion{ask("Amount requested", models.AnswerCurrency)},
	}))
	record, err := mgr.CreateRecord(ctx, models.Record{
		DraftID: w.draftID, WorkflowID: c.wf.ID, Name: "Loan terms", FormName: "Terms",
		Questions: []models.StepQuestion{ask("Months", models.AnswerNumber)},
	})
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	c.record = record
	return c
}

func newLakesideWorld(t *testing.T) computeWorld {
	t.Helper()
	return newComputeWorld(t, mwanachamaagency.WithFunctionCatalog(lakesideFunctions()))
}

func (c computeWorld) mustCreate(t *testing.T, wi models.WorkItem) models.WorkItem {
	t.Helper()
	created, err := c.mgr.CreateWorkItem(context.Background(), wi)
	if err != nil {
		t.Fatalf("CreateWorkItem %s: %v", wi.Title, err)
	}
	return created
}

func (c computeWorld) exampleBindings() []models.StepBinding {
	return []models.StepBinding{
		{Input: "principal", SourceType: models.SourceAnswer, StepID: c.apply.ID, Question: "Amount requested"},
		{Input: "monthly_rate", SourceType: models.SourceFixed, Value: "0.01"},
		{Input: "months", SourceType: models.SourceRecordField, RecordID: c.record.ID, Question: "Months"},
	}
}

func (c computeWorld) example(mutate ...func(*models.WorkItem)) models.WorkItem {
	wi := models.WorkItem{
		DraftID: c.draftID, WorkflowID: c.wf.ID, Title: "Work out the monthly payment", Ordinality: 4,
		Kind: models.StepKindCompute, Function: "fin.pmt@2", ResultName: "Monthly payment",
		Bindings:  c.exampleBindings(),
		OnFailure: &models.StepFailure{Choice: models.FailureStop},
	}
	for _, m := range mutate {
		m(&wi)
	}
	return wi
}

func (c computeWorld) stored(t *testing.T, id string) models.WorkItem {
	t.Helper()
	got, err := c.mgr.GetWorkItem(context.Background(), id)
	if err != nil {
		t.Fatalf("GetWorkItem: %v", err)
	}
	return got
}

func (c computeWorld) update(t *testing.T, id string, mutate func(*models.WorkItem)) models.WorkItem {
	t.Helper()
	wi := c.stored(t, id)
	mutate(&wi)
	out, err := c.mgr.UpdateWorkItem(context.Background(), wi)
	if err != nil {
		t.Fatalf("UpdateWorkItem %s: %v", wi.Title, err)
	}
	return out
}

func (c computeWorld) findings(t *testing.T, step models.WorkItem) []string {
	t.Helper()
	issues, err := c.mgr.ListIssues(context.Background(), c.draftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	var out []string
	for _, issue := range issues {
		if issue.EntityType == "work_item" && issue.EntityID == step.ID && issue.Severity == models.IssueSeverityError {
			out = append(out, issue.Message)
		}
	}
	return out
}

func wantRefused(t *testing.T, err error, sentinel error, sentence string) {
	t.Helper()
	if err == nil {
		t.Fatalf("saved, want refused with %q", sentence)
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want %v", err, sentinel)
	}
	if !strings.Contains(err.Error(), sentence) {
		t.Fatalf("error = %q, want the sentence %q", err.Error(), sentence)
	}
}

func wantOneFinding(t *testing.T, got []string, stepTitle, sentence string) {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("findings = %q, want exactly one", got)
	}
	if !strings.Contains(got[0], fmt.Sprintf("%q", stepTitle)) || !strings.Contains(got[0], sentence) {
		t.Fatalf("finding = %q, want the step %q and the sentence %q", got[0], stepTitle, sentence)
	}
}

func TestComputeStep_AnotherKindOfStepCarriesNoComputeField(t *testing.T) {
	c := newLakesideWorld(t)
	const sentence = "takes no function, bindings or result name"
	formStep := func(title string, kind models.StepKind) models.WorkItem {
		return models.WorkItem{
			DraftID: c.draftID, WorkflowID: c.wf.ID, Title: title, Ordinality: 6, Kind: kind,
			RoleID: c.roles[coordinator].ID, FormName: "Form",
			Questions: []models.StepQuestion{ask("Question", models.AnswerFreeText)},
		}
	}
	fill := formStep("Fill", models.StepKindFillForm)
	fill.Function = "fin.pmt@2"
	notify := formStep("Tell", models.StepKindNotify)
	notify.ResultName = "Monthly payment"
	agree := formStep("Agree", models.StepKindYesNo)
	agree.Bindings = []models.StepBinding{{Input: "months", SourceType: models.SourceFixed, Value: "12"}}
	nothing := models.WorkItem{DraftID: c.draftID, WorkflowID: c.wf.ID, Title: "Nothing", Ordinality: 6, Function: "fin.pmt@2"}
	cases := []struct {
		id   string
		name string
		step models.WorkItem
	}{
		{"CS-1", "fill_form with a function", fill},
		{"CS-2", "notify with a result name", notify},
		{"CS-3", "yes_no with a binding", agree},
		{"CS-3b", "a step with no kind and a function", nothing},
	}
	for _, tc := range cases {
		t.Run(tc.id+" "+tc.name, func(t *testing.T) {
			_, err := c.mgr.CreateWorkItem(context.Background(), tc.step)
			wantRefused(t, err, mwanachamaagency.ErrInvalidWorkItem, sentence)
		})
	}
}

func TestComputeStep_ACalculatingStepTakesNoFormRecipientRoutesOrTool(t *testing.T) {
	c := newLakesideWorld(t)
	const sentence = "takes no form, recipient, sent step, routes, workflow to run, chat fields or tool"
	cases := []struct {
		id     string
		name   string
		mutate func(*models.WorkItem)
	}{
		{"CS-4", "a form", func(wi *models.WorkItem) {
			wi.FormName = "Loan request"
			wi.Questions = []models.StepQuestion{ask("Amount", models.AnswerCurrency)}
		}},
		{"CS-5", "a recipient", func(wi *models.WorkItem) { wi.RecipientRoleID = c.roles[coordinator].ID }},
		{"CS-6", "a sent step", func(wi *models.WorkItem) { wi.SentStepID = c.apply.ID }},
		{"CS-7", "routes", func(wi *models.WorkItem) {
			wi.Routes = []models.StepRoute{{Match: models.RouteMatchAll, GoToStep: c.confirm.ID, Rules: []models.RouteRule{{Question: "Amount requested", Operator: models.RouteIs, Value: "1"}}}}
		}},
		{"CS-8", "a workflow to run", func(wi *models.WorkItem) { wi.RunWorkflowID = c.other.ID }},
		{"CS-9", "chat fields", func(wi *models.WorkItem) { wi.Topic = "Loan questions" }},
		{"CS-10", "a tool", func(wi *models.WorkItem) {
			wi.RoleID = c.roles[coordinator].ID
			wi.Tool = "market_ask"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.id+" "+tc.name, func(t *testing.T) {
			_, err := c.mgr.CreateWorkItem(context.Background(), c.example(tc.mutate))
			wantRefused(t, err, mwanachamaagency.ErrInvalidWorkItem, sentence)
		})
	}
}

func TestComputeStep_TheFunctionAndTheResultNameAreRequiredAndShaped(t *testing.T) {
	c := newLakesideWorld(t)
	cases := []struct {
		id       string
		name     string
		mutate   func(*models.WorkItem)
		sentence string
	}{
		{"CS-11", "no function", func(wi *models.WorkItem) { wi.Function = "" }, "names the function it calls"},
		{"CS-12", "no version", func(wi *models.WorkItem) { wi.Function = "fin.pmt" }, `the function "fin.pmt" has no version`},
		{"CS-13", "a fractional version", func(wi *models.WorkItem) { wi.Function = "fin.pmt@2.5" }, `the version of the function "fin.pmt@2.5" is not a whole number`},
		{"CS-14", "a named version", func(wi *models.WorkItem) { wi.Function = "fin.pmt@latest" }, `the version of the function "fin.pmt@latest" is not a whole number`},
		{"CS-15", "no result name", func(wi *models.WorkItem) { wi.ResultName = "" }, "names the result it keeps"},
		{"CS-15b", "a blank result name", func(wi *models.WorkItem) { wi.ResultName = "   " }, "names the result it keeps"},
	}
	for _, tc := range cases {
		t.Run(tc.id+" "+tc.name, func(t *testing.T) {
			_, err := c.mgr.CreateWorkItem(context.Background(), c.example(tc.mutate))
			wantRefused(t, err, mwanachamaagency.ErrInvalidWorkItem, tc.sentence)
		})
	}
}

func TestComputeStep_AResultNameIsUniqueInItsWorkflowIgnoringCase(t *testing.T) {
	c := newLakesideWorld(t)
	c.mustCreate(t, c.example())
	_, err := c.mgr.CreateWorkItem(context.Background(), c.example(func(wi *models.WorkItem) {
		wi.Title = "Work out it again"
		wi.Ordinality = 6
		wi.ResultName = "monthly payment"
		wi.Bindings = nil
	}))
	wantRefused(t, err, mwanachamaagency.ErrInvalidWorkItem, `the result name "monthly payment" is already used by another compute step of this workflow`)

	if _, err := c.mgr.CreateWorkItem(context.Background(), models.WorkItem{
		DraftID: c.draftID, WorkflowID: c.other.ID, Title: "Work out the payment", Ordinality: 2,
		Kind: models.StepKindCompute, Function: "fin.pmt@2", ResultName: "Monthly payment",
	}); err != nil {
		t.Errorf("the same result name in another workflow: %v, want it accepted", err)
	}
}

func TestComputeStep_AStepHasOneConfirmingRole(t *testing.T) {
	c := newLakesideWorld(t)
	_, err := c.mgr.CreateWorkItem(context.Background(), c.example(func(wi *models.WorkItem) {
		wi.RoleID = c.treasurer.ID + "," + c.roles[coordinator].ID
	}))
	wantRefused(t, err, mwanachamaagency.ErrInvalidWorkItem, "a step has one role")
}

func TestComputeStep_ARoleThatIsNotOfferedIsNotFound(t *testing.T) {
	c := newLakesideWorld(t)
	_, err := c.mgr.CreateWorkItem(context.Background(), c.example(func(wi *models.WorkItem) { wi.RoleID = "role-missing" }))
	if !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Fatalf("error = %v, want ErrRoleNotFound", err)
	}
}

func TestComputeStep_BindingsAreRefusedWhenTheyCannotWork(t *testing.T) {
	c := newLakesideWorld(t)
	bind := func(b ...models.StepBinding) func(*models.WorkItem) {
		return func(wi *models.WorkItem) { wi.Bindings = b }
	}
	answer := func(input, stepID, question string) models.StepBinding {
		return models.StepBinding{Input: input, SourceType: models.SourceAnswer, StepID: stepID, Question: question}
	}
	fixed := func(input, value string) models.StepBinding {
		return models.StepBinding{Input: input, SourceType: models.SourceFixed, Value: value}
	}
	cases := []struct {
		id       string
		name     string
		mutate   func(*models.WorkItem)
		sentence string
	}{
		{"CS-18", "an input the function does not have", bind(fixed("term_days", "30")), `the function fin.pmt@2 has no input "term_days"`},
		{"CS-19", "two bindings for one input", bind(fixed("monthly_rate", "0.01"), fixed("monthly_rate", "0.02")), `input "monthly_rate" has two bindings`},
		{"CS-20", "a source that is none of the four", bind(models.StepBinding{Input: "months", SourceType: "formula"}), `is none of answer, record_field, ledger_balance and fixed`},
		{"CS-21", "an answer of a step of another workflow", bind(answer("principal", c.elsewhere.ID, "Amount requested")), `the step "Open an account", which belongs to another workflow`},
		{"CS-23", "an answer of a later step", bind(answer("principal", c.confirm.ID, "Confirmed on")), `the step "Confirm the loan", which does not come before this step`},
		{"CS-24", "an answer of a step with no form", bind(answer("principal", c.greet.ID, "Amount requested")), `the step "Greet the member", which has no form to read an answer from`},
		{"CS-25", "an answer to a question the form does not ask", bind(answer("principal", c.apply.ID, "Amount asked")), `the step "Apply for a loan", which does not ask "Amount asked"`},
		{"CS-26", "a record question the record does not ask", bind(models.StepBinding{Input: "months", SourceType: models.SourceRecordField, RecordID: c.record.ID, Question: "Weeks"}), `the record "Loan terms", which does not ask "Weeks"`},
		{"CS-27", "a fixed value with no value", bind(fixed("monthly_rate", "")), `the fixed value for input "monthly_rate" is empty`},
		{"CS-28", "a fixed value of the wrong type", bind(fixed("monthly_rate", "one percent")), `the fixed value "one percent" for input "monthly_rate" is not a number`},
		{"CS-29", "a text source for a currency input", bind(answer("principal", c.apply.ID, "Reason for the loan")), `the question "Reason for the loan" is text, but input "principal" asks for currency`},
		{"CS-30", "a photo source", bind(answer("principal", c.apply.ID, "Photo of the id")), `the question "Photo of the id" is a photo question, which cannot be a source`},
		{"CS-30b", "an answer that names no step", bind(models.StepBinding{Input: "principal", SourceType: models.SourceAnswer, Question: "Amount requested"}), `the answer for input "principal" names no step`},
		{"CS-30c", "an answer that names no question", bind(models.StepBinding{Input: "principal", SourceType: models.SourceAnswer, StepID: c.apply.ID}), `the answer for input "principal" names no question`},
		{"CS-30d", "a record field that names no record", bind(models.StepBinding{Input: "months", SourceType: models.SourceRecordField, Question: "Months"}), `the record field for input "months" names no record`},
		{"CS-30e", "a binding that names no input", bind(models.StepBinding{SourceType: models.SourceFixed, Value: "1"}), "binding 1 names no input"},
		{"CS-30f", "a ledger balance for a number input", bind(models.StepBinding{Input: "months", SourceType: models.SourceLedgerBalance}), `the ledger balance is currency, but input "months" asks for number`},
	}
	for _, tc := range cases {
		t.Run(tc.id+" "+tc.name, func(t *testing.T) {
			_, err := c.mgr.CreateWorkItem(context.Background(), c.example(tc.mutate))
			wantRefused(t, err, mwanachamaagency.ErrInvalidWorkItem, tc.sentence)
		})
	}
}

func TestComputeStep_AnAnswerOfTheStepItselfIsRefused(t *testing.T) {
	c := newLakesideWorld(t)
	step := c.mustCreate(t, c.example())
	step.Bindings = []models.StepBinding{{Input: "principal", SourceType: models.SourceAnswer, StepID: step.ID, Question: "Amount requested"}}
	_, err := c.mgr.UpdateWorkItem(context.Background(), step)
	wantRefused(t, err, mwanachamaagency.ErrInvalidWorkItem, `the answer for input "principal" comes from the step itself`)
}

func TestComputeStep_ARecordOrStepOrFunctionNotInTheDraftIsNotFound(t *testing.T) {
	c := newLakesideWorld(t)
	_, err := c.mgr.CreateWorkItem(context.Background(), c.example(func(wi *models.WorkItem) {
		wi.Bindings = []models.StepBinding{{Input: "principal", SourceType: models.SourceAnswer, StepID: "wi-missing", Question: "Amount requested"}}
	}))
	if !errors.Is(err, mwanachamaagency.ErrWorkItemNotFound) {
		t.Errorf("CS-33 an answer step not in the draft: %v, want ErrWorkItemNotFound", err)
	}
	_, err = c.mgr.CreateWorkItem(context.Background(), c.example(func(wi *models.WorkItem) {
		wi.Bindings = []models.StepBinding{{Input: "months", SourceType: models.SourceRecordField, RecordID: "rc-missing", Question: "Months"}}
	}))
	if !errors.Is(err, mwanachamaagency.ErrRecordNotFound) {
		t.Errorf("CS-34 a record not in the draft: %v, want ErrRecordNotFound", err)
	}
	for _, function := range []string{"fin.nothing@1", "fin.pmt@9"} {
		_, err = c.mgr.CreateWorkItem(context.Background(), c.example(func(wi *models.WorkItem) { wi.Function = function }))
		if !errors.Is(err, mwanachamaagency.ErrFunctionNotFound) {
			t.Errorf("CS-35 the function %s: %v, want ErrFunctionNotFound", function, err)
		}
	}
}

func TestComputeStep_AnUpdateIsRefusedAsACreateIsAndLeavesTheStepAsItWas(t *testing.T) {
	c := newLakesideWorld(t)
	step := c.mustCreate(t, c.example())
	step.Function = "fin.pmt"
	_, err := c.mgr.UpdateWorkItem(context.Background(), step)
	wantRefused(t, err, mwanachamaagency.ErrInvalidWorkItem, `the function "fin.pmt" has no version`)
	if got := c.stored(t, step.ID); got.Function != "fin.pmt@2" {
		t.Errorf("stored function = %q, want fin.pmt@2 unchanged", got.Function)
	}
}

func TestComputeStep_AStepMayBeSavedWithFewerBindingsThanTheFunctionHasInputs(t *testing.T) {
	c := newLakesideWorld(t)
	step := c.mustCreate(t, c.example(func(wi *models.WorkItem) { wi.Bindings = wi.Bindings[:1] }))
	if got := c.stored(t, step.ID); len(got.Bindings) != 1 {
		t.Errorf("stored bindings = %d, want 1 of 3", len(got.Bindings))
	}
}

func TestComputeStep_TheExampleStepReadsBackUnchanged(t *testing.T) {
	c := newLakesideWorld(t)
	want := c.example()
	step := c.mustCreate(t, want)
	got := c.stored(t, step.ID)
	if got.Function != want.Function || got.ResultName != want.ResultName || got.RoleID != "" {
		t.Errorf("stored = %q %q role %q, want %q %q and no role", got.Function, got.ResultName, got.RoleID, want.Function, want.ResultName)
	}
	if !reflect.DeepEqual(got.Bindings, want.Bindings) {
		t.Errorf("stored bindings = %+v, want %+v", got.Bindings, want.Bindings)
	}
}

func TestComputeStep_ARoleOrAnOfferedGroupMayConfirm(t *testing.T) {
	group := newComputeWorld(t, mwanachamaagency.WithFunctionCatalog(lakesideFunctions()), mwanachamaagency.WithGroupLister(groupsOffered(farmersGroupID)))
	for _, role := range []string{group.treasurer.ID, farmersGroupID} {
		wi := group.example(func(wi *models.WorkItem) {
			wi.RoleID = role
			wi.ResultName = "Payment for " + role
		})
		if _, err := group.mgr.CreateWorkItem(context.Background(), wi); err != nil {
			t.Errorf("CS-38 confirming role %s: %v, want it accepted", role, err)
		}
	}
}

func TestComputeStep_AStepOfAnotherKindReadsBackWithNoComputeField(t *testing.T) {
	c := newLakesideWorld(t)
	got := c.stored(t, c.apply.ID)
	if got.Function != "" || len(got.Bindings) != 0 || got.ResultName != "" {
		t.Errorf("a fill_form step carries %q %v %q", got.Function, got.Bindings, got.ResultName)
	}
}

func TestComputeScan_TheExampleStepIsClean(t *testing.T) {
	c := newLakesideWorld(t)
	step := c.mustCreate(t, c.example())
	if got := c.findings(t, step); len(got) != 0 {
		t.Errorf("CS-60 findings = %q, want none", got)
	}
}

func TestComputeScan_AnInputWithNoSourceIsOneFindingForEachInput(t *testing.T) {
	c := newLakesideWorld(t)
	step := c.mustCreate(t, c.example())

	c.update(t, step.ID, func(wi *models.WorkItem) { wi.Bindings = wi.Bindings[:2] })
	wantOneFinding(t, c.findings(t, step), "Work out the monthly payment", `has no source for input "months"`)

	c.update(t, step.ID, func(wi *models.WorkItem) { wi.Bindings = nil })
	got := c.findings(t, step)
	if len(got) != 3 {
		t.Fatalf("CS-42 findings = %q, want three, one for each input", got)
	}
	for i, input := range []string{"principal", "monthly_rate", "months"} {
		if !strings.Contains(strings.Join(got, "\n"), fmt.Sprintf("has no source for input %q", input)) {
			t.Errorf("no finding for input %d %s in %q", i, input, got)
		}
	}
}

func TestComputeScan_WithoutACatalogTheInputsAreUnknownAndNothingIsFlaggedForThem(t *testing.T) {
	c := newComputeWorld(t)
	step := c.mustCreate(t, c.example(func(wi *models.WorkItem) { wi.Bindings = wi.Bindings[:1] }))
	if got := c.findings(t, step); len(got) != 0 {
		t.Errorf("findings = %q, want none without a catalog", got)
	}
}

func TestComputeScan_ASourceThatNoLongerFitsTheInputIsOneFinding(t *testing.T) {
	c := newLakesideWorld(t)
	step := c.mustCreate(t, c.example())
	c.update(t, c.apply.ID, func(wi *models.WorkItem) {
		wi.Questions[0].AnswerType = models.AnswerNumber
	})
	wantOneFinding(t, c.findings(t, step), "Work out the monthly payment",
		`reads input "principal", which asks for currency, from the question "Amount requested", which is number`)
}

func TestComputeScan_ADeletedOrRewordedQuestionIsOneFinding(t *testing.T) {
	deleted := newLakesideWorld(t)
	step := deleted.mustCreate(t, deleted.example())
	deleted.update(t, deleted.apply.ID, func(wi *models.WorkItem) { wi.Questions = wi.Questions[1:] })
	const gone = `reads input "principal" from the question "Amount requested", which the step "Apply for a loan" does not ask any more`
	wantOneFinding(t, deleted.findings(t, step), "Work out the monthly payment", gone)

	reworded := newLakesideWorld(t)
	step = reworded.mustCreate(t, reworded.example())
	reworded.update(t, reworded.apply.ID, func(wi *models.WorkItem) { wi.Questions[0].Prompt = "Amount you want to borrow" })
	wantOneFinding(t, reworded.findings(t, step), "Work out the monthly payment", gone)
	reworded.update(t, step.ID, func(wi *models.WorkItem) {
		wi.Bindings[0].Question = "Amount you want to borrow"
	})
	if got := reworded.findings(t, step); len(got) != 0 {
		t.Errorf("CS-57 findings after picking the question again = %q, want none", got)
	}
}

func TestComputeScan_ARewordedRecordQuestionIsOneFinding(t *testing.T) {
	c := newLakesideWorld(t)
	step := c.mustCreate(t, c.example())
	rec := c.record
	rec.Questions = []models.StepQuestion{ask("Number of months", models.AnswerNumber)}
	if _, err := c.mgr.UpdateRecord(context.Background(), rec); err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}
	wantOneFinding(t, c.findings(t, step), "Work out the monthly payment",
		`reads input "months" from the question "Months", which the record "Loan terms" does not ask any more`)
}

func TestComputeScan_ADeletedStepOrRecordIsOneFindingAndTheBindingStays(t *testing.T) {
	c := newLakesideWorld(t)
	step := c.mustCreate(t, c.example())
	if err := c.mgr.DeleteWorkItem(context.Background(), c.apply.ID); err != nil {
		t.Fatalf("DeleteWorkItem: %v", err)
	}
	wantOneFinding(t, c.findings(t, step), "Work out the monthly payment", `reads input "principal" from a step that does not exist`)
	if got := c.stored(t, step.ID); len(got.Bindings) != 3 {
		t.Errorf("CS-47 stored bindings = %d, want the 3 still stored: a delete does not cascade", len(got.Bindings))
	}

	r := newLakesideWorld(t)
	step = r.mustCreate(t, r.example())
	if err := r.mgr.DeleteRecord(context.Background(), r.record.ID); err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}
	wantOneFinding(t, r.findings(t, step), "Work out the monthly payment", `reads input "months" from a record that does not exist`)
}

func TestComputeScan_AMovedSourceStepIsOneFinding(t *testing.T) {
	away := newLakesideWorld(t)
	step := away.mustCreate(t, away.example())
	away.update(t, away.apply.ID, func(wi *models.WorkItem) { wi.WorkflowID = away.other.ID })
	wantOneFinding(t, away.findings(t, step), "Work out the monthly payment",
		`reads input "principal" from the step "Apply for a loan", which belongs to another workflow`)

	later := newLakesideWorld(t)
	step = later.mustCreate(t, later.example())
	later.update(t, later.apply.ID, func(wi *models.WorkItem) { wi.Ordinality = 6 })
	wantOneFinding(t, later.findings(t, step), "Work out the monthly payment",
		`reads input "principal" from the step "Apply for a loan", which no longer comes before it`)
}

func TestComputeScan_AConfirmingRoleThatIsGoneIsOneFindingAndAnOfferedGroupIsNot(t *testing.T) {
	c := newLakesideWorld(t)
	step := c.mustCreate(t, c.example(func(wi *models.WorkItem) { wi.RoleID = c.treasurer.ID }))
	if err := c.mgr.DeleteRole(context.Background(), c.treasurer.ID); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}
	wantOneFinding(t, c.findings(t, step), "Work out the monthly payment", "is acted on by Role")

	g := newComputeWorld(t, mwanachamaagency.WithFunctionCatalog(lakesideFunctions()), mwanachamaagency.WithGroupLister(groupsOffered(farmersGroupID)))
	step = g.mustCreate(t, g.example(func(wi *models.WorkItem) { wi.RoleID = farmersGroupID }))
	if got := g.findings(t, step); len(got) != 0 {
		t.Errorf("CS-52 findings for an offered group = %q, want none", got)
	}
}

func TestComputeScan_AStepWithNoFunctionOrNoResultNameIsFlagged(t *testing.T) {
	c := newLakesideWorld(t)
	bare, err := mwanachamaagency.PlantStep(c.mgr, models.WorkItem{
		DraftID: c.draftID, WorkflowID: c.wf.ID, Title: "Bare step", Ordinality: 6, Kind: models.StepKindCompute,
		OnFailure: &models.StepFailure{Choice: models.FailureStop},
	})
	if err != nil {
		t.Fatalf("PlantStep: %v", err)
	}
	got := c.findings(t, bare)
	if len(got) != 2 {
		t.Fatalf("CS-53 findings = %q, want two", got)
	}
	joined := strings.Join(got, "\n")
	for _, sentence := range []string{"is a compute step that names no function", "is a compute step that names no result name"} {
		if !strings.Contains(joined, sentence) {
			t.Errorf("no finding %q in %q", sentence, got)
		}
	}

	named, err := mwanachamaagency.PlantStep(c.mgr, models.WorkItem{
		DraftID: c.draftID, WorkflowID: c.wf.ID, Title: "Unnamed result", Ordinality: 7, Kind: models.StepKindCompute,
		Function: "fin.pmt@2", Bindings: c.exampleBindings(), OnFailure: &models.StepFailure{Choice: models.FailureStop},
	})
	if err != nil {
		t.Fatalf("PlantStep: %v", err)
	}
	wantOneFinding(t, c.findings(t, named), "Unnamed result", "is a compute step that names no result name")
}

func TestComputeScan_TwoFindingsAreBothListedAndPublishingIsRefusedOnce(t *testing.T) {
	c := newLakesideWorld(t)
	step := c.mustCreate(t, c.example())
	c.update(t, step.ID, func(wi *models.WorkItem) { wi.Bindings = wi.Bindings[:2] })
	c.update(t, c.apply.ID, func(wi *models.WorkItem) { wi.Questions[0].Prompt = "Amount you want to borrow" })
	if got := c.findings(t, step); len(got) != 2 {
		t.Fatalf("CS-56 findings = %q, want two", got)
	}
	_, err := c.mgr.PublishAgency(context.Background(), c.draftID, "", "")
	if !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) {
		t.Errorf("publish = %v, want ErrPublishValidationFailed", err)
	}
}

func TestComputeScan_AnAgencyWithNoComputeStepRaisesNothingNew(t *testing.T) {
	c := newLakesideWorld(t)
	issues, err := c.mgr.ListIssues(context.Background(), c.draftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	for _, issue := range issues {
		if strings.Contains(issue.Message, "compute step") || strings.Contains(issue.Message, "reads input") || strings.Contains(issue.Message, "has no source") {
			t.Errorf("CS-58 a finding about compute in an agency with none: %s", issue.Message)
		}
	}
}

func TestComputeScan_AStepWithNoRoleRaisesNoFindingAboutAConfirmingRole(t *testing.T) {
	c := newLakesideWorld(t)
	step := c.mustCreate(t, c.example())
	if got := c.findings(t, step); len(got) != 0 {
		t.Errorf("CS-59 findings = %q, want none", got)
	}
}
