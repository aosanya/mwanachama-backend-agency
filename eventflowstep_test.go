package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func seedEventFlowSteps(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID string) {
	t.Helper()
	steps := []models.EventFlowStep{
		{DraftID: draftID, Code: "planning:1", WorkflowCode: "planning", Step: "1", StepType: "start", Consumer: "codevaldwork", Ordinality: 1},
		{DraftID: draftID, Code: "planning:2", WorkflowCode: "planning", Step: "2", StepType: "handler", Consumer: "codevaldwork",
			HandlerCode: "wp-planning-step2", TriggerTopic: "planning.step1.done", Ordinality: 2},
		{DraftID: draftID, Code: "review:1", WorkflowCode: "review", Step: "1", StepType: "start", Consumer: "codevaldai", Ordinality: 1},
	}
	for _, s := range steps {
		if _, err := mgr.CreateEventFlowStep(ctx, s); err != nil {
			t.Fatalf("CreateEventFlowStep(%s): %v", s.Code, err)
		}
	}
}

func TestLookupFlowStep(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)
	seedEventFlowSteps(t, mgr, ctx, draftID)

	byHandler, err := mgr.LookupFlowStep(ctx, draftID, models.EventFlowStepLookup{HandlerCode: "wp-planning-step2"})
	if err != nil {
		t.Fatalf("lookup by HandlerCode: %v", err)
	}
	if byHandler.Code != "planning:2" {
		t.Fatalf("expected planning:2, got %q", byHandler.Code)
	}

	byTopic, err := mgr.LookupFlowStep(ctx, draftID, models.EventFlowStepLookup{TriggerTopic: "planning.step1.done", Consumer: "codevaldwork"})
	if err != nil {
		t.Fatalf("lookup by (TriggerTopic, Consumer): %v", err)
	}
	if byTopic.Code != "planning:2" {
		t.Fatalf("expected planning:2, got %q", byTopic.Code)
	}

	byCode, err := mgr.LookupFlowStep(ctx, draftID, models.EventFlowStepLookup{Code: "review:1"})
	if err != nil {
		t.Fatalf("lookup by Code: %v", err)
	}
	if byCode.Consumer != "codevaldai" {
		t.Fatalf("expected codevaldai, got %q", byCode.Consumer)
	}

	if _, err := mgr.LookupFlowStep(ctx, draftID, models.EventFlowStepLookup{}); !errors.Is(err, mwanachamaagency.ErrInvalidLookup) {
		t.Fatalf("expected ErrInvalidLookup for an empty lookup, got %v", err)
	}
	if _, err := mgr.LookupFlowStep(ctx, draftID, models.EventFlowStepLookup{TriggerTopic: "x"}); !errors.Is(err, mwanachamaagency.ErrInvalidLookup) {
		t.Fatalf("expected ErrInvalidLookup for TriggerTopic without Consumer, got %v", err)
	}
	if _, err := mgr.LookupFlowStep(ctx, draftID, models.EventFlowStepLookup{Code: "does-not-exist"}); !errors.Is(err, mwanachamaagency.ErrEventFlowStepNotFound) {
		t.Fatalf("expected ErrEventFlowStepNotFound, got %v", err)
	}
}

func TestListEventFlowSteps(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)
	seedEventFlowSteps(t, mgr, ctx, draftID)

	all, err := mgr.ListEventFlowSteps(ctx, draftID, "", mwanachamaagency.EventFlowStepFilter{})
	if err != nil {
		t.Fatalf("ListEventFlowSteps: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(all))
	}

	planning, err := mgr.ListEventFlowSteps(ctx, draftID, "planning", mwanachamaagency.EventFlowStepFilter{})
	if err != nil {
		t.Fatalf("ListEventFlowSteps(planning): %v", err)
	}
	if len(planning) != 2 || planning[0].Step != "1" || planning[1].Step != "2" {
		t.Fatalf("expected planning's 2 steps in order, got %+v", planning)
	}
}
