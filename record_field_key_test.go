package mwanachamaagency_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func fieldOf(t *testing.T, record models.Record, prompt string) models.StepQuestion {
	t.Helper()
	for _, q := range record.Questions {
		if q.Prompt == prompt {
			return q
		}
	}
	t.Fatalf("the record %s has no field %q: %+v", record.Code, prompt, record.Questions)
	return models.StepQuestion{}
}

func answerTo(record models.Record, row models.RecordRow, prompt string) string {
	for _, q := range record.Questions {
		if q.Prompt == prompt {
			return row.Answers[q.Key]
		}
	}
	return ""
}

func withoutIdentity(questions []models.StepQuestion) []models.StepQuestion {
	out := append([]models.StepQuestion(nil), questions...)
	for i := range out {
		out[i].Key, out[i].Code = "", ""
	}
	return out
}

func rowAdded(w stepWorld, o routedOrder, answers map[string]string) models.RecordRow {
	return models.RecordRow{DraftID: w.draftID, RecordID: o.record.ID, RoleID: w.roles[coordinator].ID, AddedBy: "member-1", Answers: answers}
}

func TestRecordField_CreateGivesEveryFieldAReadableCodeAndAHiddenKey(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()

	farmer := fieldOf(t, o.record, "Select the farmer")
	amount := fieldOf(t, o.record, "Enter the amount paid")
	if farmer.Code != "select_the_farmer" || amount.Code != "enter_the_amount_paid" {
		t.Errorf("codes = %q and %q, want the prompts in lower case with underscores", farmer.Code, amount.Code)
	}
	if farmer.Key == "" || amount.Key == "" || farmer.Key == amount.Key {
		t.Errorf("keys = %q and %q, want two different keys", farmer.Key, amount.Key)
	}
	for _, key := range []string{farmer.Key, amount.Key} {
		for _, q := range o.record.Questions {
			if key == q.Prompt || key == q.Code {
				t.Errorf("the key %q repeats a prompt or a code", key)
			}
		}
	}

	read, err := o.world.mgr.GetRecord(ctx, o.record.ID)
	if err != nil || fieldOf(t, read, "Select the farmer").Key != farmer.Key || fieldOf(t, read, "Select the farmer").Code != farmer.Code {
		t.Errorf("GetRecord = %+v, %v; the fields keep the code and key they were given", read.Questions, err)
	}
}

func TestRecordField_ACodeNotGivenIsProposedFromThePromptAndMadeUniqueInTheRecord(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	in := paymentsRecord(w, o.workflow)
	in.Name = "Twins"
	in.Questions = []models.StepQuestion{ask("Closing date", models.AnswerDate), ask("closing date", models.AnswerDate), ask("  Amount (USD)! ", models.AnswerNumber)}
	made, err := w.mgr.CreateRecord(context.Background(), in)
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	got := []string{made.Questions[0].Code, made.Questions[1].Code, made.Questions[2].Code}
	want := []string{"closing_date", "closing_date_2", "amount_usd"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("codes = %v, want %v", got, want)
			break
		}
	}
}

func TestRecordField_TwoFieldsCannotShareACodeAndACodeHasARule(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	for name, codes := range map[string][2]string{
		"the same code twice":             {"closing_date", "closing_date"},
		"a code with a capital":           {"Closing_date", "amount"},
		"a code with a space":             {"closing date", "amount"},
		"a code that starts with a digit": {"2nd_date", "amount"},
		"a code with a dash":              {"closing-date", "amount"},
	} {
		in := paymentsRecord(w, o.workflow)
		in.Name = "Coded " + name
		in.Questions = []models.StepQuestion{ask("Closing date", models.AnswerDate), ask("Amount", models.AnswerNumber)}
		in.Questions[0].Code, in.Questions[1].Code = codes[0], codes[1]
		if _, err := w.mgr.CreateRecord(ctx, in); !errors.Is(err, mwanachamaagency.ErrInvalidRecord) {
			t.Errorf("%s: want ErrInvalidRecord, got %v", name, err)
		}
	}
	in := paymentsRecord(w, o.workflow)
	in.Name = "Coded fine"
	in.Questions = []models.StepQuestion{ask("Closing date", models.AnswerDate), ask("Amount", models.AnswerNumber)}
	in.Questions[0].Code = "closing_date"
	made, err := w.mgr.CreateRecord(ctx, in)
	if err != nil || made.Questions[0].Code != "closing_date" || made.Questions[1].Code != "amount" {
		t.Errorf("a code given is kept and the other is proposed: %+v, %v", made.Questions, err)
	}
}

