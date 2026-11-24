package mwanachamaagency_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

const joinQuestion = "Do you want to join this order?"

type routedOrder struct {
	world    stepWorld
	workflow models.Workflow
	steps    map[string]models.WorkItem
	record   models.Record
}

func paymentsRecord(w stepWorld, workflow models.Workflow) models.Record {
	return models.Record{
		DraftID: w.draftID, WorkflowID: workflow.ID, Name: "Payments", RoleID: w.roles[coordinator].ID,
		FormName: "Payment", Questions: []models.StepQuestion{
			pickOne("Select the farmer", "The farmers in the order"),
			ask("Enter the amount paid", models.AnswerCurrency),
		},
	}
}

func routeToOpenOtherwiseThankYou(sent, open, thanks models.WorkItem) ([]models.StepRoute, string) {
	return []models.StepRoute{{
		Match:    models.RouteMatchAll,
		Rules:    []models.RouteRule{{Question: joinQuestion, Operator: models.RouteIs, Value: "yes"}},
		GoToStep: open.ID,
	}}, thanks.ID
}

func buildRoutedOrder(t *testing.T) routedOrder {
	t.Helper()
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Group order")
	steps := map[string]models.WorkItem{}
	for i, s := range groupOrderSteps {
		in := w.item(wf, i+1, s)
		switch s.title {
		case "Send the invitation":
			in.RecipientRoleID = w.roles[farmer].ID
		case "Answer the invitation":
			in.FormName, in.Questions = "", nil
			in.SentStepID = steps["Send the invitation"].ID
		}
		made, err := w.mgr.CreateWorkItem(ctx, in)
		if err != nil {
			t.Fatalf("CreateWorkItem %s: %v", s.title, err)
		}
		steps[s.title] = made
	}
	sort := steps["Sort the answers"]
	sort.SentStepID = steps["Send the invitation"].ID
	sort.Routes, sort.OtherwiseStepID = routeToOpenOtherwiseThankYou(steps["Send the invitation"], steps["Open"], steps["Thank you"])
	sorted, err := w.mgr.UpdateWorkItem(ctx, sort)
	if err != nil {
		t.Fatalf("UpdateWorkItem route: %v", err)
	}
	steps["Sort the answers"] = sorted
	record, err := w.mgr.CreateRecord(ctx, paymentsRecord(w, wf))
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	mayAdd(t, w.mgr, record)
	return routedOrder{world: w, workflow: wf, steps: steps, record: record}
}

