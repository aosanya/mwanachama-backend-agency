package mwanachamaagency_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type failureWorld struct {
	stepWorld
	db *gorm.DB
	s  *spec.Spec
	wf models.Workflow
}

func newFailureWorld(t *testing.T) failureWorld {
	t.Helper()
	mwanachamaagency.ClearStepKindRegistry()
	mwanachamaagency.RegisterComputeFailureChecks()
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	w := stepWorld{mgr: mgr, draftID: o.DraftID, objective: o, roles: map[string]models.Role{}}
	return failureWorld{stepWorld: w, db: db, s: s, wf: w.workflow(t, "Loan check")}
}

func (w failureWorld) compute(t *testing.T, workflow models.Workflow, title string, failure *models.StepFailure) models.WorkItem {
	t.Helper()
	step, err := w.mgr.CreateWorkItem(context.Background(), models.WorkItem{
		DraftID: w.draftID, WorkflowID: workflow.ID, Title: title, Kind: models.StepKindCompute,
		Function: "fin.pmt@2", ResultName: "Share each month", OnFailure: failure,
	})
	if err != nil {
		t.Fatalf("CreateWorkItem %s: %v", title, err)
	}
	return step
}

func (w failureWorld) review(t *testing.T, workflow models.Workflow, title string) models.WorkItem {
	t.Helper()
	step, err := w.mgr.CreateWorkItem(context.Background(), models.WorkItem{
		DraftID: w.draftID, WorkflowID: workflow.ID, Title: title,
	})
	if err != nil {
		t.Fatalf("CreateWorkItem %s: %v", title, err)
	}
	return step
}

func (w failureWorld) goTo(target string) *models.StepFailure {
	return &models.StepFailure{Choice: models.FailureGoToStep, StepID: target}
}

func (w failureWorld) stop() *models.StepFailure {
	return &models.StepFailure{Choice: models.FailureStop}
}

func (w failureWorld) setChoice(t *testing.T, step models.WorkItem, failure *models.StepFailure) (models.WorkItem, error) {
	t.Helper()
	current, err := w.mgr.GetWorkItem(context.Background(), step.ID)
	if err != nil {
		t.Fatalf("GetWorkItem: %v", err)
	}
	current.OnFailure = failure
	return w.mgr.UpdateWorkItem(context.Background(), current)
}

func (w failureWorld) plant(t *testing.T, step models.WorkItem, rawChoice string) {
	t.Helper()
	if err := w.db.Exec("update "+tableOf(t, w.s, "work_item")+" set on_failure = ? where id = ?", rawChoice, step.ID).Error; err != nil {
		t.Fatalf("planting the choice: %v", err)
	}
}

func (w failureWorld) errors(t *testing.T) []string {
	t.Helper()
	issues, err := w.mgr.ListIssues(context.Background(), w.draftID, mwanachamaagency.IssueFilter{Severity: string(models.IssueSeverityError)})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	messages := make([]string, 0, len(issues))
	for _, issue := range issues {
		messages = append(messages, issue.Message)
	}
	return messages
}

func (w failureWorld) errorsAbout(t *testing.T, needle string) []string {
	t.Helper()
	var found []string
	for _, message := range w.errors(t) {
		if strings.Contains(message, needle) {
			found = append(found, message)
		}
	}
	return found
}

func wantRefusal(t *testing.T, err error, sentinel error, words string) {
	t.Helper()
	if !errors.Is(err, sentinel) {
		t.Fatalf("got %v, want %v", err, sentinel)
	}
	if !strings.Contains(err.Error(), words) {
		t.Fatalf("the sentence %q does not say %q", err.Error(), words)
	}
}

func TestComputeFailure_SaveRefusesAChoiceThatIsNeitherStopNorGoToStep(t *testing.T) {
	w := newFailureWorld(t)
	_, err := w.mgr.CreateWorkItem(context.Background(), models.WorkItem{
		DraftID: w.draftID, WorkflowID: w.wf.ID, Title: "Check the loan payment", Kind: models.StepKindCompute,
		OnFailure: &models.StepFailure{Choice: "retry"},
	})
	wantRefusal(t, err, mwanachamaagency.ErrInvalidWorkItem, `the failure choice "retry" is neither stop nor go_to_step`)
}

func TestComputeFailure_SaveRefusesGoToStepWithNoStep(t *testing.T) {
	w := newFailureWorld(t)
	_, err := w.mgr.CreateWorkItem(context.Background(), models.WorkItem{
		DraftID: w.draftID, WorkflowID: w.wf.ID, Title: "Check the loan payment", Kind: models.StepKindCompute,
		OnFailure: &models.StepFailure{Choice: models.FailureGoToStep},
	})
	wantRefusal(t, err, mwanachamaagency.ErrInvalidWorkItem, "go_to_step names the step to go to")
}

