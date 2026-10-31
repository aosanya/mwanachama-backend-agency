package mwanachamaagency_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestUnguardedPathways_ListsPathwaysWithNoControl(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	_, threat, consequence := seedRiskWithBothArms(t, mgr, ctx, o.DraftID)

	gaps, err := mgr.UnguardedPathways(ctx, o.DraftID)
	if err != nil {
		t.Fatalf("UnguardedPathways: %v", err)
	}
	if len(gaps) != 2 {
		t.Fatalf("expected both pathways unguarded, got %d", len(gaps))
	}

	control, err := mgr.CreateControl(ctx, models.Control{DraftID: o.DraftID, Text: "Two approvers on payments"})
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	attachment, err := mgr.CreateControlAttachment(ctx, control.ID, models.TargetTypeRiskPathway, threat.ID)
	if err != nil {
		t.Fatalf("CreateControlAttachment: %v", err)
	}

	gaps, err = mgr.UnguardedPathways(ctx, o.DraftID)
	if err != nil {
		t.Fatalf("UnguardedPathways: %v", err)
	}
	if len(gaps) != 1 || gaps[0].ID != consequence.ID {
		t.Fatalf("expected only the consequence pathway unguarded, got %+v", gaps)
	}

	if err := mgr.DeleteControlAttachment(ctx, attachment.ID); err != nil {
		t.Fatalf("DeleteControlAttachment: %v", err)
	}
	gaps, err = mgr.UnguardedPathways(ctx, o.DraftID)
	if err != nil {
		t.Fatalf("UnguardedPathways: %v", err)
	}
	if len(gaps) != 2 {
		t.Fatalf("expected both pathways unguarded after detach, got %d", len(gaps))
	}
}

func TestUnguardedPathways_AttachmentToAnotherTargetDoesNotGuard(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	_, o, role := seedPublishableAgency(t, mgr, ctx)
	seedRiskWithBothArms(t, mgr, ctx, o.DraftID)

	control, err := mgr.CreateControl(ctx, models.Control{DraftID: o.DraftID, Text: "Roles keep their records"})
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	if _, err := mgr.CreateControlAttachment(ctx, control.ID, models.TargetTypeRole, role.ID); err != nil {
		t.Fatalf("CreateControlAttachment: %v", err)
	}

	gaps, err := mgr.UnguardedPathways(ctx, o.DraftID)
	if err != nil {
		t.Fatalf("UnguardedPathways: %v", err)
	}
	if len(gaps) != 2 {
		t.Fatalf("a role attachment must not guard a pathway, got %d gaps", len(gaps))
	}
}