func TestRecordField_AnUpdateKeepsTheKeysAndRefusesAKeyTheRecordNeverHad(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	before := fieldOf(t, o.record, "Enter the amount paid")

	same := o.record
	same.Name = "Payments made"
	same.Questions = []models.StepQuestion{ask("Select the farmer", models.AnswerSingleChoice), ask("Enter the amount paid", models.AnswerCurrency)}
	same.Questions[0].OptionsFrom = "The farmers in the order"
	updated, err := w.mgr.UpdateRecord(ctx, same)
	if err != nil {
		t.Fatalf("UpdateRecord with no keys sent: %v", err)
	}
	if got := fieldOf(t, updated, "Enter the amount paid"); got.Key != before.Key || got.Code != before.Code {
		t.Errorf("a field sent again by its prompt lost its identity: %+v, was %+v", got, before)
	}

	stranger := updated
	stranger.Questions = append([]models.StepQuestion(nil), updated.Questions...)
	stranger.Questions[1].Key = "key-the-record-never-had"
	if _, err := w.mgr.UpdateRecord(ctx, stranger); !errors.Is(err, mwanachamaagency.ErrInvalidRecord) {
		t.Errorf("an update naming a key the record never had: want ErrInvalidRecord, got %v", err)
	}

	added := updated
	added.Questions = append(append([]models.StepQuestion(nil), updated.Questions...), ask("Receipt number", models.AnswerFreeText))
	grown, err := w.mgr.UpdateRecord(ctx, added)
	if err != nil {
		t.Fatalf("UpdateRecord adding a field: %v", err)
	}
	receipt := fieldOf(t, grown, "Receipt number")
	if receipt.Key == "" || receipt.Code != "receipt_number" || receipt.Key == before.Key {
		t.Errorf("the new field = %+v", receipt)
	}
}

func TestRecordRow_IsSavedUnderTheFieldKeyNotThePromptText(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	farmerKey := fieldOf(t, o.record, "Select the farmer").Key
	amountKey := fieldOf(t, o.record, "Enter the amount paid").Key

	made, err := w.mgr.CreateRecordRow(ctx, rowAdded(w, o, map[string]string{farmerKey: "Wanjiru", amountKey: "1200"}))
	if err != nil {
		t.Fatalf("a row written by key: %v", err)
	}
	if made.Answers[farmerKey] != "Wanjiru" || made.Answers[amountKey] != "1200" || len(made.Answers) != 2 {
		t.Errorf("the saved row = %v, want the answers under the keys", made.Answers)
	}
	rows, err := w.mgr.ListRecordRows(ctx, w.draftID, mwanachamaagency.RecordRowFilter{RecordID: o.record.ID})
	if err != nil || len(rows) != 1 || rows[0].Answers[amountKey] != "1200" {
		t.Fatalf("the row read back = %+v, %v", rows, err)
	}
	if _, kept := rows[0].Answers["Enter the amount paid"]; kept {
		t.Errorf("the row still holds an answer under the prompt text: %v", rows[0].Answers)
	}
}

func TestRecordRow_AFieldMayBeNamedByItsCodeOrItsPromptAndIsStillSavedByKey(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	farmer := fieldOf(t, o.record, "Select the farmer")
	amount := fieldOf(t, o.record, "Enter the amount paid")

	byCode, err := w.mgr.CreateRecordRow(ctx, rowAdded(w, o, map[string]string{farmer.Code: "Wanjiru", amount.Code: "10"}))
	if err != nil || byCode.Answers[farmer.Key] != "Wanjiru" || byCode.Answers[amount.Key] != "10" || len(byCode.Answers) != 2 {
		t.Errorf("a row written by code = %+v, %v", byCode.Answers, err)
	}
	byPrompt, err := w.mgr.CreateRecordRow(ctx, rowAdded(w, o, map[string]string{"Select the farmer": "Otieno", "Enter the amount paid": "20"}))
	if err != nil || byPrompt.Answers[farmer.Key] != "Otieno" || byPrompt.Answers[amount.Key] != "20" || len(byPrompt.Answers) != 2 {
		t.Errorf("a row written by prompt = %+v, %v", byPrompt.Answers, err)
	}
}

