package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func recordCalled(w stepWorld, workflow models.Workflow, name string) models.Record {
	in := paymentsRecord(w, workflow)
	in.Name = name
	return in
}

func roleCarrying(t *testing.T, w stepWorld, name string, permissions ...string) models.Role {
	t.Helper()
	role, err := w.mgr.CreateRole(context.Background(), models.Role{DraftID: w.draftID, Name: name, Permissions: permissions})
	if err != nil {
		t.Fatalf("CreateRole %s: %v", name, err)
	}
	return role
}

func permissionsOf(t *testing.T, w stepWorld, roleID string) []string {
	t.Helper()
	role, err := w.mgr.GetRole(context.Background(), roleID)
	if err != nil {
		t.Fatalf("GetRole: %v", err)
	}
	return role.Permissions
}

func TestRecordCode_ANewRecordGetsACodeMadeFromItsNameAndAHiddenKey(t *testing.T) {
	o := buildRoutedOrder(t)
	if o.record.Code != "payments" {
		t.Errorf("code = %q, want payments, the name in lower case", o.record.Code)
	}
	if o.record.Key == "" || o.record.Key == o.record.Code {
		t.Errorf("key = %q, want a hidden key apart from the code %q", o.record.Key, o.record.Code)
	}
	if strings.HasPrefix(o.record.Code, "RC-") {
		t.Errorf("code = %q, the counter code is deprecated", o.record.Code)
	}
	read, err := o.world.mgr.GetRecord(context.Background(), o.record.ID)
	if err != nil || read.Code != o.record.Code || read.Key != o.record.Key {
		t.Errorf("GetRecord = %+v, %v; the record keeps the code and key it was given", read, err)
	}
}

func TestRecordCode_TwoRecordsWithTheSameNameInOneDraftGetDifferentCodes(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	other := w.workflow(t, "Second order")
	second, err := w.mgr.CreateRecord(context.Background(), recordCalled(w, other, "Payments"))
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	if second.Code != "payments_2" || second.Key == o.record.Key {
		t.Errorf("second code and key = %q, %q; first = %q, %q", second.Code, second.Key, o.record.Code, o.record.Key)
	}
}

func TestRecordCode_ACodeGivenIsKeptAndANameWithoutLettersFallsBack(t *testing.T) {
	w := newStepWorld(t)
	wf := w.workflow(t, "Orders")
	ctx := context.Background()
	in := recordCalled(w, wf, "Tender register")
	in.Code = "tender"
	made, err := w.mgr.CreateRecord(ctx, in)
	if err != nil || made.Code != "tender" {
		t.Fatalf("CreateRecord with a code = %+v, %v", made, err)
	}
	blank, err := w.mgr.CreateRecord(ctx, recordCalled(w, wf, "2024"))
	if err != nil || blank.Code != "record_2024" {
		t.Errorf("a name that starts with a digit gave %+v, %v; want the code record_2024", blank, err)
	}
}

func TestRecordCode_ADuplicateOrMalformedCodeIsRefused(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	other := w.workflow(t, "Second order")
	for name, code := range map[string]string{
		"a code another record of the draft has": o.record.Code,
		"a capital":                              "Tender",
		"a space":                                "tender register",
		"a leading digit":                        "2tender",
		"a dash":                                 "tender-register",
		"a dot":                                  "tender.register",
		"a counter code":                         "RC-9",
		"a field key":                            "f_3b5f1a39c58f1114",
	} {
		in := recordCalled(w, other, "Tenders")
		in.Code = code
		if _, err := w.mgr.CreateRecord(ctx, in); !errors.Is(err, mwanachamaagency.ErrInvalidRecord) {
			t.Errorf("create with %s (%q): err = %v, want ErrInvalidRecord", name, code, err)
		}
	}
	second, err := w.mgr.CreateRecord(ctx, recordCalled(w, other, "Tenders"))
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	for name, code := range map[string]string{"a code another record has": o.record.Code, "a capital": "Tenders", "a counter code": "RC-9"} {
		second.Code = code
		if _, err := w.mgr.UpdateRecord(ctx, second); !errors.Is(err, mwanachamaagency.ErrInvalidRecord) {
			t.Errorf("update with %s (%q): err = %v, want ErrInvalidRecord", name, code, err)
		}
	}
	read, _ := w.mgr.GetRecord(ctx, second.ID)
	if read.Code != "tenders" {
		t.Errorf("a refused update changed the code to %q", read.Code)
	}
}

