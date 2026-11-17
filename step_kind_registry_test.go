package mwanachamaagency_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

const fakeKind models.StepKind = "fake_calc"

func fakeHooks(marker string) mwanachamaagency.StepKindHooks {
	return mwanachamaagency.StepKindHooks{
		Shape: func(wi models.WorkItem) error {
			if strings.Contains(wi.Title, "refuse-shape") {
				return errors.New("a fake step does not allow this title")
			}
			return nil
		},
		References: func(ctx context.Context, lookup mwanachamaagency.StepLookup, wi models.WorkItem) error {
			if strings.Contains(wi.Title, "refuse-reference") {
				return errors.New("a fake step points at nothing")
			}
			return nil
		},
		Scan: func(report mwanachamaagency.StepScanReport, wi models.WorkItem, steps []models.WorkItem) {
			report.Add(models.IssueSeverityWarning, "work_item", wi.ID, "Fake step %q was seen by the scan among %d steps", wi.Title, len(steps))
		},
		Remap: func(wi *models.WorkItem, ids mwanachamaagency.StepIDMaps) {
			wi.Prompt = "forked:" + wi.Prompt
		},
		ToDocument: func(wi models.WorkItem, out *models.DocumentWorkItem) {
			out.Description = "fake:" + wi.Description
		},
		FromDocument: func(in models.DocumentWorkItem, row *models.WorkItem, ids mwanachamaagency.StepIDMaps) {
			row.Description = strings.TrimPrefix(in.Description, "fake:")
		},
		Hash: func(wi models.WorkItem) []string {
			return []string{mwanachamaagency.StepHashRecord("FakeStep", map[string]any{"id": wi.ID, "marker": marker})}
		},
	}
}

func seedFakeWorkflow(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context) (models.Objective, models.Workflow) {
	t.Helper()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	workflows, err := mgr.ListWorkflows(ctx, o.DraftID, mwanachamaagency.WorkflowFilter{})
	if err != nil || len(workflows) == 0 {
		t.Fatalf("ListWorkflows: %v (%d)", err, len(workflows))
	}
	return o, workflows[0]
}

func createFakeStep(mgr mwanachamaagency.AgencyManager, ctx context.Context, o models.Objective, w models.Workflow, title string) (models.WorkItem, error) {
	return mgr.CreateWorkItem(ctx, models.WorkItem{
		DraftID: o.DraftID, WorkflowID: w.ID, Title: title, Kind: fakeKind,
		Description: "worked out", Prompt: "t",
	})
}

func seedFakeStep(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, title string) (models.Objective, models.WorkItem, error) {
	t.Helper()
	o, w := seedFakeWorkflow(t, mgr, ctx)
	wi, err := createFakeStep(mgr, ctx, o, w, title)
	return o, wi, err
}

func TestStepKindRegistry_AKindWithNothingRegisteredStaysUnknown(t *testing.T) {
	mwanachamaagency.ClearStepKindRegistry()
	mgr := newTestManager(t)
	ctx := context.Background()
	if _, _, err := seedFakeStep(t, mgr, ctx, "plain"); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Fatalf("an unregistered kind must still be refused as an unknown kind, got %v", err)
	}
}

func TestStepKindRegistry_ARegisteredKindRefusesASaveAtShapeAndAtReference(t *testing.T) {
	mwanachamaagency.ClearStepKindRegistry()
	mwanachamaagency.RegisterStepKind(fakeKind, fakeHooks("a"))
	t.Cleanup(mwanachamaagency.ClearStepKindRegistry)
	mgr := newTestManager(t)
	ctx := context.Background()

	o, w := seedFakeWorkflow(t, mgr, ctx)
	if _, err := createFakeStep(mgr, ctx, o, w, "a good step"); err != nil {
		t.Fatalf("a registered kind that passes its checks must save: %v", err)
	}
	if _, err := createFakeStep(mgr, ctx, o, w, "refuse-shape"); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Fatalf("the shape check of the kind must refuse the save with ErrInvalidWorkItem, got %v", err)
	}
	if _, err := createFakeStep(mgr, ctx, o, w, "refuse-reference"); err == nil {
		t.Fatal("the reference check of the kind must refuse the save")
	}
}