func TestRecord_GroupOrderReadsBackWithEightStepsAndItsPaymentsRecord(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	w := o.world

	listed := w.stepsOf(t, w.draftID, o.workflow)
	if len(listed) != 8 {
		t.Fatalf("the Group order has %d steps, want 8", len(listed))
	}
	byTitle := map[string]models.WorkItem{}
	for _, s := range listed {
		byTitle[s.Title] = s
	}
	send, answer, route := byTitle["Send the invitation"], byTitle["Answer the invitation"], byTitle["Sort the answers"]
	if send.RecipientRoleID != w.roles[farmer].ID {
		t.Errorf("the invitation goes to %q, want the Farmer %q", send.RecipientRoleID, w.roles[farmer].ID)
	}
	if answer.SentStepID != send.ID || answer.Action != "Answers the invitation" || answer.FormName != "" {
		t.Errorf("the answering step = %+v", answer)
	}
	if route.SentStepID != send.ID || route.Action != "Routes the answers" {
		t.Errorf("the route step = %+v", route)
	}
	if len(route.Routes) != 1 || route.Routes[0].GoToStep != byTitle["Open"].ID ||
		route.Routes[0].Match != models.RouteMatchAll ||
		!reflect.DeepEqual(route.Routes[0].Rules, []models.RouteRule{{Question: joinQuestion, Operator: models.RouteIs, Value: "yes"}}) {
		t.Errorf("routes = %+v", route.Routes)
	}
	if route.OtherwiseStepID != byTitle["Thank you"].ID {
		t.Errorf("otherwise = %q, want Thank you %q", route.OtherwiseStepID, byTitle["Thank you"].ID)
	}

	records, err := w.mgr.ListRecords(ctx, w.draftID, mwanachamaagency.RecordFilter{})
	if err != nil {
		t.Fatalf("ListRecords: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("the Group order keeps %d records, want 1", len(records))
	}
	got := records[0]
	if got.Name != "Payments" || got.WorkflowID != o.workflow.ID || got.RoleID != w.roles[coordinator].ID ||
		got.FormName != "Payment" || got.Code == "" ||
		!reflect.DeepEqual(withoutIdentity(got.Questions), paymentsRecord(w, o.workflow).Questions) {
		t.Errorf("Payments came back as %+v", got)
	}
	one, err := w.mgr.GetRecord(ctx, got.ID)
	if err != nil || one.ID != got.ID {
		t.Errorf("GetRecord = %+v, %v", one, err)
	}
}

func TestRecord_NeedsANameARoleAWorkflowAndAForm(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	w := o.world
	base := paymentsRecord(w, o.workflow)
	base.Name = "Deliveries"

	invalid := map[string]func(*models.Record){
		"no name":                   func(r *models.Record) { r.Name = " " },
		"no role":                   func(r *models.Record) { r.RoleID = "" },
		"two roles":                 func(r *models.Record) { r.RoleID = w.roles[coordinator].ID + "," + w.roles[farmer].ID },
		"no workflow":               func(r *models.Record) { r.WorkflowID = "" },
		"no form name":              func(r *models.Record) { r.FormName = "" },
		"no questions":              func(r *models.Record) { r.Questions = nil },
		"a question with no prompt": func(r *models.Record) { r.Questions = []models.StepQuestion{ask("", models.AnswerNumber)} },
		"the name of another record in the workflow": func(r *models.Record) { r.Name = "payments" },
	}
	for name, mutate := range invalid {
		in := base
		mutate(&in)
		if _, err := w.mgr.CreateRecord(ctx, in); !errors.Is(err, mwanachamaagency.ErrInvalidRecord) {
			t.Errorf("%s: want ErrInvalidRecord, got %v", name, err)
		}
	}

	in := base
	in.RoleID = "no-such-role"
	if _, err := w.mgr.CreateRecord(ctx, in); !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Errorf("a role that does not exist: want ErrRoleNotFound, got %v", err)
	}
	in = base
	in.WorkflowID = "no-such-workflow"
	if _, err := w.mgr.CreateRecord(ctx, in); !errors.Is(err, mwanachamaagency.ErrWorkflowNotFound) {
		t.Errorf("a workflow that does not exist: want ErrWorkflowNotFound, got %v", err)
	}

	o.record.RoleID = "no-such-role"
	if _, err := w.mgr.UpdateRecord(ctx, o.record); !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Errorf("update to a role that does not exist: want ErrRoleNotFound, got %v", err)
	}
	o.record.RoleID = w.roles[supplier].ID
	o.record.Name = "Paid"
	updated, err := w.mgr.UpdateRecord(ctx, o.record)
	if err != nil || updated.RoleID != w.roles[supplier].ID || updated.Name != "Paid" || updated.Code != o.record.Code {
		t.Errorf("UpdateRecord = %+v, %v", updated, err)
	}
}