func TestRecordCode_AnUpdateThatLeavesTheCodeOutKeepsIt(t *testing.T) {
	o := buildRoutedOrder(t)
	in := o.record
	in.Code = ""
	in.FormName = "Payment made"
	updated, err := o.world.mgr.UpdateRecord(context.Background(), in)
	if err != nil || updated.Code != o.record.Code || updated.Key != o.record.Key {
		t.Errorf("UpdateRecord = %+v, %v; want code %q and key %q kept", updated, err, o.record.Code, o.record.Key)
	}
}

func TestRecordCode_RecordPermissionsReadTheReadableCode(t *testing.T) {
	if got := mwanachamaagency.RecordPermissionCode("tender", "edit"); got != "record.tender.edit" {
		t.Errorf("RecordPermissionCode = %q", got)
	}
	o := buildRoutedOrder(t)
	role := roleCarrying(t, o.world, "Clerk", "record.payments.add", "record.payments.edit")
	if got := permissionsOf(t, o.world, role.ID); !reflect.DeepEqual(got, []string{"record.payments.add", "record.payments.edit"}) {
		t.Errorf("permissions = %v", got)
	}
	_, err := o.world.mgr.CreateRole(context.Background(), models.Role{
		DraftID: o.world.draftID, Name: "Old clerk", Permissions: []string{"record." + o.record.Key + ".add"},
	})
	if !errors.Is(err, mwanachamaagency.ErrUnknownPermission) {
		t.Errorf("a role carrying the counter code %s: err = %v, want ErrUnknownPermission", o.record.Key, err)
	}
}

func TestRecordCode_RenamingTheCodeKeepsTheKeyAndRewritesTheRolesInTheSameSave(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	other := w.workflow(t, "Second order")
	sibling, err := w.mgr.CreateRecord(ctx, recordCalled(w, other, "Payments extra"))
	if err != nil || sibling.Code != "payments_extra" {
		t.Fatalf("CreateRecord = %+v, %v", sibling, err)
	}
	clerk := roleCarrying(t, w, "Clerk", "record.payments.add", "record.payments.edit", "report.read")
	auditor := roleCarrying(t, w, "Auditor", "record.payments.view", "record.payments_extra.view")

	rowBefore, err := w.mgr.CreateRecordRow(ctx, rowAdded(w, o, map[string]string{"Select the farmer": "Wanjiru", "Enter the amount paid": "10"}))
	if err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}

	renamed := o.record
	renamed.Code = "payment_register"
	got, err := w.mgr.UpdateRecord(ctx, renamed)
	if err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}
	if got.Code != "payment_register" || got.Key != o.record.Key {
		t.Errorf("renamed record = code %q key %q, want payment_register and %q", got.Code, got.Key, o.record.Key)
	}
	if got := permissionsOf(t, w, clerk.ID); !reflect.DeepEqual(got, []string{"record.payment_register.add", "record.payment_register.edit", "report.read"}) {
		t.Errorf("Clerk = %v, want the permissions moved to the new code", got)
	}
	if got := permissionsOf(t, w, auditor.ID); !reflect.DeepEqual(got, []string{"record.payment_register.view", "record.payments_extra.view"}) {
		t.Errorf("Auditor = %v, want only the permission of the renamed record moved", got)
	}

	rows, err := w.mgr.ListRecordRows(ctx, w.draftID, mwanachamaagency.RecordRowFilter{RecordCode: "payment_register"})
	if err != nil || len(rows) != 1 || rows[0].ID != rowBefore.ID || rows[0].RecordCode != o.record.Key {
		t.Errorf("rows read by the new code = %+v, %v; want the one row, still quoting %q", rows, err, o.record.Key)
	}
	byKey, err := w.mgr.ListRecordRows(ctx, w.draftID, mwanachamaagency.RecordRowFilter{RecordCode: o.record.Key})
	if err != nil || len(byKey) != 1 {
		t.Errorf("rows read by the key = %+v, %v", byKey, err)
	}

	workflows, _ := w.mgr.ListWorkflows(ctx, w.draftID, mwanachamaagency.WorkflowFilter{})
	generated := 0
	for _, wf := range workflows {
		if wf.Generated && wf.GeneratedFor == o.record.Key {
			generated++
		}
	}
	if generated != 3 {
		t.Errorf("%d generated workflows still made for the key %s, want 3: a rename must not remake them", generated, o.record.Key)
	}
	if _, err := w.mgr.CreateRole(ctx, models.Role{DraftID: w.draftID, Name: "Late", Permissions: []string{"record.payments.add"}}); !errors.Is(err, mwanachamaagency.ErrUnknownPermission) {
		t.Errorf("the old code still names the record: err = %v", err)
	}
}

