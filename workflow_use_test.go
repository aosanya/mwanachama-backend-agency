package mwanachamaagency_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func workflowUseManager(t *testing.T, runs mwanachamaagency.RunLister) mwanachamaagency.AgencyManager {
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
	mgr, err := mwanachamaagency.NewAgencyManager(db, s, mwanachamaagency.WithRunLister(runs))
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	return mgr
}

func workflowNamed(t *testing.T, list []models.Workflow, name string) models.Workflow {
	t.Helper()
	for _, w := range list {
		if w.Name == name {
			return w
		}
	}
	t.Fatalf("no workflow named %q in %v", name, list)
	return models.Workflow{}
}

func TestListDefaultWorkflows_CancelIsOneAndIsMarkedAsADefault(t *testing.T) {
	mgr := newTestManager(t)

	got, err := mgr.ListDefaultWorkflows(context.Background())
	if err != nil {
		t.Fatalf("ListDefaultWorkflows: %v", err)
	}
	cancel := workflowNamed(t, got, "Cancel")
	if !cancel.Default || cancel.Code != models.CancelWorkflowCode || cancel.InUse {
		t.Fatalf("Cancel = %+v, want a default coded %q that is not in use", cancel, models.CancelWorkflowCode)
	}
}

func TestListDefaultWorkflows_CancelIsInUseWhenAnOrderRunsIt(t *testing.T) {
	runs := func(context.Context) ([]models.WorkflowRunUse, error) {
		return []models.WorkflowRunUse{{WorkflowCode: models.CancelWorkflowCode, WorkflowVersion: 3}}, nil
	}
	mgr := workflowUseManager(t, runs)

	got, err := mgr.ListDefaultWorkflows(context.Background())
	if err != nil {
		t.Fatalf("ListDefaultWorkflows: %v", err)
	}
	cancel := workflowNamed(t, got, "Cancel")
	if !cancel.InUse || !reflect.DeepEqual(cancel.OrderVersions, []int{3}) {
		t.Fatalf("Cancel = %+v, want in use on version 3", cancel)
	}
}

func TestListWorkflows_AWorkflowWithAnOrderUnderWayIsInUseOnTheVersionTheOrdersAreOn(t *testing.T) {
	var running []models.WorkflowRunUse
	runs := func(context.Context) ([]models.WorkflowRunUse, error) { return running, nil }
	mgr := workflowUseManager(t, runs)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	idle, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: o.DraftID, Name: "Find it"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	linkWorkflowToObjective(t, mgr, o.DraftID, idle.ID, o.ID)
	if _, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: o.DraftID, WorkflowID: idle.ID, Title: "Ask"}); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	running = []models.WorkflowRunUse{
		{WorkflowCode: "W-1", WorkflowVersion: 1},
		{WorkflowCode: "W-1", WorkflowVersion: 1},
	}

	got, err := mgr.ListWorkflows(ctx, o.DraftID, mwanachamaagency.WorkflowFilter{})
	if err != nil {
		t.Fatalf("ListWorkflows: %v", err)
	}
	busy := workflowNamed(t, got, "Onboarding")
	if !busy.InUse || busy.OrdersUnderWay != 2 || !reflect.DeepEqual(busy.OrderVersions, []int{1}) || busy.Version != 1 {
		t.Fatalf("Onboarding = %+v, want in use by 2 orders on version 1, published at version 1", busy)
	}
	quiet := workflowNamed(t, got, "Find it")
	if quiet.InUse || quiet.OrdersUnderWay != 0 || len(quiet.OrderVersions) != 0 || quiet.Version != 1 {
		t.Fatalf("Find it = %+v, want not in use, published at version 1", quiet)
	}

	read, err := mgr.GetWorkflow(ctx, busy.ID)
	if err != nil {
		t.Fatalf("GetWorkflow: %v", err)
	}
	if !read.InUse || !reflect.DeepEqual(read.OrderVersions, []int{1}) {
		t.Fatalf("GetWorkflow = %+v, want in use on version 1", read)
	}
}

func TestListWorkflows_OrdersOnTwoVersionsListBothOldestFirst(t *testing.T) {
	runs := func(context.Context) ([]models.WorkflowRunUse, error) {
		return []models.WorkflowRunUse{
			{WorkflowCode: "W-1", WorkflowVersion: 2},
			{WorkflowCode: "W-1", WorkflowVersion: 1},
			{WorkflowCode: "W-1", WorkflowVersion: 2},
		}, nil
	}
	mgr := workflowUseManager(t, runs)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)
	if _, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftID, Name: "Group order"}); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	got, err := mgr.ListWorkflows(ctx, draftID, mwanachamaagency.WorkflowFilter{})
	if err != nil {
		t.Fatalf("ListWorkflows: %v", err)
	}
	w := workflowNamed(t, got, "Group order")
	if w.OrdersUnderWay != 3 || !reflect.DeepEqual(w.OrderVersions, []int{1, 2}) {
		t.Fatalf("Group order = %+v, want 3 orders on versions 1 and 2", w)
	}
}

func TestListWorkflows_WithoutARunListerNothingIsInUse(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)
	if _, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftID, Name: "Group order"}); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	got, err := mgr.ListWorkflows(ctx, draftID, mwanachamaagency.WorkflowFilter{})
	if err != nil {
		t.Fatalf("ListWorkflows: %v", err)
	}
	if w := workflowNamed(t, got, "Group order"); w.InUse || w.Version != 0 {
		t.Fatalf("Group order = %+v, want not in use and unpublished", w)
	}
}
