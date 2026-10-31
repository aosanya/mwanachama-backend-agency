package mwanachamaagency_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func joinsYes(goTo string) models.StepRoute {
	return models.StepRoute{
		Match:    models.RouteMatchAll,
		Rules:    []models.RouteRule{{Question: joinQuestion, Operator: models.RouteIs, Value: "yes"}},
		GoToStep: goTo,
	}
}

func TestRouteStep_NamesOnlyStepsThatExist(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	w := o.world
	route := o.steps["Sort the answers"]
	open := o.steps["Open"]

	missingRoute := route
	missingRoute.Routes = []models.StepRoute{joinsYes("no-such-step")}
	if _, err := w.mgr.UpdateWorkItem(ctx, missingRoute); !errors.Is(err, mwanachamaagency.ErrWorkItemNotFound) {
		t.Errorf("a route to a step that does not exist: want ErrWorkItemNotFound, got %v", err)
	}

	missingOtherwise := route
	missingOtherwise.OtherwiseStepID = "no-such-step"
	if _, err := w.mgr.UpdateWorkItem(ctx, missingOtherwise); !errors.Is(err, mwanachamaagency.ErrWorkItemNotFound) {
		t.Errorf("an otherwise to a step that does not exist: want ErrWorkItemNotFound, got %v", err)
	}

	elsewhere := w.workflow(t, "Buy now")
	stranger, err := w.mgr.CreateWorkItem(ctx, w.item(elsewhere, 1, buyNowSteps[0]))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	foreignRoute := route
	foreignRoute.Routes = []models.StepRoute{joinsYes(stranger.ID)}
	if _, err := w.mgr.UpdateWorkItem(ctx, foreignRoute); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Errorf("a route to a step of another workflow: want ErrInvalidWorkItem, got %v", err)
	}
	foreignOtherwise := route
	foreignOtherwise.OtherwiseStepID = stranger.ID
	if _, err := w.mgr.UpdateWorkItem(ctx, foreignOtherwise); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Errorf("an otherwise to a step of another workflow: want ErrInvalidWorkItem, got %v", err)
	}

	toSelf := route
	toSelf.Routes = []models.StepRoute{joinsYes(route.ID)}
	if _, err := w.mgr.UpdateWorkItem(ctx, toSelf); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Errorf("a route to its own step: want ErrInvalidWorkItem, got %v", err)
	}

	ok := route
	ok.Routes = []models.StepRoute{joinsYes(open.ID)}
	ok.OtherwiseStepID = ""
	if _, err := w.mgr.UpdateWorkItem(ctx, ok); err != nil {
		t.Errorf("a route to a step that exists and no otherwise: %v", err)
	}
	reread, _ := w.mgr.GetWorkItem(ctx, route.ID)
	if reread.OtherwiseStepID != "" || reread.Routes[0].GoToStep != open.ID {
		t.Errorf("route step after the accepted update = %+v", reread)
	}
}

func TestRouteStep_RulesAreReadFromTheFormThatWasSent(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	w := o.world
	route := o.steps["Sort the answers"]
	open := o.steps["Open"].ID

	with := func(mutate func(*models.StepRoute)) models.WorkItem {
		in := route
		r := joinsYes(open)
		mutate(&r)
		in.Routes = []models.StepRoute{r}
		return in
	}
	invalid := map[string]models.WorkItem{
		"a question the sent form does not ask": with(func(r *models.StepRoute) { r.Rules[0].Question = "Colour?" }),
		"an unknown operator":                   with(func(r *models.StepRoute) { r.Rules[0].Operator = "resembles" }),
		"no value":                              with(func(r *models.StepRoute) { r.Rules[0].Value = " " }),
		"a number operator with a word":         with(func(r *models.StepRoute) { r.Rules[0].Operator = models.RouteMoreThan }),
		"neither all nor any":                   with(func(r *models.StepRoute) { r.Match = "most" }),
		"a route with no rule":                  with(func(r *models.StepRoute) { r.Rules = nil }),
		"a route going nowhere":                 with(func(r *models.StepRoute) { r.GoToStep = "" }),
	}
	noForm := route
	noForm.SentStepID = ""
	invalid["routes but no form that was sent"] = noForm
	notSent := route
	notSent.SentStepID = o.steps["Start"].ID
	invalid["a form that was not sent"] = notSent

	for name, in := range invalid {
		if _, err := w.mgr.UpdateWorkItem(ctx, in); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
			t.Errorf("%s: want ErrInvalidWorkItem, got %v", name, err)
		}
	}

	any := with(func(r *models.StepRoute) {
		r.Match = models.RouteMatchAny
		r.Rules = append(r.Rules, models.RouteRule{Question: joinQuestion, Operator: models.RouteContains, Value: "y"})
	})
	if _, err := w.mgr.UpdateWorkItem(ctx, any); err != nil {
		t.Errorf("a route matching any of two rules: %v", err)
	}
}

