package mwanachamaagency_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestPostStep_SavedWithOnlyAnAccount(t *testing.T) {
	w := newPostHoldWorld(t)
	w.workOutThePayment(t, 1, nil)
	if _, err := w.step("Post the payment", 2, models.StepKindPost, func(wi *models.WorkItem) { wi.Account = "Member loans" }); err != nil {
		t.Fatalf("a post step may be saved before its amount step, counter account and direction are picked: %v", err)
	}
}

func TestPostStep_CompleteStepSavesAndScansClean(t *testing.T) {
	w := newPostHoldWorld(t)
	calc := w.workOutThePayment(t, 1, nil)
	post := w.postThePayment(t, 2, calc.ID, nil)
	requireNoErrors(t, w.errorsAbout(t, post.ID))
	requireNoErrors(t, w.errorsAbout(t, calc.ID))
}

func TestPostStep_SaveRefusals(t *testing.T) {
	cases := []struct {
		name   string
		change func(t *testing.T, w postHoldWorld, calc models.WorkItem) func(*models.WorkItem)
		want   error
		says   string
	}{
		{"a direction that is neither debit nor credit (P6)", func(t *testing.T, w postHoldWorld, calc models.WorkItem) func(*models.WorkItem) {
			return func(wi *models.WorkItem) { wi.Direction = "credit side" }
		}, mwanachamaagency.ErrInvalidWorkItem, `the direction of a post step is debit or credit, not "credit side"`},
		{"a blank reason", func(t *testing.T, w postHoldWorld, calc models.WorkItem) func(*models.WorkItem) {
			return func(wi *models.WorkItem) { wi.Reason = "   " }
		}, mwanachamaagency.ErrInvalidWorkItem, "the reason of a post step is written in words, not left blank"},
		{"the same account on both sides (P10)", func(t *testing.T, w postHoldWorld, calc models.WorkItem) func(*models.WorkItem) {
			return func(wi *models.WorkItem) { wi.Account, wi.CounterAccount = "Member loans", "Member loans" }
		}, mwanachamaagency.ErrInvalidWorkItem, `a post step debits one account and credits another, so "Member loans" cannot be both sides`},
		{"a function on a post step", func(t *testing.T, w postHoldWorld, calc models.WorkItem) func(*models.WorkItem) {
			return func(wi *models.WorkItem) { wi.Function = "loan.cap@3" }
		}, mwanachamaagency.ErrInvalidWorkItem, "a post step takes no function"},
		{"a failure choice on a post step", func(t *testing.T, w postHoldWorld, calc models.WorkItem) func(*models.WorkItem) {
			return func(wi *models.WorkItem) { wi.OnFailure = &models.StepFailure{Choice: models.FailureStop} }
		}, mwanachamaagency.ErrInvalidWorkItem, "a post step"},
		{"a tool on a post step", func(t *testing.T, w postHoldWorld, calc models.WorkItem) func(*models.WorkItem) {
			return func(wi *models.WorkItem) { wi.Tool = "market_ask"; wi.RoleID = w.roles[coordinator].ID }
		}, mwanachamaagency.ErrInvalidWorkItem, "a post step names no tool"},
		{"a form on a post step", func(t *testing.T, w postHoldWorld, calc models.WorkItem) func(*models.WorkItem) {
			return func(wi *models.WorkItem) { wi.FormName = "Loan form" }
		}, mwanachamaagency.ErrInvalidWorkItem, "takes no form"},
		{"a recipient on a post step", func(t *testing.T, w postHoldWorld, calc models.WorkItem) func(*models.WorkItem) {
			return func(wi *models.WorkItem) { wi.RecipientRoleID = w.roles[farmer].ID }
		}, mwanachamaagency.ErrInvalidWorkItem, "only a send_form or notify step goes to a recipient"},
		{"routes on a post step", func(t *testing.T, w postHoldWorld, calc models.WorkItem) func(*models.WorkItem) {
			return func(wi *models.WorkItem) { wi.OtherwiseStepID = calc.ID }
		}, mwanachamaagency.ErrInvalidWorkItem, "only a route step has routes or an otherwise"},
		{"chat fields on a post step", func(t *testing.T, w postHoldWorld, calc models.WorkItem) func(*models.WorkItem) {
			return func(wi *models.WorkItem) { wi.Topic = "Loans" }
		}, mwanachamaagency.ErrInvalidWorkItem, "only a chat step names an agent"},
		{"a workflow to run on a post step", func(t *testing.T, w postHoldWorld, calc models.WorkItem) func(*models.WorkItem) {
			return func(wi *models.WorkItem) { wi.RunWorkflowID = w.wf.ID }
		}, mwanachamaagency.ErrInvalidWorkItem, "only a run_workflow step names a workflow to run"},
		{"an amount step that is not a step of the draft (P9)", func(t *testing.T, w postHoldWorld, calc models.WorkItem) func(*models.WorkItem) {
			return func(wi *models.WorkItem) { wi.AmountStepID = "st-404" }
		}, mwanachamaagency.ErrWorkItemNotFound, "the amount step st-404 is not a step of this draft"},
		{"an amount step that is not a calculating step (P7)", func(t *testing.T, w postHoldWorld, calc models.WorkItem) func(*models.WorkItem) {
			apply := w.mustStep(t, "Apply for a loan", 0, models.StepKindFillForm, func(wi *models.WorkItem) {
				wi.RoleID = w.roles[farmer].ID
				withForm(wi)
			})
			return func(wi *models.WorkItem) { wi.AmountStepID = apply.ID }
		}, mwanachamaagency.ErrInvalidWorkItem, `the amount step "Apply for a loan" is not a calculating step`},
		{"an amount step in another workflow", func(t *testing.T, w postHoldWorld, calc models.WorkItem) func(*models.WorkItem) {
			other := w.workflow(t, "Loan review")
			elsewhere, err := w.mgr.CreateWorkItem(context.Background(), models.WorkItem{
				DraftID: w.draftID, WorkflowID: other.ID, Title: "Work out the review", Ordinality: 1, Kind: models.StepKindCompute,
			})
			if err != nil {
				t.Fatalf("CreateWorkItem: %v", err)
			}
			return func(wi *models.WorkItem) { wi.AmountStepID = elsewhere.ID }
		}, mwanachamaagency.ErrInvalidWorkItem, `the amount step "Work out the review" belongs to another workflow`},
		{"an amount step that is not earlier", func(t *testing.T, w postHoldWorld, calc models.WorkItem) func(*models.WorkItem) {
			return func(wi *models.WorkItem) { wi.Ordinality = calc.Ordinality }
		}, mwanachamaagency.ErrInvalidWorkItem, `the amount step "Work out the payment" is not earlier than this step`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := newPostHoldWorld(t)
			calc := w.workOutThePayment(t, 1, nil)
			change := tc.change(t, w, calc)
			_, err := w.step("Post the payment", 2, models.StepKindPost, func(wi *models.WorkItem) {
				wi.AmountStepID = calc.ID
				wi.Account = "Member loans"
				wi.CounterAccount = "Cash"
				wi.Direction = "debit"
				wi.Reason = "Loan payment"
				change(wi)
			})
			if !errors.Is(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
			if !strings.Contains(err.Error(), tc.says) {
				t.Fatalf("error %q does not say %q", err, tc.says)
			}
		})
	}
}

