package mwanachamaagency_test

import (
	"errors"
	"maps"
	"testing"

	"gorm.io/gorm"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

func TestTheReviewTableHoldsAUniqueIndexOnItsRunningReviews(t *testing.T) {
	_, db, s := newTestManagerDB(t)
	table := tableOf(t, s, "review")

	var named []string
	err := db.Raw(`select name from pragma_index_list(?) where origin = 'c' and "unique" = 1 and partial = 1`, table).Scan(&named).Error
	if err != nil {
		t.Fatalf("read the indexes of %s: %v", table, err)
	}
	if len(named) == 0 {
		t.Fatalf("%s has no unique partial index: two starts at once for one draft can both pass the running check on a read-committed database", table)
	}
}

func TestAReviewThatLandsBesideARunningOneIsRefusedAsRunning(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	mustCreateAgency(t, mgr)
	draft, err := mgr.CreateDraft(t.Context(), "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	table := tableOf(t, s, "review")
	planted := false
	err = db.Callback().Create().Before("gorm:create").Register("land_a_rival_start", func(tx *gorm.DB) {
		if planted || tx.Statement.Table != table {
			return
		}
		planted = true
		row, ok := tx.Statement.Dest.(map[string]any)
		if !ok {
			t.Errorf("the review is written as %T, want a column map", tx.Statement.Dest)
			return
		}
		rival := maps.Clone(row)
		rival["id"] = "rival"
		if rivalErr := tx.Session(&gorm.Session{NewDB: true}).Table(table).Create(rival).Error; rivalErr != nil {
			t.Errorf("land the rival start: %v", rivalErr)
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = mgr.StartReview(t.Context(), draft.ID, "amina", "")

	if !planted {
		t.Fatal("the rival start never landed")
	}
	if !errors.Is(err, mwanachamaagency.ErrReviewRunning) {
		t.Fatalf("a start that loses to a rival written after the check: %v, want ErrReviewRunning", err)
	}
}

func TestADraftStartsAgainOnceItsReviewHasSettled(t *testing.T) {
	b := newReviewBench(t)
	first := b.start("")
	if _, err := b.mgr.StartReview(b.ctx, b.draft, "amina", ""); !errors.Is(err, mwanachamaagency.ErrReviewRunning) {
		t.Fatalf("a second start while one runs: %v, want ErrReviewRunning", err)
	}

	b.recordEveryPart(first, nil)

	if _, err := b.mgr.StartReview(b.ctx, b.draft, "amina", ""); err != nil {
		t.Fatalf("a start after the review settled: %v", err)
	}
}

func TestTwoDraftsRunReviewsAtTheSameTime(t *testing.T) {
	b := newReviewBench(t)
	other, err := b.mgr.CreateDraft(b.ctx, "other", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	b.start("")

	if _, err := b.mgr.StartReview(b.ctx, other.ID, "amina", ""); err != nil {
		t.Fatalf("a start on another draft: %v", err)
	}
}
