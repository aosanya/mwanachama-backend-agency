package mwanachamaagency

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

type reviewShelf struct {
	st *specstore.Store
}

func newReviewShelf(t *testing.T) reviewShelf {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	st, err := newStore(db, s)
	if err != nil {
		t.Fatalf("newStore: %v", err)
	}
	return reviewShelf{st: st}
}

func (r reviewShelf) save(t *testing.T, role string, v any) {
	t.Helper()
	if err := r.st.Insert(context.Background(), role, v); err != nil {
		t.Fatalf("save %s: %v", role, err)
	}
}

func readWhere[T any](t *testing.T, r reviewShelf, role, column, value string) []T {
	t.Helper()
	q := r.st.Query(context.Background(), role).Where(column+" = ?", value)
	out, err := specstore.List[T](r.st, q, role)
	if err != nil {
		t.Fatalf("read %s by %s: %v", role, column, err)
	}
	return out
}

func idsOf[T any](rows []T, id func(T) string) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, id(row))
	}
	sort.Strings(out)
	return out
}

func TestReviewIsSavedReadBackAndFoundByDraft(t *testing.T) {
	r := newReviewShelf(t)
	first := models.Review{
		ID: "r1", DraftID: "d1", Status: models.ReviewDone, StartedBy: "amina",
		StartedAt: "2026-10-31T09:00:00.000000000Z", FinishedAt: "2026-10-31T09:02:00.000000000Z", Tokens: 4200,
	}
	second := models.Review{
		ID: "r2", DraftID: "d1", Status: models.ReviewRunning, PreviousReviewID: "r1", StartedBy: "amina",
		StartedAt: "2026-10-31T10:00:00.000000000Z",
	}
	other := models.Review{ID: "r3", DraftID: "d2", Status: models.ReviewFailed, StartedAt: "2026-10-31T11:00:00.000000000Z"}
	for _, v := range []models.Review{first, second, other} {
		r.save(t, roleReview, v)
	}

	got := readWhere[models.Review](t, r, roleReview, "draft_id", "d1")
	if !reflect.DeepEqual(idsOf(got, func(v models.Review) string { return v.ID }), []string{"r1", "r2"}) {
		t.Fatalf("reviews of d1 = %+v", got)
	}
	byID := map[string]models.Review{got[0].ID: got[0], got[1].ID: got[1]}
	if !reflect.DeepEqual(byID["r1"], first) {
		t.Errorf("r1 read back as %+v, want %+v", byID["r1"], first)
	}
	if !reflect.DeepEqual(byID["r2"], second) {
		t.Errorf("r2 read back as %+v, want %+v", byID["r2"], second)
	}
	if n := len(readWhere[models.Review](t, r, roleReview, "draft_id", "d9")); n != 0 {
		t.Errorf("reviews of an unreviewed draft = %d, want 0", n)
	}
}

func TestReviewAreaIsSavedReadBackAndFoundByReview(t *testing.T) {
	r := newReviewShelf(t)
	read := models.ReviewArea{
		ID: "a1", ReviewID: "r1", Area: models.ReviewAreaRolesAgents, Status: models.ReviewAreaDone,
		Summary: "Two roles lack a description. Both agents have tools.", Tokens: 900,
	}
	failed := models.ReviewArea{
		ID: "a2", ReviewID: "r1", Area: models.ReviewAreaRisks, Status: models.ReviewAreaFailed,
		FailureReason: "The AI account ran out of allowance.",
	}
	reading := models.ReviewArea{ID: "a3", ReviewID: "r2", Area: models.ReviewAreaGoals, Status: models.ReviewAreaReading}
	for _, v := range []models.ReviewArea{read, failed, reading} {
		r.save(t, roleReviewArea, v)
	}

	got := readWhere[models.ReviewArea](t, r, roleReviewArea, "review_id", "r1")
	if !reflect.DeepEqual(idsOf(got, func(v models.ReviewArea) string { return v.ID }), []string{"a1", "a2"}) {
		t.Fatalf("areas of r1 = %+v", got)
	}
	byID := map[string]models.ReviewArea{got[0].ID: got[0], got[1].ID: got[1]}
	if !reflect.DeepEqual(byID["a1"], read) || !reflect.DeepEqual(byID["a2"], failed) {
		t.Errorf("areas read back as %+v", byID)
	}
	only := readWhere[models.ReviewArea](t, r, roleReviewArea, "review_id", "r2")
	if len(only) != 1 || only[0].Status != models.ReviewAreaReading {
		t.Errorf("areas of r2 = %+v, want the one still reading", only)
	}
}

