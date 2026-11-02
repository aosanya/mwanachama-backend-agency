package mwanachamaagency_test

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (b reviewBench) draftNamed(name string) string {
	b.t.Helper()
	draft, err := b.mgr.CreateDraft(b.ctx, name, "", "")
	if err != nil {
		b.t.Fatalf("CreateDraft(%s): %v", name, err)
	}
	return draft.ID
}

func (b reviewBench) startOn(draftID string) models.ReviewView {
	b.t.Helper()
	view, err := b.mgr.StartReview(b.ctx, draftID, "amina", "")
	if err != nil {
		b.t.Fatalf("StartReview(%s): %v", draftID, err)
	}
	return view
}

func (b reviewBench) runningIDs() []string {
	b.t.Helper()
	running, err := b.mgr.ListRunningReviews(b.ctx)
	if err != nil {
		b.t.Fatalf("ListRunningReviews: %v", err)
	}
	ids := make([]string, 0, len(running))
	for _, r := range running {
		if r.Status != models.ReviewRunning {
			b.t.Fatalf("listed review %s has status %q", r.ID, r.Status)
		}
		ids = append(ids, r.ID)
	}
	return ids
}

func sameIDs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestNoReviewIsRunningBeforeAnyStarts(t *testing.T) {
	b := newReviewBench(t)

	running, err := b.mgr.ListRunningReviews(b.ctx)
	if err != nil {
		t.Fatalf("ListRunningReviews: %v", err)
	}
	if running == nil || len(running) != 0 {
		t.Fatalf("running reviews = %#v, want an empty list", running)
	}
}

func TestRunningReviewsAcrossDraftsAreListedNewestFirst(t *testing.T) {
	b := newReviewBench(t)
	other := b.draftNamed("other")
	first := b.start("")
	second := b.startOn(other)

	got := b.runningIDs()

	want := []string{second.Review.ID, first.Review.ID}
	if !sameIDs(got, want) {
		t.Fatalf("running reviews = %v, want %v (newest first, one per draft)", got, want)
	}
}

func TestASettledReviewLeavesTheRunningList(t *testing.T) {
	b := newReviewBench(t)
	other := b.draftNamed("other")
	done := b.start("")
	stillRunning := b.startOn(other)

	b.recordEveryPart(done, nil)

	got := b.runningIDs()
	if !sameIDs(got, []string{stillRunning.Review.ID}) {
		t.Fatalf("running reviews = %v, want only %s", got, stillRunning.Review.ID)
	}
}

func TestAFailedReviewLeavesTheRunningList(t *testing.T) {
	b := newReviewBench(t)
	started := b.start("")

	b.recordAs(started, func(models.ReviewAreaName) bool { return true })

	if got := b.runningIDs(); len(got) != 0 {
		t.Fatalf("running reviews after the review failed = %v, want none", got)
	}
}

func TestAReviewWithOnePartStillReadingStaysInTheRunningList(t *testing.T) {
	b := newReviewBench(t)
	started := b.start("")
	b.record(started.Review.ID, started.Areas[0].Area)

	got := b.runningIDs()

	if !sameIDs(got, []string{started.Review.ID}) {
		t.Fatalf("running reviews = %v, want %s", got, started.Review.ID)
	}
}
