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

const (
	farmersGroupID = "group-farmers"
	ridersGroupID  = "group-riders"
)

func groupsOffered(ids ...string) mwanachamaagency.GroupLister {
	return func(context.Context) ([]models.PickerActor, error) {
		out := make([]models.PickerActor, len(ids))
		for i, id := range ids {
			out[i] = models.PickerActor{ID: id, Name: id, Kind: models.PickerGroup}
		}
		return out, nil
	}
}

func newStepWorldWithGroups(t *testing.T, groups mwanachamaagency.GroupLister) stepWorld {
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
	var opts []mwanachamaagency.Option
	if groups != nil {
		opts = append(opts, mwanachamaagency.WithGroupLister(groups))
	}
	mgr, err := mwanachamaagency.NewAgencyManager(db, s, opts...)
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	w := stepWorld{mgr: mgr, draftID: o.DraftID, objective: o, roles: map[string]models.Role{}}
	for _, name := range []string{coordinator, farmer, supplier} {
		r, err := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: name})
		if err != nil {
			t.Fatalf("CreateRole %s: %v", name, err)
		}
		w.roles[name] = r
	}
	return w
}

func stepActedOnBy(w stepWorld, wf models.Workflow, id string) models.WorkItem {
	in := w.item(wf, 1, buyNowSteps[0])
	in.RoleID = id
	return in
}

func stepSentTo(w stepWorld, wf models.Workflow, id string) models.WorkItem {
	in := w.item(wf, 1, groupOrderSteps[1])
	in.RecipientRoleID = id
	return in
}

func TestWhoDoesIt_AGroupThePickersOfferActsOnAStep(t *testing.T) {
	w := newStepWorldWithGroups(t, groupsOffered(farmersGroupID, ridersGroupID))
	ctx := context.Background()
	wf := w.workflow(t, "Buy now")

	created, err := w.mgr.CreateWorkItem(ctx, stepActedOnBy(w, wf, farmersGroupID))
	if err != nil {
		t.Fatalf("a step done by an offered group: %v", err)
	}
	if got, _ := w.mgr.GetWorkItem(ctx, created.ID); got.RoleID != farmersGroupID {
		t.Errorf("read back actor = %q, want %q", got.RoleID, farmersGroupID)
	}
	created.RoleID = ridersGroupID
	if _, err := w.mgr.UpdateWorkItem(ctx, created); err != nil {
		t.Errorf("update to another offered group: %v", err)
	}
}

func TestWhoDoesIt_AnActorThePickersDoNotOfferIsRefused(t *testing.T) {
	w := newStepWorldWithGroups(t, groupsOffered(farmersGroupID))
	ctx := context.Background()
	wf := w.workflow(t, "Buy now")

	if _, err := w.mgr.CreateWorkItem(ctx, stepActedOnBy(w, wf, "group-nobody-offers")); !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Errorf("a group the pickers do not offer: want ErrRoleNotFound, got %v", err)
	}
	if _, err := w.mgr.CreateWorkItem(ctx, stepActedOnBy(w, wf, farmersGroupID+","+w.roles[farmer].ID)); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Errorf("a group and a role together: want ErrInvalidWorkItem, got %v", err)
	}
}

func TestWhoDoesIt_WithoutAGroupSourceNoGroupActs(t *testing.T) {
	w := newStepWorldWithGroups(t, nil)
	wf := w.workflow(t, "Buy now")

	if _, err := w.mgr.CreateWorkItem(context.Background(), stepActedOnBy(w, wf, farmersGroupID)); !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Errorf("a group with no group source: want ErrRoleNotFound, got %v", err)
	}
}

func TestWhoDoesIt_AFailingGroupSourceIsReportedNotTakenForNotFound(t *testing.T) {
	down := errors.New("actor module down")
	w := newStepWorldWithGroups(t, func(context.Context) ([]models.PickerActor, error) { return nil, down })
	wf := w.workflow(t, "Buy now")

	_, err := w.mgr.CreateWorkItem(context.Background(), stepActedOnBy(w, wf, farmersGroupID))
	if !errors.Is(err, down) || errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Errorf("a failing group source: want the source's error, got %v", err)
	}
	if _, err := w.mgr.CreateWorkItem(context.Background(), stepActedOnBy(w, wf, w.roles[farmer].ID)); err != nil {
		t.Errorf("a role never asks the group source: %v", err)
	}
}