func TestRecordCode_ARenameThatIsRefusedLeavesTheRolesAlone(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	clerk := roleCarrying(t, w, "Clerk", "record.payments.add")
	bad := o.record
	bad.Code = "Not A Code"
	if _, err := w.mgr.UpdateRecord(context.Background(), bad); !errors.Is(err, mwanachamaagency.ErrInvalidRecord) {
		t.Fatalf("UpdateRecord: %v", err)
	}
	if got := permissionsOf(t, w, clerk.ID); !reflect.DeepEqual(got, []string{"record.payments.add"}) {
		t.Errorf("Clerk = %v after a refused rename", got)
	}
}

func TestRecordCode_AStepAWorkflowSubjectAndAHomeEntryNameTheRecordByItsKey(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	if _, err := w.mgr.CreateWorkflow(ctx, models.Workflow{DraftID: w.draftID, Name: "Pay", SubjectRecord: o.record.Key}); err != nil {
		t.Errorf("a workflow whose subject is the record's key: %v", err)
	}
	renamed := o.record
	renamed.Code = "payment_register"
	if _, err := w.mgr.UpdateRecord(ctx, renamed); err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}
	if _, err := w.mgr.CreateWorkflow(ctx, models.Workflow{DraftID: w.draftID, Name: "Pay again", SubjectRecord: o.record.Key}); err != nil {
		t.Errorf("a subject that names the key still resolves after a rename: %v", err)
	}
	issues, err := w.mgr.ListIssues(ctx, w.draftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	for _, i := range issues {
		if strings.Contains(i.Message, "does not exist") {
			t.Errorf("the scan reports %q after a rename", i.Message)
		}
	}
}

