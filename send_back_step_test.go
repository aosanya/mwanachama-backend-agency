package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

const (
	decisionPrompt = "Go or no-go"
	reasonPrompt   = "Reason"
	notesPrompt    = "Notes"
)

func approvalStep(w stepWorld, workflowID string) models.WorkItem {
	return models.WorkItem{
		DraftID: w.draftID, WorkflowID: workflowID, Title: "Approval for submission", Ordinality: 1, Kind: models.StepKindFillForm,
		RoleID: w.roles[coordinator].ID, FormName: "Approval",
		Questions: []models.StepQuestion{
			ask(decisionPrompt, models.AnswerYesNo),
			ask(reasonPrompt, models.AnswerFreeText),
			ask(notesPrompt, models.AnswerFreeText),
		},
		SendBackDecision: decisionPrompt, SendBackReason: reasonPrompt,
	}
}

func TestSendBackStep_AFillFormStepNamesItsDecisionAndReasonAndReadsThemBack(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	made, err := w.mgr.CreateWorkItem(ctx, approvalStep(w, w.workflow(t, "Submission").ID))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	got, listed := readBack(t, w, made.ID, w.draftID)
	for label, step := range map[string]models.WorkItem{"read": got, "listed": listed} {
		if step.SendBackDecision != decisionPrompt || step.SendBackReason != reasonPrompt {
			t.Errorf("%s: send back prompts = %q and %q", label, step.SendBackDecision, step.SendBackReason)
		}
	}
}

func TestSendBackStep_IsRefusedWhenItDoesNotFit(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	cases := map[string]func(*models.WorkItem){
		"a decision with no reason":              func(s *models.WorkItem) { s.SendBackReason = "" },
		"a reason with no decision":              func(s *models.WorkItem) { s.SendBackDecision = "" },
		"a decision that is no question":         func(s *models.WorkItem) { s.SendBackDecision = "Approve it" },
		"a reason that is no question":           func(s *models.WorkItem) { s.SendBackReason = "Why" },
		"a decision that is not yes or no":       func(s *models.WorkItem) { s.SendBackDecision = notesPrompt },
		"a reason that is not free text":         func(s *models.WorkItem) { s.SendBackReason = decisionPrompt },
		"a step that is not fill_form":           func(s *models.WorkItem) { s.Kind = models.StepKindYesNo },
		"a step no role acts on":                 func(s *models.WorkItem) { s.RoleID = "" },
		"a step that writes a row of a record":   func(s *models.WorkItem) { s.RowRecord, s.RowAction = "RC-1", models.RowActionAdd },
		"a decision with surrounding whitespace": func(s *models.WorkItem) { s.SendBackDecision = " " + decisionPrompt },
	}
	for name, mutate := range cases {
		in := approvalStep(w, w.workflow(t, "Refused "+name).ID)
		mutate(&in)
		if _, err := w.mgr.CreateWorkItem(ctx, in); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
			t.Errorf("%s: want ErrInvalidWorkItem, got %v", name, err)
		}
	}
	plain := approvalStep(w, w.workflow(t, "Plain").ID)
	plain.SendBackDecision, plain.SendBackReason = "", ""
	if _, err := w.mgr.CreateWorkItem(ctx, plain); err != nil {
		t.Errorf("a step that names neither prompt is a step as before: %v", err)
	}
}

func TestSendBackStep_UpdateKeepsThePromptsABodyLeavesOutAndAClearingBodyClearsThem(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	made, err := w.mgr.CreateWorkItem(ctx, approvalStep(w, w.workflow(t, "Submission").ID))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	body := map[string]any{
		"draft_id": w.draftID, "workflow_id": made.WorkflowID, "title": "Approval for submission", "ordinality": 1,
		"kind": "fill_form", "role_id": made.RoleID, "form_name": made.FormName, "questions": made.Questions,
	}
	var keep models.WorkItem
	consoleBody(t, &keep, body)
	keep.ID = made.ID
	if _, err := w.mgr.UpdateWorkItem(ctx, keep); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}
	got, _ := w.mgr.GetWorkItem(ctx, made.ID)
	if got.SendBackDecision != decisionPrompt || got.SendBackReason != reasonPrompt {
		t.Errorf("a body that leaves the prompts out keeps them: %+v", got)
	}

	body["send_back_decision"], body["send_back_reason"] = "", ""
	var clear models.WorkItem
	consoleBody(t, &clear, body)
	clear.ID = made.ID
	if _, err := w.mgr.UpdateWorkItem(ctx, clear); err != nil {
		t.Fatalf("UpdateWorkItem clearing: %v", err)
	}
	got, _ = w.mgr.GetWorkItem(ctx, made.ID)
	if got.SendBackDecision != "" || got.SendBackReason != "" {
		t.Errorf("a body that sends them empty clears them: %+v", got)
	}
}

func TestSendBackStep_AForkAPublicationAndAnImportCarryItAndAChangeIsPublishable(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	made, err := w.mgr.CreateWorkItem(ctx, approvalStep(w, w.workflow(t, "Submission").ID))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	check := func(label, draftID string) {
		t.Helper()
		step := copiedSteps(t, w.mgr, draftID)["Approval for submission"]
		if step.SendBackDecision != decisionPrompt || step.SendBackReason != reasonPrompt {
			t.Errorf("%s: send back prompts lost: %+v", label, step)
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
		if wi.SendBackDecision == decisionPrompt && wi.SendBackReason == reasonPrompt {
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

	made.SendBackReason = notesPrompt
	if _, err := w.mgr.UpdateWorkItem(ctx, made); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "", ""); err != nil {
		t.Errorf("a changed reason prompt is a change to publish, got %v", err)
	}
}
