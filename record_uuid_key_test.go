package mwanachamaagency_test

import (
	"context"
	"regexp"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

var uuidShape = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func generatedFor(t *testing.T, w stepWorld, key string) []models.Workflow {
	t.Helper()
	workflows, err := w.mgr.ListWorkflows(context.Background(), w.draftID, mwanachamaagency.WorkflowFilter{})
	if err != nil {
		t.Fatalf("ListWorkflows: %v", err)
	}
	var out []models.Workflow
	for _, wf := range workflows {
		if wf.Generated && wf.GeneratedFor == key {
			out = append(out, wf)
		}
	}
	return out
}

func TestRecordKey_ANewRecordGetsAUuidKeyAndTwoRecordsNeverShareOne(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	if !uuidShape.MatchString(o.record.Key) {
		t.Errorf("key = %q, want a UUID", o.record.Key)
	}
	second, err := w.mgr.CreateRecord(ctx, recordCalled(w, w.workflow(t, "Second order"), "Tenders"))
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	if !uuidShape.MatchString(second.Key) || second.Key == o.record.Key {
		t.Errorf("second key = %q beside %q, want another UUID", second.Key, o.record.Key)
	}
	read, err := w.mgr.GetRecord(ctx, o.record.ID)
	if err != nil || read.Key != o.record.Key {
		t.Errorf("GetRecord = %+v, %v; the record keeps the key it was made with", read, err)
	}
}

func TestRecordKey_TheUuidStaysThroughUpdateRenameDraftForkAndPublicationFork(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	key := o.record.Key

	sameCode, err := w.mgr.UpdateRecord(ctx, o.record)
	if err != nil || sameCode.Key != key {
		t.Fatalf("UpdateRecord = %+v, %v; want the key %s", sameCode, err, key)
	}
	renamed := o.record
	renamed.Code, renamed.Key = "payment_register", "an-attempt-to-change-the-key"
	got, err := w.mgr.UpdateRecord(ctx, renamed)
	if err != nil || got.Key != key {
		t.Fatalf("UpdateRecord with a rename = %+v, %v; want the key %s", got, err, key)
	}

	draftFork, err := w.mgr.CreateDraft(ctx, "fork", w.draftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft from a draft: %v", err)
	}
	pub, err := w.mgr.PublishAgency(ctx, w.draftID, "uuid-key", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	pubFork, err := w.mgr.CreateDraft(ctx, "next", pub.Tag, "publication")
	if err != nil {
		t.Fatalf("CreateDraft from a publication: %v", err)
	}
	for name, draftID := range map[string]string{"draft fork": draftFork.ID, "publication fork": pubFork.ID} {
		records, err := w.mgr.ListRecords(ctx, draftID, mwanachamaagency.RecordFilter{})
		if err != nil || len(records) != 1 || records[0].Key != key || records[0].Code != "payment_register" {
			t.Errorf("%s records = %+v, %v; want payment_register with the key %s", name, records, err, key)
		}
	}
}

func TestRecordKey_AnOldRecordKeepsItsCounterKeyThroughProvisionUpdateAndFork(t *testing.T) {
	d := newRecordCodesDatabase(t)
	w := d.world
	ctx := context.Background()
	if err := d.db.Exec("update "+tableOf(t, d.s, "record")+" set key = ? where id = ?", "RC-7", d.record.ID).Error; err != nil {
		t.Fatalf("write the older key: %v", err)
	}
	if err := mwanachamaagency.Provision(d.db, d.s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	held, err := w.mgr.GetRecord(ctx, d.record.ID)
	if err != nil || held.Key != "RC-7" || held.Code != "payments" {
		t.Fatalf("GetRecord = %+v, %v; want payments keeping RC-7", held, err)
	}
	renamed := held
	renamed.Code = "payment_register"
	updated, err := w.mgr.UpdateRecord(ctx, renamed)
	if err != nil || updated.Key != "RC-7" {
		t.Fatalf("UpdateRecord = %+v, %v; want the key RC-7", updated, err)
	}
	fork, err := w.mgr.CreateDraft(ctx, "fork", w.draftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	records, _ := w.mgr.ListRecords(ctx, fork.ID, mwanachamaagency.RecordFilter{})
	if len(records) != 1 || records[0].Key != "RC-7" {
		t.Errorf("forked records = %+v, want the key RC-7", records)
	}
	later, err := w.mgr.CreateRecord(ctx, recordCalled(w, w.workflow(t, "Later"), "Later"))
	if err != nil || !uuidShape.MatchString(later.Key) {
		t.Errorf("a record made after an old one = %+v, %v; want a UUID key", later, err)
	}
}

func TestRecordKey_GeneratedWorkflowsOfANewRecordAreMadeForItsUuidAndNamedByItsCode(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	key := o.record.Key

	made := generatedFor(t, w, key)
	if len(made) != 3 {
		t.Fatalf("%d generated workflows for the key %s, want 3", len(made), key)
	}
	for _, wf := range made {
		if !strings.HasPrefix(wf.Code, "record."+o.record.Code+".") {
			t.Errorf("generated workflow code = %q, want it built from the record code %s", wf.Code, o.record.Code)
		}
	}
	steps, err := w.mgr.ListWorkItems(ctx, w.draftID, mwanachamaagency.WorkItemFilter{})
	if err != nil {
		t.Fatalf("ListWorkItems: %v", err)
	}
	addStep := 0
	for _, s := range steps {
		if s.RowRecord == key && s.RowAction == models.RowActionAdd {
			addStep++
		}
	}
	if addStep != 1 {
		t.Errorf("%d add steps name the key %s, want 1", addStep, key)
	}
	row, err := w.mgr.CreateRecordRow(ctx, rowAdded(w, o, map[string]string{"Select the farmer": "Wanjiru", "Enter the amount paid": "10"}))
	if err != nil || row.RecordCode != key {
		t.Fatalf("CreateRecordRow = %+v, %v; want the row to quote the key %s", row, err, key)
	}

	renamed := o.record
	renamed.Code = "payment_register"
	if _, err := w.mgr.UpdateRecord(ctx, renamed); err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}
	if again := generatedFor(t, w, key); len(again) != 3 {
		t.Errorf("%d generated workflows after a rename, want the same 3", len(again))
	}
	issues, err := w.mgr.ListIssues(ctx, w.draftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	for _, i := range issues {
		if i.Severity == models.IssueSeverityError || strings.Contains(i.Message, "does not exist") {
			t.Errorf("the scan reports %q", i.Message)
		}
	}
}

func TestRecordKey_AStableIdentityOfANewRecordResolvesBackToIt(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	raw, err := w.mgr.StableIdentity(ctx, "record", o.record.ID)
	if err != nil {
		t.Fatalf("StableIdentity: %v", err)
	}
	object, id, err := w.mgr.Resolve(ctx, raw)
	if err != nil || object != "record" || id != o.record.ID {
		t.Errorf("Resolve(%q) = (%q, %q, %v), want the record %s", raw, object, id, err, o.record.ID)
	}
}

func homeRecordTarget(ref string) *models.HomeTarget {
	return &models.HomeTarget{Type: models.HomeTargetRecord, Ref: ref}
}

func TestHomeEntry_ARecordNamedByCodeIsSavedAsTheKeyAndSurvivesARename(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	list, err := w.mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: w.draftID, Key: "payments-list", Title: "Payments", TargetType: models.HomeTargetRecord, Target: o.record.Code})
	if err != nil || list.Target != o.record.Key {
		t.Fatalf("CreateHomeEntry by code = %+v, %v; want the target saved as the key %s", list, err, o.record.Key)
	}
	page, err := w.mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: w.draftID, Key: "payment-page", Title: "A payment", PageOf: o.record.Code})
	if err != nil || page.PageOf != o.record.Key {
		t.Fatalf("CreateHomeEntry page by code = %+v, %v; want page_of saved as the key %s", page, err, o.record.Key)
	}

	renamed := o.record
	renamed.Code = "payment_register"
	if _, err := w.mgr.UpdateRecord(ctx, renamed); err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}
	rows := homeRowsOf(t, w.mgr, ctx, w.draftID)
	if got := homeKeyed(rows, "payments-list"); got.Target != o.record.Key {
		t.Errorf("target after a rename = %q, want the key %s", got.Target, o.record.Key)
	}
	if got := homeKeyed(rows, "payment-page"); got.PageOf != o.record.Key {
		t.Errorf("page_of after a rename = %q, want the key %s", got.PageOf, o.record.Key)
	}
	pub, err := w.mgr.PublishAgency(ctx, w.draftID, "renamed-home", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if err := models.HomePartsOf(pub.Document).CheckHome(pub.Document.Home); err != nil {
		t.Errorf("the home no longer resolves after a rename: %v", err)
	}
}

