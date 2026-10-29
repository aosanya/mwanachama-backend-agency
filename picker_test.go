package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func pickerManager(t *testing.T, groups mwanachamaagency.GroupLister) mwanachamaagency.AgencyManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := mwanachamaagency.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	mgr, err := mwanachamaagency.NewAgencyManager(db, s, mwanachamaagency.WithGroupLister(groups))
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	return mgr
}

func actorsOf(p models.Pickers, kind models.PickerKind) []string {
	var names []string
	for _, a := range p.Actors {
		if a.Kind == kind {
			names = append(names, a.Name)
		}
	}
	return names
}

func TestListPickers_TheAgencyItsRolesAndItsGroupsAreOneList(t *testing.T) {
	groups := func(context.Context) ([]models.PickerActor, error) {
		return []models.PickerActor{{ID: "g1", Name: "Farmers"}, {ID: "g2", Name: "Riders"}}, nil
	}
	mgr := pickerManager(t, groups)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)
	for _, name := range []string{"Rider", "Farmer"} {
		if _, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: name}); err != nil {
			t.Fatalf("CreateRole %s: %v", name, err)
		}
	}

	got, err := mgr.ListPickers(ctx, draftID)
	if err != nil {
		t.Fatalf("ListPickers: %v", err)
	}
	if names := actorsOf(got, models.PickerAgency); len(names) != 1 || names[0] != "The agency" {
		t.Errorf("agency actors = %v, want the agency once", names)
	}
	if names := actorsOf(got, models.PickerRole); len(names) != 2 {
		t.Errorf("role actors = %v, want the draft's two roles", names)
	}
	groupActors := actorsOf(got, models.PickerGroup)
	if len(groupActors) != 2 || groupActors[0] != "Farmers" || groupActors[1] != "Riders" {
		t.Errorf("group actors = %v, want Farmers and Riders", groupActors)
	}
	if got.Actors[0].Kind != models.PickerAgency {
		t.Errorf("the agency is not first: %+v", got.Actors[0])
	}
}

func TestListPickers_AnswerTypesAreTheFormsModulesTen(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	got, err := mgr.ListPickers(ctx, newDraftAgency(t, mgr, ctx))
	if err != nil {
		t.Fatalf("ListPickers: %v", err)
	}
	want := []models.AnswerType{"yes_no", "single_choice", "multi_select", "free_text", "date", "time", "number", "currency", "photo", "file"}
	if len(got.AnswerTypes) != len(want) {
		t.Fatalf("answer types = %v, want %v", got.AnswerTypes, want)
	}
	for i := range want {
		if got.AnswerTypes[i] != want[i] {
			t.Errorf("answer type %d = %q, want %q", i, got.AnswerTypes[i], want[i])
		}
	}
}

func TestListPickers_WithoutAGroupSourceTheListHasNoGroups(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	got, err := mgr.ListPickers(ctx, newDraftAgency(t, mgr, ctx))
	if err != nil {
		t.Fatalf("ListPickers: %v", err)
	}
	if names := actorsOf(got, models.PickerGroup); len(names) != 0 {
		t.Errorf("group actors = %v, want none", names)
	}
}

func TestListPickers_AFailingGroupSourceIsReportedNotHidden(t *testing.T) {
	boom := errors.New("actor store down")
	mgr := pickerManager(t, func(context.Context) ([]models.PickerActor, error) { return nil, boom })
	ctx := context.Background()
	if _, err := mgr.ListPickers(ctx, newDraftAgency(t, mgr, ctx)); !errors.Is(err, boom) {
		t.Fatalf("ListPickers = %v, want the group source's error", err)
	}
}

func TestListPickers_RolesComeFromTheAskedDraftOnly(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	first := newDraftAgency(t, mgr, ctx)
	if _, err := mgr.CreateRole(ctx, models.Role{DraftID: first, Name: "Rider"}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	second, err := mgr.CreateDraft(ctx, "empty", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	got, err := mgr.ListPickers(ctx, second.ID)
	if err != nil {
		t.Fatalf("ListPickers: %v", err)
	}
	if names := actorsOf(got, models.PickerRole); len(names) != 0 {
		t.Errorf("role actors in an empty draft = %v, want none", names)
	}
}

func TestWorkItemStep_TheFormsModulesTenAnswerTypesAreAccepted(t *testing.T) {
	w := newStepWorld(t)
	wf := w.workflow(t, "Every type")
	for _, answer := range models.AnswerTypes() {
		q := ask("Question?", answer)
		if answer.HasOptions() {
			q.Options = []string{"a", "b"}
		}
		in := w.item(wf, 1, stepSpec{title: "Step " + string(answer), kind: models.StepKindFillForm, role: farmer, form: "Form", questions: []models.StepQuestion{q}})
		if _, err := w.mgr.CreateWorkItem(context.Background(), in); err != nil {
			t.Errorf("answer type %s: %v", answer, err)
		}
	}
}

func TestWorkItemStep_AMultiSelectNamesItsOptions(t *testing.T) {
	w := newStepWorld(t)
	wf := w.workflow(t, "Multi")
	in := w.item(wf, 1, stepSpec{title: "Pick", kind: models.StepKindFillForm, role: farmer, form: "Form", questions: []models.StepQuestion{ask("Which?", models.AnswerMultiSelect)}})
	if _, err := w.mgr.CreateWorkItem(context.Background(), in); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Fatalf("multi_select with no options: want ErrInvalidWorkItem, got %v", err)
	}
}