func TestWhoIsTold_AGroupThePickersOfferIsTheRecipient(t *testing.T) {
	w := newStepWorldWithGroups(t, groupsOffered(farmersGroupID))
	ctx := context.Background()
	wf := w.workflow(t, "Group order")

	sent, err := w.mgr.CreateWorkItem(ctx, stepSentTo(w, wf, farmersGroupID))
	if err != nil {
		t.Fatalf("a send_form to an offered group: %v", err)
	}
	if got, _ := w.mgr.GetWorkItem(ctx, sent.ID); got.RecipientRoleID != farmersGroupID {
		t.Errorf("read back recipient = %q, want %q", got.RecipientRoleID, farmersGroupID)
	}
	notify := w.item(wf, 2, groupOrderSteps[5])
	notify.RecipientRoleID = farmersGroupID
	if _, err := w.mgr.CreateWorkItem(ctx, notify); err != nil {
		t.Errorf("a notify to an offered group: %v", err)
	}
}

func TestWhoIsTold_ARecipientThePickersDoNotOfferIsRefused(t *testing.T) {
	w := newStepWorldWithGroups(t, groupsOffered(farmersGroupID))
	wf := w.workflow(t, "Group order")

	if _, err := w.mgr.CreateWorkItem(context.Background(), stepSentTo(w, wf, "group-nobody-offers")); !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Errorf("a recipient the pickers do not offer: want ErrRoleNotFound, got %v", err)
	}
	if _, err := w.mgr.CreateWorkItem(context.Background(), stepSentTo(w, wf, farmersGroupID+","+ridersGroupID)); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Errorf("two groups: want ErrInvalidWorkItem, got %v", err)
	}
	fill := w.item(wf, 2, buyNowSteps[0])
	fill.RecipientRoleID = farmersGroupID
	if _, err := w.mgr.CreateWorkItem(context.Background(), fill); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Errorf("a fill_form with a group recipient: want ErrInvalidWorkItem, got %v", err)
	}
}

func TestGroupActors_TheDraftScanFlagsOnlyTheOnesThePickersDoNotOffer(t *testing.T) {
	groups := groupsOffered(farmersGroupID)
	w := newStepWorldWithGroups(t, groups)
	ctx := context.Background()
	wf := w.workflow(t, "Group order")
	if _, err := w.mgr.CreateWorkItem(ctx, stepActedOnBy(w, wf, farmersGroupID)); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	sent := stepSentTo(w, wf, farmersGroupID)
	sent.Ordinality = 2
	if _, err := w.mgr.CreateWorkItem(ctx, sent); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	if _, err := w.mgr.CreateRecord(ctx, paymentsRecord(w, wf)); err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}

	issues, err := w.mgr.ListIssues(ctx, w.draftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	for _, i := range issues {
		if i.Severity == models.IssueSeverityError && (i.EntityType == "work_item" || i.EntityType == "record") {
			t.Errorf("a group actor was flagged: %s", i.Message)
		}
	}
}

func TestGroupActors_AnImportedDocumentKeepsTheGroupThatActsOnAStep(t *testing.T) {
	w := newStepWorldWithGroups(t, groupsOffered(farmersGroupID))
	ctx := context.Background()
	doc := models.AgencyDocument{
		Agency:             models.DocumentAgency{Name: "Ward office"},
		Goals:              []models.DocumentGoal{{ID: "g1", Code: "G-1", Title: "Hear residents"}},
		Objectives:         []models.DocumentObjective{{ID: "o1", Code: "O-1", GoalID: "g1", Title: "Ask residents"}},
		Roles:              []models.DocumentRole{{ID: "r1", Code: "R-1", Name: "Clerk"}},
		Workflows:          []models.DocumentWorkflow{{ID: "w1", Code: "W-1", Name: "Order"}},
		WorkflowObjectives: []models.DocumentWorkflowObjective{{ID: "wo1", WorkflowID: "w1", ObjectiveID: "o1"}},
		WorkItems: []models.DocumentWorkItem{{
			ID: "s1", Code: "S-1", WorkflowID: "w1", Title: "Ask", Ordinality: 1,
			Kind: models.StepKindFillForm, RoleID: farmersGroupID, FormName: "Ask form",
			Questions: []models.StepQuestion{ask("Enter the number of bags", models.AnswerNumber)},
		}},
	}

	out, err := w.mgr.ImportDocument(ctx, "ward", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	steps, err := w.mgr.ListWorkItems(ctx, out.Draft.ID, mwanachamaagency.WorkItemFilter{})
	if err != nil || len(steps) != 1 {
		t.Fatalf("ListWorkItems = %v, %v", steps, err)
	}
	if steps[0].RoleID != farmersGroupID {
		t.Errorf("imported actor = %q, want the group %q", steps[0].RoleID, farmersGroupID)
	}
}
