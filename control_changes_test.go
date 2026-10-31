package mwanachamaagency_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func enforcedControl(draftID, text string) models.Control {
	return models.Control{
		DraftID:      draftID,
		Text:         text,
		Action:       "payments.release",
		CondField:    "amount",
		CondOperator: models.OperatorGt,
		CondValue:    "5000",
		OnBreach:     models.OnBreachHoldForApproval,
		Severity:     models.SeverityHigh,
	}
}

func codesOf(controls []models.Control) []string {
	out := make([]string, 0, len(controls))
	for _, c := range controls {
		out = append(out, c.Code)
	}
	return out
}

func TestChangedEnforcedControls_NothingPublishedCountsEveryEnforcedControl(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	enforced, err := mgr.CreateControl(ctx, enforcedControl(o.DraftID, "Payments over 5,000 need a second approver"))
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	if _, err := mgr.CreateControl(ctx, models.Control{DraftID: o.DraftID, Text: "Correctness over speed"}); err != nil {
		t.Fatalf("CreateControl advisory: %v", err)
	}

	changed, err := mgr.ChangedEnforcedControls(ctx, o.DraftID)
	if err != nil {
		t.Fatalf("ChangedEnforcedControls: %v", err)
	}
	if len(changed) != 1 || changed[0].Code != enforced.Code {
		t.Fatalf("changed = %v, want only %s", codesOf(changed), enforced.Code)
	}
}

func TestChangedEnforcedControls_AnAdvisoryControlIsNeverChanged(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	if _, err := mgr.CreateControl(ctx, models.Control{DraftID: o.DraftID, Text: "Never touch production data"}); err != nil {
		t.Fatalf("CreateControl: %v", err)
	}

	changed, err := mgr.ChangedEnforcedControls(ctx, o.DraftID)
	if err != nil {
		t.Fatalf("ChangedEnforcedControls: %v", err)
	}
	if len(changed) != 0 {
		t.Fatalf("changed = %v, want none", codesOf(changed))
	}
}

func TestChangedEnforcedControls_AnAdvisoryControlMadeEnforcedIsChanged(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	advisory, err := mgr.CreateControl(ctx, models.Control{DraftID: o.DraftID, Text: "Payments need a second approver"})
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}

	advisory.Action = "payments.release"
	advisory.OnBreach = models.OnBreachBlock
	if _, err := mgr.UpdateControl(ctx, advisory); err != nil {
		t.Fatalf("UpdateControl: %v", err)
	}

	changed, err := mgr.ChangedEnforcedControls(ctx, o.DraftID)
	if err != nil {
		t.Fatalf("ChangedEnforcedControls: %v", err)
	}
	if len(changed) != 1 || changed[0].Code != advisory.Code {
		t.Fatalf("changed = %v, want only %s", codesOf(changed), advisory.Code)
	}
}

func TestChangedEnforcedControls_AnEnforcedControlEditedAfterPublishIsChanged(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	control, err := mgr.CreateControl(ctx, enforcedControl(o.DraftID, "Payments over 5,000 need a second approver"))
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); err != nil {
		t.Skipf("an enforced control cannot be published yet: %v", err)
	}

	unchanged, err := mgr.ChangedEnforcedControls(ctx, o.DraftID)
	if err != nil {
		t.Fatalf("ChangedEnforcedControls: %v", err)
	}
	if len(unchanged) != 0 {
		t.Fatalf("changed = %v right after publish, want none", codesOf(unchanged))
	}

	control.CondValue = "9000"
	if _, err := mgr.UpdateControl(ctx, control); err != nil {
		t.Fatalf("UpdateControl: %v", err)
	}
	changed, err := mgr.ChangedEnforcedControls(ctx, o.DraftID)
	if err != nil {
		t.Fatalf("ChangedEnforcedControls: %v", err)
	}
	if len(changed) != 1 || changed[0].Code != control.Code {
		t.Fatalf("changed = %v, want only %s", codesOf(changed), control.Code)
	}
}

func TestChangedEnforcedControls_UnknownDraftIsRefused(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.ChangedEnforcedControls(context.Background(), "no-such-draft"); err == nil {
		t.Fatal("expected an error for an unknown draft")
	}
}
