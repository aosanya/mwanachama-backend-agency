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

func (w stepWorld) chatWorld(t *testing.T) (chat, other models.Workflow, step models.WorkItem) {
	t.Helper()
	ctx := context.Background()
	chat = w.chatWorkflow(t, "Buy inputs")
	other = w.workflow(t, "Group order")
	var err error
	step, err = w.mgr.CreateWorkItem(ctx, w.boxedChatStep(chat, other))
	if err != nil {
		t.Fatalf("CreateWorkItem chat: %v", err)
	}
	w.build(t, other, groupOrderSteps[:1])
	return chat, other, step
}

func issueMessages(t *testing.T, w stepWorld) string {
	t.Helper()
	issues, err := w.mgr.ListIssues(context.Background(), w.draftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	var lines []string
	for _, i := range issues {
		if i.Severity == models.IssueSeverityError {
			lines = append(lines, i.Message)
		}
	}
	return strings.Join(lines, "\n")
}

func documentWorkItem(t *testing.T, doc models.AgencyDocument, kind models.StepKind) models.DocumentWorkItem {
	t.Helper()
	for _, wi := range doc.WorkItems {
		if wi.Kind == kind {
			return wi
		}
	}
	t.Fatalf("no %s step in the document", kind)
	return models.DocumentWorkItem{}
}

func TestChatWorkflow_ReachesThePublicationDocument(t *testing.T) {
	w := newStepWorld(t)
	chat, other, step := w.chatWorld(t)
	pub, err := w.mgr.PublishAgency(context.Background(), w.draftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	opens := map[string]models.OpensIn{}
	for _, wf := range pub.Document.Workflows {
		opens[wf.ID] = wf.OpensIn
	}
	if opens[chat.ID] != models.OpensInChat || opens[other.ID] != "" {
		t.Errorf("opens_in in the document = %v", opens)
	}
	got := documentWorkItem(t, pub.Document, models.StepKindChat)
	if got.AgentRoleID != step.AgentRoleID || got.Topic != step.Topic || got.OpeningLine != step.OpeningLine {
		t.Errorf("document chat step = %+v", got)
	}
	if !reflect.DeepEqual(got.Prompts, step.Prompts) || !reflect.DeepEqual(got.Tools, step.Tools) || !reflect.DeepEqual(got.StartWorkflowIDs, step.StartWorkflowIDs) {
		t.Errorf("document chat lists = %v %v %v", got.Prompts, got.Tools, got.StartWorkflowIDs)
	}
}

func TestChatWorkflow_ComesBackWhenADraftIsForkedFromItsTag(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	w.chatWorld(t)
	pub, err := w.mgr.PublishAgency(ctx, w.draftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	fork, err := w.mgr.CreateDraft(ctx, "from the tag", pub.Tag, "publication")
	if err != nil {
		t.Fatalf("CreateDraft from the tag: %v", err)
	}
	assertForkedChat(t, w, fork)
}

func TestChatWorkflow_ComesBackWhenADraftIsForkedFromADraft(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	w.chatWorld(t)
	fork, err := w.mgr.CreateDraft(ctx, "from the draft", w.draftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft from the draft: %v", err)
	}
	assertForkedChat(t, w, fork)
}

func assertForkedChat(t *testing.T, w stepWorld, fork models.Draft) {
	t.Helper()
	ctx := context.Background()
	workflows, err := w.mgr.ListWorkflows(ctx, fork.ID, mwanachamaagency.WorkflowFilter{})
	if err != nil {
		t.Fatalf("ListWorkflows: %v", err)
	}
	var chat, other models.Workflow
	for _, wf := range workflows {
		if wf.IsChat() {
			chat = wf
		} else if wf.Name == "Group order" {
			other = wf
		}
	}
	if chat.ID == "" || other.ID == "" {
		t.Fatalf("the fork has workflows %v, want a chat one and Group order", workflows)
	}
	steps, err := w.mgr.ListWorkItems(ctx, fork.ID, mwanachamaagency.WorkItemFilter{})
	if err != nil {
		t.Fatalf("ListWorkItems: %v", err)
	}
	var got models.WorkItem
	for _, s := range steps {
		if s.Kind == models.StepKindChat {
			got = s
		}
	}
	if got.ID == "" {
		t.Fatalf("the fork has no chat step")
	}
	roles, err := w.mgr.ListRoles(ctx, fork.ID, mwanachamaagency.RoleFilter{})
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	forkRoles := map[string]string{}
	for _, r := range roles {
		forkRoles[r.ID] = r.Name
	}
	if forkRoles[got.AgentRoleID] != coordinator || forkRoles[got.RoleID] != farmer {
		t.Errorf("the fork's chat step names roles %q and %q, want ids of the fork's %s and %s", got.AgentRoleID, got.RoleID, coordinator, farmer)
	}
	if got.WorkflowID != chat.ID || !reflect.DeepEqual(got.StartWorkflowIDs, []string{other.ID}) {
		t.Errorf("the fork's chat step is in %q and may start %v, want %q and [%s]", got.WorkflowID, got.StartWorkflowIDs, chat.ID, other.ID)
	}
	if got.Topic != "Buying farm inputs" || !reflect.DeepEqual(got.Tools, []string{"See what is on sale", "Ask a shop"}) {
		t.Errorf("the fork's chat step topic = %q, tools = %v", got.Topic, got.Tools)
	}
}

func TestPublishHash_AFormWorkflowAndItsStepsHashAsTheyDidBefore(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	plain := w.workflow(t, "Group order")
	w.build(t, plain, groupOrderSteps[:1])
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "", ""); err != nil {
		t.Fatalf("first publish: %v", err)
	}
	plain.OpensIn = models.OpensInForm
	if _, err := w.mgr.UpdateWorkflow(ctx, plain); err != nil {
		t.Fatalf("UpdateWorkflow saying form: %v", err)
	}
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "", ""); !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("publishing after a workflow said it opens in a form: err = %v, want ErrNoChangesDetected", err)
	}
}