func TestRecordRow_IsRefusedForARoleThatDoesNotAddRows(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	w := o.world
	row := models.RecordRow{
		DraftID: w.draftID, RecordID: o.record.ID, RoleID: w.roles[coordinator].ID, AddedBy: "member-1",
		Answers: map[string]string{"Select the farmer": "Wanjiru", "Enter the amount paid": "1200"},
	}

	made, err := w.mgr.CreateRecordRow(ctx, row)
	if err != nil {
		t.Fatalf("the Group coordinator adds a row: %v", err)
	}
	if made.ID == "" || made.RecordCode != o.record.Key || made.CreatedAt == "" {
		t.Errorf("row = %+v", made)
	}

	for name, as := range map[string]string{
		"the Farmer": w.roles[farmer].ID, "the Supplier": w.roles[supplier].ID, "nobody": "", "a role that does not exist": "no-such-role",
	} {
		refused := row
		refused.RoleID = as
		if _, err := w.mgr.CreateRecordRow(ctx, refused); !errors.Is(err, mwanachamaagency.ErrRecordRowRefused) {
			t.Errorf("%s adds a row: want ErrRecordRowRefused, got %v", name, err)
		}
	}

	rows, err := w.mgr.ListRecordRows(ctx, w.draftID, mwanachamaagency.RecordRowFilter{RecordID: o.record.ID})
	if err != nil {
		t.Fatalf("ListRecordRows: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != made.ID || answerTo(o.record, rows[0], "Enter the amount paid") != "1200" {
		t.Fatalf("rows = %+v, want only the one the coordinator added", rows)
	}
}

func TestRecordRow_AnswersMustFitTheRecordsForm(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	w := o.world
	good := map[string]string{"Select the farmer": "Wanjiru", "Enter the amount paid": "1200"}
	row := func(answers map[string]string) models.RecordRow {
		return models.RecordRow{DraftID: w.draftID, RecordID: o.record.ID, RoleID: w.roles[coordinator].ID, Answers: answers}
	}

	cases := map[string]map[string]string{
		"an answer to a question the form does not ask": {"Select the farmer": "Wanjiru", "Enter the amount paid": "1", "Colour": "red"},
		"a required question left out":                  {"Select the farmer": "Wanjiru"},
		"a required question left blank":                {"Select the farmer": "Wanjiru", "Enter the amount paid": "  "},
		"an amount that is not a number":                {"Select the farmer": "Wanjiru", "Enter the amount paid": "plenty"},
	}
	for name, answers := range cases {
		if _, err := w.mgr.CreateRecordRow(ctx, row(answers)); !errors.Is(err, mwanachamaagency.ErrInvalidRecordRow) {
			t.Errorf("%s: want ErrInvalidRecordRow, got %v", name, err)
		}
	}
	if _, err := w.mgr.CreateRecordRow(ctx, row(good)); err != nil {
		t.Errorf("a fitting row: %v", err)
	}

	missing := row(good)
	missing.RecordID = "no-such-record"
	if _, err := w.mgr.CreateRecordRow(ctx, missing); !errors.Is(err, mwanachamaagency.ErrRecordNotFound) {
		t.Errorf("a record that does not exist: want ErrRecordNotFound, got %v", err)
	}
	other := row(good)
	other.DraftID = ""
	if _, err := w.mgr.CreateRecordRow(ctx, other); !errors.Is(err, mwanachamaagency.ErrDraftRequired) {
		t.Errorf("a row naming no draft: want ErrDraftRequired, got %v", err)
	}
}

func TestRecordRow_AForkCarriesTheRecordButNotItsRows(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	w := o.world
	if _, err := w.mgr.CreateRecordRow(ctx, models.RecordRow{
		DraftID: w.draftID, RecordID: o.record.ID, RoleID: w.roles[coordinator].ID,
		Answers: map[string]string{"Select the farmer": "Wanjiru", "Enter the amount paid": "1200"},
	}); err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}

	fork, err := w.mgr.CreateDraft(ctx, "fork", w.draftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	records, _ := w.mgr.ListRecords(ctx, fork.ID, mwanachamaagency.RecordFilter{})
	roles, _ := w.mgr.ListRoles(ctx, fork.ID, mwanachamaagency.RoleFilter{})
	roleNames := map[string]string{}
	for _, r := range roles {
		roleNames[r.ID] = r.Name
	}
	if len(records) != 1 || records[0].ID == o.record.ID || records[0].Code != o.record.Code ||
		roleNames[records[0].RoleID] != coordinator || records[0].DraftID != fork.ID {
		t.Fatalf("forked records = %+v (role %q)", records, roleNames[records[0].RoleID])
	}
	forkedWorkflows, _ := w.mgr.ListWorkflows(ctx, fork.ID, mwanachamaagency.WorkflowFilter{})
	forkedWorkflowNames := map[string]string{}
	for _, wf := range forkedWorkflows {
		forkedWorkflowNames[wf.ID] = wf.Name
	}
	if forkedWorkflowNames[records[0].WorkflowID] != "Group order" {
		t.Errorf("forked record keeps workflow %q (%q), want the forked Group order", records[0].WorkflowID, forkedWorkflowNames[records[0].WorkflowID])
	}
	rows, err := w.mgr.ListRecordRows(ctx, fork.ID, mwanachamaagency.RecordRowFilter{})
	if err != nil || len(rows) != 0 {
		t.Errorf("the fork holds rows %+v, %v", rows, err)
	}
	byCode, err := w.mgr.ListRecordRows(ctx, w.draftID, mwanachamaagency.RecordRowFilter{RecordCode: o.record.Code})
	if err != nil || len(byCode) != 1 {
		t.Errorf("the row stays where it was added: %+v, %v", byCode, err)
	}
}

func stepsByTitle(t *testing.T, w stepWorld, draftID string) map[string]models.WorkItem {
	t.Helper()
	all, err := w.mgr.ListWorkItems(context.Background(), draftID, mwanachamaagency.WorkItemFilter{})
	if err != nil {
		t.Fatalf("ListWorkItems: %v", err)
	}
	out := map[string]models.WorkItem{}
	for _, s := range all {
		out[s.Title] = s
	}
	return out
}

func checkRoutedOrderIn(t *testing.T, w stepWorld, draftID string) {
	t.Helper()
	ctx := context.Background()
	steps := stepsByTitle(t, w, draftID)
	roles, _ := w.mgr.ListRoles(ctx, draftID, mwanachamaagency.RoleFilter{})
	roleNames := map[string]string{}
	for _, r := range roles {
		roleNames[r.ID] = r.Name
	}
	send, answer, route := steps["Send the invitation"], steps["Answer the invitation"], steps["Sort the answers"]
	if roleNames[send.RecipientRoleID] != farmer {
		t.Errorf("the invitation goes to %q (%q)", roleNames[send.RecipientRoleID], send.RecipientRoleID)
	}
	if answer.SentStepID != send.ID || send.ID == "" {
		t.Errorf("the answering step reads %q, want the invitation %q", answer.SentStepID, send.ID)
	}
	if route.SentStepID != send.ID || len(route.Routes) != 1 || route.Routes[0].GoToStep != steps["Open"].ID ||
		route.OtherwiseStepID != steps["Thank you"].ID || steps["Open"].ID == "" {
		t.Errorf("route = %+v", route)
	}
	records, _ := w.mgr.ListRecords(ctx, draftID, mwanachamaagency.RecordFilter{})
	if len(records) != 1 || records[0].Name != "Payments" || roleNames[records[0].RoleID] != coordinator {
		t.Errorf("records = %+v", records)
	}
}

func TestRecord_APublicationKeepsTheRecordAndTheRoutingAndForksBack(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	w := o.world
	assignOn(t, w.mgr, ctx, w.draftID, w.roles[coordinator], o.steps["Open"])
	pub, err := w.mgr.PublishAgency(ctx, w.draftID, "release-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if len(pub.Document.Records) != 1 || pub.Document.Records[0].Name != "Payments" {
		t.Fatalf("the publication document holds records %+v", pub.Document.Records)
	}

	back, err := w.mgr.CreateDraft(ctx, "revisit", "release-1", "publication")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	checkRoutedOrderIn(t, w, back.ID)

	fork, err := w.mgr.CreateDraft(ctx, "copy", w.draftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	checkRoutedOrderIn(t, w, fork.ID)
}

func TestRecord_ChangingOnlyARecordIsAChangeToPublish(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	w := o.world
	assignOn(t, w.mgr, ctx, w.draftID, w.roles[coordinator], o.steps["Open"])
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "release-1", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "release-2", ""); !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("publishing the same content: want ErrNoChangesDetected, got %v", err)
	}
	o.record.RoleID = w.roles[supplier].ID
	if _, err := w.mgr.UpdateRecord(ctx, o.record); err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "release-2", ""); err != nil {
		t.Fatalf("a changed record is a change: %v", err)
	}

	route := o.steps["Sort the answers"]
	route.OtherwiseStepID = o.steps["Open"].ID
	if _, err := w.mgr.UpdateWorkItem(ctx, route); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "release-3", ""); err != nil {
		t.Fatalf("a changed route is a change: %v", err)
	}
}

