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

const (
	acceptedPrompt = "Accepted"
	statusPrompt   = "Status"
)

type settingWorld struct {
	world  stepWorld
	record models.Record
}

func buildSettingWorld(t *testing.T) settingWorld {
	t.Helper()
	w := newStepWorld(t)
	home := w.workflow(t, "Order keeping")
	if _, err := w.mgr.CreateWorkItem(context.Background(), models.WorkItem{DraftID: w.draftID, WorkflowID: home.ID, Title: "Keep the order", Ordinality: 1}); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	record, err := w.mgr.CreateRecord(context.Background(), models.Record{
		DraftID: w.draftID, WorkflowID: home.ID, Name: "Orders", FormName: "Order",
		Questions: []models.StepQuestion{
			ask("Title", models.AnswerFreeText),
			{Prompt: statusPrompt, AnswerType: models.AnswerSingleChoice, Required: true, Options: []string{"Proposed", "Accepted", "Fulfilled"}},
		},
	})
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	mayAdd(t, w.mgr, record)
	return settingWorld{world: w, record: record}
}

func (s settingWorld) workflowOn(t *testing.T, name, subject string) models.Workflow {
	t.Helper()
	made, err := s.world.mgr.CreateWorkflow(context.Background(), models.Workflow{DraftID: s.world.draftID, Name: name, SubjectRecord: subject})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	linkWorkflowToObjective(t, s.world.mgr, s.world.draftID, made.ID, s.world.objective.ID)
	return made
}

func acceptedSetting() models.SubjectSetting {
	return models.SubjectSetting{
		When:  models.RouteRule{Question: acceptedPrompt, Operator: models.RouteIs, Value: "Yes"},
		Field: statusPrompt, To: "Accepted",
	}
}

func (s settingWorld) deciding(workflowID string) models.WorkItem {
	return models.WorkItem{
		DraftID: s.world.draftID, WorkflowID: workflowID, Title: "Decide the order", Ordinality: 1, Kind: models.StepKindFillForm,
		RoleID: s.world.roles[coordinator].ID, FormName: "Decision",
		Questions:   []models.StepQuestion{ask(acceptedPrompt, models.AnswerYesNo), ask(reasonPrompt, models.AnswerFreeText)},
		SetsSubject: []models.SubjectSetting{acceptedSetting()},
	}
}

func TestSetsSubject_AFormStepNamesTheChoiceItSetsAndReadsItBack(t *testing.T) {
	s := buildSettingWorld(t)
	made, err := s.world.mgr.CreateWorkItem(context.Background(), s.deciding(s.workflowOn(t, "Orders", s.record.Key).ID))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	got, listed := readBack(t, s.world, made.ID, s.world.draftID)
	for label, step := range map[string]models.WorkItem{"read": got, "listed": listed} {
		if !reflect.DeepEqual(step.SetsSubject, []models.SubjectSetting{acceptedSetting()}) {
			t.Errorf("%s: sets_subject = %+v", label, step.SetsSubject)
		}
	}
}

func TestSetsSubject_IsRefusedWhenItDoesNotFit(t *testing.T) {
	s := buildSettingWorld(t)
	ctx := context.Background()
	withSubject := s.workflowOn(t, "With subject", s.record.Key)
	cases := map[string]struct {
		workflowID string
		mutate     func(*models.WorkItem)
		says       string
	}{
		"a workflow with no subject record": {s.workflowOn(t, "No subject", "").ID, func(*models.WorkItem) {}, "subject"},
		"a question the step does not ask": {withSubject.ID, func(w *models.WorkItem) {
			w.SetsSubject[0].When.Question = "Approved"
		}, "Approved"},
		"a rule with no question": {withSubject.ID, func(w *models.WorkItem) { w.SetsSubject[0].When.Question = "" }, "question"},
		"an unknown operator": {withSubject.ID, func(w *models.WorkItem) {
			w.SetsSubject[0].When.Operator = "resembles"
		}, "operator"},
		"a rule with no value": {withSubject.ID, func(w *models.WorkItem) { w.SetsSubject[0].When.Value = "" }, "value"},
		"a field the subject record does not ask": {withSubject.ID, func(w *models.WorkItem) {
			w.SetsSubject[0].Field = "Stage"
		}, "Stage"},
		"a field that is not a choice": {withSubject.ID, func(w *models.WorkItem) {
			w.SetsSubject[0].Field = "Title"
		}, "Title"},
		"an option the choice does not offer": {withSubject.ID, func(w *models.WorkItem) {
			w.SetsSubject[0].To = "Shipped"
		}, "Shipped"},
		"no option": {withSubject.ID, func(w *models.WorkItem) { w.SetsSubject[0].To = "" }, "option"},
		"a step that is not fill_form": {withSubject.ID, func(w *models.WorkItem) {
			w.Kind = models.StepKindYesNo
		}, "fill_form"},
		"a step no role acts on": {withSubject.ID, func(w *models.WorkItem) { w.RoleID = "" }, "role"},
		"a step that writes a row": {withSubject.ID, func(w *models.WorkItem) {
			w.RowRecord, w.RowAction = s.record.Key, models.RowActionAdd
		}, "row"},
	}
	for name, c := range cases {
		in := s.deciding(c.workflowID)
		c.mutate(&in)
		_, err := s.world.mgr.CreateWorkItem(ctx, in)
		if !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
			t.Errorf("%s: want ErrInvalidWorkItem, got %v", name, err)
			continue
		}
		if !strings.Contains(err.Error(), c.says) {
			t.Errorf("%s: the refusal %q does not say %q", name, err, c.says)
		}
	}
	plain := s.deciding(s.workflowOn(t, "Plain", "").ID)
	plain.SetsSubject = nil
	if _, err := s.world.mgr.CreateWorkItem(ctx, plain); err != nil {
		t.Errorf("a step with no setting is a step as before: %v", err)
	}
}