func TestPostStep_AStepCannotTakeItsAmountFromItself(t *testing.T) {
	w := newPostHoldWorld(t)
	calc := w.workOutThePayment(t, 1, nil)
	post := w.postThePayment(t, 2, calc.ID, nil)
	post.AmountStepID = post.ID
	_, err := w.mgr.UpdateWorkItem(context.Background(), post)
	if !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) || !strings.Contains(err.Error(), "a post step cannot take its amount from itself") {
		t.Fatalf("got %v", err)
	}
}

func TestPostStep_PostFieldsBelongToAPostStepOnly(t *testing.T) {
	w := newPostHoldWorld(t)
	for _, kind := range []models.StepKind{models.StepKindCompute, models.StepKindNotify, ""} {
		_, err := w.step("Not a post step", 1, kind, func(wi *models.WorkItem) {
			if kind == models.StepKindNotify {
				withForm(wi)
			}
			wi.Account = "Member loans"
		})
		if !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) || !strings.Contains(err.Error(), "belong to a post step") {
			t.Fatalf("kind %q: got %v", kind, err)
		}
	}
}

func TestPostStep_AnAccountTheAgencyDoesNotHaveIsRefused(t *testing.T) {
	w := newPostHoldWorld(t)
	mwanachamaagency.SetAccountDirectory(lakesideAccounts{"Member loans": true, "Cash": true})
	calc := w.workOutThePayment(t, 1, nil)
	_, err := w.step("Post the payment", 2, models.StepKindPost, func(wi *models.WorkItem) {
		wi.AmountStepID = calc.ID
		wi.Account = "Member savings"
		wi.CounterAccount = "Cash"
		wi.Direction = "credit"
		wi.Reason = "Loan payment"
	})
	if !errors.Is(err, mwanachamaagency.ErrAccountNotFound) || !strings.Contains(err.Error(), `the agency has no account "Member savings"`) {
		t.Fatalf("got %v", err)
	}
	_, err = w.step("Post the payment", 2, models.StepKindPost, func(wi *models.WorkItem) {
		wi.AmountStepID = calc.ID
		wi.Account = "Member loans"
		wi.CounterAccount = "Petty cash"
	})
	if !errors.Is(err, mwanachamaagency.ErrAccountNotFound) || !strings.Contains(err.Error(), `the agency has no account "Petty cash"`) {
		t.Fatalf("the counter account is checked too, got %v", err)
	}
	if _, err := w.step("Post the payment", 2, models.StepKindPost, func(wi *models.WorkItem) {
		wi.AmountStepID = calc.ID
		wi.Account = "Member loans"
		wi.CounterAccount = "Cash"
	}); err != nil {
		t.Fatalf("accounts the agency has must save: %v", err)
	}
}

