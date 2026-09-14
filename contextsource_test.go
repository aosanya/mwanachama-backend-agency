package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func newTestWorkPlan(t *testing.T, mgr mwanachamaagency.AgencyManager) models.WorkPlan {
	t.Helper()
	w, err := mgr.CreateWorkPlan(context.Background(), models.WorkPlan{
		Name:         "test plan",
		TriggerTopic: ".*",
	})
	if err != nil {
		t.Fatalf("CreateWorkPlan: %v", err)
	}
	return w
}

func TestContextSourceCRUD(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	w := newTestWorkPlan(t, mgr)

	git, err := mgr.AddContextSource(ctx, w.ID, models.ContextSource{
		SourceType: models.ContextSourceGit,
		Git:        &models.GitContextSourceConfig{Signals: "authority,contributor", MaxResults: 20},
	})
	if err != nil {
		t.Fatalf("AddContextSource (git): %v", err)
	}
	if git.ID == "" {
		t.Fatal("expected minted ID")
	}
	if git.Git == nil || git.Git.Signals != "authority,contributor" {
		t.Fatalf("git config not round-tripped: %+v", git)
	}

	comm, err := mgr.AddContextSource(ctx, w.ID, models.ContextSource{
		SourceType: models.ContextSourceComm,
		Comm:       &models.CommContextSourceConfig{LookbackDays: 7},
	})
	if err != nil {
		t.Fatalf("AddContextSource (comm): %v", err)
	}
	if comm.Git != nil || comm.Work != nil {
		t.Fatalf("expected only Comm populated: %+v", comm)
	}

	sources, err := mgr.ListContextSources(ctx, w.ID, mwanachamaagency.ContextSourceFilter{})
	if err != nil {
		t.Fatalf("ListContextSources: %v", err)
	}
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}

	if err := mgr.RemoveContextSource(ctx, w.ID, git.ID); err != nil {
		t.Fatalf("RemoveContextSource: %v", err)
	}
	sources, err = mgr.ListContextSources(ctx, w.ID, mwanachamaagency.ContextSourceFilter{})
	if err != nil {
		t.Fatalf("ListContextSources after remove: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source after remove, got %d", len(sources))
	}
}

func TestAddContextSource_WorkPlanNotFound(t *testing.T) {
	mgr := newTestManager(t)
	_, err := mgr.AddContextSource(context.Background(), "unknown", models.ContextSource{
		SourceType: models.ContextSourceWork,
		Work:       &models.WorkContextSourceConfig{IncludeHistory: true},
	})
	if !errors.Is(err, mwanachamaagency.ErrWorkPlanNotFound) {
		t.Fatalf("expected ErrWorkPlanNotFound, got %v", err)
	}
}

// TestRemoveContextSource_WrongWorkPlan proves the fix over the source's
// RemoveContextSource, which accepted a workPlanID it never checked: a
// caller cannot delete a ContextSource by giving the wrong owning
// WorkPlanID.
func TestRemoveContextSource_WrongWorkPlan(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	w1 := newTestWorkPlan(t, mgr)
	w2 := newTestWorkPlan(t, mgr)

	cs, err := mgr.AddContextSource(ctx, w1.ID, models.ContextSource{
		SourceType: models.ContextSourceWork,
		Work:       &models.WorkContextSourceConfig{},
	})
	if err != nil {
		t.Fatalf("AddContextSource: %v", err)
	}

	err = mgr.RemoveContextSource(ctx, w2.ID, cs.ID)
	if !errors.Is(err, mwanachamaagency.ErrContextSourceNotFound) {
		t.Fatalf("expected ErrContextSourceNotFound removing via wrong work plan, got %v", err)
	}

	sources, err := mgr.ListContextSources(ctx, w1.ID, mwanachamaagency.ContextSourceFilter{})
	if err != nil {
		t.Fatalf("ListContextSources: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("source should survive the wrong-work-plan removal attempt, got %d", len(sources))
	}
}
