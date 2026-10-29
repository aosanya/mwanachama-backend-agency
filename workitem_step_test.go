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
	coordinator = "Group coordinator"
	farmer      = "Farmer"
	supplier    = "Supplier"
	theAgency   = ""
)

type stepSpec struct {
	title     string
	kind      models.StepKind
	role      string
	form      string
	questions []models.StepQuestion
	action    string
}

func ask(prompt string, answer models.AnswerType) models.StepQuestion {
	return models.StepQuestion{Prompt: prompt, AnswerType: answer, Required: true}
}

func pickOne(prompt, from string) models.StepQuestion {
	return models.StepQuestion{Prompt: prompt, AnswerType: models.AnswerSingleChoice, Required: true, OptionsFrom: from}
}

var groupOrderSteps = []stepSpec{
	{"Start", models.StepKindFillForm, coordinator, "Group order form", []models.StepQuestion{
		pickOne("Select the target group", "Siaya farmers"),
		pickOne("Select the supplier", "Supplier"),
		ask("Enter the bulk price per bag", models.AnswerCurrency),
		ask("Enter the minimum total of bags", models.AnswerNumber),
	}, "Fills the Group order form"},
	{"Send the invitation", models.StepKindSendForm, theAgency, "Invitation", []models.StepQuestion{
		ask("Do you want to join this order?", models.AnswerYesNo),
	}, "Sends the invitation"},
	{"Answer the invitation", models.StepKindYesNo, farmer, "The farmer's answer", []models.StepQuestion{
		ask("Do you want to join this order?", models.AnswerYesNo),
	}, "Gives the farmer's answer"},
	{"Sort the answers", models.StepKindRoute, theAgency, "", nil, "Routes the answers"},
	{"Open", models.StepKindFillForm, farmer, "Join form", []models.StepQuestion{
		ask("Enter the number of bags", models.AnswerNumber),
	}, "Fills the Join form"},
	{"Send", models.StepKindNotify, coordinator, "Order", []models.StepQuestion{
		ask("Write the message", models.AnswerFreeText),
	}, "Notifies with the order"},
	{"Answered", models.StepKindYesNo, supplier, "The supplier's answer", []models.StepQuestion{
		ask("Can you fill this order?", models.AnswerYesNo),
		ask("Enter the date you can deliver", models.AnswerDate),
	}, "Gives the supplier's answer"},
	{"Thank you", models.StepKindNotify, theAgency, "Thank-you message", []models.StepQuestion{
		{Prompt: "Message sent", AnswerType: models.AnswerFreeText, Required: true, Fixed: "Thank you for the response"},
	}, "Notifies with the thank-you message"},
}

var buyNowSteps = []stepSpec{
	{"Ask", models.StepKindFillForm, farmer, "Buy now form", []models.StepQuestion{
		pickOne("Select the supplier", "Supplier"),
		ask("Enter the product", models.AnswerFreeText),
		ask("Enter the number of bags", models.AnswerNumber),
	}, "Fills the Buy now form"},
	{"Answered", models.StepKindYesNo, supplier, "The supplier's answer", []models.StepQuestion{
		ask("Can you fill this order?", models.AnswerYesNo),
		ask("Enter the date you can deliver", models.AnswerDate),
	}, "Gives the supplier's answer"},
	{"Paid", models.StepKindNotify, supplier, "Paid mark", []models.StepQuestion{
		ask("Write the message", models.AnswerFreeText),
	}, "Notifies with the paid mark"},
}

type stepWorld struct {
	mgr       mwanachamaagency.AgencyManager
	draftID   string
	objective models.Objective
	roles     map[string]models.Role
}

func newStepWorld(t *testing.T) stepWorld {
	t.Helper()
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	w := stepWorld{mgr: mgr, draftID: o.DraftID, objective: o, roles: map[string]models.Role{}}
	for _, name := range []string{coordinator, farmer, supplier} {
		r, err := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: name})
		if err != nil {
			t.Fatalf("CreateRole %s: %v", name, err)
		}
		w.roles[name] = r
	}
	return w
}

func (w stepWorld) workflow(t *testing.T, name string) models.Workflow {
	t.Helper()
	wf, err := w.mgr.CreateWorkflow(context.Background(), models.Workflow{DraftID: w.draftID, Name: name})
	if err != nil {
		t.Fatalf("CreateWorkflow %s: %v", name, err)
	}
	return wf
}