func TestComputeFailure_SaveRefusesStopCarryingAStep(t *testing.T) {
	w := newFailureWorld(t)
	_, err := w.mgr.CreateWorkItem(context.Background(), models.WorkItem{
		DraftID: w.draftID, WorkflowID: w.wf.ID, Title: "Check the loan payment", Kind: models.StepKindCompute,
		OnFailure: &models.StepFailure{Choice: models.FailureStop, StepID: "st-7"},
	})
	wantRefusal(t, err, mwanachamaagency.ErrInvalidWorkItem, "stop names no step, got st-7")
}

func TestComputeFailure_SaveRefusesAFailureChoiceOnAStepThatIsNotCalculating(t *testing.T) {
	w := newFailureWorld(t)
	_, err := w.mgr.CreateWorkItem(context.Background(), models.WorkItem{
		DraftID: w.draftID, WorkflowID: w.wf.ID, Title: "Tell the member", Kind: models.StepKindNotify,
		OnFailure: w.stop(),
	})
	wantRefusal(t, err, mwanachamaagency.ErrInvalidWorkItem, "a notify step has no failure choice; only a compute step can say what happens when it cannot work out a result")
}

func TestComputeFailure_SaveRefusesAFailureChoiceOnAStepWithNoKind(t *testing.T) {
	w := newFailureWorld(t)
	_, err := w.mgr.CreateWorkItem(context.Background(), models.WorkItem{
		DraftID: w.draftID, WorkflowID: w.wf.ID, Title: "Plain step", OnFailure: w.stop(),
	})
	wantRefusal(t, err, mwanachamaagency.ErrInvalidWorkItem, "only a compute step can say what happens when it cannot work out a result")
}

func TestComputeFailure_SaveRefusesAStepOfAnotherWorkflow(t *testing.T) {
	w := newFailureWorld(t)
	savings := w.workflow(t, "Savings check")
	elsewhere := w.review(t, savings, "Review the savings")
	_, err := w.mgr.CreateWorkItem(context.Background(), models.WorkItem{
		DraftID: w.draftID, WorkflowID: w.wf.ID, Title: "Check the loan payment", Kind: models.StepKindCompute,
		OnFailure: w.goTo(elsewhere.ID),
	})
	wantRefusal(t, err, mwanachamaagency.ErrInvalidWorkItem, "belongs to another workflow")
}

func TestComputeFailure_SaveRefusesTheStepItself(t *testing.T) {
	w := newFailureWorld(t)
	step := w.compute(t, w.wf, "Check the loan payment", nil)
	_, err := w.setChoice(t, step, w.goTo(step.ID))
	wantRefusal(t, err, mwanachamaagency.ErrInvalidWorkItem, "a step cannot go to itself on failure")
}

func TestComputeFailure_SaveReportsAStepOfNoDraftAsNotFound(t *testing.T) {
	w := newFailureWorld(t)
	_, err := w.mgr.CreateWorkItem(context.Background(), models.WorkItem{
		DraftID: w.draftID, WorkflowID: w.wf.ID, Title: "Check the loan payment", Kind: models.StepKindCompute,
		OnFailure: w.goTo("st-404"),
	})
	wantRefusal(t, err, mwanachamaagency.ErrWorkItemNotFound, "st-404")
}

func TestComputeFailure_SaveAllowsNoChoice(t *testing.T) {
	w := newFailureWorld(t)
	step := w.compute(t, w.wf, "Check the loan payment", nil)
	got, err := w.mgr.GetWorkItem(context.Background(), step.ID)
	if err != nil || got.OnFailure != nil {
		t.Fatalf("a step saved with no choice must read back with none, got %v %v", got.OnFailure, err)
	}
}

func TestComputeFailure_SaveAllowsStopAndAnyOtherStepOfTheWorkflow(t *testing.T) {
	w := newFailureWorld(t)
	review := w.review(t, w.wf, "Review the loan payment")
	term := w.compute(t, w.wf, "Check the term", w.stop())
	for _, target := range []string{review.ID, term.ID} {
		step := w.compute(t, w.wf, "Check the loan payment", nil)
		got, err := w.setChoice(t, step, w.goTo(target))
		if err != nil {
			t.Fatalf("a target of %s must be allowed: %v", target, err)
		}
		if got.OnFailure == nil || got.OnFailure.StepID != target {
			t.Fatalf("the choice was not kept: %+v", got.OnFailure)
		}
	}
	if _, err := w.setChoice(t, term, w.stop()); err != nil {
		t.Fatalf("stop must be allowed: %v", err)
	}
}

