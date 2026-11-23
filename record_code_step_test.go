package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

const bidDecisionCode = "bid_decision"

func recordCodeStep(w stepWorld, workflowID string) models.WorkItem {
	return models.WorkItem{
		DraftID: w.draftID, WorkflowID: workflowID, Title: "Bid decision", Ordinality: 1, Kind: models.StepKindFillForm,
		RoleID: w.roles[coordinator].ID, FormName: "Bid decision",
		Questions:  []models.StepQuestion{ask("Go or no-go", models.AnswerYesNo)},
		RecordCode: bidDecisionCode,
	}
}

func TestRecordCodeStep_AFillFormStepNamesTheRecordItWritesAndReadsItBack(t *testing.T) {
	w := newStepWorld(t)
	made, err := w.mgr.CreateWorkItem(context.Background(), recordCodeStep(w, w.workflow(t, "Bidding").ID))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	got, listed := readBack(t, w, made.ID, w.draftID)
	for label, step := range map[string]models.WorkItem{"read": got, "listed": listed} {
		if step.RecordCode != bidDecisionCode {
			t.Errorf("%s: record_code = %q", label, step.RecordCode)
		}
	}
}

func TestRecordCodeStep_IsRefusedOnAStepThatIsNotAFillFormStep(t *testing.T) {
	w := newStepWorld(t)
	in := recordCodeStep(w, w.workflow(t, "Not a form").ID)
	in.Kind = models.StepKindYesNo
	if _, err := w.mgr.CreateWorkItem(context.Background(), in); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Errorf("want ErrInvalidWorkItem, got %v", err)
	}
	plain := recordCodeStep(w, w.workflow(t, "Plain").ID)
	plain.RecordCode = ""
	if _, err := w.mgr.CreateWorkItem(context.Background(), plain); err != nil {
		t.Errorf("a step that names no record code is a step as before: %v", err)
	}
}

func TestRecordCodeStep_UpdateKeepsTheCodeABodyLeavesOutAndAClearingBodyClearsIt(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	made, err := w.mgr.CreateWorkItem(ctx, recordCodeStep(w, w.workflow(t, "Bidding").ID))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	body := map[string]any{
		"draft_id": w.draftID, "workflow_id": made.WorkflowID, "title": "Bid decision", "ordinality": 1,
		"kind": "fill_form", "role_id": made.RoleID, "form_name": made.FormName, "questions": made.Questions,
	}
	var keep models.WorkItem
	consoleBody(t, &keep, body)
	keep.ID = made.ID
	if _, err := w.mgr.UpdateWorkItem(ctx, keep); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}
	got, _ := w.mgr.GetWorkItem(ctx, made.ID)
	if got.RecordCode != bidDecisionCode {
		t.Errorf("a body that leaves record_code out keeps it: %q", got.RecordCode)
	}

	body["record_code"] = ""
	var clear models.WorkItem
	consoleBody(t, &clear, body)
	clear.ID = made.ID
	if _, err := w.mgr.UpdateWorkItem(ctx, clear); err != nil {
		t.Fatalf("UpdateWorkItem clearing: %v", err)
	}
	got, _ = w.mgr.GetWorkItem(ctx, made.ID)
	if got.RecordCode != "" {
		t.Errorf("a body that sends it empty clears it: %q", got.RecordCode)
	}
}

func TestRecordCodeStep_AForkAPublicationAndAnImportCarryItAndAChangeIsPublishable(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	made, err := w.mgr.CreateWorkItem(ctx, recordCodeStep(w, w.workflow(t, "Bidding").ID))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	check := func(label, draftID string) {
		t.Helper()
		if step := copiedSteps(t, w.mgr, draftID)["Bid decision"]; step.RecordCode != bidDecisionCode {
			t.Errorf("%s: record_code lost: %q", label, step.RecordCode)
		}
	}
	fork, err := w.mgr.CreateDraft(ctx, "fork", w.draftID, "draft")
	if err != nil {
		t.Fatalf("fork: %v", err)
	}
	check("fork", fork.ID)

	pub, err := w.mgr.PublishAgency(ctx, w.draftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	carried := 0
	for _, wi := range pub.Document.WorkItems {
		if wi.RecordCode == bidDecisionCode {
			carried++
		}
	}
	if carried != 1 {
		t.Errorf("the document carries the step once, found %d", carried)
	}
	restored, err := w.mgr.CreateDraft(ctx, "restored", pub.Tag, "publication")
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	check("restore", restored.ID)
	imported, err := w.mgr.ImportDocument(ctx, "imported", pub.Document)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	check("import", imported.Draft.ID)

	made.RecordCode = "tender_handover"
	if _, err := w.mgr.UpdateWorkItem(ctx, made); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "", ""); err != nil {
		t.Errorf("a changed record code is a change to publish, got %v", err)
	}
}

func TestRecordCodeStep_ADocumentWithTheFieldExportsItUnderRecordCode(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	if _, err := w.mgr.CreateWorkItem(ctx, recordCodeStep(w, w.workflow(t, "Bidding").ID)); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	pub, err := w.mgr.PublishAgency(ctx, w.draftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	raw, err := json.Marshal(pub.Document.WorkItems)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"record_code":"bid_decision"`) {
		t.Errorf("the document step has no record_code: %s", raw)
	}
}