func TestRouteStep_OnlyARouteStepHasRoutes(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	w := o.world
	for _, title := range []string{"Start", "Send the invitation", "Open", "Send"} {
		in := o.steps[title]
		in.Routes = []models.StepRoute{joinsYes(o.steps["Open"].ID)}
		if _, err := w.mgr.UpdateWorkItem(ctx, in); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
			t.Errorf("%s with routes: want ErrInvalidWorkItem, got %v", title, err)
		}
		in = o.steps[title]
		in.OtherwiseStepID = o.steps["Open"].ID
		if _, err := w.mgr.UpdateWorkItem(ctx, in); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
			t.Errorf("%s with an otherwise: want ErrInvalidWorkItem, got %v", title, err)
		}
	}
}

func TestSendFormStep_GoesToOneRoleThatExists(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	w := o.world
	send := o.steps["Send the invitation"]

	send.RecipientRoleID = "no-such-role"
	if _, err := w.mgr.UpdateWorkItem(ctx, send); !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Errorf("a recipient that does not exist: want ErrRoleNotFound, got %v", err)
	}
	send.RecipientRoleID = w.roles[farmer].ID + "," + w.roles[supplier].ID
	if _, err := w.mgr.UpdateWorkItem(ctx, send); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Errorf("two recipients: want ErrInvalidWorkItem, got %v", err)
	}
	send.RecipientRoleID = ""
	if _, err := w.mgr.UpdateWorkItem(ctx, send); err != nil {
		t.Errorf("a send form to everyone in the group: %v", err)
	}

	start := o.steps["Start"]
	start.RecipientRoleID = w.roles[farmer].ID
	if _, err := w.mgr.UpdateWorkItem(ctx, start); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Errorf("a fill_form step with a recipient: want ErrInvalidWorkItem, got %v", err)
	}
}

func TestAnsweringStep_AnswersASentFormOfItsWorkflow(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	w := o.world
	answer := o.steps["Answer the invitation"]

	withOwnForm := answer
	withOwnForm.FormName, withOwnForm.Questions = "The farmer's answer", []models.StepQuestion{ask(joinQuestion, models.AnswerYesNo)}
	if _, err := w.mgr.UpdateWorkItem(ctx, withOwnForm); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Errorf("an answering step with a form of its own: want ErrInvalidWorkItem, got %v", err)
	}
	notSent := answer
	notSent.SentStepID = o.steps["Start"].ID
	if _, err := w.mgr.UpdateWorkItem(ctx, notSent); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Errorf("answering a step that sends nothing: want ErrInvalidWorkItem, got %v", err)
	}
	missing := answer
	missing.SentStepID = "no-such-step"
	if _, err := w.mgr.UpdateWorkItem(ctx, missing); !errors.Is(err, mwanachamaagency.ErrWorkItemNotFound) {
		t.Errorf("answering a step that does not exist: want ErrWorkItemNotFound, got %v", err)
	}
	elsewhere := w.workflow(t, "Another order")
	otherSend, err := w.mgr.CreateWorkItem(ctx, models.WorkItem{
		DraftID: w.draftID, WorkflowID: elsewhere.ID, Title: "Send",
		Kind: models.StepKindSendForm, FormName: "Invitation", Questions: groupOrderSteps[1].questions,
	})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	foreign := answer
	foreign.SentStepID = otherSend.ID
	if _, err := w.mgr.UpdateWorkItem(ctx, foreign); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Errorf("answering a form of another workflow: want ErrInvalidWorkItem, got %v", err)
	}

	fillForm := o.steps["Start"]
	fillForm.SentStepID = o.steps["Send the invitation"].ID
	if _, err := w.mgr.UpdateWorkItem(ctx, fillForm); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Errorf("a fill_form step answering a sent form: want ErrInvalidWorkItem, got %v", err)
	}
}

