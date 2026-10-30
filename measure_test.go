package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func objectiveIn(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID, goalID, title string) models.Objective {
	t.Helper()
	o, err := mgr.CreateObjective(ctx, models.Objective{DraftID: draftID, GoalID: goalID, Title: title})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	return o
}

func measuresOf(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID, objectiveID string) []models.Measure {
	t.Helper()
	out, err := mgr.ListMeasures(ctx, draftID, mwanachamaagency.MeasureFilter{ObjectiveID: objectiveID})
	if err != nil {
		t.Fatalf("ListMeasures: %v", err)
	}
	return out
}

func TestSetMeasure_AMeasureSetOnAnObjectiveIsListedWithIt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	set, err := mgr.SetMeasure(ctx, models.Measure{ObjectiveID: o.ID, Name: "Suppliers posting prices", Target: "40", Unit: "suppliers", Status: "on track"})
	if err != nil {
		t.Fatalf("SetMeasure: %v", err)
	}
	if set.ID == "" || set.DraftID != o.DraftID || set.ObjectiveID != o.ID || set.CreatedAt == "" {
		t.Fatalf("set = %+v, want a minted id in the objective's draft", set)
	}

	listed := measuresOf(t, mgr, ctx, o.DraftID, o.ID)
	if len(listed) != 1 {
		t.Fatalf("listed = %+v, want the one measure", listed)
	}
	got := listed[0]
	if got.ID != set.ID || got.Name != "Suppliers posting prices" || got.Target != "40" || got.Unit != "suppliers" || got.Status != "on track" || got.Current != "" {
		t.Fatalf("listed = %+v, want the measure as set with no current reading", got)
	}
}

func TestSetMeasure_UnitAndStatusAreOptional(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	got, err := mgr.SetMeasure(ctx, models.Measure{ObjectiveID: o.ID, Name: "Shops joined", Target: "12"})
	if err != nil {
		t.Fatalf("SetMeasure: %v", err)
	}
	if got.Unit != "" || got.Status != "" {
		t.Fatalf("got = %+v, want no unit and no status", got)
	}
}

func TestSetMeasure_ASecondSetWithTheSameNameReplacesTheFirst(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	first, err := mgr.SetMeasure(ctx, models.Measure{ObjectiveID: o.ID, Name: "Orders per week", Target: "20", Unit: "orders", Status: "behind"})
	if err != nil {
		t.Fatalf("SetMeasure (first): %v", err)
	}
	second, err := mgr.SetMeasure(ctx, models.Measure{ObjectiveID: o.ID, Name: "  orders PER week ", Target: "50"})
	if err != nil {
		t.Fatalf("SetMeasure (second): %v", err)
	}
	if second.ID != first.ID {
		t.Errorf("second id = %q, want the first measure's %q kept", second.ID, first.ID)
	}

	listed := measuresOf(t, mgr, ctx, o.DraftID, o.ID)
	if len(listed) != 1 {
		t.Fatalf("listed = %+v, want one measure after two sets of one name", listed)
	}
	if listed[0].Target != "50" || listed[0].Unit != "" || listed[0].Status != "" || listed[0].Name != "orders PER week" {
		t.Fatalf("listed = %+v, want the second set to have replaced target, unit, status and spelling", listed[0])
	}
}

func TestSetMeasure_TheManagerCannotSetCurrentThroughTheTool(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	set, err := mgr.SetMeasure(ctx, models.Measure{ObjectiveID: o.ID, Name: "Shops joined", Target: "12", Current: "9"})
	if err != nil {
		t.Fatalf("SetMeasure: %v", err)
	}
	if set.Current != "" {
		t.Fatalf("set.Current = %q, want the supplied reading ignored", set.Current)
	}
	if got := measuresOf(t, mgr, ctx, o.DraftID, o.ID)[0]; got.Current != "" {
		t.Fatalf("listed current = %q, want it empty", got.Current)
	}
}

func TestSetMeasure_AReplacedMeasureKeepsItsCurrentReading(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	first, err := mgr.SetMeasure(ctx, models.Measure{ObjectiveID: o.ID, Name: "Shops joined", Target: "12"})
	if err != nil {
		t.Fatalf("SetMeasure: %v", err)
	}
	if err := db.Table(tableOf(t, s, "measure")).Where("id = ?", first.ID).Update("current", "9").Error; err != nil {
		t.Fatalf("writing a reading: %v", err)
	}

	if _, err := mgr.SetMeasure(ctx, models.Measure{ObjectiveID: o.ID, Name: "Shops joined", Target: "15"}); err != nil {
		t.Fatalf("SetMeasure (second): %v", err)
	}
	got := measuresOf(t, mgr, ctx, o.DraftID, o.ID)
	if len(got) != 1 || got[0].Current != "9" || got[0].Target != "15" {
		t.Fatalf("listed = %+v, want the reading 9 kept and the target now 15", got)
	}
}