func olderRecordCodesDocument(t *testing.T, o routedOrder) models.AgencyDocument {
	t.Helper()
	w := o.world
	roleCarrying(t, w, "Clerk", "record.payments.add", "record.payments.edit")
	pub, err := w.mgr.PublishAgency(context.Background(), w.draftID, "older-codes", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	return oldRecordIdentityIn(pub.Document, o.record)
}

func oldRecordIdentityIn(doc models.AgencyDocument, record models.Record) models.AgencyDocument {
	doc.Records = append([]models.DocumentRecord(nil), doc.Records...)
	for i := range doc.Records {
		doc.Records[i].Key, doc.Records[i].Code = "", record.Key
	}
	doc.Roles = append([]models.DocumentRole(nil), doc.Roles...)
	for i := range doc.Roles {
		var kept []string
		for _, p := range doc.Roles[i].Permissions {
			kept = append(kept, strings.Replace(p, "record.payments.", "record."+record.Key+".", 1))
		}
		doc.Roles[i].Permissions = kept
	}
	return doc
}

func TestImport_AnOldFileGivesEveryRecordAReadableCodeAndKeepsTheCounterCodeAsItsKey(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	doc := olderRecordCodesDocument(t, o)
	if !strings.HasPrefix(doc.Records[0].Code, "RC-") {
		t.Fatalf("the older file's record code = %q", doc.Records[0].Code)
	}

	first, err := w.mgr.ImportDocument(ctx, "old codes", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	second, err := w.mgr.ImportDocument(ctx, "old codes again", doc)
	if err != nil {
		t.Fatalf("ImportDocument again: %v", err)
	}
	a, _ := w.mgr.ListRecords(ctx, first.Draft.ID, mwanachamaagency.RecordFilter{})
	b, _ := w.mgr.ListRecords(ctx, second.Draft.ID, mwanachamaagency.RecordFilter{})
	if len(a) != 1 || len(b) != 1 {
		t.Fatalf("imported records = %+v and %+v", a, b)
	}
	if a[0].Code != "payments" || a[0].Key != o.record.Key {
		t.Errorf("imported record = code %q key %q, want payments and %q", a[0].Code, a[0].Key, o.record.Key)
	}
	if b[0].Code != a[0].Code || b[0].Key != a[0].Key {
		t.Errorf("the same file imported twice gave %q/%q and %q/%q", a[0].Code, a[0].Key, b[0].Code, b[0].Key)
	}
	roles, _ := w.mgr.ListRoles(ctx, first.Draft.ID, mwanachamaagency.RoleFilter{Name: "Clerk"})
	if len(roles) != 1 || !reflect.DeepEqual(roles[0].Permissions, []string{"record.payments.add", "record.payments.edit"}) {
		t.Errorf("the role of the old file = %+v, want its permissions on the readable code", roles)
	}
	steps, _ := w.mgr.ListWorkItems(ctx, first.Draft.ID, mwanachamaagency.WorkItemFilter{})
	for _, s := range steps {
		if s.RowRecord != "" && s.RowRecord != a[0].Key {
			t.Errorf("the step %s names the record %q, want the key %q", s.Code, s.RowRecord, a[0].Key)
		}
	}
}

func TestImport_AFileWhoseRecordsShareACodeIsRefusedAndNothingIsKept(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	other := w.workflow(t, "Second order")
	if _, err := w.mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: w.draftID, WorkflowID: other.ID, Title: "Start", Ordinality: 1}); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	second, err := w.mgr.CreateRecord(ctx, recordCalled(w, other, "Tenders"))
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	pub, err := w.mgr.PublishAgency(ctx, w.draftID, "two-records", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	doc := pub.Document
	for i := range doc.Records {
		if doc.Records[i].ID == second.ID {
			doc.Records[i].Code = o.record.Code
		}
	}
	if _, err := w.mgr.ImportDocument(ctx, "clash", doc); !errors.Is(err, mwanachamaagency.ErrInvalidRecord) {
		t.Errorf("a file whose two records share a code: err = %v, want ErrInvalidRecord", err)
	}
}

func TestImport_ANewRecordAfterAFileWithCounterKeysGetsAKeyThatIsFree(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	doc := olderRecordCodesDocument(t, o)
	doc.Records[0].Code = "RC-40"
	for i := range doc.WorkItems {
		if doc.WorkItems[i].RowRecord != "" {
			doc.WorkItems[i].RowRecord = "RC-40"
		}
	}
	for i := range doc.Workflows {
		if doc.Workflows[i].GeneratedFor != "" {
			doc.Workflows[i].GeneratedFor = "RC-40"
		}
	}
	imported, err := w.mgr.ImportDocument(ctx, "high counter", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	wf := w.workflow(t, "Later")
	made, err := w.mgr.CreateRecord(ctx, models.Record{
		DraftID: imported.Draft.ID, WorkflowID: wf.ID, Name: "Later", RoleID: w.roles[coordinator].ID, FormName: "Later",
		Questions: []models.StepQuestion{ask("When", models.AnswerDate)},
	})
	if err == nil && made.Key == "RC-40" {
		t.Errorf("a new record was given the key RC-40, which the imported record already holds")
	}
}

func TestFork_APublicationMadeBeforeReadableCodesForksIntoRecordsWithReadableCodes(t *testing.T) {
	d := newRecordCodesDatabase(t)
	w := d.world
	ctx := context.Background()
	pub, err := w.mgr.PublishAgency(ctx, w.draftID, "before-codes", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	older := oldRecordIdentityIn(pub.Document, d.record)
	raw, err := json.Marshal(older)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := d.db.Exec("update "+tableOf(t, d.s, "agency_publication")+" set document = ? where tag = ?", string(raw), pub.Tag).Error; err != nil {
		t.Fatalf("write the older publication: %v", err)
	}
	fork, err := w.mgr.CreateDraft(ctx, "fork of the older publication", pub.Tag, "publication")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	records, _ := w.mgr.ListRecords(ctx, fork.ID, mwanachamaagency.RecordFilter{})
	if len(records) != 1 || records[0].Code != "payments" || records[0].Key != d.record.Key {
		t.Fatalf("forked records = %+v, want payments with the key %s", records, d.record.Key)
	}
	roles, _ := w.mgr.ListRoles(ctx, fork.ID, mwanachamaagency.RoleFilter{Name: "Clerk"})
	if len(roles) != 1 || !reflect.DeepEqual(roles[0].Permissions, []string{"record.payments.add", "record.payments.view", "report.read"}) {
		t.Errorf("the forked role = %+v", roles)
	}
}

func TestFork_ADraftForkKeepsTheCodeAndTheKeyOfEveryRecord(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	renamed := o.record
	renamed.Code = "payment_register"
	if _, err := w.mgr.UpdateRecord(ctx, renamed); err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}
	pub, err := w.mgr.PublishAgency(ctx, w.draftID, "renamed", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	fork, err := w.mgr.CreateDraft(ctx, "next", pub.Tag, "publication")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	records, _ := w.mgr.ListRecords(ctx, fork.ID, mwanachamaagency.RecordFilter{})
	if len(records) != 1 || records[0].Code != "payment_register" || records[0].Key != o.record.Key {
		t.Errorf("forked records = %+v", records)
	}
}

func TestPublish_ARenamedRecordCodeIsAChangeToPublish(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	ctx := context.Background()
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "first", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	renamed := o.record
	renamed.Code = "payment_register"
	if _, err := w.mgr.UpdateRecord(ctx, renamed); err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "second", ""); err != nil {
		t.Errorf("PublishAgency after the rename: %v", err)
	}
}

type olderRecordCodesDatabase struct {
	db     *gorm.DB
	s      *spec.Spec
	world  stepWorld
	record models.Record
	clerk  models.Role
	rowID  string
}

func newOlderRecordCodesDatabase(t *testing.T) olderRecordCodesDatabase {
	t.Helper()
	d := newRecordCodesDatabase(t)
	d.makeOlder(t)
	return d
}

func newRecordCodesDatabase(t *testing.T) olderRecordCodesDatabase {
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
	clerk := roleCarrying(t, w, "Clerk", "record.payments.add", "record.payments.view", "report.read")
	o := routedOrder{world: w, workflow: workflow, record: record}
	row, err := mgr.CreateRecordRow(ctx, rowAdded(w, o, map[string]string{"Select the farmer": "Wanjiru", "Enter the amount paid": "1200"}))
	if err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	return olderRecordCodesDatabase{db: db, s: s, world: w, record: record, clerk: clerk, rowID: row.ID}
}

func (d olderRecordCodesDatabase) makeOlder(t *testing.T) {
	t.Helper()
	if err := d.db.Exec("update "+tableOf(t, d.s, "record")+" set code = ?, key = '' where id = ?", d.record.Key, d.record.ID).Error; err != nil {
		t.Fatalf("write the older record: %v", err)
	}
	older, _ := json.Marshal([]string{"record." + d.record.Key + ".add", "record." + d.record.Key + ".view", "report.read"})
	if err := d.db.Exec("update "+tableOf(t, d.s, "role")+" set permissions = ? where id = ?", string(older), d.clerk.ID).Error; err != nil {
		t.Fatalf("write the older role: %v", err)
	}
}

func TestProvision_AnOlderDatabaseGivesItsRecordsAReadableCodeAndMovesTheRolesWithoutChangingTheRows(t *testing.T) {
	d := newOlderRecordCodesDatabase(t)
	ctx := context.Background()
	w := d.world
	rowsBefore, err := w.mgr.ListRecordRows(ctx, w.draftID, mwanachamaagency.RecordRowFilter{})
	if err != nil || len(rowsBefore) != 1 {
		t.Fatalf("ListRecordRows = %+v, %v", rowsBefore, err)
	}

	if err := mwanachamaagency.Provision(d.db, d.s); err != nil {
		t.Fatalf("Provision over the older database: %v", err)
	}
	after, err := w.mgr.GetRecord(ctx, d.record.ID)
	if err != nil {
		t.Fatalf("GetRecord: %v", err)
	}
	if after.Code != "payments" || after.Key != d.record.Key {
		t.Errorf("migrated record = code %q key %q, want payments and %q", after.Code, after.Key, d.record.Key)
	}
	if got := permissionsOf(t, w, d.clerk.ID); !reflect.DeepEqual(got, []string{"record.payments.add", "record.payments.view", "report.read"}) {
		t.Errorf("Clerk = %v, want the permissions on the readable code", got)
	}
	rowsAfter, err := w.mgr.ListRecordRows(ctx, w.draftID, mwanachamaagency.RecordRowFilter{})
	if err != nil || !reflect.DeepEqual(rowsBefore, rowsAfter) {
		t.Errorf("rows changed: %+v, then %+v, %v", rowsBefore, rowsAfter, err)
	}
	byCode, err := w.mgr.ListRecordRows(ctx, w.draftID, mwanachamaagency.RecordRowFilter{RecordCode: "payments"})
	if err != nil || len(byCode) != 1 {
		t.Errorf("rows read by the readable code = %+v, %v", byCode, err)
	}
	issues, err := w.mgr.ListIssues(ctx, w.draftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	for _, i := range issues {
		if i.Severity == models.IssueSeverityError {
			t.Errorf("the migrated draft has the error %q", i.Message)
		}
	}

	if err := mwanachamaagency.Provision(d.db, d.s); err != nil {
		t.Fatalf("Provision a second time: %v", err)
	}
	again, _ := w.mgr.GetRecord(ctx, d.record.ID)
	if again.Code != after.Code || again.Key != after.Key || again.LastUpdated != after.LastUpdated {
		t.Errorf("a second Provision changed the record: %+v then %+v", after, again)
	}
}

func TestProvision_TwoOlderRecordsWithTheSameNameGetDifferentCodes(t *testing.T) {
	d := newOlderRecordCodesDatabase(t)
	ctx := context.Background()
	w := d.world
	other := w.workflow(t, "Second order")
	second, err := w.mgr.CreateRecord(ctx, recordCalled(w, other, "Payments"))
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	if err := d.db.Exec("update "+tableOf(t, d.s, "record")+" set code = ?, key = '' where id = ?", second.Key, second.ID).Error; err != nil {
		t.Fatalf("write the second older record: %v", err)
	}
	if err := mwanachamaagency.Provision(d.db, d.s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	first, _ := w.mgr.GetRecord(ctx, d.record.ID)
	again, _ := w.mgr.GetRecord(ctx, second.ID)
	if first.Code != "payments" || again.Code != "payments_2" {
		t.Errorf("codes = %q and %q, want payments and payments_2 in the order the records were made", first.Code, again.Code)
	}
}

func TestProvision_AnOlderPublicationStillForksAfterTheDatabaseWasMigrated(t *testing.T) {
	d := newOlderRecordCodesDatabase(t)
	ctx := context.Background()
	w := d.world
	if err := mwanachamaagency.Provision(d.db, d.s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	pub, err := w.mgr.PublishAgency(ctx, w.draftID, "after-migration", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	fork, err := w.mgr.CreateDraft(ctx, "fork", pub.Tag, "publication")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	records, _ := w.mgr.ListRecords(ctx, fork.ID, mwanachamaagency.RecordFilter{})
	if len(records) != 1 || records[0].Code != "payments" || records[0].Key != d.record.Key {
		t.Errorf("forked records = %+v", records)
	}
}