func TestRecordRow_NamingOneFieldTwiceOrAFieldThatIsNotThereIsRefused(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	farmer := fieldOf(t, o.record, "Select the farmer")
	amount := fieldOf(t, o.record, "Enter the amount paid")

	for name, answers := range map[string]map[string]string{
		"one field by key and by code": {farmer.Key: "Wanjiru", farmer.Code: "Otieno", amount.Key: "1"},
		"a field that is not there":    {farmer.Key: "Wanjiru", amount.Key: "1", "no_such_field": "x"},
	} {
		if _, err := w.mgr.CreateRecordRow(ctx, rowAdded(w, o, answers)); !errors.Is(err, mwanachamaagency.ErrInvalidRecordRow) {
			t.Errorf("%s: want ErrInvalidRecordRow, got %v", name, err)
		}
	}
}

func TestRecordField_RenamingACodeKeepsTheSavedAnswersAndTheKey(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	farmer := fieldOf(t, o.record, "Select the farmer")
	amount := fieldOf(t, o.record, "Enter the amount paid")
	if _, err := w.mgr.CreateRecordRow(ctx, rowAdded(w, o, map[string]string{farmer.Key: "Wanjiru", amount.Key: "1200"})); err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}

	renamed := o.record
	renamed.Questions = append([]models.StepQuestion(nil), o.record.Questions...)
	renamed.Questions[1].Code = "amount_paid"
	updated, err := w.mgr.UpdateRecord(ctx, renamed)
	if err != nil {
		t.Fatalf("UpdateRecord renaming a code: %v", err)
	}
	if got := fieldOf(t, updated, "Enter the amount paid"); got.Code != "amount_paid" || got.Key != amount.Key {
		t.Errorf("after the rename the field = %+v, want the new code on the old key %q", got, amount.Key)
	}

	rows, err := w.mgr.ListRecordRows(ctx, w.draftID, mwanachamaagency.RecordRowFilter{RecordID: o.record.ID})
	if err != nil || len(rows) != 1 || rows[0].Answers[amount.Key] != "1200" || rows[0].Answers[farmer.Key] != "Wanjiru" {
		t.Fatalf("the saved row after the rename = %+v, %v", rows, err)
	}

	byNewCode, err := w.mgr.CreateRecordRow(ctx, rowAdded(w, o, map[string]string{farmer.Key: "Otieno", "amount_paid": "5"}))
	if err != nil || byNewCode.Answers[amount.Key] != "5" {
		t.Errorf("a row written by the new code = %+v, %v", byNewCode.Answers, err)
	}
	if _, err := w.mgr.CreateRecordRow(ctx, rowAdded(w, o, map[string]string{farmer.Key: "Otieno", amount.Code: "6"})); !errors.Is(err, mwanachamaagency.ErrInvalidRecordRow) {
		t.Errorf("a row written by the old code: want ErrInvalidRecordRow, got %v", err)
	}
}

func TestRecordField_ARenamedPromptWithItsKeyIsTheSameField(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	amount := fieldOf(t, o.record, "Enter the amount paid")
	renamed := o.record
	renamed.Questions = append([]models.StepQuestion(nil), o.record.Questions...)
	renamed.Questions[1].Prompt = "How much was paid"
	updated, err := w.mgr.UpdateRecord(ctx, renamed)
	if err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}
	if got := fieldOf(t, updated, "How much was paid"); got.Key != amount.Key || got.Code != amount.Code {
		t.Errorf("the reworded field = %+v, want key %q and code %q kept", got, amount.Key, amount.Code)
	}
}