func TestStepKindRegistry_AFindingOfTheKindAppearsInTheScanAndAKindFreeCheckRunsOncePerScan(t *testing.T) {
	mwanachamaagency.ClearStepKindRegistry()
	mwanachamaagency.RegisterStepKind(fakeKind, fakeHooks("a"))
	runs := 0
	mwanachamaagency.RegisterIssueCheck(func(report mwanachamaagency.StepScanReport, steps []models.WorkItem) error {
		runs++
		report.Add(models.IssueSeverityInfo, "work_item", "", "The kind free check saw %d steps", len(steps))
		return nil
	})
	t.Cleanup(mwanachamaagency.ClearStepKindRegistry)
	mgr := newTestManager(t)
	ctx := context.Background()
	o, _, err := seedFakeStep(t, mgr, ctx, "a good step")
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	before := runs
	issues, err := mgr.ListIssues(ctx, o.DraftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	var fakeSeen, freeSeen int
	for _, is := range issues {
		if strings.Contains(is.Message, `Fake step "a good step" was seen by the scan`) {
			fakeSeen++
		}
		if strings.Contains(is.Message, "The kind free check saw") {
			freeSeen++
		}
	}
	if fakeSeen != 1 {
		t.Fatalf("the kind's finding must appear once, got %d among %d issues", fakeSeen, len(issues))
	}
	if freeSeen != 1 || runs-before < 1 {
		t.Fatalf("the kind free check must run once per scan, finding %d, runs %d", freeSeen, runs-before)
	}
}

func TestStepKindRegistry_AForkRemapsTheKindsIdsAndTheDocumentCarriesItsFields(t *testing.T) {
	mwanachamaagency.ClearStepKindRegistry()
	mwanachamaagency.RegisterStepKind(fakeKind, fakeHooks("a"))
	t.Cleanup(mwanachamaagency.ClearStepKindRegistry)
	mgr := newTestManager(t)
	ctx := context.Background()
	o, _, err := seedFakeStep(t, mgr, ctx, "a good step")
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	fork, err := mgr.CreateDraft(ctx, "fork", o.DraftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	forked, err := mgr.ListWorkItems(ctx, fork.ID, mwanachamaagency.WorkItemFilter{})
	if err != nil {
		t.Fatalf("ListWorkItems(fork): %v", err)
	}
	var found bool
	for _, wi := range forked {
		if wi.Kind == fakeKind {
			found = true
			if wi.Prompt != "forked:t" {
				t.Fatalf("the kind's remap must run on a fork, prompt is %q", wi.Prompt)
			}
		}
	}
	if !found {
		t.Fatal("the fake step must be forked")
	}

	pub, err := mgr.PublishAgency(ctx, o.DraftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	var docStep models.DocumentWorkItem
	for _, wi := range pub.Document.WorkItems {
		if wi.Kind == fakeKind {
			docStep = wi
		}
	}
	if docStep.Description != "fake:worked out" {
		t.Fatalf("the kind's fields must be written into the document, description is %q", docStep.Description)
	}
	out, err := mgr.ImportDocument(ctx, "imported", pub.Document)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	imported, err := mgr.ListWorkItems(ctx, out.Draft.ID, mwanachamaagency.WorkItemFilter{})
	if err != nil {
		t.Fatalf("ListWorkItems(imported): %v", err)
	}
	var back bool
	for _, wi := range imported {
		if wi.Kind == fakeKind && wi.Description == "worked out" {
			back = true
		}
	}
	if !back {
		t.Fatalf("the kind's fields must be read back on import: %+v", imported)
	}
}

func TestStepKindRegistry_TheKindsRecordEntersTheHashOnlyWhenItIsRegistered(t *testing.T) {
	mwanachamaagency.ClearStepKindRegistry()
	mwanachamaagency.RegisterStepKind(fakeKind, fakeHooks("a"))
	t.Cleanup(mwanachamaagency.ClearStepKindRegistry)
	mgr := newTestManager(t)
	ctx := context.Background()
	o, _, err := seedFakeStep(t, mgr, ctx, "a good step")
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("the same content must hash the same, got %v", err)
	}
	mwanachamaagency.ClearStepKindRegistry()
	mwanachamaagency.RegisterStepKind(fakeKind, fakeHooks("b"))
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); err != nil {
		t.Fatalf("a changed record of the kind must change the hash: %v", err)
	}
}