func (w stepWorld) item(workflow models.Workflow, position int, s stepSpec) models.WorkItem {
	return models.WorkItem{
		DraftID: w.draftID, WorkflowID: workflow.ID, ObjectiveID: w.objective.ID,
		Title: s.title, Ordinality: position, Kind: s.kind, RoleID: w.roles[s.role].ID,
		FormName: s.form, Questions: s.questions,
	}
}

func (w stepWorld) build(t *testing.T, workflow models.Workflow, steps []stepSpec) {
	t.Helper()
	for i, s := range steps {
		if _, err := w.mgr.CreateWorkItem(context.Background(), w.item(workflow, i+1, s)); err != nil {
			t.Fatalf("CreateWorkItem %s: %v", s.title, err)
		}
	}
}

func (w stepWorld) stepsOf(t *testing.T, draftID string, workflow models.Workflow) []models.WorkItem {
	t.Helper()
	all, err := w.mgr.ListWorkItems(context.Background(), draftID, mwanachamaagency.WorkItemFilter{})
	if err != nil {
		t.Fatalf("ListWorkItems: %v", err)
	}
	var out []models.WorkItem
	for _, wi := range all {
		if wi.WorkflowID == workflow.ID {
			out = append(out, wi)
		}
	}
	return out
}

func checkReadBack(t *testing.T, w stepWorld, got []models.WorkItem, want []stepSpec) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("read back %d steps, want %d", len(got), len(want))
	}
	for i, s := range want {
		g := got[i]
		if g.Title != s.title || g.Kind != s.kind {
			t.Errorf("step %d = %q/%q, want %q/%q", i+1, g.Title, g.Kind, s.title, s.kind)
		}
		if g.RoleID != w.roles[s.role].ID {
			t.Errorf("step %q role = %q, want %q (%q)", s.title, g.RoleID, w.roles[s.role].ID, s.role)
		}
		if g.FormName != s.form || !reflect.DeepEqual(g.Questions, s.questions) {
			t.Errorf("step %q form = %q %+v, want %q %+v", s.title, g.FormName, g.Questions, s.form, s.questions)
		}
		if g.Action != s.action {
			t.Errorf("step %q action = %q, want %q", s.title, g.Action, s.action)
		}
	}
}

func TestWorkItemStep_GroupOrderReadsBackWithKindsRolesAndForms(t *testing.T) {
	w := newStepWorld(t)
	wf := w.workflow(t, "Group order")
	w.build(t, wf, groupOrderSteps)

	checkReadBack(t, w, w.stepsOf(t, w.draftID, wf), groupOrderSteps)
}

func TestWorkItemStep_BuyNowReadsBackWithKindsRolesAndForms(t *testing.T) {
	w := newStepWorld(t)
	wf := w.workflow(t, "Buy now")
	w.build(t, wf, buyNowSteps)

	checkReadBack(t, w, w.stepsOf(t, w.draftID, wf), buyNowSteps)
}

func TestWorkItemStep_GetReadsOneStepBack(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Buy now")
	created, err := w.mgr.CreateWorkItem(ctx, w.item(wf, 1, buyNowSteps[0]))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	if created.Action != "Fills the Buy now form" {
		t.Errorf("create answered action %q", created.Action)
	}
	got, err := w.mgr.GetWorkItem(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetWorkItem: %v", err)
	}
	if got.Kind != models.StepKindFillForm || got.RoleID != w.roles[farmer].ID || got.Action != "Fills the Buy now form" {
		t.Fatalf("GetWorkItem = %+v", got)
	}
}

func TestWorkItemStep_ActionIsNeverStored(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Buy now")
	in := w.item(wf, 1, buyNowSteps[0])
	in.Action = "Does something else"
	created, err := w.mgr.CreateWorkItem(ctx, in)
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	reread, err := w.mgr.GetWorkItem(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetWorkItem: %v", err)
	}
	if created.Action != "Fills the Buy now form" || reread.Action != "Fills the Buy now form" {
		t.Fatalf("action = %q then %q, want the derived one", created.Action, reread.Action)
	}
}

func TestWorkItemStep_StepWithTwoRolesIsRefused(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Buy now")
	two := w.roles[farmer].ID + "," + w.roles[supplier].ID

	in := w.item(wf, 1, buyNowSteps[0])
	in.RoleID = two
	if _, err := w.mgr.CreateWorkItem(ctx, in); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Fatalf("create with two roles: want ErrInvalidWorkItem, got %v", err)
	}

	ok, err := w.mgr.CreateWorkItem(ctx, w.item(wf, 1, buyNowSteps[0]))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	ok.RoleID = w.roles[farmer].ID + " " + w.roles[supplier].ID
	if _, err := w.mgr.UpdateWorkItem(ctx, ok); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Fatalf("update with two roles: want ErrInvalidWorkItem, got %v", err)
	}
	reread, _ := w.mgr.GetWorkItem(ctx, ok.ID)
	if reread.RoleID != w.roles[farmer].ID {
		t.Fatalf("a refused update changed the role to %q", reread.RoleID)
	}
}

