package mwanachamaagency_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestStepKindRegistry_TwoRegistrationsOfOneKindBothRun(t *testing.T) {
	mwanachamaagency.ClearStepKindRegistry()
	t.Cleanup(mwanachamaagency.ClearStepKindRegistry)
	mwanachamaagency.RegisterStepKind(fakeKind, mwanachamaagency.StepKindHooks{
		Shape: func(wi models.WorkItem) error {
			if strings.Contains(wi.Title, "first-refuses") {
				return errors.New("the first registration refuses this title")
			}
			return nil
		},
		Scan: func(report mwanachamaagency.StepScanReport, wi models.WorkItem, steps []models.WorkItem) {
			report.Add(models.IssueSeverityInfo, "work_item", wi.ID, "first saw %q", wi.Title)
		},
	})
	mwanachamaagency.RegisterStepKind(fakeKind, mwanachamaagency.StepKindHooks{
		Shape: func(wi models.WorkItem) error {
			if strings.Contains(wi.Title, "second-refuses") {
				return errors.New("the second registration refuses this title")
			}
			return nil
		},
		References: func(ctx context.Context, lookup mwanachamaagency.StepLookup, wi models.WorkItem) error {
			if strings.Contains(wi.Title, "second-refuses-reference") {
				return errors.New("the second registration refuses this reference")
			}
			return nil
		},
		Scan: func(report mwanachamaagency.StepScanReport, wi models.WorkItem, steps []models.WorkItem) {
			report.Add(models.IssueSeverityInfo, "work_item", wi.ID, "second saw %q", wi.Title)
		},
		Hash: func(wi models.WorkItem) []string {
			return []string{mwanachamaagency.StepHashRecord("SecondOnly", map[string]any{"id": wi.ID})}
		},
	})
	mgr := newTestManager(t)
	ctx := context.Background()
	o, w := seedFakeWorkflow(t, mgr, ctx)

	if _, err := createFakeStep(mgr, ctx, o, w, "fine"); err != nil {
		t.Fatalf("a step both registrations accept must save: %v", err)
	}
	if _, err := createFakeStep(mgr, ctx, o, w, "first-refuses"); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Fatalf("the first registration's shape check must still refuse, got %v", err)
	}
	if _, err := createFakeStep(mgr, ctx, o, w, "second-refuses"); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Fatalf("the second registration's shape check must refuse too, got %v", err)
	}
	if _, err := createFakeStep(mgr, ctx, o, w, "second-refuses-reference"); err == nil {
		t.Fatal("a hook only the second registration has must run")
	}

	issues, err := mgr.ListIssues(ctx, o.DraftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	var first, second int
	for _, issue := range issues {
		first += strings.Count(issue.Message, "first saw")
		second += strings.Count(issue.Message, "second saw")
	}
	if first != 1 || second != 1 {
		t.Fatalf("each registration's finding must appear once, got first %d second %d", first, second)
	}
}
