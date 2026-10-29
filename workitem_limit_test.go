package mwanachamaagency_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type limitWorld struct {
	stepWorld
	ride   models.Workflow
	cancel models.Workflow
}

func newLimitWorld(t *testing.T) limitWorld {
	t.Helper()
	w := newStepWorld(t)
	l := limitWorld{stepWorld: w, ride: w.workflow(t, "Pick a rider"), cancel: w.workflow(t, "Rider ask closed")}
	l.build(t, l.cancel, buyNowSteps[2:])
	return l
}

func (l limitWorld) waiting(minutes int) models.WorkItem {
	item := l.item(l.ride, 1, buyNowSteps[1])
	item.TimeLimitMinutes = minutes
	item.CancelWorkflowID = l.cancel.ID
	return item
}

func TestStepLimit_AWaitingStepKeepsItsLimitAndCancelWorkflow(t *testing.T) {
	l := newLimitWorld(t)
	ctx := context.Background()
	made, err := l.mgr.CreateWorkItem(ctx, l.waiting(15))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	got, err := l.mgr.GetWorkItem(ctx, made.ID)
	if err != nil {
		t.Fatalf("GetWorkItem: %v", err)
	}
	if got.TimeLimitMinutes != 15 || got.CancelWorkflowID != l.cancel.ID {
		t.Fatalf("read back %d minutes, cancel %q; want 15 and %q", got.TimeLimitMinutes, got.CancelWorkflowID, l.cancel.ID)
	}
	listed := l.stepsOf(t, l.draftID, l.ride)
	if len(listed) != 1 || listed[0].TimeLimitMinutes != 15 || listed[0].CancelWorkflowID != l.cancel.ID {
		t.Fatalf("listed = %+v", listed)
	}
}

func TestStepLimit_AFillFormStepCanWaitToo(t *testing.T) {
	l := newLimitWorld(t)
	item := l.item(l.ride, 1, buyNowSteps[0])
	item.TimeLimitMinutes = 5
	item.CancelWorkflowID = l.cancel.ID
	if _, err := l.mgr.CreateWorkItem(context.Background(), item); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
}

func TestStepLimit_RefusesWhatCannotWait(t *testing.T) {
	l := newLimitWorld(t)
	ctx := context.Background()
	noRole := l.waiting(5)
	noRole.RoleID = ""
	notify := l.item(l.ride, 1, buyNowSteps[2])
	notify.TimeLimitMinutes, notify.CancelWorkflowID = 5, l.cancel.ID
	send := l.item(l.ride, 1, groupOrderSteps[1])
	send.TimeLimitMinutes, send.CancelWorkflowID = 5, l.cancel.ID
	bare := models.WorkItem{DraftID: l.draftID, WorkflowID: l.ride.ID, ObjectiveID: l.objective.ID, Title: "Do it", TimeLimitMinutes: 5, CancelWorkflowID: l.cancel.ID}
	negative := l.waiting(-3)
	noCancel := l.waiting(5)
	noCancel.CancelWorkflowID = ""
	noLimit := l.waiting(0)
	ownCancel := l.waiting(5)
	ownCancel.CancelWorkflowID = l.ride.ID

	cases := map[string]models.WorkItem{
		"a step the agency takes":         noRole,
		"a notify step":                   notify,
		"a send_form step":                send,
		"a step with no kind":             bare,
		"a negative limit":                negative,
		"a limit with no cancel workflow": noCancel,
		"a cancel workflow with no limit": noLimit,
		"the workflow that owns the step": ownCancel,
	}
	for name, item := range cases {
		if _, err := l.mgr.CreateWorkItem(ctx, item); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
			t.Errorf("%s: err = %v, want ErrInvalidWorkItem", name, err)
		}
	}
	if len(l.stepsOf(t, l.draftID, l.ride)) != 0 {
		t.Errorf("a refused step was stored")
	}
}

func TestStepLimit_RefusesACancelWorkflowThatIsNotInTheDraft(t *testing.T) {
	l := newLimitWorld(t)
	ctx := context.Background()
	missing := l.waiting(5)
	missing.CancelWorkflowID = "no-such-workflow"
	if _, err := l.mgr.CreateWorkItem(ctx, missing); !errors.Is(err, mwanachamaagency.ErrWorkflowNotFound) {
		t.Errorf("missing workflow: err = %v, want ErrWorkflowNotFound", err)
	}

	fork, err := l.mgr.CreateDraft(ctx, "other", l.draftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	forkedWorkflows, _ := l.mgr.ListWorkflows(ctx, fork.ID, mwanachamaagency.WorkflowFilter{})
	elsewhere := l.waiting(5)
	elsewhere.CancelWorkflowID = forkedWorkflows[0].ID
	if _, err := l.mgr.CreateWorkItem(ctx, elsewhere); !errors.Is(err, mwanachamaagency.ErrWorkflowNotFound) {
		t.Errorf("a workflow of another draft: err = %v, want ErrWorkflowNotFound", err)
	}
}

func TestStepLimit_UpdateSetsAndClearsTheLimit(t *testing.T) {
	l := newLimitWorld(t)
	ctx := context.Background()
	made, err := l.mgr.CreateWorkItem(ctx, l.item(l.ride, 1, buyNowSteps[1]))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	if made.TimeLimitMinutes != 0 || made.CancelWorkflowID != "" {
		t.Fatalf("a step made with no limit has %d minutes and %q", made.TimeLimitMinutes, made.CancelWorkflowID)
	}

	made.TimeLimitMinutes, made.CancelWorkflowID = 5, l.cancel.ID
	set, err := l.mgr.UpdateWorkItem(ctx, made)
	if err != nil || set.TimeLimitMinutes != 5 {
		t.Fatalf("set: %+v, %v", set, err)
	}

	set.TimeLimitMinutes = 0
	if _, err := l.mgr.UpdateWorkItem(ctx, set); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Fatalf("clearing only the minutes: err = %v, want ErrInvalidWorkItem", err)
	}

	set.TimeLimitMinutes, set.CancelWorkflowID = 0, ""
	cleared, err := l.mgr.UpdateWorkItem(ctx, set)
	if err != nil || cleared.TimeLimitMinutes != 0 || cleared.CancelWorkflowID != "" {
		t.Fatalf("clear: %+v, %v", cleared, err)
	}
	got, _ := l.mgr.GetWorkItem(ctx, made.ID)
	if got.TimeLimitMinutes != 0 || got.CancelWorkflowID != "" {
		t.Errorf("stored after clearing = %d minutes, %q", got.TimeLimitMinutes, got.CancelWorkflowID)
	}
}