func TestHomeEntry_ARecordNamedByNameOrCodeInSetHomeIsSavedAsTheKeyAndAnUnknownOneIsKept(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	saved, err := w.mgr.SetHome(ctx, w.draftID, []map[string]any{
		{"key": "by-name", "title": "Payments", "target": map[string]any{"type": "record", "ref": "Payments"}},
		{"key": "by-code", "title": "Payments again", "target": map[string]any{"type": "record", "ref": o.record.Code}},
		{"key": "page", "title": "A payment", "page_of": o.record.Code},
		{"key": "not-yet", "title": "Later", "target": map[string]any{"type": "record", "ref": "tenders"}},
	})
	if err != nil {
		t.Fatalf("SetHome: %v", err)
	}
	want := map[string]string{"by-name": o.record.Key, "by-code": o.record.Key, "not-yet": "tenders"}
	for key, ref := range want {
		if got := homeKeyed(saved, key).Target; got != ref {
			t.Errorf("SetHome %s target = %q, want %q", key, got, ref)
		}
	}
	if got := homeKeyed(saved, "page").PageOf; got != o.record.Key {
		t.Errorf("SetHome page_of = %q, want the key %s", got, o.record.Key)
	}
}

func TestHomeEntry_ARecordNamedByCodeInAnImportedFileIsSavedAsTheKey(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	pub, err := w.mgr.PublishAgency(ctx, w.draftID, "to-import", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	doc := pub.Document
	doc.Home = []models.DocumentHomeNode{
		{Key: "payments-list", Title: "Payments", Target: homeRecordTarget(o.record.Code)},
		{Key: "payment-page", Title: "A payment", PageOf: o.record.Code},
	}
	imported, err := w.mgr.ImportDocument(ctx, "home by code", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	rows := homeRowsOf(t, w.mgr, ctx, imported.Draft.ID)
	if got := homeKeyed(rows, "payments-list").Target; got != o.record.Key {
		t.Errorf("imported target = %q, want the key %s", got, o.record.Key)
	}
	if got := homeKeyed(rows, "payment-page").PageOf; got != o.record.Key {
		t.Errorf("imported page_of = %q, want the key %s", got, o.record.Key)
	}
}