func TestPublishHash_EachChatFieldIsPartOfIt(t *testing.T) {
	var w stepWorld
	changes := map[string]func(*models.WorkItem, models.Workflow){
		"the topic":     func(s *models.WorkItem, _ models.Workflow) { s.Topic = "Selling the harvest" },
		"the opening":   func(s *models.WorkItem, _ models.Workflow) { s.OpeningLine = "Hello" },
		"the prompts":   func(s *models.WorkItem, _ models.Workflow) { s.Prompts = []string{"Only one"} },
		"the tools":     func(s *models.WorkItem, _ models.Workflow) { s.Tools = []string{"Ask a shop"} },
		"the may-start": func(s *models.WorkItem, _ models.Workflow) { s.StartWorkflowIDs = nil },
		"the agent":     func(s *models.WorkItem, _ models.Workflow) { s.AgentRoleID = w.roles[supplier].ID },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			w = newStepWorld(t)
			ctx := context.Background()
			_, other, step := w.chatWorld(t)
			if _, err := w.mgr.PublishAgency(ctx, w.draftID, "", ""); err != nil {
				t.Fatalf("first publish: %v", err)
			}
			change(&step, other)
			if _, err := w.mgr.UpdateWorkItem(ctx, step); err != nil {
				t.Fatalf("UpdateWorkItem: %v", err)
			}
			if _, err := w.mgr.PublishAgency(ctx, w.draftID, "", ""); err != nil {
				t.Fatalf("publish after changing %s: %v (a hash that ignores it answers ErrNoChangesDetected)", name, err)
			}
		})
	}
}

func TestScan_ChatStepNamesWhatItNeeds(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	_, other, _ := w.chatWorld(t)
	if got := issueMessages(t, w); got != "" {
		t.Fatalf("a complete chat workflow has errors: %s", got)
	}

	steps, err := w.mgr.ListWorkItems(ctx, w.draftID, mwanachamaagency.WorkItemFilter{})
	if err != nil {
		t.Fatalf("ListWorkItems: %v", err)
	}
	var chatStep models.WorkItem
	for _, s := range steps {
		if s.Kind == models.StepKindChat {
			chatStep = s
		}
	}
	chatStep.RoleID, chatStep.AgentRoleID, chatStep.Topic = "", "", ""
	if _, err := w.mgr.UpdateWorkItem(ctx, chatStep); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}
	got := issueMessages(t, w)
	for _, want := range []string{"names no role that opens it", "names no agent to chat with", "names no topic"} {
		if !strings.Contains(got, want) {
			t.Errorf("errors %q lack %q", got, want)
		}
	}

	other.OpensIn = models.OpensInChat
	if _, err := w.mgr.UpdateWorkflow(ctx, other); err != nil {
		t.Fatalf("UpdateWorkflow: %v", err)
	}
	if got := issueMessages(t, w); !strings.Contains(got, "opens in a chat but has no chat step") {
		t.Errorf("errors %q lack the chat workflow with no chat step", got)
	}
}
