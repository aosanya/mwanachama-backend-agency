package mwanachamaagency_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (w postHoldWorld) workOutTheInterest(t *testing.T, ordinality int, days int) models.WorkItem {
	return w.mustStep(t, "Work out the interest", ordinality, models.StepKindCompute, func(wi *models.WorkItem) {
		wi.Function = "loan.interest@1"
		wi.ResultName = "Interest"
		if days > 0 {
			wi.Hold = &models.StepHold{ExpiresAfterDays: days}
		}
	})
}

func (w postHoldWorld) approveTheInterest(t *testing.T, ordinality int, heldStepID string, change func(*models.WorkItem)) models.WorkItem {
	return w.mustStep(t, "Approve the interest", ordinality, models.StepKindYesNo, func(wi *models.WorkItem) {
		wi.RoleID = w.roles[coordinator].ID
		withForm(wi)
		wi.HeldStepID = heldStepID
		if change != nil {
			change(wi)
		}
	})
}

func requireRefused(t *testing.T, err error, want error, says string) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("want %v, got %v", want, err)
	}
	if !strings.Contains(err.Error(), says) {
		t.Fatalf("error %q does not say %q", err, says)
	}
}

func TestHold_AHeldStepReleasedByAnApprovalScansClean(t *testing.T) {
	w := newPostHoldWorld(t)
	calc := w.workOutTheInterest(t, 1, 14)
	approve := w.approveTheInterest(t, 2, calc.ID, nil)
	post := w.postThePayment(t, 3, calc.ID, nil)
	requireNoErrors(t, w.errorsAbout(t, calc.ID))
	requireNoErrors(t, w.errorsAbout(t, approve.ID))
	requireNoErrors(t, w.errorsAbout(t, post.ID))
}

func TestHold_APostStepWithARoleMayReleaseTheFigureItPosts(t *testing.T) {
	w := newPostHoldWorld(t)
	calc := w.workOutTheInterest(t, 1, 14)
	post := w.postThePayment(t, 2, calc.ID, func(wi *models.WorkItem) {
		wi.HeldStepID = calc.ID
		wi.RoleID = w.roles[coordinator].ID
	})
	requireNoErrors(t, w.errorsAbout(t, post.ID))
	requireNoErrors(t, w.errorsAbout(t, calc.ID))
}

func TestHold_AHoldOrAHeldStepOnTheWrongKindIsRefused(t *testing.T) {
	w := newPostHoldWorld(t)
	calc := w.workOutTheInterest(t, 1, 14)

	_, err := w.step("Fill the form", 2, models.StepKindFillForm, func(wi *models.WorkItem) {
		withForm(wi)
		wi.RoleID = w.roles[farmer].ID
		wi.Hold = &models.StepHold{ExpiresAfterDays: 14}
	})
	requireRefused(t, err, mwanachamaagency.ErrInvalidWorkItem, "only a calculating step can hold its result, not a fill_form step")

	_, err = w.step("Tell the farmer", 3, models.StepKindNotify, func(wi *models.WorkItem) {
		withForm(wi)
		wi.HeldStepID = calc.ID
	})
	requireRefused(t, err, mwanachamaagency.ErrInvalidWorkItem, "only a yes_no or post step names the held figure it decides, not a notify step")

	_, err = w.step("A step without a kind", 4, "", func(wi *models.WorkItem) { wi.Hold = &models.StepHold{ExpiresAfterDays: 14} })
	requireRefused(t, err, mwanachamaagency.ErrInvalidWorkItem, "only a calculating step can hold its result")
}

func TestHold_ACalculatingStepThatAlsoNamesARoleCannotHold(t *testing.T) {
	w := newPostHoldWorld(t)
	_, err := w.step("Work out the interest", 1, models.StepKindCompute, func(wi *models.WorkItem) {
		wi.RoleID = w.roles[coordinator].ID
		wi.Hold = &models.StepHold{ExpiresAfterDays: 14}
	})
	requireRefused(t, err, mwanachamaagency.ErrInvalidWorkItem, "either through the role that confirms it or through a hold, not both")
}

func TestHold_TheExpiryIsAWholeNumberOfDaysFromOneToNinety(t *testing.T) {
	for _, days := range []int{0, -3, 91} {
		w := newPostHoldWorld(t)
		_, err := w.step("Work out the interest", 1, models.StepKindCompute, func(wi *models.WorkItem) { wi.Hold = &models.StepHold{ExpiresAfterDays: days} })
		requireRefused(t, err, mwanachamaagency.ErrInvalidWorkItem, "a hold expires after a whole number of days from 1 to 90")
	}
	for _, days := range []int{1, 14, 90} {
		w := newPostHoldWorld(t)
		if _, err := w.step("Work out the interest", 1, models.StepKindCompute, func(wi *models.WorkItem) { wi.Hold = &models.StepHold{ExpiresAfterDays: days} }); err != nil {
			t.Fatalf("%d days must be accepted: %v", days, err)
		}
	}
}