func TestComputeFailure_ScanFlagsAStepWithNoChoice(t *testing.T) {
	w := newFailureWorld(t)
	step := w.compute(t, w.wf, "Check the loan payment", nil)
	want := fmt.Sprintf("Work item %s %q has no answer for a failure; choose stop or a step to go to", step.Code, step.Title)
	if got := w.errorsAbout(t, step.Code); len(got) != 1 || got[0] != want {
		t.Fatalf("got %q, want exactly %q", got, want)
	}
}

func TestComputeFailure_ScanLeavesAStepWithAChoiceAlone(t *testing.T) {
	w := newFailureWorld(t)
	review := w.review(t, w.wf, "Review the loan payment")
	stopping := w.compute(t, w.wf, "Check the term", w.stop())
	routing := w.compute(t, w.wf, "Check the loan payment", w.goTo(review.ID))
	for _, step := range []models.WorkItem{stopping, routing} {
		for _, message := range w.errorsAbout(t, step.Code+" ") {
			if strings.Contains(message, "failure") || strings.Contains(message, "fail onto") {
				t.Fatalf("a step with a choice was flagged: %s", message)
			}
		}
	}
}

func TestComputeFailure_ScanFlagsAStepThatNoLongerExists(t *testing.T) {
	w := newFailureWorld(t)
	review := w.review(t, w.wf, "Review the loan payment")
	step := w.compute(t, w.wf, "Check the loan payment", w.goTo(review.ID))
	if err := w.mgr.DeleteWorkItem(context.Background(), review.ID); err != nil {
		t.Fatalf("DeleteWorkItem: %v", err)
	}
	want := fmt.Sprintf("Work item %s %q goes on failure to Step %s, which does not exist", step.Code, step.Title, review.ID)
	if got := w.errorsAbout(t, step.Code); len(got) != 1 || got[0] != want {
		t.Fatalf("got %q, want exactly %q", got, want)
	}
}

func TestComputeFailure_ScanFlagsAStepNowInAnotherWorkflow(t *testing.T) {
	w := newFailureWorld(t)
	savings := w.workflow(t, "Savings check")
	elsewhere := w.review(t, savings, "Review the savings")
	step := w.compute(t, w.wf, "Check the loan payment", nil)
	w.plant(t, step, fmt.Sprintf(`{"choice":"go_to_step","step_id":%q}`, elsewhere.ID))
	want := fmt.Sprintf("Work item %s %q goes on failure to Step %s, which belongs to another workflow", step.Code, step.Title, elsewhere.ID)
	if got := w.errorsAbout(t, step.Code); len(got) != 1 || got[0] != want {
		t.Fatalf("got %q, want exactly %q", got, want)
	}
}

func TestComputeFailure_ScanFlagsAStepThatPointsAtItself(t *testing.T) {
	w := newFailureWorld(t)
	step := w.compute(t, w.wf, "Check the loan payment", nil)
	w.plant(t, step, fmt.Sprintf(`{"choice":"go_to_step","step_id":%q}`, step.ID))
	want := fmt.Sprintf("Work item %s %q goes on failure to itself", step.Code, step.Title)
	got := w.errorsAbout(t, step.Code)
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %q, want exactly %q (one error, not one more for a loop of one)", got, want)
	}
}

func TestComputeFailure_ScanRaisesOneErrorForALoopOfTwo(t *testing.T) {
	w := newFailureWorld(t)
	term := w.compute(t, w.wf, "Check the term", nil)
	payment := w.compute(t, w.wf, "Check the loan payment", w.goTo(term.ID))
	if _, err := w.setChoice(t, term, w.goTo(payment.ID)); err != nil {
		t.Fatalf("both saves must succeed: %v", err)
	}
	loops := w.errorsAbout(t, "fail onto each other")
	if len(loops) != 1 {
		t.Fatalf("want one error for the loop, got %q", loops)
	}
	for _, step := range []models.WorkItem{payment, term} {
		if !strings.Contains(loops[0], fmt.Sprintf("%s %q", step.Code, step.Title)) {
			t.Fatalf("the loop sentence %q does not name %s %q", loops[0], step.Code, step.Title)
		}
	}
	if !strings.HasSuffix(loops[0], "fail onto each other with no way out; one of them must stop or go to a step outside the loop") {
		t.Fatalf("the loop sentence is %q", loops[0])
	}
}

