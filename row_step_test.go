package mwanachamaagency_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

var reasonQuestion = models.StepQuestion{Prompt: "Why is it removed", AnswerType: models.AnswerFreeText, Required: true}

type rowStepWorld struct {
	world  stepWorld
	record models.Record
}

func buildRowStepWorld(t *testing.T) rowStepWorld {
	t.Helper()
	w := newStepWorld(t)
	home := w.workflow(t, "Tender pipeline")
	if _, err := w.mgr.CreateWorkItem(context.Background(), models.WorkItem{DraftID: w.draftID, WorkflowID: home.ID, Title: "Find the tender", Ordinality: 1}); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	record, err := w.mgr.CreateRecord(context.Background(), models.Record{
		DraftID: w.draftID, WorkflowID: home.ID, Name: "Tenders", RoleID: w.roles[coordinator].ID, FormName: "Tender",
		Questions: []models.StepQuestion{ask("Title", models.AnswerFreeText), ask("Value", models.AnswerCurrency)},
	})
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	mayAdd(t, w.mgr, record)
	return rowStepWorld{world: w, record: record}
}

func (r rowStepWorld) step(t *testing.T, workflowName, action string) models.WorkItem {
	t.Helper()
	wf := r.world.workflow(t, workflowName)
	wi := models.WorkItem{
		DraftID: r.world.draftID, WorkflowID: wf.ID, Title: workflowName, Ordinality: 1, Kind: models.StepKindFillForm,
		RoleID: r.record.RoleID, RowRecord: r.record.Key, RowAction: action,
		FormName: r.record.FormName, Questions: r.record.Questions,
	}
	if action == "delete" {
		wi.FormName, wi.Questions = "Why delete this tender", []models.StepQuestion{reasonQuestion}
	}
	return wi
}

func TestRowStep_AFillFormStepNamesTheRecordAndTheActionAndReadsThemBack(t *testing.T) {
	r := buildRowStepWorld(t)
	ctx := context.Background()
	for _, action := range []string{"add", "edit", "delete"} {
		made, err := r.world.mgr.CreateWorkItem(ctx, r.step(t, "Row "+action, action))
		if err != nil {
			t.Fatalf("CreateWorkItem %s: %v", action, err)
		}
		got, err := r.world.mgr.GetWorkItem(ctx, made.ID)
		if err != nil || got.RowRecord != r.record.Key || got.RowAction != action {
			t.Errorf("%s: read back %+v, %v", action, got, err)
		}
	}
}

func TestRowStep_IsRefusedWhenItDoesNotFit(t *testing.T) {
	r := buildRowStepWorld(t)
	ctx := context.Background()
	cases := map[string]func(*models.WorkItem){
		"an action with no record":         func(w *models.WorkItem) { w.RowRecord = "" },
		"a record with no action":          func(w *models.WorkItem) { w.RowAction = "" },
		"an unknown action":                func(w *models.WorkItem) { w.RowAction = "replace" },
		"a record the draft does not keep": func(w *models.WorkItem) { w.RowRecord = "RC-99" },
		"a step that is not fill_form": func(w *models.WorkItem) {
			w.Kind = models.StepKindYesNo
		},
		"a step with no kind":             func(w *models.WorkItem) { w.Kind = ""; w.RoleID = ""; w.FormName, w.Questions = "", nil },
		"a role other than the record's":  func(w *models.WorkItem) { w.RoleID = r.world.roles[farmer].ID },
		"an add with another form":        func(w *models.WorkItem) { w.Questions = []models.StepQuestion{ask("Title", models.AnswerFreeText)} },
		"an edit with another form name":  func(w *models.WorkItem) { w.RowAction = "edit"; w.FormName = "Something else" },
		"a delete with the record's form": func(w *models.WorkItem) { w.RowAction = "delete" },
		"a delete asking two questions": func(w *models.WorkItem) {
			w.RowAction = "delete"
			w.Questions = []models.StepQuestion{reasonQuestion, ask("Who", models.AnswerFreeText)}
		},
		"a delete whose reason is optional": func(w *models.WorkItem) {
			w.RowAction = "delete"
			optional := reasonQuestion
			optional.Required = false
			w.Questions = []models.StepQuestion{optional}
		},
		"a delete whose reason is a date": func(w *models.WorkItem) {
			w.RowAction = "delete"
			dated := reasonQuestion
			dated.AnswerType = models.AnswerDate
			w.Questions = []models.StepQuestion{dated}
		},
	}
	for name, mutate := range cases {
		in := r.step(t, "Refused "+name, "add")
		mutate(&in)
		if _, err := r.world.mgr.CreateWorkItem(ctx, in); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
			t.Errorf("%s: want ErrInvalidWorkItem, got %v", name, err)
		}
	}
}

