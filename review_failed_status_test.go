package mwanachamaagency_test

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (b reviewBench) recordAs(started models.ReviewView, failed func(models.ReviewAreaName) bool) (models.ReviewStatus, models.ReviewStatus) {
	b.t.Helper()
	var last, beforeLast models.ReviewStatus
	for _, part := range started.Areas {
		status, reason := models.ReviewAreaDone, ""
		if failed(part.Area) {
			status, reason = models.ReviewAreaFailed, "the reviewer gave up"
		}
		_, got, err := b.mgr.RecordReviewArea(b.ctx, started.Review.ID, part.Area, status, "s", reason, 1, nil)
		if err != nil {
			b.t.Fatalf("RecordReviewArea(%s): %v", part.Area, err)
		}
		beforeLast, last = last, got
	}
	return beforeLast, last
}

func (b reviewBench) stored(reviewID string) models.Review {
	b.t.Helper()
	view, err := b.mgr.GetReview(b.ctx, reviewID, "")
	if err != nil {
		b.t.Fatalf("GetReview: %v", err)
	}
	return *view.Review
}

func TestAReviewWhoseEveryPartFailedIsFailed(t *testing.T) {
	b := newReviewBench(t)
	started := b.start("")

	beforeLast, last := b.recordAs(started, func(models.ReviewAreaName) bool { return true })

	if beforeLast != models.ReviewRunning {
		t.Fatalf("status with a part still reading = %q, want running", beforeLast)
	}
	if last != models.ReviewFailed {
		t.Fatalf("status once every part failed = %q, want failed", last)
	}
	review := b.stored(started.Review.ID)
	if review.Status != models.ReviewFailed || review.Running || review.FinishedAt == "" {
		t.Fatalf("stored review = %+v, want failed, not running, with finished_at", review)
	}
}

func TestAReviewOfOnePartThatFailedIsFailed(t *testing.T) {
	b := newReviewBench(t)
	started := b.start("goals")

	_, last := b.recordAs(started, func(models.ReviewAreaName) bool { return true })

	if last != models.ReviewFailed {
		t.Fatalf("status of a one-part review whose part failed = %q, want failed", last)
	}
}

func TestAReviewWithSomePartsFailedAndSomeDoneStaysDone(t *testing.T) {
	b := newReviewBench(t)
	started := b.start("")

	_, last := b.recordAs(started, func(area models.ReviewAreaName) bool { return area == models.ReviewAreaGoals })

	if last != models.ReviewDone {
		t.Fatalf("status with one part failed and the rest done = %q, want done", last)
	}
	if got := b.stored(started.Review.ID).Status; got != models.ReviewDone {
		t.Fatalf("stored status = %q, want done", got)
	}
}

func TestAReviewWithOnlyOnePartDoneAmongFailuresStaysDone(t *testing.T) {
	b := newReviewBench(t)
	started := b.start("")

	_, last := b.recordAs(started, func(area models.ReviewAreaName) bool { return area != models.ReviewAreaGoals })

	if last != models.ReviewDone {
		t.Fatalf("status with one part done and the rest failed = %q, want done", last)
	}
}

func TestAReviewWithNoPartFailedIsDone(t *testing.T) {
	b := newReviewBench(t)
	started := b.start("")

	_, last := b.recordAs(started, func(models.ReviewAreaName) bool { return false })

	if last != models.ReviewDone {
		t.Fatalf("status with no part failed = %q, want done", last)
	}
}

func TestADraftStartsAgainOnceItsReviewHasFailed(t *testing.T) {
	b := newReviewBench(t)
	started := b.start("")
	b.recordAs(started, func(models.ReviewAreaName) bool { return true })

	again, err := b.mgr.StartReview(b.ctx, b.draft, "amina", "")
	if err != nil {
		t.Fatalf("a start after the review failed: %v", err)
	}
	if again.Review.PreviousReviewID != started.Review.ID {
		t.Fatalf("previous_review_id = %q, want the failed review %q", again.Review.PreviousReviewID, started.Review.ID)
	}
}