func TestRecord_TheIssueScanNamesWhatARecordRelies(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	w := o.world
	if err := w.mgr.DeleteRole(ctx, w.roles[coordinator].ID); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}
	issues, err := w.mgr.ListIssues(ctx, w.draftID, mwanachamaagency.IssueFilter{EntityType: "record"})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(issues) != 1 || !strings.Contains(issues[0].Message, "adds rows as Role") {
		t.Fatalf("issues = %+v, want one naming the missing role", issues)
	}
}

func TestRecord_DeletingAWorkflowDeletesItsRecordsAndTheirRows(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	w := o.world
	if _, err := w.mgr.CreateRecordRow(ctx, models.RecordRow{
		DraftID: w.draftID, RecordID: o.record.ID, RoleID: w.roles[coordinator].ID,
		Answers: map[string]string{"Select the farmer": "Wanjiru", "Enter the amount paid": "1"},
	}); err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	if err := w.mgr.DeleteWorkflow(ctx, o.workflow.ID); err != nil {
		t.Fatalf("DeleteWorkflow: %v", err)
	}
	if records, _ := w.mgr.ListRecords(ctx, w.draftID, mwanachamaagency.RecordFilter{}); len(records) != 0 {
		t.Errorf("records left behind: %+v", records)
	}
	if rows, _ := w.mgr.ListRecordRows(ctx, w.draftID, mwanachamaagency.RecordRowFilter{}); len(rows) != 0 {
		t.Errorf("rows left behind: %+v", rows)
	}
	if _, err := w.mgr.GetRecord(ctx, o.record.ID); !errors.Is(err, mwanachamaagency.ErrRecordNotFound) {
		t.Errorf("GetRecord after the workflow went: %v", err)
	}
}

func TestRecord_DeleteRemovesTheRecordAndItsRows(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	w := o.world
	if _, err := w.mgr.CreateRecordRow(ctx, models.RecordRow{
		DraftID: w.draftID, RecordID: o.record.ID, RoleID: w.roles[coordinator].ID,
		Answers: map[string]string{"Select the farmer": "Wanjiru", "Enter the amount paid": "1"},
	}); err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	if err := w.mgr.DeleteRecord(ctx, o.record.ID); err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}
	if rows, _ := w.mgr.ListRecordRows(ctx, w.draftID, mwanachamaagency.RecordRowFilter{}); len(rows) != 0 {
		t.Errorf("rows left behind: %+v", rows)
	}
	if err := w.mgr.DeleteRecord(ctx, o.record.ID); !errors.Is(err, mwanachamaagency.ErrRecordNotFound) {
		t.Errorf("deleting twice: want ErrRecordNotFound, got %v", err)
	}
}
