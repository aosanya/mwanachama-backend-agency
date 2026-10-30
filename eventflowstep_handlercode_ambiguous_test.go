package mwanachamaagency_test

import (
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAG65_LookupFlowStepByHandlerCodeRefusesWhenSeveralStepsShareIt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := t.Context()
	draftID := newDraftAgency(t, mgr, ctx)

	first, err := mgr.CreateEventFlowStep(ctx, models.EventFlowStep{
		DraftID:      draftID,
		WorkflowCode: "planning",
		Step:         "1",
		HandlerCode:  "shared-handler",
		Name:         "Validate",
	})
	if err != nil {
		t.Fatalf("CreateEventFlowStep(first): %v", err)
	}
	second, err := mgr.CreateEventFlowStep(ctx, models.EventFlowStep{
		DraftID:      draftID,
		WorkflowCode: "planning",
		Step:         "2",
		HandlerCode:  "shared-handler",
		Name:         "Notify",
	})
	if err != nil {
		t.Fatalf("CreateEventFlowStep(second): %v", err)
	}

	got, err := mgr.LookupFlowStep(ctx, draftID, models.EventFlowStepLookup{HandlerCode: "shared-handler"})
	if err == nil {
		t.Fatalf("AG65: LookupFlowStep by handler_code %q matches two steps (%q and %q) the blueprint's own handler_code index describes as \"which steps a given handler runs\" — legitimately more than one — yet the call answered one of them (%q, id %s) with no error; want it to refuse an ambiguous handler_code lookup rather than silently picking one of %s/%s",
			"shared-handler", first.ID, second.ID, got.Name, got.ID, first.ID, second.ID)
	}
}

func TestAG65_LookupFlowStepErrorNamesEveryStepSharingTheHandler(t *testing.T) {
	mgr := newTestManager(t)
	ctx := t.Context()
	draftID := newDraftAgency(t, mgr, ctx)
	var ids []string
	for _, step := range []string{"1", "2"} {
		created, err := mgr.CreateEventFlowStep(ctx, models.EventFlowStep{DraftID: draftID, WorkflowCode: "planning", Step: step, HandlerCode: "shared-handler", Name: "Step " + step})
		if err != nil {
			t.Fatalf("CreateEventFlowStep(%s): %v", step, err)
		}
		ids = append(ids, created.ID)
	}

	_, err := mgr.LookupFlowStep(ctx, draftID, models.EventFlowStepLookup{HandlerCode: "shared-handler"})
	if !errors.Is(err, mwanachamaagency.ErrAmbiguousLookup) {
		t.Fatalf("LookupFlowStep by shared handler_code = %v, want ErrAmbiguousLookup", err)
	}
	for _, id := range ids {
		if !strings.Contains(err.Error(), id) {
			t.Errorf("error %q does not name step %s", err, id)
		}
	}
}

func TestAG65_LookupFlowStepByTriggerTopicAndConsumerRefusesWhenSeveralStepsShareThem(t *testing.T) {
	mgr := newTestManager(t)
	ctx := t.Context()
	draftID := newDraftAgency(t, mgr, ctx)
	for _, step := range []string{"1", "2"} {
		if _, err := mgr.CreateEventFlowStep(ctx, models.EventFlowStep{DraftID: draftID, WorkflowCode: "planning", Step: step, TriggerTopic: "t.fired", Consumer: "svc", Name: "Step " + step}); err != nil {
			t.Fatalf("CreateEventFlowStep(%s): %v", step, err)
		}
	}

	_, err := mgr.LookupFlowStep(ctx, draftID, models.EventFlowStepLookup{TriggerTopic: "t.fired", Consumer: "svc"})
	if !errors.Is(err, mwanachamaagency.ErrAmbiguousLookup) {
		t.Fatalf("LookupFlowStep by shared topic and consumer = %v, want ErrAmbiguousLookup", err)
	}
}

func TestAG65_LookupFlowStepStillAnswersAUniqueHandler(t *testing.T) {
	mgr := newTestManager(t)
	ctx := t.Context()
	draftID := newDraftAgency(t, mgr, ctx)
	created, err := mgr.CreateEventFlowStep(ctx, models.EventFlowStep{DraftID: draftID, WorkflowCode: "planning", Step: "1", HandlerCode: "only-one", Name: "Solo"})
	if err != nil {
		t.Fatalf("CreateEventFlowStep: %v", err)
	}
	got, err := mgr.LookupFlowStep(ctx, draftID, models.EventFlowStepLookup{HandlerCode: "only-one"})
	if err != nil || got.ID != created.ID {
		t.Fatalf("LookupFlowStep by unique handler_code = %v, %v; want %s", got.ID, err, created.ID)
	}
	if _, err := mgr.LookupFlowStep(ctx, draftID, models.EventFlowStepLookup{HandlerCode: "nobody"}); !errors.Is(err, mwanachamaagency.ErrEventFlowStepNotFound) {
		t.Fatalf("LookupFlowStep by unknown handler_code = %v, want ErrEventFlowStepNotFound", err)
	}
}
