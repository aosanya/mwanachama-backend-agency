package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"testing"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func olderDocumentOf(t *testing.T, o routedOrder) models.AgencyDocument {
	t.Helper()
	w := o.world
	assignOn(t, w.mgr, context.Background(), w.draftID, w.roles[coordinator], o.steps["Open"])
	pub, err := w.mgr.PublishAgency(context.Background(), w.draftID, "older-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	return withoutFieldIdentityIn(pub.Document)
}

func withoutFieldIdentityIn(doc models.AgencyDocument) models.AgencyDocument {
	doc.Records = append([]models.DocumentRecord(nil), doc.Records...)
	for i := range doc.Records {
		doc.Records[i].Questions = withoutIdentity(doc.Records[i].Questions)
	}
	doc.WorkItems = append([]models.DocumentWorkItem(nil), doc.WorkItems...)
	for i := range doc.WorkItems {
		doc.WorkItems[i].Questions = withoutIdentity(doc.WorkItems[i].Questions)
	}
	return doc
}

func TestImport_AnOldFileGivesEveryFieldAKeyAndACodeProposedFromItsPrompt(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	doc := olderDocumentOf(t, o)

	first, err := w.mgr.ImportDocument(ctx, "old file", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	second, err := w.mgr.ImportDocument(ctx, "old file again", doc)
	if err != nil {
		t.Fatalf("ImportDocument again: %v", err)
	}
	a, _ := w.mgr.ListRecords(ctx, first.Draft.ID, mwanachamaagency.RecordFilter{})
	b, _ := w.mgr.ListRecords(ctx, second.Draft.ID, mwanachamaagency.RecordFilter{})
	if len(a) != 1 || len(b) != 1 || len(a[0].Questions) != 2 {
		t.Fatalf("imported records = %+v and %+v", a, b)
	}
	for i, q := range a[0].Questions {
		if q.Key == "" || q.Key == q.Prompt || q.Code == "" {
			t.Errorf("imported field %q has key %q and code %q", q.Prompt, q.Key, q.Code)
		}
		if b[0].Questions[i].Key != q.Key || b[0].Questions[i].Code != q.Code {
			t.Errorf("the same old file imported twice gave %+v and %+v; a field must get the same key in every import", q, b[0].Questions[i])
		}
	}
	if a[0].Questions[0].Code != "select_the_farmer" || a[0].Questions[1].Code != "enter_the_amount_paid" {
		t.Errorf("proposed codes = %q and %q", a[0].Questions[0].Code, a[0].Questions[1].Code)
	}
	if a[0].Questions[0].Key == a[0].Questions[1].Key {
		t.Errorf("two fields share the key %q", a[0].Questions[0].Key)
	}
}

func TestImport_ADocumentWithTwoFieldsOfOnePromptKeepsThemApart(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	doc := olderDocumentOf(t, o)
	doc.Records[0].Questions = []models.StepQuestion{ask("Date", models.AnswerDate), ask("Date", models.AnswerDate)}
	imported, err := w.mgr.ImportDocument(ctx, "twins", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	records, _ := w.mgr.ListRecords(ctx, imported.Draft.ID, mwanachamaagency.RecordFilter{})
	q := records[0].Questions
	if q[0].Key == q[1].Key || q[0].Code != "date" || q[1].Code != "date_2" {
		t.Errorf("two fields of one prompt = %+v", q)
	}
}

func TestImport_AFileWhoseFieldsShareACodeIsRefused(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	doc := olderDocumentOf(t, o)
	doc.Records[0].Questions[0].Code, doc.Records[0].Questions[1].Code = "same", "same"
	if _, err := w.mgr.ImportDocument(context.Background(), "clash", doc); err == nil {
		t.Errorf("a file whose two fields share a code was imported")
	}
}

type olderFieldsDatabase struct {
	db     *gorm.DB
	s      *spec.Spec
	world  stepWorld
	record models.Record
	rowID  string
}

func newOlderFieldsDatabase(t *testing.T) (olderFieldsDatabase, routedOrder) {
	t.Helper()
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	_, _, objective, _, _ := seedLiveAgency(t, mgr, ctx)
	w := stepWorld{mgr: mgr, draftID: objective.DraftID, objective: objective, roles: map[string]models.Role{}}
	for _, name := range []string{coordinator, farmer, supplier} {
		role, err := mgr.CreateRole(ctx, models.Role{DraftID: w.draftID, Name: name})
		if err != nil {
			t.Fatalf("CreateRole %s: %v", name, err)
		}
		w.roles[name] = role
	}
	workflow := w.workflow(t, "Group order")
	if _, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: w.draftID, WorkflowID: workflow.ID, Title: "Start", Ordinality: 1}); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	record, err := mgr.CreateRecord(ctx, paymentsRecord(w, workflow))
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	w.letAdd(t, w.roles[coordinator], record)
	o := routedOrder{world: w, workflow: workflow, record: record}
	row, err := mgr.CreateRecordRow(ctx, rowAdded(w, o, map[string]string{
		"Select the farmer": "Wanjiru", "Enter the amount paid": "1200", "Approved by": "amina", "Changed: Enter the amount paid": "was 90",
	}))
	if err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	older := olderFieldsDatabase{db: db, s: s, world: w, record: record, rowID: row.ID}
	older.makeOlder(t, o)
	return older, o
}

func (d olderFieldsDatabase) makeOlder(t *testing.T, o routedOrder) {
	t.Helper()
	asked, _ := json.Marshal(withoutIdentity(d.record.Questions))
	if err := d.db.Exec("update "+tableOf(t, d.s, "record")+" set questions = ? where id = ?", string(asked), d.record.ID).Error; err != nil {
		t.Fatalf("write the older record: %v", err)
	}
	if err := d.db.Exec("update "+tableOf(t, d.s, "work_item")+" set questions = ? where row_record = ?", string(asked), d.record.Code).Error; err != nil {
		t.Fatalf("write the older generated steps: %v", err)
	}
	answers, _ := json.Marshal(map[string]string{
		"Select the farmer": "Wanjiru", "Enter the amount paid": "1200", "Approved by": "amina", "Changed: Enter the amount paid": "was 90",
	})
	if err := d.db.Exec("update "+tableOf(t, d.s, "record_row")+" set answers = ? where id = ?", string(answers), d.rowID).Error; err != nil {
		t.Fatalf("write the older row: %v", err)
	}
}

func TestProvision_AnOlderDatabaseGetsKeysAndItsRowsMoveFromPromptsToKeys(t *testing.T) {
	d, o := newOlderFieldsDatabase(t)
	ctx := context.Background()
	w := d.world

	before, _ := w.mgr.GetRecord(ctx, d.record.ID)
	if before.Questions[0].Key != "" {
		t.Fatalf("the older record still has keys: %+v", before.Questions)
	}
	if err := mwanachamaagency.Provision(d.db, d.s); err != nil {
		t.Fatalf("Provision over the older database: %v", err)
	}

	after, err := w.mgr.GetRecord(ctx, d.record.ID)
	if err != nil {
		t.Fatalf("GetRecord: %v", err)
	}
	farmer, amount := fieldOf(t, after, "Select the farmer"), fieldOf(t, after, "Enter the amount paid")
	if farmer.Key == "" || amount.Key == "" || farmer.Code != "select_the_farmer" || amount.Code != "enter_the_amount_paid" {
		t.Fatalf("the migrated fields = %+v", after.Questions)
	}
	rows, err := w.mgr.ListRecordRows(ctx, w.draftID, mwanachamaagency.RecordRowFilter{RecordID: d.record.ID})
	if err != nil || len(rows) != 1 {
		t.Fatalf("ListRecordRows = %+v, %v", rows, err)
	}
	answers := rows[0].Answers
	if answers[farmer.Key] != "Wanjiru" || answers[amount.Key] != "1200" || answers["Approved by"] != "amina" || answers["Changed: Enter the amount paid"] != "was 90" || len(answers) != 4 {
		t.Errorf("the migrated row = %v, want the form answers under the keys and the stamps as they were", answers)
	}

	if err := mwanachamaagency.Provision(d.db, d.s); err != nil {
		t.Fatalf("Provision a second time: %v", err)
	}
	again, _ := w.mgr.GetRecord(ctx, d.record.ID)
	if fieldOf(t, again, "Enter the amount paid").Key != amount.Key {
		t.Errorf("a second Provision changed a key")
	}

	steps, _ := w.mgr.ListWorkItems(ctx, w.draftID, mwanachamaagency.WorkItemFilter{})
	for _, s := range steps {
		if s.RowRecord == d.record.Code && s.RowAction == models.RowActionAdd {
			if got := fieldOf(t, models.Record{Code: s.RowRecord, Questions: s.Questions}, "Enter the amount paid"); got.Key != amount.Key {
				t.Errorf("the generated add step asks %+v, want the migrated key", got)
			}
		}
	}

	next, err := w.mgr.CreateRecordRow(ctx, rowAdded(w, o, map[string]string{farmer.Key: "Otieno", amount.Key: "5"}))
	if err != nil || next.Answers[amount.Key] != "5" {
		t.Errorf("a row written by key after the migration = %+v, %v", next.Answers, err)
	}
}

func TestProvision_TheKeysAnOlderDatabaseGetsAreTheOnesAnOldFileGets(t *testing.T) {
	d, _ := newOlderFieldsDatabase(t)
	ctx := context.Background()
	w := d.world
	if err := mwanachamaagency.Provision(d.db, d.s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	migrated, _ := w.mgr.GetRecord(ctx, d.record.ID)
	pub, err := w.mgr.PublishAgency(ctx, w.draftID, "older-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	imported, err := w.mgr.ImportDocument(ctx, "old file", withoutFieldIdentityIn(pub.Document))
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	records, _ := w.mgr.ListRecords(ctx, imported.Draft.ID, mwanachamaagency.RecordFilter{})
	if len(records) != 1 || records[0].Code != migrated.Code {
		t.Fatalf("imported %+v beside the migrated record %s", records, migrated.Code)
	}
	for i, q := range migrated.Questions {
		if records[0].Questions[i].Key != q.Key || records[0].Questions[i].Code != q.Code {
			t.Errorf("the migration gave %+v and the import %+v: a row must read the same field in every version", q, records[0].Questions[i])
		}
	}
}

func TestProvision_ARowStillSavedByPromptAfterItsRecordWasKeyedIsMovedByTheNextProvision(t *testing.T) {
	d, o := newOlderFieldsDatabase(t)
	ctx := context.Background()
	w := d.world
	if err := mwanachamaagency.Provision(d.db, d.s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	record, _ := w.mgr.GetRecord(ctx, d.record.ID)
	amount := fieldOf(t, record, "Enter the amount paid")
	stray, err := w.mgr.CreateRecordRow(ctx, rowAdded(w, o, map[string]string{
		fieldOf(t, record, "Select the farmer").Key: "Otieno", amount.Key: "7",
	}))
	if err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	byPrompt, _ := json.Marshal(map[string]string{"Select the farmer": "Otieno", "Enter the amount paid": "7"})
	if err := d.db.Exec("update "+tableOf(t, d.s, "record_row")+" set answers = ? where id = ?", string(byPrompt), stray.ID).Error; err != nil {
		t.Fatalf("write the row an older process left: %v", err)
	}

	if err := mwanachamaagency.Provision(d.db, d.s); err != nil {
		t.Fatalf("Provision again: %v", err)
	}
	rows, err := w.mgr.ListRecordRows(ctx, w.draftID, mwanachamaagency.RecordRowFilter{RecordID: d.record.ID})
	if err != nil {
		t.Fatalf("ListRecordRows: %v", err)
	}
	moved := false
	for _, row := range rows {
		if row.ID == stray.ID {
			moved = row.Answers[amount.Key] == "7" && len(row.Answers) == 2
		}
	}
	if !moved {
		t.Errorf("the row an older process left by prompt was not moved to keys: %+v", rows)
	}
}
