package mwanachamaagency_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCreateScenario_ShowsTheControlsAttachedToItsWorkflowOnly(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	onWorkflow, err := w.mgr.CreateControl(ctx, models.Control{DraftID: w.draftID, Text: "Ask the shop once"})
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	onOther, err := w.mgr.CreateControl(ctx, models.Control{DraftID: w.draftID, Text: "Pay on delivery"})
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	if _, err := w.mgr.CreateControl(ctx, models.Control{DraftID: w.draftID, Text: "Agency wide"}); err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	if _, err := w.mgr.CreateControlAttachment(ctx, onWorkflow.ID, models.TargetTypeWorkflow, w.buyNow.ID); err != nil {
		t.Fatalf("CreateControlAttachment: %v", err)
	}
	if _, err := w.mgr.CreateControlAttachment(ctx, onOther.ID, models.TargetTypeWorkflow, w.groupOrd.ID); err != nil {
		t.Fatalf("CreateControlAttachment: %v", err)
	}
	s := w.scenario(t, "Shop says yes", w.buyNow)
	if !reflect.DeepEqual(s.Rules, []string{onWorkflow.Code}) {
		t.Errorf("rules = %v, want [%s]", s.Rules, onWorkflow.Code)
	}
}