func TestSetMeasure_Refusals(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	cases := []struct {
		name string
		in   models.Measure
		want error
	}{
		{"unknown objective", models.Measure{ObjectiveID: "no-such-objective", Name: "Shops", Target: "1"}, mwanachamaagency.ErrObjectiveNotFound},
		{"no objective", models.Measure{Name: "Shops", Target: "1"}, mwanachamaagency.ErrObjectiveNotFound},
		{"no name", models.Measure{ObjectiveID: o.ID, Target: "1"}, mwanachamaagency.ErrInvalidMeasure},
		{"blank name", models.Measure{ObjectiveID: o.ID, Name: "   ", Target: "1"}, mwanachamaagency.ErrInvalidMeasure},
		{"no target", models.Measure{ObjectiveID: o.ID, Name: "Shops"}, mwanachamaagency.ErrInvalidMeasure},
		{"blank target", models.Measure{ObjectiveID: o.ID, Name: "Shops", Target: " "}, mwanachamaagency.ErrInvalidMeasure},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := mgr.SetMeasure(ctx, c.in); !errors.Is(err, c.want) {
				t.Fatalf("SetMeasure = %v, want %v", err, c.want)
			}
		})
	}
	if got := measuresOf(t, mgr, ctx, o.DraftID, ""); len(got) != 0 {
		t.Fatalf("listed = %+v, want a refused set to store nothing", got)
	}
}

func TestSetMeasure_AnArchivedDraftRefusesIt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	if _, err := mgr.ArchiveDraft(ctx, o.DraftID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}
	if _, err := mgr.SetMeasure(ctx, models.Measure{ObjectiveID: o.ID, Name: "Shops", Target: "1"}); !errors.Is(err, mwanachamaagency.ErrDraftNotOpen) {
		t.Fatalf("SetMeasure = %v, want ErrDraftNotOpen", err)
	}
}

func TestListMeasures_ByObjectiveAndByDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, first, _, _ := seedLiveAgency(t, mgr, ctx)
	second := objectiveIn(t, mgr, ctx, first.DraftID, g.ID, "Second")

	for _, in := range []models.Measure{
		{ObjectiveID: first.ID, Name: "A", Target: "1"},
		{ObjectiveID: first.ID, Name: "B", Target: "2"},
		{ObjectiveID: second.ID, Name: "C", Target: "3"},
	} {
		if _, err := mgr.SetMeasure(ctx, in); err != nil {
			t.Fatalf("SetMeasure %s: %v", in.Name, err)
		}
	}

	if got := measuresOf(t, mgr, ctx, first.DraftID, first.ID); len(got) != 2 || got[0].Name != "A" || got[1].Name != "B" {
		t.Errorf("first objective = %+v, want A then B", got)
	}
	if got := measuresOf(t, mgr, ctx, first.DraftID, second.ID); len(got) != 1 || got[0].Name != "C" {
		t.Errorf("second objective = %+v, want C", got)
	}
	if got := measuresOf(t, mgr, ctx, first.DraftID, ""); len(got) != 3 {
		t.Errorf("whole draft = %+v, want all three", got)
	}

	other, err := mgr.CreateDraft(ctx, "another", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if got := measuresOf(t, mgr, ctx, other.ID, ""); len(got) != 0 {
		t.Errorf("another draft = %+v, want none of this draft's measures", got)
	}
}

func TestSetMeasure_TheSameNameOnTwoObjectivesIsTwoMeasures(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, first, _, _ := seedLiveAgency(t, mgr, ctx)
	second := objectiveIn(t, mgr, ctx, first.DraftID, g.ID, "Second")

	for _, o := range []models.Objective{first, second} {
		if _, err := mgr.SetMeasure(ctx, models.Measure{ObjectiveID: o.ID, Name: "Shops joined", Target: "12"}); err != nil {
			t.Fatalf("SetMeasure: %v", err)
		}
	}
	if got := measuresOf(t, mgr, ctx, first.DraftID, ""); len(got) != 2 {
		t.Fatalf("listed = %+v, want one per objective", got)
	}
}

func TestDeleteObjective_TakesItsMeasuresWithIt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, doomed, _, _ := seedLiveAgency(t, mgr, ctx)
	kept := objectiveIn(t, mgr, ctx, doomed.DraftID, g.ID, "Kept")
	for _, o := range []models.Objective{doomed, kept} {
		if _, err := mgr.SetMeasure(ctx, models.Measure{ObjectiveID: o.ID, Name: "Shops joined", Target: "12"}); err != nil {
			t.Fatalf("SetMeasure: %v", err)
		}
	}

	if err := mgr.DeleteObjective(ctx, doomed.ID); err != nil {
		t.Fatalf("DeleteObjective: %v", err)
	}
	if got := measuresOf(t, mgr, ctx, doomed.DraftID, ""); len(got) != 1 || got[0].ObjectiveID != kept.ID {
		t.Fatalf("listed = %+v, want only the surviving objective's measure", got)
	}
}