func TestStepLimit_AForkCarriesTheLimitOntoTheForkedCancelWorkflow(t *testing.T) {
	l := newLimitWorld(t)
	ctx := context.Background()
	if _, err := l.mgr.CreateWorkItem(ctx, l.waiting(15)); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	fork, err := l.mgr.CreateDraft(ctx, "fork", l.draftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	forkedCancel := cancelWorkflowOf(t, l, fork.ID)
	if forkedCancel.ID == l.cancel.ID || forkedCancel.Name != "Rider ask closed" {
		t.Fatalf("forked cancel workflow = %+v", forkedCancel)
	}
}

func TestStepLimit_APublicationKeepsTheLimitAndForksBackOntoTheRemappedWorkflow(t *testing.T) {
	l := newLimitWorld(t)
	ctx := context.Background()
	if _, err := l.mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		DraftID: l.draftID, RoleID: l.roles[coordinator].ID, ObjectiveID: l.objective.ID, Label: models.RACILabelAccountable,
	}); err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}
	if _, err := l.mgr.CreateWorkItem(ctx, l.waiting(15)); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	if _, err := l.mgr.PublishAgency(ctx, l.draftID, "release-1", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	back, err := l.mgr.CreateDraft(ctx, "revisit", "release-1", "publication")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	backCancel := cancelWorkflowOf(t, l, back.ID)
	if backCancel.ID == l.cancel.ID || backCancel.Name != "Rider ask closed" {
		t.Fatalf("restored cancel workflow = %+v", backCancel)
	}
}

func cancelWorkflowOf(t *testing.T, l limitWorld, draftID string) models.Workflow {
	t.Helper()
	ctx := context.Background()
	items, _ := l.mgr.ListWorkItems(ctx, draftID, mwanachamaagency.WorkItemFilter{})
	var limited models.WorkItem
	for _, wi := range items {
		if wi.TimeLimitMinutes != 0 {
			limited = wi
		}
	}
	if limited.TimeLimitMinutes != 15 || limited.CancelWorkflowID == "" {
		t.Fatalf("no step with a 15 minute limit in draft %s: %+v", draftID, items)
	}
	workflows, _ := l.mgr.ListWorkflows(ctx, draftID, mwanachamaagency.WorkflowFilter{})
	for _, wf := range workflows {
		if wf.ID == limited.CancelWorkflowID {
			return wf
		}
	}
	t.Fatalf("cancel workflow %s is not in draft %s", limited.CancelWorkflowID, draftID)
	return models.Workflow{}
}

func TestStepLimit_ALimitChangeIsAChangeToPublish(t *testing.T) {
	l := newLimitWorld(t)
	ctx := context.Background()
	if _, err := l.mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		DraftID: l.draftID, RoleID: l.roles[coordinator].ID, ObjectiveID: l.objective.ID, Label: models.RACILabelAccountable,
	}); err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}
	made, err := l.mgr.CreateWorkItem(ctx, l.item(l.ride, 1, buyNowSteps[1]))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	if _, err := l.mgr.PublishAgency(ctx, l.draftID, "release-1", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	next, err := l.mgr.CreateDraft(ctx, "limit", "release-1", "publication")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	items, _ := l.mgr.ListWorkItems(ctx, next.ID, mwanachamaagency.WorkItemFilter{})
	var step models.WorkItem
	for _, wi := range items {
		if wi.Title == made.Title {
			step = wi
		}
	}
	workflows, _ := l.mgr.ListWorkflows(ctx, next.ID, mwanachamaagency.WorkflowFilter{})
	for _, wf := range workflows {
		if wf.Name == "Rider ask closed" {
			step.CancelWorkflowID = wf.ID
		}
	}
	step.TimeLimitMinutes = 5
	if _, err := l.mgr.UpdateWorkItem(ctx, step); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}
	if _, err := l.mgr.PublishAgency(ctx, next.ID, "release-2", ""); errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("a new time limit was read as no change: %v", err)
	}
}

func TestStepLimit_TheIssueScanNamesAStepWhoseCancelWorkflowIsGone(t *testing.T) {
	l := newLimitWorld(t)
	ctx := context.Background()
	if _, err := l.mgr.CreateWorkItem(ctx, l.waiting(15)); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	if err := l.mgr.DeleteWorkflow(ctx, l.cancel.ID); err != nil {
		t.Fatalf("DeleteWorkflow: %v", err)
	}
	issues, err := l.mgr.ListIssues(ctx, l.draftID, mwanachamaagency.IssueFilter{EntityType: "work_item"})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	var messages []string
	for _, i := range issues {
		messages = append(messages, i.Message)
	}
	if joined := strings.Join(messages, "\n"); !strings.Contains(joined, "cancels with Workflow") {
		t.Fatalf("issues = %q, want one naming the missing cancel workflow", joined)
	}
}