func TestWorkItemStep_RoleMustExistInTheStepsDraft(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Buy now")

	in := w.item(wf, 1, buyNowSteps[0])
	in.RoleID = "no-such-role"
	if _, err := w.mgr.CreateWorkItem(ctx, in); !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Fatalf("unknown role: want ErrRoleNotFound, got %v", err)
	}

	other, err := w.mgr.CreateDraft(ctx, "other", w.draftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	otherWorkflow, err := w.mgr.CreateWorkflow(ctx, models.Workflow{DraftID: other.ID, Name: "Elsewhere"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	in = w.item(otherWorkflow, 1, buyNowSteps[0])
	in.DraftID = other.ID
	in.ObjectiveID = w.objective.ID
	if _, err := w.mgr.CreateWorkItem(ctx, in); !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Fatalf("a role from another draft: want ErrRoleNotFound, got %v", err)
	}
}

func TestWorkItemStep_AgencyOnlyKindsNameNoRole(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Group order")
	for _, s := range []stepSpec{groupOrderSteps[1], groupOrderSteps[3]} {
		in := w.item(wf, 1, s)
		in.RoleID = w.roles[coordinator].ID
		if _, err := w.mgr.CreateWorkItem(ctx, in); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
			t.Errorf("%s step with a role: want ErrInvalidWorkItem, got %v", s.kind, err)
		}
	}
}

func TestWorkItemStep_RunWorkflowNamesOneThatExists(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	groupOrder := w.workflow(t, "Group order")
	cancel := w.workflow(t, "Cancel after the Supplier says yes")

	run := models.WorkItem{
		DraftID: w.draftID, WorkflowID: groupOrder.ID, ObjectiveID: w.objective.ID,
		Title: "Cancel", Kind: models.StepKindRunWorkflow, RoleID: w.roles[farmer].ID, RunWorkflowID: cancel.ID,
	}
	made, err := w.mgr.CreateWorkItem(ctx, run)
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	if made.RunWorkflowID != cancel.ID || made.Action != "Starts Cancel after the Supplier says yes" {
		t.Fatalf("run step = %+v", made)
	}
	listed := w.stepsOf(t, w.draftID, groupOrder)
	if len(listed) != 1 || listed[0].Action != "Starts Cancel after the Supplier says yes" {
		t.Fatalf("listed run step = %+v", listed)
	}

	missing := run
	missing.RunWorkflowID = "no-such-workflow"
	if _, err := w.mgr.CreateWorkItem(ctx, missing); !errors.Is(err, mwanachamaagency.ErrWorkflowNotFound) {
		t.Fatalf("a workflow that does not exist: want ErrWorkflowNotFound, got %v", err)
	}

	made.RunWorkflowID = "no-such-workflow"
	if _, err := w.mgr.UpdateWorkItem(ctx, made); !errors.Is(err, mwanachamaagency.ErrWorkflowNotFound) {
		t.Fatalf("update to a workflow that does not exist: want ErrWorkflowNotFound, got %v", err)
	}

	none := run
	none.RunWorkflowID = ""
	if _, err := w.mgr.CreateWorkItem(ctx, none); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Fatalf("run_workflow naming nothing: want ErrInvalidWorkItem, got %v", err)
	}

	self := run
	self.RunWorkflowID = groupOrder.ID
	if _, err := w.mgr.CreateWorkItem(ctx, self); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Fatalf("a workflow running itself: want ErrInvalidWorkItem, got %v", err)
	}

	notRun := w.item(groupOrder, 1, buyNowSteps[0])
	notRun.RunWorkflowID = cancel.ID
	if _, err := w.mgr.CreateWorkItem(ctx, notRun); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Fatalf("a fill_form step naming a workflow to run: want ErrInvalidWorkItem, got %v", err)
	}
}

