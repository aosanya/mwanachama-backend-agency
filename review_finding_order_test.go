package mwanachamaagency_test

import (
	"reflect"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestFindingsComeBackInTheOrderTheReviewerWroteThem(t *testing.T) {
	b := newReviewBench(t)
	started := b.start("goals")
	written := []string{"Zebra goal has no owner", "Add a measure to goal G-2", "Mention the second language", "Rename goal G-1"}

	recorded := b.record(started.Review.ID, models.ReviewAreaGoals, finding(written[0]), finding(written[1]), finding(written[2]), finding(written[3]))

	if got := titlesOf(recorded); !reflect.DeepEqual(got, written) {
		t.Fatalf("the answer to the recording lists %v, want %v", got, written)
	}
	read, err := b.mgr.GetReview(b.ctx, started.Review.ID, "")
	if err != nil {
		t.Fatalf("GetReview: %v", err)
	}
	if got := titlesOf(partOf(read, models.ReviewAreaGoals)); !reflect.DeepEqual(got, written) {
		t.Fatalf("reading the review lists %v, want %v", got, written)
	}
}

func TestFindingsKeepTheReviewersOrderInEveryPartAndOnTheNextReview(t *testing.T) {
	b := newReviewBench(t)
	first := b.start("")
	goals := []string{"Yes first", "Beta second", "Alpha third"}
	risks := []string{"Writing risk two", "Writing risk one"}
	recorded := b.recordEveryPart(first, map[models.ReviewAreaName][]models.ReviewFindingInput{
		models.ReviewAreaGoals: {finding(goals[0]), finding(goals[1]), finding(goals[2])},
		models.ReviewAreaRisks: {finding(risks[0]), finding(risks[1])},
	})
	if got := titlesOf(recorded[models.ReviewAreaGoals]); !reflect.DeepEqual(got, goals) {
		t.Fatalf("goals = %v, want %v", got, goals)
	}

	second := b.start("")
	input, err := b.mgr.ReviewInput(b.ctx, second.Review.ID, models.ReviewAreaGoals)
	if err != nil {
		t.Fatalf("ReviewInput: %v", err)
	}
	var previous []string
	for _, f := range input.Previous {
		previous = append(previous, f.Title)
	}
	if !reflect.DeepEqual(previous, goals) {
		t.Fatalf("the next review reads %v, want %v", previous, goals)
	}
	read, err := b.mgr.GetReview(b.ctx, first.Review.ID, "")
	if err != nil {
		t.Fatalf("GetReview: %v", err)
	}
	if got := titlesOf(partOf(read, models.ReviewAreaRisks)); !reflect.DeepEqual(got, risks) {
		t.Fatalf("risks = %v, want %v", got, risks)
	}
}

func TestAFindingWrittenTwiceKeepsTheFirstPlace(t *testing.T) {
	b := newReviewBench(t)
	started := b.start("goals")

	recorded := b.record(started.Review.ID, models.ReviewAreaGoals, finding("Zulu"), finding("Mike"), finding("zulu"), finding("Alpha"))

	if got, want := titlesOf(recorded), []string{"Zulu", "Mike", "Alpha"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("lists %v, want %v", got, want)
	}
}