func TestPostStep_ScanOfAStepWithNothingButAnAccount(t *testing.T) {
	w := newPostHoldWorld(t)
	w.workOutThePayment(t, 1, nil)
	post := w.mustStep(t, "Post the payment", 2, models.StepKindPost, func(wi *models.WorkItem) { wi.Account = "Member loans" })

	messages := w.errorsAbout(t, post.ID)
	if len(messages) != 4 {
		t.Fatalf("want 4 errors (P5), got %d: %q", len(messages), messages)
	}
	wants := []string{
		"is a post step with no amount source: it names no calculating step",
		"is a post step with no counter account",
		"is a post step with no direction",
		"is a post step with no reason",
	}
	for _, want := range wants {
		found := false
		for _, message := range messages {
			found = found || strings.Contains(message, want)
		}
		if !found {
			t.Fatalf("no error says %q in %q", want, messages)
		}
	}
}

func TestPostStep_ScanFollowsTheAmountStepAfterItChanges(t *testing.T) {
	ctx := context.Background()
	t.Run("the calculating step is deleted", func(t *testing.T) {
		w := newPostHoldWorld(t)
		calc := w.workOutThePayment(t, 1, nil)
		post := w.postThePayment(t, 2, calc.ID, nil)
		if err := w.mgr.DeleteWorkItem(ctx, calc.ID); err != nil {
			t.Fatalf("DeleteWorkItem: %v", err)
		}
		requireOneError(t, w.errorsAbout(t, post.ID), "which does not exist")
	})
	t.Run("the calculating step becomes another kind", func(t *testing.T) {
		w := newPostHoldWorld(t)
		calc := w.workOutThePayment(t, 1, nil)
		post := w.postThePayment(t, 2, calc.ID, nil)
		calc.Kind = models.StepKindNotify
		calc.Function, calc.ResultName = "", ""
		withForm(&calc)
		if _, err := w.mgr.UpdateWorkItem(ctx, calc); err != nil {
			t.Fatalf("UpdateWorkItem: %v", err)
		}
		requireOneError(t, w.errorsAbout(t, post.ID), "which is not a calculating step")
	})
	t.Run("the calculating step is no longer earlier", func(t *testing.T) {
		w := newPostHoldWorld(t)
		calc := w.workOutThePayment(t, 1, nil)
		post := w.postThePayment(t, 2, calc.ID, nil)
		calc.Ordinality = 3
		if _, err := w.mgr.UpdateWorkItem(ctx, calc); err != nil {
			t.Fatalf("UpdateWorkItem: %v", err)
		}
		requireOneError(t, w.errorsAbout(t, post.ID), "which is not earlier than it")
	})
	t.Run("an account the agency no longer has", func(t *testing.T) {
		w := newPostHoldWorld(t)
		accounts := lakesideAccounts{"Member loans": true, "Cash": true}
		mwanachamaagency.SetAccountDirectory(accounts)
		calc := w.workOutThePayment(t, 1, nil)
		post := w.postThePayment(t, 2, calc.ID, nil)
		requireNoErrors(t, w.errorsAbout(t, post.ID))
		delete(accounts, "Cash")
		requireOneError(t, w.errorsAbout(t, post.ID), `posts to account "Cash", which the agency does not have`)
	})
}

func TestPostStep_AnUnpublishableDraftRefusesToPublish(t *testing.T) {
	w := newPostHoldWorld(t)
	w.workOutThePayment(t, 1, nil)
	w.mustStep(t, "Post the payment", 2, models.StepKindPost, func(wi *models.WorkItem) { wi.Account = "Member loans" })
	_, err := w.mgr.PublishAgency(context.Background(), w.draftID, "", "")
	if !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) {
		t.Fatalf("P5: the draft with a bare post step must refuse to publish, got %v", err)
	}
}
