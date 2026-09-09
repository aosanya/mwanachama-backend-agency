package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestWorkPlanCRUD(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	w, err := mgr.CreateWorkPlan(ctx, models.WorkPlan{
		Code:         "notify-on-push",
		Name:         "Notify on push",
		TriggerTopic: `^git\.push\.`,
		Enabled:      true,
		Ordinality:   1,
	})
	if err != nil {
		t.Fatalf("CreateWorkPlan: %v", err)
	}
	if w.ID == "" {
		t.Fatal("expected minted ID")
	}
	if w.Code != "notify-on-push" {
		t.Fatalf("Code not round-tripped: %+v", w)
	}

	got, err := mgr.GetWorkPlan(ctx, w.ID)
	if err != nil {
		t.Fatalf("GetWorkPlan: %v", err)
	}
	if got.Name != "Notify on push" {
		t.Fatalf("unexpected work plan: %+v", got)
	}

	w.Name = "Notify on push v2"
	w.Ordinality = 0 // exercise the zero-value update path
	updated, err := mgr.UpdateWorkPlan(ctx, w)
	if err != nil {
		t.Fatalf("UpdateWorkPlan: %v", err)
	}
	if updated.Name != "Notify on push v2" || updated.Ordinality != 0 {
		t.Fatalf("update did not apply zero-value field: %+v", updated)
	}

	live, err := mgr.ListWorkPlans(ctx)
	if err != nil {
		t.Fatalf("ListWorkPlans: %v", err)
	}
	if len(live) != 1 {
		t.Fatalf("expected 1 live work plan, got %d", len(live))
	}
}

func TestGetWorkPlan_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.GetWorkPlan(context.Background(), "unknown"); !errors.Is(err, mwanachamaagency.ErrWorkPlanNotFound) {
		t.Fatalf("expected ErrWorkPlanNotFound, got %v", err)
	}
}

func TestCreateWorkPlan_InvalidRegex(t *testing.T) {
	mgr := newTestManager(t)
	_, err := mgr.CreateWorkPlan(context.Background(), models.WorkPlan{
		Name:         "bad regex",
		TriggerTopic: "(unclosed",
	})
	if !errors.Is(err, mwanachamaagency.ErrInvalidRegex) {
		t.Fatalf("expected ErrInvalidRegex, got %v", err)
	}
}

func TestMatchWorkPlans(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	matching, err := mgr.CreateWorkPlan(ctx, models.WorkPlan{
		Name:         "matches",
		TriggerTopic: `^git\.push\..*`,
		Enabled:      true,
		Ordinality:   2,
	})
	if err != nil {
		t.Fatalf("CreateWorkPlan (matching): %v", err)
	}
	if _, err := mgr.AddContextSource(ctx, matching.ID, models.ContextSource{
		SourceType: models.ContextSourceGit,
		Git:        &models.GitContextSourceConfig{Signals: "authority"},
	}); err != nil {
		t.Fatalf("AddContextSource: %v", err)
	}

	if _, err := mgr.CreateWorkPlan(ctx, models.WorkPlan{
		Name:         "wrong topic",
		TriggerTopic: `^comm\..*`,
		Enabled:      true,
		Ordinality:   1,
	}); err != nil {
		t.Fatalf("CreateWorkPlan (wrong topic): %v", err)
	}

	if _, err := mgr.CreateWorkPlan(ctx, models.WorkPlan{
		Name:         "disabled",
		TriggerTopic: `^git\.push\..*`,
		Enabled:      false,
		Ordinality:   0,
	}); err != nil {
		t.Fatalf("CreateWorkPlan (disabled): %v", err)
	}

	matches, err := mgr.MatchWorkPlans(ctx, "git.push.main", "{}")
	if err != nil {
		t.Fatalf("MatchWorkPlans: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d: %+v", len(matches), matches)
	}
	if matches[0].WorkPlan.ID != matching.ID {
		t.Fatalf("matched wrong work plan: %+v", matches[0].WorkPlan)
	}
	if len(matches[0].ContextSources) != 1 {
		t.Fatalf("expected 1 context source on the match, got %d", len(matches[0].ContextSources))
	}
}

func TestMatchWorkPlans_PayloadCondition(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.CreateWorkPlan(ctx, models.WorkPlan{
		Name:             "payload gated",
		TriggerTopic:     `^git\.push\..*`,
		PayloadCondition: `"branch":"main"`,
		Enabled:          true,
	}); err != nil {
		t.Fatalf("CreateWorkPlan: %v", err)
	}

	matches, err := mgr.MatchWorkPlans(ctx, "git.push.main", `{"branch":"dev"}`)
	if err != nil {
		t.Fatalf("MatchWorkPlans: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected no matches for non-matching payload, got %d", len(matches))
	}

	matches, err = mgr.MatchWorkPlans(ctx, "git.push.main", `{"branch":"main"}`)
	if err != nil {
		t.Fatalf("MatchWorkPlans: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match for matching payload, got %d", len(matches))
	}
}