func TestReviewFindingKeepsItsToolAndStepsAndIsFoundByReview(t *testing.T) {
	r := newReviewShelf(t)
	applied := models.ReviewFinding{
		ID: "f1", ReviewID: "r1", AreaID: "a1", Title: "Describe Farmer R-47", Detail: "The seat has no description.",
		Tool:      "agency_update_role",
		Arguments: map[string]any{"role_id": "role-47", "description": "Grows and sells maize."},
		ThenSteps: []models.FindingStep{
			{Tool: "agency_attach_value", Arguments: map[string]any{"role_id": "role-47"}},
			{Tool: "agency_scan"},
		},
		Status: models.FindingApplied, Fate: models.FateFixed, FateReason: "The role now has a description.", CarriedFromID: "f0",
	}
	advice := models.ReviewFinding{
		ID: "f2", ReviewID: "r1", AreaID: "a1", Title: "Add a second owner", Status: models.FindingOpen, Fate: models.FateNew,
	}
	later := models.ReviewFinding{
		ID: "f3", ReviewID: "r2", AreaID: "a3", Title: "Add a second owner", Status: models.FindingDismissed,
		Fate: models.FateStillOpen, CarriedFromID: "f2",
	}
	for _, v := range []models.ReviewFinding{applied, advice, later} {
		r.save(t, roleReviewFinding, v)
	}

	got := readWhere[models.ReviewFinding](t, r, roleReviewFinding, "review_id", "r1")
	if !reflect.DeepEqual(idsOf(got, func(v models.ReviewFinding) string { return v.ID }), []string{"f1", "f2"}) {
		t.Fatalf("findings of r1 = %+v", got)
	}
	byID := map[string]models.ReviewFinding{got[0].ID: got[0], got[1].ID: got[1]}
	if !reflect.DeepEqual(byID["f1"], applied) {
		t.Errorf("f1 read back as %+v, want %+v", byID["f1"], applied)
	}
	if !reflect.DeepEqual(byID["f2"], advice) {
		t.Errorf("f2 read back as %+v, want %+v", byID["f2"], advice)
	}
	next := readWhere[models.ReviewFinding](t, r, roleReviewFinding, "review_id", "r2")
	if len(next) != 1 || next[0].CarriedFromID != "f2" || next[0].Status != models.FindingDismissed || next[0].Fate != models.FateStillOpen {
		t.Errorf("findings of r2 = %+v", next)
	}
}

func TestReviewCommentIsSavedReadBackAndFoundByFinding(t *testing.T) {
	r := newReviewShelf(t)
	note := models.ReviewComment{
		ID: "c1", FindingID: "f1", Author: "amina", Body: "Two owners is deliberate here.", CreatedAt: "2026-10-31T09:30:00.000000000Z",
	}
	reply := models.ReviewComment{ID: "c2", FindingID: "f1", Author: "amina", Body: "Still true.", CreatedAt: "2026-10-31T09:40:00.000000000Z"}
	elsewhere := models.ReviewComment{ID: "c3", FindingID: "f2", Author: "amina", Body: "Noted.", CreatedAt: "2026-10-31T09:50:00.000000000Z"}
	for _, v := range []models.ReviewComment{note, reply, elsewhere} {
		r.save(t, roleReviewComment, v)
	}

	got := readWhere[models.ReviewComment](t, r, roleReviewComment, "finding_id", "f1")
	if !reflect.DeepEqual(idsOf(got, func(v models.ReviewComment) string { return v.ID }), []string{"c1", "c2"}) {
		t.Fatalf("comments on f1 = %+v", got)
	}
	byID := map[string]models.ReviewComment{got[0].ID: got[0], got[1].ID: got[1]}
	if !reflect.DeepEqual(byID["c1"], note) || !reflect.DeepEqual(byID["c2"], reply) {
		t.Errorf("comments read back as %+v", byID)
	}
}

func TestEveryReviewObjectIsFilteredByAnIndex(t *testing.T) {
	b, err := Blueprint()
	if err != nil {
		t.Fatalf("blueprint: %v", err)
	}
	want := map[string]string{
		roleReview:        "draft_id",
		roleReviewArea:    "review_id",
		roleReviewFinding: "review_id",
		roleReviewComment: "finding_id",
	}
	for role, column := range want {
		o, ok := b.Object(role)
		if !ok {
			t.Fatalf("the blueprint declares no %s", role)
		}
		if len(o.Indexes) != 1 || !reflect.DeepEqual(o.Indexes[0].Fields, []string{column}) {
			t.Errorf("%s indexes = %+v, want one on %s", role, o.Indexes, column)
		}
	}
}