func TestRowStep_AWorkflowWithARowStepHoldsThatOneStepOnly(t *testing.T) {
	r := buildRowStepWorld(t)
	ctx := context.Background()
	made, err := r.world.mgr.CreateWorkItem(ctx, r.step(t, "Tender add form", "add"))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	other := models.WorkItem{DraftID: r.world.draftID, WorkflowID: made.WorkflowID, Title: "Tell the team", Ordinality: 2}
	if _, err := r.world.mgr.CreateWorkItem(ctx, other); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Errorf("a second step beside a row step: want ErrInvalidWorkItem, got %v", err)
	}

	plain := r.world.workflow(t, "Plain")
	first := models.WorkItem{DraftID: r.world.draftID, WorkflowID: plain.ID, Title: "First", Ordinality: 1}
	if _, err := r.world.mgr.CreateWorkItem(ctx, first); err != nil {
		t.Fatalf("CreateWorkItem plain: %v", err)
	}
	row := r.step(t, "unused", "add")
	row.WorkflowID, row.Ordinality = plain.ID, 2
	if _, err := r.world.mgr.CreateWorkItem(ctx, row); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Errorf("a row step beside another step: want ErrInvalidWorkItem, got %v", err)
	}
}

func TestRowStep_UpdateKeepsTheFieldsAConsoleBodyLeavesOutAndAClearingBodyClearsThem(t *testing.T) {
	r := buildRowStepWorld(t)
	ctx := context.Background()
	made, err := r.world.mgr.CreateWorkItem(ctx, r.step(t, "Tender add form", "add"))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	body := map[string]any{
		"draft_id": r.world.draftID, "workflow_id": made.WorkflowID, "title": "Tender add form", "ordinality": 1,
		"kind": "fill_form", "role_id": made.RoleID, "form_name": made.FormName, "questions": made.Questions,
	}
	var keep models.WorkItem
	consoleBody(t, &keep, body)
	keep.ID = made.ID
	if _, err := r.world.mgr.UpdateWorkItem(ctx, keep); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}
	got, _ := r.world.mgr.GetWorkItem(ctx, made.ID)
	if got.RowRecord != r.record.Key || got.RowAction != "add" {
		t.Errorf("a body that leaves the fields out keeps them: %+v", got)
	}

	body["row_record"], body["row_action"] = "", ""
	var clear models.WorkItem
	consoleBody(t, &clear, body)
	clear.ID = made.ID
	if _, err := r.world.mgr.UpdateWorkItem(ctx, clear); err != nil {
		t.Fatalf("UpdateWorkItem clearing: %v", err)
	}
	got, _ = r.world.mgr.GetWorkItem(ctx, made.ID)
	if got.RowRecord != "" || got.RowAction != "" {
		t.Errorf("a body that sends them empty clears them: %+v", got)
	}
}

func TestRowStep_AForkAPublicationAndAnImportCarryIt(t *testing.T) {
	r := buildRowStepWorld(t)
	ctx := context.Background()
	r.world.mgr.CreateWorkItem(ctx, r.step(t, "Tender add form", "add"))
	edit, err := r.world.mgr.CreateWorkItem(ctx, r.step(t, "Tender edit form", "edit"))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	check := func(label, draftID string) {
		t.Helper()
		steps := copiedSteps(t, r.world.mgr, draftID)
		if steps["Tender add form"].RowRecord != r.record.Key || steps["Tender add form"].RowAction != "add" ||
			steps["Tender edit form"].RowRecord != r.record.Key || steps["Tender edit form"].RowAction != "edit" {
			t.Errorf("%s: row fields lost: add %+v edit %+v", label, steps["Tender add form"], steps["Tender edit form"])
		}
	}
	fork, err := r.world.mgr.CreateDraft(ctx, "fork", r.world.draftID, "draft")
	if err != nil {
		t.Fatalf("fork: %v", err)
	}
	check("fork", fork.ID)

	pub, err := r.world.mgr.PublishAgency(ctx, r.world.draftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	carried := 0
	for _, wi := range pub.Document.WorkItems {
		if wi.RowRecord == r.record.Key && (wi.Title == "Tender add form" || wi.Title == "Tender edit form") && (wi.RowAction == "add" || wi.RowAction == "edit") {
			carried++
		}
	}
	if carried != 2 {
		t.Errorf("the document carries both row steps, found %d", carried)
	}
	restored, err := r.world.mgr.CreateDraft(ctx, "restored", pub.Tag, "publication")
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	check("restore", restored.ID)

	imported, err := r.world.mgr.ImportDocument(ctx, "imported", pub.Document)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	check("import", imported.Draft.ID)

	edit.RowAction = "add"
	if _, err := r.world.mgr.UpdateWorkItem(ctx, edit); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}
	if _, err := r.world.mgr.PublishAgency(ctx, r.world.draftID, "", ""); err != nil {
		t.Errorf("a changed row action is a change to publish, got %v", err)
	}
}

func TestRowStep_TheIssueScanNamesARowStepWhoseRecordIsGone(t *testing.T) {
	r := buildRowStepWorld(t)
	ctx := context.Background()
	if _, err := r.world.mgr.CreateWorkItem(ctx, r.step(t, "Tender add form", "add")); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	if err := r.world.mgr.DeleteRecord(ctx, r.record.ID); err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}
	issues, err := r.world.mgr.ListIssues(ctx, r.world.draftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	for _, i := range issues {
		if i.Severity == models.IssueSeverityError && i.EntityType == "work_item" && strings.Contains(i.Message, "Tender add form") && strings.Contains(i.Message, r.record.Key) {
			return
		}
	}
	t.Errorf("no error names the step and the missing record %s: %+v", r.record.Key, issues)
}
