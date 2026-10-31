package mwanachamaagency_test

import (
	"context"
	"reflect"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func mustCreateAgency(t *testing.T, mgr mwanachamaagency.AgencyManager) {
	t.Helper()
	if _, err := mgr.CreateAgency(t.Context(), models.Agency{Name: "Farmers"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
}

func areaNames(areas []models.ReviewAreaView) []models.ReviewAreaName {
	out := make([]models.ReviewAreaName, 0, len(areas))
	for _, a := range areas {
		out = append(out, a.Area)
	}
	return out
}

func TestStartReviewReadsAllSixPartsUnlessTheHubNarrowsThem(t *testing.T) {
	ctx := t.Context()
	mgr, db, s := newTestManagerDB(t)
	mustCreateAgency(t, mgr)
	draft, err := mgr.CreateDraft(ctx, "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	started, err := mgr.StartReview(ctx, draft.ID, "amina", "")
	if err != nil {
		t.Fatalf("StartReview: %v", err)
	}
	if !reflect.DeepEqual(areaNames(started.Areas), models.ReviewAreaOrder) {
		t.Fatalf("areas = %v, want all six in order", areaNames(started.Areas))
	}

	farmless := func(_ context.Context, _ string) []models.ReviewAreaName {
		return []models.ReviewAreaName{models.ReviewAreaGoals, models.ReviewAreaRisks}
	}
	narrow, err := mwanachamaagency.NewAgencyManager(db, s, mwanachamaagency.WithReviewAreas(farmless))
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	other, err := narrow.CreateDraft(ctx, "other", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	narrowed, err := narrow.StartReview(ctx, other.ID, "", "")
	if err != nil {
		t.Fatalf("StartReview: %v", err)
	}
	if got := areaNames(narrowed.Areas); !reflect.DeepEqual(got, []models.ReviewAreaName{models.ReviewAreaGoals, models.ReviewAreaRisks}) {
		t.Fatalf("narrowed areas = %v", got)
	}
}

func TestReviewFinishesWhenTheNarrowedPartsAreRecorded(t *testing.T) {
	ctx := t.Context()
	mgr, db, s := newTestManagerDB(t)
	only := func(context.Context, string) []models.ReviewAreaName {
		return []models.ReviewAreaName{models.ReviewAreaGoals}
	}
	narrow, err := mwanachamaagency.NewAgencyManager(db, s, mwanachamaagency.WithReviewAreas(only))
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	mustCreateAgency(t, mgr)
	draft, err := mgr.CreateDraft(ctx, "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	started, err := narrow.StartReview(ctx, draft.ID, "", "")
	if err != nil {
		t.Fatalf("StartReview: %v", err)
	}
	_, status, err := narrow.RecordReviewArea(ctx, started.Review.ID, models.ReviewAreaGoals, models.ReviewAreaDone, "ok", "", 3, nil)
	if err != nil || status != models.ReviewDone {
		t.Fatalf("RecordReviewArea = %v, %v, want the review done", status, err)
	}
}
