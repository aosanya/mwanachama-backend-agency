package mwanachamaagency_test

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAG65_OpenBug_LookupFlowStepByHandlerCodePicksAnArbitraryStepWhenSeveralShareIt(t *testing.T) {
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
		t.Fatalf("AG65 open bug: LookupFlowStep by handler_code %q matches two steps (%q and %q) the blueprint's own handler_code index describes as \"which steps a given handler runs\" — legitimately more than one — yet the call answered one of them (%q, id %s) with no error; want it to refuse an ambiguous handler_code lookup rather than silently picking one of %s/%s",
			"shared-handler", first.ID, second.ID, got.Name, got.ID, first.ID, second.ID)
	}
}
