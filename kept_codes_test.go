package mwanachamaagency_test

import (
	"context"
	"fmt"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func workflowNumber(t *testing.T, code string) int {
	t.Helper()
	var n int
	if _, err := fmt.Sscanf(code, "W-%d", &n); err != nil {
		t.Fatalf("workflow code %q: %v", code, err)
	}
	return n
}

func TestKeptCodes_AFileCodeIsUsedAndTheSequenceRisesPastIt(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	kept, err := w.mgr.CreateWorkflow(mwanachamaagency.WithKeptCode(ctx, "W-90"), models.Workflow{DraftID: w.draftID, Name: "Kept"})
	if err != nil {
		t.Fatalf("kept create: %v", err)
	}
	if kept.Code != "W-90" {
		t.Errorf("kept code = %q, want W-90", kept.Code)
	}
	next, err := w.mgr.CreateWorkflow(ctx, models.Workflow{DraftID: w.draftID, Name: "Minted"})
	if err != nil {
		t.Fatalf("minted create: %v", err)
	}
	if next.Code != "W-91" {
		t.Errorf("minted code after a kept W-90 = %q, want W-91", next.Code)
	}
}

func TestKeptCodes_MintingSkipsReservedCodes(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	probe, err := w.mgr.CreateWorkflow(ctx, models.Workflow{DraftID: w.draftID, Name: "Probe"})
	if err != nil {
		t.Fatalf("probe create: %v", err)
	}
	n := workflowNumber(t, probe.Code)
	reserved := map[string]bool{fmt.Sprintf("W-%d", n+1): true, fmt.Sprintf("W-%d", n+2): true}
	minted, err := w.mgr.CreateWorkflow(mwanachamaagency.WithReservedCodes(ctx, reserved), models.Workflow{DraftID: w.draftID, Name: "Minted"})
	if err != nil {
		t.Fatalf("minted create: %v", err)
	}
	if want := fmt.Sprintf("W-%d", n+3); minted.Code != want {
		t.Errorf("minted code with the next two reserved = %q, want %q", minted.Code, want)
	}
}