func TestComputeFailure_ScanRaisesNothingForALoopWithAStopInIt(t *testing.T) {
	w := newFailureWorld(t)
	term := w.compute(t, w.wf, "Check the term", nil)
	payment := w.compute(t, w.wf, "Check the loan payment", w.goTo(term.ID))
	if _, err := w.setChoice(t, term, w.goTo(payment.ID)); err != nil {
		t.Fatalf("setChoice: %v", err)
	}
	if _, err := w.setChoice(t, term, w.stop()); err != nil {
		t.Fatalf("setChoice: %v", err)
	}
	if loops := w.errorsAbout(t, "fail onto each other"); len(loops) != 0 {
		t.Fatalf("a way out was left, got %q", loops)
	}
}

func TestComputeFailure_ScanRaisesNothingForALoopThatLeavesToAnotherKindOfStep(t *testing.T) {
	w := newFailureWorld(t)
	review := w.review(t, w.wf, "Review the loan payment")
	term := w.compute(t, w.wf, "Check the term", nil)
	payment := w.compute(t, w.wf, "Check the loan payment", w.goTo(term.ID))
	if _, err := w.setChoice(t, term, w.goTo(payment.ID)); err != nil {
		t.Fatalf("setChoice: %v", err)
	}
	if _, err := w.setChoice(t, term, w.goTo(review.ID)); err != nil {
		t.Fatalf("setChoice: %v", err)
	}
	if loops := w.errorsAbout(t, "fail onto each other"); len(loops) != 0 {
		t.Fatalf("a way out was left, got %q", loops)
	}
}

func TestComputeFailure_ScanNamesOnlyTheLoopNotTheStepThatFeedsIt(t *testing.T) {
	w := newFailureWorld(t)
	last := w.compute(t, w.wf, "Check the last month", nil)
	term := w.compute(t, w.wf, "Check the term", w.goTo(last.ID))
	if _, err := w.setChoice(t, last, w.goTo(term.ID)); err != nil {
		t.Fatalf("setChoice: %v", err)
	}
	feeder := w.compute(t, w.wf, "Check the loan payment", w.goTo(term.ID))
	loops := w.errorsAbout(t, "fail onto each other")
	if len(loops) != 1 {
		t.Fatalf("want one error for the loop, got %q", loops)
	}
	if strings.Contains(loops[0], feeder.Code+" ") {
		t.Fatalf("the step that only feeds the loop is not in it: %s", loops[0])
	}
}

func TestComputeFailure_ScanRaisesOneErrorForALoopOfThree(t *testing.T) {
	w := newFailureWorld(t)
	a := w.compute(t, w.wf, "Check the balance", nil)
	b := w.compute(t, w.wf, "Check the payment", w.goTo(a.ID))
	c := w.compute(t, w.wf, "Check the term", w.goTo(b.ID))
	if _, err := w.setChoice(t, a, w.goTo(c.ID)); err != nil {
		t.Fatalf("setChoice: %v", err)
	}
	if loops := w.errorsAbout(t, "fail onto each other"); len(loops) != 1 {
		t.Fatalf("want one error for the loop, got %q", loops)
	}
}

func TestComputeFailure_PublishIsRefusedForNoChoiceAndForALoop(t *testing.T) {
	w := newFailureWorld(t)
	step := w.compute(t, w.wf, "Check the loan payment", nil)
	_, err := w.mgr.PublishAgency(context.Background(), w.draftID, "", "")
	if !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) || !strings.Contains(err.Error(), "has no answer for a failure") {
		t.Fatalf("a step with no choice must block publishing, got %v", err)
	}
	other := w.compute(t, w.wf, "Check the term", w.goTo(step.ID))
	if _, err := w.setChoice(t, step, w.goTo(other.ID)); err != nil {
		t.Fatalf("setChoice: %v", err)
	}
	_, err = w.mgr.PublishAgency(context.Background(), w.draftID, "", "")
	if !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) || !strings.Contains(err.Error(), "fail onto each other") {
		t.Fatalf("a loop must block publishing, got %v", err)
	}
}

func TestComputeFailure_AStepOfAnotherKindIsNeverFlaggedForNoChoice(t *testing.T) {
	w := newFailureWorld(t)
	review := w.review(t, w.wf, "Review the loan payment")
	if got := w.errorsAbout(t, review.Code+" "); len(got) != 0 {
		for _, message := range got {
			if strings.Contains(message, "answer for a failure") {
				t.Fatalf("only a calculating step needs a choice: %s", message)
			}
		}
	}
}