func TestDeleteGoal_TakesTheMeasuresOfItsObjectivesWithIt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, o, _, _ := seedLiveAgency(t, mgr, ctx)
	if _, err := mgr.SetMeasure(ctx, models.Measure{ObjectiveID: o.ID, Name: "Shops joined", Target: "12"}); err != nil {
		t.Fatalf("SetMeasure: %v", err)
	}
	if err := mgr.DeleteGoal(ctx, g.ID); err != nil {
		t.Fatalf("DeleteGoal: %v", err)
	}
	if got := measuresOf(t, mgr, ctx, o.DraftID, ""); len(got) != 0 {
		t.Fatalf("listed = %+v, want the goal's measures gone", got)
	}
}

func TestCreateDraft_ForkFromDraftCarriesMeasuresOnTheCopiedObjectives(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	source, err := mgr.SetMeasure(ctx, models.Measure{ObjectiveID: o.ID, Name: "Shops joined", Target: "12", Unit: "shops", Status: "on track"})
	if err != nil {
		t.Fatalf("SetMeasure: %v", err)
	}

	fork, err := mgr.CreateDraft(ctx, "fork", o.DraftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	copied, err := mgr.ListObjectives(ctx, fork.ID, mwanachamaagency.ObjectiveFilter{})
	if err != nil || len(copied) != 1 {
		t.Fatalf("ListObjectives = %+v, %v", copied, err)
	}

	got := measuresOf(t, mgr, ctx, fork.ID, "")
	if len(got) != 1 {
		t.Fatalf("fork measures = %+v, want the one copied", got)
	}
	if got[0].ID == source.ID || got[0].ObjectiveID != copied[0].ID || got[0].DraftID != fork.ID {
		t.Errorf("fork measure = %+v, want a new row on the copied objective %q in the fork", got[0], copied[0].ID)
	}
	if got[0].Name != "Shops joined" || got[0].Target != "12" || got[0].Unit != "shops" || got[0].Status != "on track" {
		t.Errorf("fork measure = %+v, want the fields copied", got[0])
	}

	if _, err := mgr.SetMeasure(ctx, models.Measure{ObjectiveID: copied[0].ID, Name: "Shops joined", Target: "99"}); err != nil {
		t.Fatalf("SetMeasure on the fork: %v", err)
	}
	if again := measuresOf(t, mgr, ctx, o.DraftID, ""); len(again) != 1 || again[0].Target != "12" {
		t.Fatalf("source measures = %+v, want the source untouched by the fork's edit", again)
	}
}

func TestPublishAgency_CarriesMeasuresAndAForkFromTheTagGetsThem(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	if _, err := mgr.SetMeasure(ctx, models.Measure{ObjectiveID: o.ID, Name: "Shops joined", Target: "12", Unit: "shops", Status: "on track"}); err != nil {
		t.Fatalf("SetMeasure: %v", err)
	}

	pub, err := mgr.PublishAgency(ctx, o.DraftID, "release-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if len(pub.Document.Measures) != 1 {
		t.Fatalf("document measures = %+v, want the one", pub.Document.Measures)
	}
	dm := pub.Document.Measures[0]
	if dm.ObjectiveID != o.ID || dm.Name != "Shops joined" || dm.Target != "12" || dm.Unit != "shops" || dm.Status != "on track" {
		t.Fatalf("document measure = %+v", dm)
	}

	fork, err := mgr.CreateDraft(ctx, "back to release-1", "release-1", "publication")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	objectives, err := mgr.ListObjectives(ctx, fork.ID, mwanachamaagency.ObjectiveFilter{})
	if err != nil || len(objectives) != 1 {
		t.Fatalf("ListObjectives = %+v, %v", objectives, err)
	}
	got := measuresOf(t, mgr, ctx, fork.ID, objectives[0].ID)
	if len(got) != 1 || got[0].Name != "Shops joined" || got[0].Target != "12" || got[0].Unit != "shops" || got[0].Status != "on track" {
		t.Fatalf("fork measures = %+v, want the published measure on the copied objective", got)
	}
}

func TestPublishAgency_ChangingAMeasureIsAChange(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)

	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if _, err := mgr.SetMeasure(ctx, models.Measure{ObjectiveID: o.ID, Name: "Shops joined", Target: "12"}); err != nil {
		t.Fatalf("SetMeasure: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); err != nil {
		t.Fatalf("PublishAgency after a new measure: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("PublishAgency unchanged = %v, want ErrNoChangesDetected", err)
	}
	if _, err := mgr.SetMeasure(ctx, models.Measure{ObjectiveID: o.ID, Name: "Shops joined", Target: "20"}); err != nil {
		t.Fatalf("SetMeasure (retarget): %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); err != nil {
		t.Fatalf("PublishAgency after a new target: %v", err)
	}
}