func TestHold_AHeldStepThatNamesTheWrongStepIsRefused(t *testing.T) {
	ctx := context.Background()

	t.Run("a step that is not a calculating step (H9a)", func(t *testing.T) {
		w := newPostHoldWorld(t)
		apply := w.mustStep(t, "Apply for a loan", 1, models.StepKindFillForm, func(wi *models.WorkItem) {
			withForm(wi)
			wi.RoleID = w.roles[farmer].ID
		})
		_, err := w.step("Approve the interest", 2, models.StepKindYesNo, func(wi *models.WorkItem) {
			withForm(wi)
			wi.RoleID = w.roles[coordinator].ID
			wi.HeldStepID = apply.ID
		})
		requireRefused(t, err, mwanachamaagency.ErrInvalidWorkItem, `the held step "Apply for a loan" is not a calculating step`)
	})
	t.Run("a calculating step with no hold (H9b)", func(t *testing.T) {
		w := newPostHoldWorld(t)
		calc := w.workOutTheInterest(t, 1, 0)
		_, err := w.step("Approve the interest", 2, models.StepKindYesNo, func(wi *models.WorkItem) {
			withForm(wi)
			wi.RoleID = w.roles[coordinator].ID
			wi.HeldStepID = calc.ID
		})
		requireRefused(t, err, mwanachamaagency.ErrInvalidWorkItem, `the calculating step "Work out the interest" has no hold`)
	})
	t.Run("a step that is not earlier (H9c)", func(t *testing.T) {
		w := newPostHoldWorld(t)
		calc := w.workOutTheInterest(t, 3, 14)
		_, err := w.step("Approve the interest", 2, models.StepKindYesNo, func(wi *models.WorkItem) {
			withForm(wi)
			wi.RoleID = w.roles[coordinator].ID
			wi.HeldStepID = calc.ID
		})
		requireRefused(t, err, mwanachamaagency.ErrInvalidWorkItem, `the held step "Work out the interest" is not earlier than this step`)
	})
	t.Run("the step itself (H9d)", func(t *testing.T) {
		w := newPostHoldWorld(t)
		calc := w.workOutTheInterest(t, 1, 14)
		approve := w.approveTheInterest(t, 2, calc.ID, nil)
		approve.HeldStepID = approve.ID
		_, err := w.mgr.UpdateWorkItem(ctx, approve)
		requireRefused(t, err, mwanachamaagency.ErrInvalidWorkItem, "a step cannot release its own figure")
	})
	t.Run("a step of another workflow (H9e)", func(t *testing.T) {
		w := newPostHoldWorld(t)
		other := w.workflow(t, "Loan review")
		elsewhere, err := w.mgr.CreateWorkItem(ctx, models.WorkItem{
			DraftID: w.draftID, WorkflowID: other.ID, Title: "Work out the review", Ordinality: 1, Kind: models.StepKindCompute,
			Hold: &models.StepHold{ExpiresAfterDays: 14},
		})
		if err != nil {
			t.Fatalf("CreateWorkItem: %v", err)
		}
		_, err = w.step("Approve the interest", 2, models.StepKindYesNo, func(wi *models.WorkItem) {
			withForm(wi)
			wi.RoleID = w.roles[coordinator].ID
			wi.HeldStepID = elsewhere.ID
		})
		requireRefused(t, err, mwanachamaagency.ErrInvalidWorkItem, `the held step "Work out the review" belongs to another workflow`)
	})
	t.Run("a second step naming the same calculating step (H9f)", func(t *testing.T) {
		w := newPostHoldWorld(t)
		calc := w.workOutTheInterest(t, 1, 14)
		w.approveTheInterest(t, 2, calc.ID, nil)
		_, err := w.step("Approve again", 3, models.StepKindYesNo, func(wi *models.WorkItem) {
			withForm(wi)
			wi.RoleID = w.roles[coordinator].ID
			wi.HeldStepID = calc.ID
		})
		requireRefused(t, err, mwanachamaagency.ErrInvalidWorkItem, "is already released by step")
	})
	t.Run("an update that keeps its own naming is not a second naming", func(t *testing.T) {
		w := newPostHoldWorld(t)
		calc := w.workOutTheInterest(t, 1, 14)
		approve := w.approveTheInterest(t, 2, calc.ID, nil)
		approve.Description = "Approves the interest"
		if _, err := w.mgr.UpdateWorkItem(ctx, approve); err != nil {
			t.Fatalf("UpdateWorkItem: %v", err)
		}
	})
	t.Run("a post step whose held step differs from its amount step (H9g)", func(t *testing.T) {
		w := newPostHoldWorld(t)
		calc := w.workOutTheInterest(t, 1, 14)
		other := w.mustStep(t, "Work out the fee", 2, models.StepKindCompute, func(wi *models.WorkItem) { wi.Function = "loan.fee@1" })
		_, err := w.step("Post the interest", 3, models.StepKindPost, func(wi *models.WorkItem) {
			wi.RoleID = w.roles[coordinator].ID
			wi.HeldStepID = calc.ID
			wi.AmountStepID = other.ID
		})
		requireRefused(t, err, mwanachamaagency.ErrInvalidWorkItem, "its held step and its amount step are one step")
	})
	t.Run("a step that is not in the draft (H9h)", func(t *testing.T) {
		w := newPostHoldWorld(t)
		_, err := w.step("Approve the interest", 2, models.StepKindYesNo, func(wi *models.WorkItem) {
			withForm(wi)
			wi.RoleID = w.roles[coordinator].ID
			wi.HeldStepID = "st-404"
		})
		requireRefused(t, err, mwanachamaagency.ErrWorkItemNotFound, "the held step st-404 is not a step of this draft")
	})
}