func TestWorkItemStep_FormShapeIsChecked(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Buy now")
	base := w.item(wf, 1, buyNowSteps[0])

	cases := map[string]func(*models.WorkItem){
		"an unknown kind":           func(i *models.WorkItem) { i.Kind = "shout" },
		"a form with no name":       func(i *models.WorkItem) { i.FormName = "" },
		"a form with no questions":  func(i *models.WorkItem) { i.Questions = nil },
		"a question with no prompt": func(i *models.WorkItem) { i.Questions = []models.StepQuestion{ask("  ", models.AnswerYesNo)} },
		"an unknown answer type":    func(i *models.WorkItem) { i.Questions = []models.StepQuestion{ask("Why?", "essay")} },
		"pick one with no options": func(i *models.WorkItem) {
			i.Questions = []models.StepQuestion{ask("Which?", models.AnswerSingleChoice)}
		},
		"options on a text question": func(i *models.WorkItem) {
			i.Questions = []models.StepQuestion{{Prompt: "Why?", AnswerType: models.AnswerFreeText, Options: []string{"a"}}}
		},
		"a role without a kind": func(i *models.WorkItem) { i.Kind = ""; i.FormName = ""; i.Questions = nil },
	}
	for name, mutate := range cases {
		in := base
		mutate(&in)
		if _, err := w.mgr.CreateWorkItem(ctx, in); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
			t.Errorf("%s: want ErrInvalidWorkItem, got %v", name, err)
		}
	}

	route := w.item(wf, 2, groupOrderSteps[3])
	route.FormName = "Answers"
	if _, err := w.mgr.CreateWorkItem(ctx, route); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Errorf("a route step with a form: want ErrInvalidWorkItem, got %v", err)
	}
}

func TestWorkItemStep_AnInstructionOnlyStepStillWorks(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Onboarding")
	made, err := w.mgr.CreateWorkItem(ctx, models.WorkItem{
		DraftID: w.draftID, WorkflowID: wf.ID, ObjectiveID: w.objective.ID, Title: "Write charter", Prompt: "Draft it",
	})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	if made.Kind != "" || made.RoleID != "" || made.Action != "" || made.Prompt != "Draft it" {
		t.Fatalf("instruction-only step = %+v", made)
	}
}

func TestWorkItemStep_AForkCarriesTheStepOntoTheForkedRolesAndWorkflows(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	groupOrder := w.workflow(t, "Group order")
	cancel := w.workflow(t, "Cancel before the Supplier says yes")
	w.build(t, groupOrder, groupOrderSteps[:3])
	if _, err := w.mgr.CreateWorkItem(ctx, models.WorkItem{
		DraftID: w.draftID, WorkflowID: groupOrder.ID, ObjectiveID: w.objective.ID, Ordinality: 4,
		Title: "Cancel", Kind: models.StepKindRunWorkflow, RoleID: w.roles[farmer].ID, RunWorkflowID: cancel.ID,
	}); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}

	fork, err := w.mgr.CreateDraft(ctx, "fork", w.draftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	roles, _ := w.mgr.ListRoles(ctx, fork.ID, mwanachamaagency.RoleFilter{})
	roleNames := map[string]string{}
	for _, r := range roles {
		roleNames[r.ID] = r.Name
	}
	workflows, _ := w.mgr.ListWorkflows(ctx, fork.ID, mwanachamaagency.WorkflowFilter{})
	workflowNames := map[string]string{}
	for _, wf := range workflows {
		workflowNames[wf.ID] = wf.Name
	}
	forked, _ := w.mgr.ListWorkItems(ctx, fork.ID, mwanachamaagency.WorkItemFilter{})
	var items []models.WorkItem
	for _, wi := range forked {
		if wi.Kind != "" {
			items = append(items, wi)
		}
	}
	if len(items) != 4 {
		t.Fatalf("forked %d steps, want 4", len(items))
	}
	wantRoles := []string{coordinator, "", farmer, farmer}
	for i, wi := range items {
		got := roleNames[wi.RoleID]
		if got != wantRoles[i] {
			t.Errorf("forked step %q role = %q (%q), want %q", wi.Title, got, wi.RoleID, wantRoles[i])
		}
	}
	run := items[3]
	if workflowNames[run.RunWorkflowID] != "Cancel before the Supplier says yes" {
		t.Errorf("forked run step starts %q (%q)", workflowNames[run.RunWorkflowID], run.RunWorkflowID)
	}
	if run.Action != "Starts Cancel before the Supplier says yes" {
		t.Errorf("forked run step action = %q", run.Action)
	}
	if !reflect.DeepEqual(items[0].Questions, groupOrderSteps[0].questions) || items[0].FormName != "Group order form" {
		t.Errorf("forked form = %q %+v", items[0].FormName, items[0].Questions)
	}
}