func TestRouteStep_TheIssueScanFlagsATargetThatIsGone(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	w := o.world
	if err := w.mgr.DeleteWorkItem(ctx, o.steps["Open"].ID); err != nil {
		t.Fatalf("DeleteWorkItem Open: %v", err)
	}
	if err := w.mgr.DeleteWorkItem(ctx, o.steps["Thank you"].ID); err != nil {
		t.Fatalf("DeleteWorkItem Thank you: %v", err)
	}
	if err := w.mgr.DeleteWorkItem(ctx, o.steps["Send the invitation"].ID); err != nil {
		t.Fatalf("DeleteWorkItem Send: %v", err)
	}

	issues, err := w.mgr.ListIssues(ctx, w.draftID, mwanachamaagency.IssueFilter{EntityType: "work_item", Severity: "error"})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	var messages []string
	for _, i := range issues {
		messages = append(messages, i.Message)
	}
	joined := strings.Join(messages, "\n")
	for _, want := range []string{
		"routes to Step " + o.steps["Open"].ID + ", which does not exist",
		"otherwise goes to Step " + o.steps["Thank you"].ID + ", which does not exist",
		"reads the form sent by Step " + o.steps["Send the invitation"].ID + ", which does not exist",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("issues do not say %q:\n%s", want, joined)
		}
	}
}

func TestRouteStep_PublishingIsRefusedWhileATargetIsGone(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	w := o.world
	if err := w.mgr.DeleteWorkItem(ctx, o.steps["Open"].ID); err != nil {
		t.Fatalf("DeleteWorkItem: %v", err)
	}
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "release-1", ""); !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) {
		t.Fatalf("publishing with a dangling route: want ErrPublishValidationFailed, got %v", err)
	}
}

func TestRouteStep_TheIssueScanFlagsARouteStepLeftEmpty(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Group order")
	if _, err := w.mgr.CreateWorkItem(ctx, w.item(wf, 1, groupOrderSteps[3])); err != nil {
		t.Fatalf("a route step still being filled in: %v", err)
	}
	issues, err := w.mgr.ListIssues(ctx, w.draftID, mwanachamaagency.IssueFilter{EntityType: "work_item", Severity: "error"})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	var messages []string
	for _, i := range issues {
		messages = append(messages, i.Message)
	}
	joined := strings.Join(messages, "\n")
	for _, want := range []string{"names no sent form", "has no route"} {
		if !strings.Contains(joined, want) {
			t.Errorf("issues do not say %q:\n%s", want, joined)
		}
	}
}

func TestRouteStep_TheIssueScanFlagsATargetMovedToAnotherWorkflow(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	w := o.world
	elsewhere := w.workflow(t, "Another order")
	open := o.steps["Open"]
	open.WorkflowID = elsewhere.ID
	if _, err := w.mgr.UpdateWorkItem(ctx, open); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}
	issues, err := w.mgr.ListIssues(ctx, w.draftID, mwanachamaagency.IssueFilter{EntityType: "work_item", Severity: "error"})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	for _, i := range issues {
		if strings.Contains(i.Message, "routes to Step "+open.ID+", which belongs to another workflow") {
			return
		}
	}
	t.Fatalf("no issue says the route's target moved to another workflow: %+v", issues)
}

func TestRouteStep_AStepWithNoKindHasNoLinks(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	w := o.world
	plain, err := w.mgr.CreateWorkItem(ctx, models.WorkItem{
		DraftID: w.draftID, WorkflowID: o.workflow.ID, Title: "Write it down",
	})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	for name, mutate := range map[string]func(*models.WorkItem){
		"a recipient":  func(i *models.WorkItem) { i.RecipientRoleID = w.roles[farmer].ID },
		"a sent form":  func(i *models.WorkItem) { i.SentStepID = o.steps["Send the invitation"].ID },
		"an otherwise": func(i *models.WorkItem) { i.OtherwiseStepID = o.steps["Open"].ID },
		"a route":      func(i *models.WorkItem) { i.Routes = []models.StepRoute{joinsYes(o.steps["Open"].ID)} },
	} {
		in := plain
		mutate(&in)
		if _, err := w.mgr.UpdateWorkItem(ctx, in); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
			t.Errorf("a step with no kind and %s: want ErrInvalidWorkItem, got %v", name, err)
		}
	}
}