func TestRecordField_AForkAndAPublicationCarryTheKeysAndCodes(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	amount := fieldOf(t, o.record, "Enter the amount paid")

	fork, err := w.mgr.CreateDraft(ctx, "fork", w.draftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	forked, _ := w.mgr.ListRecords(ctx, fork.ID, mwanachamaagency.RecordFilter{})
	if len(forked) != 1 || fieldOf(t, forked[0], "Enter the amount paid").Key != amount.Key {
		t.Errorf("the fork's fields = %+v, want the key %q", forked, amount.Key)
	}

	pub, err := w.mgr.PublishAgency(ctx, w.draftID, "release-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if len(pub.Document.Records) != 1 || pub.Document.Records[0].Questions[1].Key != amount.Key || pub.Document.Records[0].Questions[1].Code != amount.Code {
		t.Errorf("the publication's fields = %+v", pub.Document.Records)
	}
	back, err := w.mgr.CreateDraft(ctx, "revisit", "release-1", "publication")
	if err != nil {
		t.Fatalf("CreateDraft from the publication: %v", err)
	}
	restored, _ := w.mgr.ListRecords(ctx, back.ID, mwanachamaagency.RecordFilter{})
	if len(restored) != 1 || fieldOf(t, restored[0], "Enter the amount paid").Key != amount.Key {
		t.Errorf("the restored fields = %+v, want the key %q", restored, amount.Key)
	}
}

func TestRecordField_RenamingACodeIsAChangeToPublishAndKeysAloneAreNot(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	if _, err := w.mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		DraftID: w.draftID, WorkItemID: o.steps["Open"].ID, RoleID: w.roles[coordinator].ID, Label: models.RACILabelResponsible,
	}); err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "release-1", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	again := o.record
	again.Questions = withoutIdentity(o.record.Questions)
	if _, err := w.mgr.UpdateRecord(ctx, again); err != nil {
		t.Fatalf("UpdateRecord with the same form: %v", err)
	}
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "release-2", ""); !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("the same fields sent again: want ErrNoChangesDetected, got %v", err)
	}

	renamed := o.record
	renamed.Questions = append([]models.StepQuestion(nil), o.record.Questions...)
	renamed.Questions[1].Code = "amount_paid"
	if _, err := w.mgr.UpdateRecord(ctx, renamed); err != nil {
		t.Fatalf("UpdateRecord renaming a code: %v", err)
	}
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "release-2", ""); err != nil {
		t.Errorf("a renamed code is a change to publish: %v", err)
	}
}

func TestRecordField_TheGeneratedStepsFollowACodeRename(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	renamed := o.record
	renamed.Questions = append([]models.StepQuestion(nil), o.record.Questions...)
	renamed.Questions[1].Code = "amount_paid"
	if _, err := w.mgr.UpdateRecord(ctx, renamed); err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}
	steps, err := w.mgr.ListWorkItems(ctx, w.draftID, mwanachamaagency.WorkItemFilter{})
	if err != nil {
		t.Fatalf("ListWorkItems: %v", err)
	}
	seen := 0
	for _, s := range steps {
		if s.RowRecord != o.record.Code || s.RowAction == models.RowActionDelete {
			continue
		}
		seen++
		if got := fieldOf(t, models.Record{Code: s.RowRecord, Questions: s.Questions}, "Enter the amount paid"); got.Code != "amount_paid" {
			t.Errorf("the %s step asks %+v, want the renamed code", s.RowAction, got)
		}
	}
	if seen != 2 {
		t.Errorf("saw %d generated steps that write the record's form, want 2", seen)
	}
	issues, err := w.mgr.ListIssues(ctx, w.draftID, mwanachamaagency.IssueFilter{EntityType: "record"})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	for _, issue := range issues {
		if issue.Severity == "error" && strings.Contains(issue.Message, "generated") {
			t.Errorf("the scan reports %+v after a rename", issue)
		}
	}
}

func TestRecordField_ADesignedRowStepWithTheFormAndNoKeysStillFits(t *testing.T) {
	r := buildRowStepWorld(t)
	step := r.step(t, "Stand-in add", models.RowActionAdd)
	step.Questions = withoutIdentity(r.record.Questions)
	if _, err := r.world.mgr.CreateWorkItem(context.Background(), step); err != nil {
		t.Errorf("a row step whose form is the record's, with no keys: %v", err)
	}
}
