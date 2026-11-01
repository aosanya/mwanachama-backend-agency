package mwanachamaagency_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type reviewBench struct {
	t     *testing.T
	ctx   context.Context
	mgr   mwanachamaagency.AgencyManager
	draft string
}

func newReviewBench(t *testing.T) reviewBench {
	t.Helper()
	mgr, _, _ := newTestManagerDB(t)
	mustCreateAgency(t, mgr)
	draft, err := mgr.CreateDraft(t.Context(), "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	return reviewBench{t: t, ctx: t.Context(), mgr: mgr, draft: draft.ID}
}

func (b reviewBench) start(area string) models.ReviewView {
	b.t.Helper()
	view, err := b.mgr.StartReview(b.ctx, b.draft, "amina", area)
	if err != nil {
		b.t.Fatalf("StartReview(%q): %v", area, err)
	}
	return view
}

func (b reviewBench) record(reviewID string, area models.ReviewAreaName, findings ...models.ReviewFindingInput) models.ReviewAreaView {
	b.t.Helper()
	view, _, err := b.mgr.RecordReviewArea(b.ctx, reviewID, area, models.ReviewAreaDone, "ok", "", 1, findings)
	if err != nil {
		b.t.Fatalf("RecordReviewArea(%s): %v", area, err)
	}
	return view
}

func (b reviewBench) recordEveryPart(started models.ReviewView, findings map[models.ReviewAreaName][]models.ReviewFindingInput) map[models.ReviewAreaName]models.ReviewAreaView {
	b.t.Helper()
	out := map[models.ReviewAreaName]models.ReviewAreaView{}
	for _, part := range started.Areas {
		out[part.Area] = b.record(started.Review.ID, part.Area, findings[part.Area]...)
	}
	return out
}

func finding(title string) models.ReviewFindingInput {
	return models.ReviewFindingInput{Title: title, Detail: "d", Fate: models.FateNew}
}

func carrying(title, earlierID string) models.ReviewFindingInput {
	return models.ReviewFindingInput{Title: title, Detail: "d", Fate: models.FateStillOpen, CarriedFromID: earlierID}
}

func partOf(view models.ReviewView, area models.ReviewAreaName) models.ReviewAreaView {
	for _, p := range view.Areas {
		if p.Area == area {
			return p
		}
	}
	return models.ReviewAreaView{}
}

func titlesOf(part models.ReviewAreaView) []string {
	out := []string{}
	for _, f := range part.Findings {
		out = append(out, f.Title)
	}
	return out
}

func TestAnAreaRunCreatesOnlyThatPartAndReadsIt(t *testing.T) {
	b := newReviewBench(t)

	started := b.start("goals")

	if got := areaNames(started.Areas); !reflect.DeepEqual(got, []models.ReviewAreaName{models.ReviewAreaGoals}) {
		t.Fatalf("parts = %v, want only goals", got)
	}
	if started.Areas[0].Status != models.ReviewAreaReading {
		t.Fatalf("status = %s, want reading", started.Areas[0].Status)
	}
}

func TestAnAreaRunRefusesAnUnknownPartAndAPartTheHubDoesNotRead(t *testing.T) {
	b := newReviewBench(t)
	if _, err := b.mgr.StartReview(b.ctx, b.draft, "amina", "nonsense"); !errors.Is(err, mwanachamaagency.ErrInvalidReview) {
		t.Fatalf("unknown part: %v", err)
	}

	mgr, db, s := newTestManagerDB(t)
	mustCreateAgency(t, mgr)
	narrow, err := mwanachamaagency.NewAgencyManager(db, s, mwanachamaagency.WithReviewAreas(func(context.Context, string) []models.ReviewAreaName {
		return []models.ReviewAreaName{models.ReviewAreaGoals}
	}))
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	draft, err := narrow.CreateDraft(b.ctx, "narrow", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if _, err := narrow.StartReview(b.ctx, draft.ID, "amina", "roles_agents"); !errors.Is(err, mwanachamaagency.ErrInvalidReview) {
		t.Fatalf("unread part: %v", err)
	}
	if _, err := narrow.StartReview(b.ctx, draft.ID, "amina", "goals"); err != nil {
		t.Fatalf("read part: %v", err)
	}
}

func TestAnAreaRunStillRefusesWhileAnotherReviewOfTheDraftIsRunning(t *testing.T) {
	b := newReviewBench(t)
	b.start("goals")

	if _, err := b.mgr.StartReview(b.ctx, b.draft, "amina", "roles_agents"); !errors.Is(err, mwanachamaagency.ErrReviewRunning) {
		t.Fatalf("second run: %v", err)
	}
}

func TestAPartTakesItsPreviousFindingsFromTheNewestReviewThatCoveredIt(t *testing.T) {
	b := newReviewBench(t)
	first := b.start("")
	firstParts := b.recordEveryPart(first, map[models.ReviewAreaName][]models.ReviewFindingInput{
		models.ReviewAreaGoals:       {finding("goal one")},
		models.ReviewAreaRolesAgents: {finding("role one")},
	})

	second := b.start("goals")
	input, err := b.mgr.ReviewInput(b.ctx, second.Review.ID, models.ReviewAreaGoals)
	if err != nil || len(input.Previous) != 1 || input.Previous[0].ID != firstParts[models.ReviewAreaGoals].Findings[0].ID {
		t.Fatalf("goals input after a full run = %+v, %v", input, err)
	}
	secondGoals := b.record(second.Review.ID, models.ReviewAreaGoals, carrying("goal one again", input.Previous[0].ID))

	third := b.start("goals")
	input, err = b.mgr.ReviewInput(b.ctx, third.Review.ID, models.ReviewAreaGoals)
	if err != nil || len(input.Previous) != 1 || input.Previous[0].ID != secondGoals.Findings[0].ID {
		t.Fatalf("goals input after a goals-only run = %+v, %v", input, err)
	}
	b.record(third.Review.ID, models.ReviewAreaGoals, carrying("goal one a third time", input.Previous[0].ID))

	fourth := b.start("")
	roles, err := b.mgr.ReviewInput(b.ctx, fourth.Review.ID, models.ReviewAreaRolesAgents)
	if err != nil || len(roles.Previous) != 1 || roles.Previous[0].ID != firstParts[models.ReviewAreaRolesAgents].Findings[0].ID {
		t.Fatalf("roles input across two goals-only runs = %+v, %v", roles, err)
	}
	b.record(fourth.Review.ID, models.ReviewAreaRolesAgents, carrying("role one again", roles.Previous[0].ID))
}

func TestACarriedFindingMustComeFromTheSamePartOfTheCoveringReview(t *testing.T) {
	b := newReviewBench(t)
	first := b.start("")
	firstParts := b.recordEveryPart(first, map[models.ReviewAreaName][]models.ReviewFindingInput{
		models.ReviewAreaGoals:       {finding("goal one")},
		models.ReviewAreaRolesAgents: {finding("role one")},
	})

	second := b.start("goals")
	roleID := firstParts[models.ReviewAreaRolesAgents].Findings[0].ID
	_, _, err := b.mgr.RecordReviewArea(b.ctx, second.Review.ID, models.ReviewAreaGoals, models.ReviewAreaDone, "ok", "", 1, []models.ReviewFindingInput{carrying("wrong part", roleID)})
	if !errors.Is(err, mwanachamaagency.ErrInvalidReview) {
		t.Fatalf("carrying a roles finding into goals: %v", err)
	}

	b.record(second.Review.ID, models.ReviewAreaGoals)
	third := b.start("roles_agents")
	if _, _, err := b.mgr.RecordReviewArea(b.ctx, third.Review.ID, models.ReviewAreaRolesAgents, models.ReviewAreaDone, "ok", "", 1, []models.ReviewFindingInput{carrying("role one again", roleID)}); err != nil {
		t.Fatalf("carrying across the goals-only run: %v", err)
	}
}

func TestReadingADraftShowsEachPartFromTheNewestRunThatCoveredIt(t *testing.T) {
	b := newReviewBench(t)
	first := b.start("")
	firstParts := b.recordEveryPart(first, map[models.ReviewAreaName][]models.ReviewFindingInput{
		models.ReviewAreaGoals:       {finding("goal one")},
		models.ReviewAreaRolesAgents: {finding("role one")},
	})
	note, err := b.mgr.CommentFinding(b.ctx, firstParts[models.ReviewAreaRolesAgents].Findings[0].ID, "known", "amina")
	if err != nil {
		t.Fatalf("CommentFinding: %v", err)
	}

	second := b.start("goals")
	running, err := b.mgr.GetReview(b.ctx, "", b.draft)
	if err != nil {
		t.Fatalf("GetReview: %v", err)
	}
	if running.Review.ID != second.Review.ID {
		t.Fatalf("review = %s, want the running goals run %s", running.Review.ID, second.Review.ID)
	}
	if partOf(running, models.ReviewAreaGoals).Status != models.ReviewAreaReading {
		t.Fatalf("goals = %s, want reading from the running review", partOf(running, models.ReviewAreaGoals).Status)
	}
	if got := titlesOf(partOf(running, models.ReviewAreaRolesAgents)); !reflect.DeepEqual(got, []string{"role one"}) {
		t.Fatalf("roles while goals run = %v", got)
	}

	input, err := b.mgr.ReviewInput(b.ctx, second.Review.ID, models.ReviewAreaGoals)
	if err != nil {
		t.Fatalf("ReviewInput: %v", err)
	}
	b.record(second.Review.ID, models.ReviewAreaGoals, carrying("goal one again", input.Previous[0].ID))

	view, err := b.mgr.GetReview(b.ctx, "", b.draft)
	if err != nil {
		t.Fatalf("GetReview: %v", err)
	}
	if view.Review.ID != second.Review.ID {
		t.Fatalf("review = %s, want %s", view.Review.ID, second.Review.ID)
	}
	if got := areaNames(view.Areas); !reflect.DeepEqual(got, models.ReviewAreaOrder) {
		t.Fatalf("parts = %v, want every part in order", got)
	}
	goals, roles := partOf(view, models.ReviewAreaGoals), partOf(view, models.ReviewAreaRolesAgents)
	if goals.ReviewID != second.Review.ID || !reflect.DeepEqual(titlesOf(goals), []string{"goal one again"}) {
		t.Fatalf("goals = review %s %v", goals.ReviewID, titlesOf(goals))
	}
	if roles.ReviewID != first.Review.ID || !reflect.DeepEqual(titlesOf(roles), []string{"role one"}) {
		t.Fatalf("roles = review %s %v", roles.ReviewID, titlesOf(roles))
	}
	if len(roles.Findings[0].Comments) != 1 || roles.Findings[0].Comments[0].ID != note.ID {
		t.Fatalf("the note on the roles finding is gone: %+v", roles.Findings[0].Comments)
	}
}

func TestReadingADraftWhoseEveryRunCoveredEveryPartIsTheLatestReviewAsBefore(t *testing.T) {
	b := newReviewBench(t)
	first := b.start("")
	b.recordEveryPart(first, map[models.ReviewAreaName][]models.ReviewFindingInput{models.ReviewAreaGoals: {finding("goal one")}})
	second := b.start("")
	b.recordEveryPart(second, nil)

	byDraft, err := b.mgr.GetReview(b.ctx, "", b.draft)
	if err != nil {
		t.Fatalf("GetReview by draft: %v", err)
	}
	byID, err := b.mgr.GetReview(b.ctx, second.Review.ID, "")
	if err != nil {
		t.Fatalf("GetReview by id: %v", err)
	}
	if !reflect.DeepEqual(byDraft, byID) {
		t.Fatalf("by draft = %+v\nby id    = %+v", byDraft, byID)
	}
}

func TestReadingADraftWithNoReviewAnswersNone(t *testing.T) {
	b := newReviewBench(t)

	view, err := b.mgr.GetReview(b.ctx, "", b.draft)
	if err != nil {
		t.Fatalf("GetReview: %v", err)
	}
	if view.Review != nil || len(view.Areas) != 0 {
		t.Fatalf("view = %+v, want none", view)
	}
}

func TestFindingsWithTheSameTitleAreStoredOnceSoACarriedRepeatDoesNotGrow(t *testing.T) {
	b := newReviewBench(t)
	first := b.start("")
	b.recordEveryPart(first, map[models.ReviewAreaName][]models.ReviewFindingInput{
		models.ReviewAreaRolesAgents: {finding("Describe Farmer R-47"), finding("Describe  farmer r-47")},
	})
	view, err := b.mgr.GetReview(b.ctx, first.Review.ID, b.draft)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(partOf(view, models.ReviewAreaRolesAgents).Findings); got != 1 {
		t.Fatalf("roles findings after a repeated title = %d, want 1", got)
	}
}