func TestSetsSubject_UpdateKeepsWhatABodyLeavesOutAndAClearingBodyClearsIt(t *testing.T) {
	s := buildSettingWorld(t)
	ctx := context.Background()
	made, err := s.world.mgr.CreateWorkItem(ctx, s.deciding(s.workflowOn(t, "Orders", s.record.Key).ID))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	body := map[string]any{
		"draft_id": s.world.draftID, "workflow_id": made.WorkflowID, "title": "Decide the order", "ordinality": 1,
		"kind": "fill_form", "role_id": made.RoleID, "form_name": made.FormName, "questions": made.Questions,
	}
	var keep models.WorkItem
	consoleBody(t, &keep, body)
	keep.ID = made.ID
	if _, err := s.world.mgr.UpdateWorkItem(ctx, keep); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}
	got, _ := s.world.mgr.GetWorkItem(ctx, made.ID)
	if !reflect.DeepEqual(got.SetsSubject, []models.SubjectSetting{acceptedSetting()}) {
		t.Errorf("a body that leaves sets_subject out keeps it: %+v", got.SetsSubject)
	}

	body["sets_subject"] = []models.SubjectSetting{}
	var clear models.WorkItem
	consoleBody(t, &clear, body)
	clear.ID = made.ID
	if _, err := s.world.mgr.UpdateWorkItem(ctx, clear); err != nil {
		t.Fatalf("UpdateWorkItem clearing: %v", err)
	}
	got, _ = s.world.mgr.GetWorkItem(ctx, made.ID)
	if len(got.SetsSubject) != 0 {
		t.Errorf("a body that sends it empty clears it: %+v", got.SetsSubject)
	}
}

func TestSetsSubject_AForkAPublicationAndAnImportCarryItAndAChangeIsPublishable(t *testing.T) {
	s := buildSettingWorld(t)
	ctx := context.Background()
	mgr := s.world.mgr
	made, err := mgr.CreateWorkItem(ctx, s.deciding(s.workflowOn(t, "Orders", s.record.Key).ID))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	check := func(label, draftID string) {
		t.Helper()
		step := copiedSteps(t, mgr, draftID)["Decide the order"]
		if !reflect.DeepEqual(step.SetsSubject, []models.SubjectSetting{acceptedSetting()}) {
			t.Errorf("%s: sets_subject = %+v", label, step.SetsSubject)
		}
	}
	fork, err := mgr.CreateDraft(ctx, "fork", s.world.draftID, "draft")
	if err != nil {
		t.Fatalf("fork: %v", err)
	}
	check("fork", fork.ID)

	pub, err := mgr.PublishAgency(ctx, s.world.draftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	carried := 0
	for _, wi := range pub.Document.WorkItems {
		if reflect.DeepEqual(wi.SetsSubject, []models.SubjectSetting{acceptedSetting()}) {
			carried++
		}
	}
	if carried != 1 {
		t.Errorf("the document carries the step once, found %d", carried)
	}
	restored, err := mgr.CreateDraft(ctx, "restored", pub.Tag, "publication")
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	check("restore", restored.ID)
	imported, err := mgr.ImportDocument(ctx, "imported", pub.Document)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	check("import", imported.Draft.ID)

	made.SetsSubject = []models.SubjectSetting{{When: acceptedSetting().When, Field: statusPrompt, To: "Fulfilled"}}
	if _, err := mgr.UpdateWorkItem(ctx, made); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, s.world.draftID, "", ""); err != nil {
		t.Errorf("a changed option is a change to publish, got %v", err)
	}
}

func TestSetsSubject_AFileWithAnOptionTheRecordDoesNotOfferIsRefusedOnImport(t *testing.T) {
	s := buildSettingWorld(t)
	ctx := context.Background()
	if _, err := s.world.mgr.CreateWorkItem(ctx, s.deciding(s.workflowOn(t, "Orders", s.record.Key).ID)); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	pub, err := s.world.mgr.PublishAgency(ctx, s.world.draftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	doc := pub.Document
	for i := range doc.WorkItems {
		for j := range doc.WorkItems[i].SetsSubject {
			doc.WorkItems[i].SetsSubject[j].To = "Shipped"
		}
	}
	_, err = s.world.mgr.ImportDocument(ctx, "refused", doc)
	if !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) || !strings.Contains(err.Error(), "Shipped") {
		t.Errorf("ImportDocument error = %v, want ErrInvalidWorkItem naming Shipped", err)
	}
}

func TestSetsSubject_TheIssueScanReportsAnOptionThatLeftTheRecord(t *testing.T) {
	s := buildSettingWorld(t)
	ctx := context.Background()
	if _, err := s.world.mgr.CreateWorkItem(ctx, s.deciding(s.workflowOn(t, "Orders", s.record.Key).ID)); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	questions := append([]models.StepQuestion(nil), s.record.Questions...)
	questions[1].Options = []string{"Proposed", "Fulfilled"}
	s.record.Questions = questions
	if _, err := s.world.mgr.UpdateRecord(ctx, s.record); err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}
	issues, err := s.world.mgr.ListIssues(ctx, s.world.draftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	for _, issue := range issues {
		if issue.Severity == models.IssueSeverityError && strings.Contains(issue.Message, "Decide the order") && strings.Contains(issue.Message, "Accepted") {
			return
		}
	}
	t.Errorf("no error names the step and the option Accepted: %+v", issues)
}
