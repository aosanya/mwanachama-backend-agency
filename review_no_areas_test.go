package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func readsNoPart(context.Context, string) []models.ReviewAreaName { return nil }

type noPartBench struct {
	t     *testing.T
	ctx   context.Context
	bare  mwanachamaagency.AgencyManager
	usual mwanachamaagency.AgencyManager
	draft string
}

func newNoPartBench(t *testing.T) noPartBench {
	t.Helper()
	usual, db, s := newTestManagerDB(t)
	mustCreateAgency(t, usual)
	bare, err := mwanachamaagency.NewAgencyManager(db, s, mwanachamaagency.WithReviewAreas(readsNoPart))
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	draft, err := usual.CreateDraft(t.Context(), "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	return noPartBench{t: t, ctx: t.Context(), bare: bare, usual: usual, draft: draft.ID}
}

func TestAStartWhenTheHubReadsNoPartIsRefusedWithNoPartToReview(t *testing.T) {
	b := newNoPartBench(t)

	_, err := b.bare.StartReview(b.ctx, b.draft, "amina", "")

	if !errors.Is(err, mwanachamaagency.ErrNoReviewAreas) {
		t.Fatalf("a start with no part to read: %v, want ErrNoReviewAreas", err)
	}
}

func TestAStartRefusedForWantOfPartsLeavesNoReviewBehind(t *testing.T) {
	b := newNoPartBench(t)

	if _, err := b.bare.StartReview(b.ctx, b.draft, "amina", ""); err == nil {
		t.Fatal("a start with no part to read was accepted")
	}

	reviews, err := b.bare.ListReviews(b.ctx, b.draft)
	if err != nil {
		t.Fatalf("ListReviews: %v", err)
	}
	if len(reviews) != 0 {
		t.Fatalf("reviews after the refused start = %d, want none", len(reviews))
	}
	running, err := b.bare.ListRunningReviews(b.ctx)
	if err != nil {
		t.Fatalf("ListRunningReviews: %v", err)
	}
	if len(running) != 0 {
		t.Fatalf("running reviews after the refused start = %d, want none", len(running))
	}
}

func TestADraftTheHubReadsNothingOfStartsOnceTheHubReadsParts(t *testing.T) {
	b := newNoPartBench(t)
	if _, err := b.bare.StartReview(b.ctx, b.draft, "amina", ""); err == nil {
		t.Fatal("a start with no part to read was accepted")
	}

	started, err := b.usual.StartReview(b.ctx, b.draft, "amina", "")

	if err != nil {
		t.Fatalf("a start once parts are read: %v, want it accepted and not blocked as running", err)
	}
	if len(started.Areas) != len(models.ReviewAreaOrder) {
		t.Fatalf("areas = %d, want %d", len(started.Areas), len(models.ReviewAreaOrder))
	}
}

func TestANamedPartTheHubDoesNotReadIsStillRefusedAsInvalid(t *testing.T) {
	b := newNoPartBench(t)

	_, err := b.bare.StartReview(b.ctx, b.draft, "amina", "goals")

	if !errors.Is(err, mwanachamaagency.ErrInvalidReview) {
		t.Fatalf("a named part that is not read: %v, want ErrInvalidReview", err)
	}
}
