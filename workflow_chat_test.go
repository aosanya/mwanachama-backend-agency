package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (w stepWorld) chatWorkflow(t *testing.T, name string) models.Workflow {
	t.Helper()
	wf, err := w.mgr.CreateWorkflow(context.Background(), models.Workflow{DraftID: w.draftID, Name: name, OpensIn: models.OpensInChat})
	if err != nil {
		t.Fatalf("CreateWorkflow %s: %v", name, err)
	}
	linkWorkflowToObjective(t, w.mgr, w.draftID, wf.ID, w.objective.ID)
	return wf
}

func (w stepWorld) chatStep(workflow models.Workflow, position int) models.WorkItem {
	return models.WorkItem{
		DraftID: w.draftID, WorkflowID: workflow.ID, Title: "Chat with an agent",
		Ordinality: position, Kind: models.StepKindChat, RoleID: w.roles[farmer].ID,
	}
}

func TestWorkflowOpensIn_AWorkflowSaidToOpenInAChatReadsBackAsOne(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	chat := w.chatWorkflow(t, "Buy inputs")
	plain := w.workflow(t, "Group order")

	got, err := w.mgr.GetWorkflow(ctx, chat.ID)
	if err != nil {
		t.Fatalf("GetWorkflow: %v", err)
	}
	if got.OpensIn != models.OpensInChat || !got.IsChat() {
		t.Errorf("opens_in of the chat workflow = %q, want %q", got.OpensIn, models.OpensInChat)
	}
	listed, err := w.mgr.ListWorkflows(ctx, w.draftID, mwanachamaagency.WorkflowFilter{})
	if err != nil {
		t.Fatalf("ListWorkflows: %v", err)
	}
	opens := map[string]models.OpensIn{}
	for _, wf := range listed {
		opens[wf.ID] = wf.OpensIn
	}
	if opens[chat.ID] != models.OpensInChat || opens[plain.ID] != "" {
		t.Errorf("listed opens_in = %v, want the chat workflow chat and the other empty", opens)
	}
}

func TestWorkflowOpensIn_AFormIsTheDefaultAndIsStoredAsEmpty(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	explicit, err := w.mgr.CreateWorkflow(ctx, models.Workflow{DraftID: w.draftID, Name: "Said form", OpensIn: models.OpensInForm})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	if explicit.OpensIn != "" || explicit.IsChat() {
		t.Errorf("opens_in of a workflow said to open in a form = %q, want it stored as empty", explicit.OpensIn)
	}
	got, err := w.mgr.GetWorkflow(ctx, explicit.ID)
	if err != nil {
		t.Fatalf("GetWorkflow: %v", err)
	}
	if got.OpensIn != "" {
		t.Errorf("opens_in read back = %q, want empty", got.OpensIn)
	}
}

func TestWorkflowOpensIn_AnythingButChatOrFormIsRefused(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	_, err := w.mgr.CreateWorkflow(ctx, models.Workflow{DraftID: w.draftID, Name: "Voice", OpensIn: "voice"})
	if !errors.Is(err, mwanachamaagency.ErrInvalidWorkflow) {
		t.Fatalf("CreateWorkflow with opens_in voice: err = %v, want ErrInvalidWorkflow", err)
	}
	made := w.workflow(t, "Plain")
	made.OpensIn = "voice"
	if _, err := w.mgr.UpdateWorkflow(ctx, made); !errors.Is(err, mwanachamaagency.ErrInvalidWorkflow) {
		t.Fatalf("UpdateWorkflow with opens_in voice: err = %v, want ErrInvalidWorkflow", err)
	}
}

func TestChatStep_IsAKindOfStepAndHasAnAction(t *testing.T) {
	if !models.StepKindChat.Valid() {
		t.Fatalf("chat is not among the step kinds %v", models.StepKinds())
	}
	if models.StepKindChat.TakesForm() {
		t.Errorf("a chat step takes a form")
	}
	if got := models.ActionOf(models.StepKindChat, "", ""); got != "Chats with an agent" {
		t.Errorf("action of a chat step = %q", got)
	}
}

func TestChatStep_BelongsOnlyToAWorkflowThatOpensInAChat(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	plain := w.workflow(t, "Group order")
	if _, err := w.mgr.CreateWorkItem(ctx, w.chatStep(plain, 1)); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Fatalf("a chat step in a workflow that opens in a form: err = %v, want ErrInvalidWorkItem", err)
	}
	chat := w.chatWorkflow(t, "Buy inputs")
	if _, err := w.mgr.CreateWorkItem(ctx, w.chatStep(chat, 1)); err != nil {
		t.Fatalf("a chat step in a workflow that opens in a chat: %v", err)
	}
}

func TestChatStep_AWorkflowHasOneChatStep(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	chat := w.chatWorkflow(t, "Buy inputs")
	if _, err := w.mgr.CreateWorkItem(ctx, w.chatStep(chat, 1)); err != nil {
		t.Fatalf("first chat step: %v", err)
	}
	if _, err := w.mgr.CreateWorkItem(ctx, w.chatStep(chat, 2)); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Fatalf("second chat step in one workflow: err = %v, want ErrInvalidWorkItem", err)
	}
}

func TestChatStep_AnotherStepMayFollowIt(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	chat := w.chatWorkflow(t, "Buy inputs")
	if _, err := w.mgr.CreateWorkItem(ctx, w.chatStep(chat, 1)); err != nil {
		t.Fatalf("chat step: %v", err)
	}
	after := w.item(chat, 2, stepSpec{"Thank you", models.StepKindNotify, theAgency, "Thank-you message", []models.StepQuestion{ask("Write the message", models.AnswerFreeText)}, ""})
	if _, err := w.mgr.CreateWorkItem(ctx, after); err != nil {
		t.Fatalf("a step after the chat step: %v", err)
	}
}

func TestChatStep_ANoteStepCannotBeTurnedIntoOneInAFormWorkflow(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	plain := w.workflow(t, "Group order")
	step, err := w.mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: w.draftID, WorkflowID: plain.ID, Title: "Look", Ordinality: 1})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	step.Kind = models.StepKindChat
	step.RoleID = w.roles[farmer].ID
	if _, err := w.mgr.UpdateWorkItem(ctx, step); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Fatalf("UpdateWorkItem to a chat step in a form workflow: err = %v, want ErrInvalidWorkItem", err)
	}
}

func TestWorkflowOpensIn_CannotLeaveAChatWhileItHoldsAChatStep(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	chat := w.chatWorkflow(t, "Buy inputs")
	if _, err := w.mgr.CreateWorkItem(ctx, w.chatStep(chat, 1)); err != nil {
		t.Fatalf("chat step: %v", err)
	}
	chat.OpensIn = models.OpensInForm
	if _, err := w.mgr.UpdateWorkflow(ctx, chat); !errors.Is(err, mwanachamaagency.ErrInvalidWorkflow) {
		t.Fatalf("UpdateWorkflow from chat to form with a chat step: err = %v, want ErrInvalidWorkflow", err)
	}
	chat.OpensIn = models.OpensInChat
	chat.Name = "Buy inputs renamed"
	if _, err := w.mgr.UpdateWorkflow(ctx, chat); err != nil {
		t.Fatalf("UpdateWorkflow keeping the chat: %v", err)
	}
}