func TestWorkItemStep_APublicationKeepsTheStepAndForksBackIntoADraft(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	if _, err := w.mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		DraftID: w.draftID, RoleID: w.roles[coordinator].ID, ObjectiveID: w.objective.ID, Label: models.RACILabelAccountable,
	}); err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}
	groupOrder := w.workflow(t, "Group order")
	cancel := w.workflow(t, "Cancel before the Supplier says yes")
	w.build(t, cancel, buyNowSteps[:1])
	w.build(t, groupOrder, groupOrderSteps[:3])
	if _, err := w.mgr.CreateWorkItem(ctx, models.WorkItem{
		DraftID: w.draftID, WorkflowID: groupOrder.ID, ObjectiveID: w.objective.ID, Ordinality: 4,
		Title: "Cancel", Kind: models.StepKindRunWorkflow, RoleID: w.roles[farmer].ID, RunWorkflowID: cancel.ID,
	}); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "release-1", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}

	back, err := w.mgr.CreateDraft(ctx, "revisit", "release-1", "publication")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	roles, _ := w.mgr.ListRoles(ctx, back.ID, mwanachamaagency.RoleFilter{})
	roleNames := map[string]string{}
	for _, r := range roles {
		roleNames[r.ID] = r.Name
	}
	workflows, _ := w.mgr.ListWorkflows(ctx, back.ID, mwanachamaagency.WorkflowFilter{})
	workflowNames := map[string]string{}
	for _, wf := range workflows {
		workflowNames[wf.ID] = wf.Name
	}
	items, _ := w.mgr.ListWorkItems(ctx, back.ID, mwanachamaagency.WorkItemFilter{})
	byTitle := map[string]models.WorkItem{}
	for _, wi := range items {
		byTitle[wi.Title] = wi
	}
	start := byTitle["Start"]
	if start.Kind != models.StepKindFillForm || roleNames[start.RoleID] != coordinator || start.FormName != "Group order form" ||
		!reflect.DeepEqual(start.Questions, groupOrderSteps[0].questions) {
		t.Errorf("Start came back as %+v (role %q)", start, roleNames[start.RoleID])
	}
	send := byTitle["Send the invitation"]
	if send.Kind != models.StepKindSendForm || send.RoleID != "" {
		t.Errorf("Send the invitation came back as %+v", send)
	}
	cancelStep := byTitle["Cancel"]
	if cancelStep.Kind != models.StepKindRunWorkflow || workflowNames[cancelStep.RunWorkflowID] != "Cancel before the Supplier says yes" ||
		roleNames[cancelStep.RoleID] != farmer {
		t.Errorf("Cancel came back as %+v", cancelStep)
	}
}

func TestWorkItemStep_TheIssueScanNamesAStepWhoseRoleOrWorkflowIsGone(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	groupOrder := w.workflow(t, "Group order")
	cancel := w.workflow(t, "Cancel")
	w.build(t, cancel, buyNowSteps[:1])
	w.build(t, groupOrder, groupOrderSteps[:1])
	if _, err := w.mgr.CreateWorkItem(ctx, models.WorkItem{
		DraftID: w.draftID, WorkflowID: groupOrder.ID, ObjectiveID: w.objective.ID, Ordinality: 2,
		Title: "Cancel", Kind: models.StepKindRunWorkflow, RunWorkflowID: cancel.ID,
	}); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}

	if err := w.mgr.DeleteRole(ctx, w.roles[coordinator].ID); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}
	if err := w.mgr.DeleteWorkflow(ctx, cancel.ID); err != nil {
		t.Fatalf("DeleteWorkflow: %v", err)
	}

	issues, err := w.mgr.ListIssues(ctx, w.draftID, mwanachamaagency.IssueFilter{EntityType: "work_item"})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	var messages []string
	for _, i := range issues {
		messages = append(messages, i.Message)
	}
	joined := strings.Join(messages, "\n")
	if !strings.Contains(joined, "is acted on by Role") || !strings.Contains(joined, "runs Workflow") {
		t.Fatalf("issues = %q, want one naming the missing role and one naming the missing workflow", joined)
	}
}

func TestWorkItemStep_UpdateKeepsTheStepInItsDraft(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Buy now")
	made, err := w.mgr.CreateWorkItem(ctx, w.item(wf, 1, buyNowSteps[0]))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	made.DraftID = ""
	made.Title = "Ask for bags"
	updated, err := w.mgr.UpdateWorkItem(ctx, made)
	if err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}
	if updated.DraftID != w.draftID {
		t.Fatalf("update answered draft %q, want %q", updated.DraftID, w.draftID)
	}
	listed := w.stepsOf(t, w.draftID, wf)
	if len(listed) != 1 || listed[0].Title != "Ask for bags" {
		t.Fatalf("the step left its draft: %+v", listed)
	}
}