func TestHold_ACalculatingStepWithAHoldAndNoReleasingStepCanBeSavedButNotPublished(t *testing.T) {
	w := newPostHoldWorld(t)
	calc := w.workOutTheInterest(t, 1, 14)
	requireOneError(t, w.errorsAbout(t, calc.ID), "holds its result, and no step releases it")
	_, err := w.mgr.PublishAgency(context.Background(), w.draftID, "", "")
	if !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) {
		t.Fatalf("H6: publishing must be refused, got %v", err)
	}
	w.approveTheInterest(t, 2, calc.ID, nil)
	requireNoErrors(t, w.errorsAbout(t, calc.ID))
}

func TestHold_AReleasingStepWithNoRoleIsFlagged(t *testing.T) {
	w := newPostHoldWorld(t)
	calc := w.workOutTheInterest(t, 1, 14)
	approve := w.approveTheInterest(t, 2, calc.ID, nil)
	approve.RoleID = ""
	if _, err := w.mgr.UpdateWorkItem(context.Background(), approve); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}
	requireOneError(t, w.errorsAbout(t, approve.ID), "names no role: the agency cannot approve its own figure")
}

func TestHold_AReleasingStepWhoseRoleWasDeletedIsFlaggedLikeAnyStepWhoseRoleIsGone(t *testing.T) {
	w := newPostHoldWorld(t)
	calc := w.workOutTheInterest(t, 1, 14)
	approve := w.approveTheInterest(t, 2, calc.ID, nil)
	if err := w.mgr.DeleteRole(context.Background(), w.roles[coordinator].ID); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}
	requireOneError(t, w.errorsAbout(t, approve.ID), "which does not exist")
}

func TestHold_TheScanFollowsTheHeldStepAfterItChanges(t *testing.T) {
	ctx := context.Background()
	t.Run("the calculating step is deleted", func(t *testing.T) {
		w := newPostHoldWorld(t)
		calc := w.workOutTheInterest(t, 1, 14)
		approve := w.approveTheInterest(t, 2, calc.ID, nil)
		if err := w.mgr.DeleteWorkItem(ctx, calc.ID); err != nil {
			t.Fatalf("DeleteWorkItem: %v", err)
		}
		requireOneError(t, w.errorsAbout(t, approve.ID), "releases the figure of step", "which does not exist")
	})
	t.Run("the calculating step loses its hold", func(t *testing.T) {
		w := newPostHoldWorld(t)
		calc := w.workOutTheInterest(t, 1, 14)
		approve := w.approveTheInterest(t, 2, calc.ID, nil)
		calc.Hold = nil
		if _, err := w.mgr.UpdateWorkItem(ctx, calc); err != nil {
			t.Fatalf("UpdateWorkItem: %v", err)
		}
		requireOneError(t, w.errorsAbout(t, approve.ID), "which has no hold")
	})
	t.Run("the calculating step is no longer earlier", func(t *testing.T) {
		w := newPostHoldWorld(t)
		calc := w.workOutTheInterest(t, 1, 14)
		approve := w.approveTheInterest(t, 2, calc.ID, nil)
		calc.Ordinality = 5
		if _, err := w.mgr.UpdateWorkItem(ctx, calc); err != nil {
			t.Fatalf("UpdateWorkItem: %v", err)
		}
		requireOneError(t, w.errorsAbout(t, approve.ID), "which is not earlier than it")
	})
	t.Run("the calculating step is no longer a calculating step", func(t *testing.T) {
		w := newPostHoldWorld(t)
		calc := w.workOutTheInterest(t, 1, 14)
		approve := w.approveTheInterest(t, 2, calc.ID, nil)
		calc.Kind = models.StepKindNotify
		calc.Function, calc.ResultName, calc.Hold = "", "", nil
		withForm(&calc)
		if _, err := w.mgr.UpdateWorkItem(ctx, calc); err != nil {
			t.Fatalf("UpdateWorkItem: %v", err)
		}
		requireOneError(t, w.errorsAbout(t, approve.ID), "which is no longer a calculating step")
	})
}

func TestHold_APostStepThatReadsAHeldFigureBeforeItIsReleasedIsFlagged(t *testing.T) {
	w := newPostHoldWorld(t)
	calc := w.workOutTheInterest(t, 1, 14)
	post := w.postThePayment(t, 2, calc.ID, nil)
	w.approveTheInterest(t, 3, calc.ID, nil)
	requireOneError(t, w.errorsAbout(t, post.ID), "posts the held figure of step", "before the step that releases it")
}
