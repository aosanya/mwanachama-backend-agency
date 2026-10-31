package mwanachamaagency_test

import (
	"context"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type libraryUsageReader interface {
	ThreatUsage(ctx context.Context, draftID string) ([]mwanachamaagency.LibraryUsage, error)
	ConsequenceUsage(ctx context.Context, draftID string) ([]mwanachamaagency.LibraryUsage, error)
	ListUnguardedParts(ctx context.Context, draftID string) ([]mwanachamaagency.UnguardedPart, error)
}

func seedSharedThreat(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID string) (models.Threat, models.Threat, models.Consequence) {
	t.Helper()
	shared, err := mgr.CreateThreat(ctx, models.Threat{DraftID: draftID, Title: "Supplier fails to deliver"})
	if err != nil {
		t.Fatalf("CreateThreat: %v", err)
	}
	orphan, err := mgr.CreateThreat(ctx, models.Threat{DraftID: draftID, Title: "Nobody uses this"})
	if err != nil {
		t.Fatalf("CreateThreat: %v", err)
	}
	consequence, err := mgr.CreateConsequence(ctx, models.Consequence{DraftID: draftID, Title: "Produce spoils"})
	if err != nil {
		t.Fatalf("CreateConsequence: %v", err)
	}
	for _, title := range []string{"First risk", "Second risk"} {
		risk, err := mgr.CreateRisk(ctx, models.Risk{DraftID: draftID, Title: title, TopEvent: "Something happens"})
		if err != nil {
			t.Fatalf("CreateRisk: %v", err)
		}
		if _, err := mgr.CreateRiskThreat(ctx, models.RiskThreat{DraftID: draftID, RiskCode: risk.Code, ThreatCode: shared.Code}); err != nil {
			t.Fatalf("CreateRiskThreat: %v", err)
		}
		if title == "First risk" {
			if _, err := mgr.CreateRiskConsequence(ctx, models.RiskConsequence{DraftID: draftID, RiskCode: risk.Code, ConsequenceCode: consequence.Code}); err != nil {
				t.Fatalf("CreateRiskConsequence: %v", err)
			}
		}
	}
	return shared, orphan, consequence
}

func TestLibraryUsage_CountsRisksLinkingEachPart(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	shared, orphan, consequence := seedSharedThreat(t, mgr, ctx, o.DraftID)
	reader := mgr.(libraryUsageReader)

	threats, err := reader.ThreatUsage(ctx, o.DraftID)
	if err != nil {
		t.Fatalf("ThreatUsage: %v", err)
	}
	uses := map[string]int{}
	for _, u := range threats {
		uses[u.Code] = u.Uses
	}
	if len(threats) != 2 || uses[shared.Code] != 2 || uses[orphan.Code] != 0 {
		t.Fatalf("unexpected threat usage %+v", threats)
	}

	consequences, err := reader.ConsequenceUsage(ctx, o.DraftID)
	if err != nil {
		t.Fatalf("ConsequenceUsage: %v", err)
	}
	if len(consequences) != 1 || consequences[0].Code != consequence.Code || consequences[0].Uses != 1 || consequences[0].Title != "Produce spoils" {
		t.Fatalf("unexpected consequence usage %+v", consequences)
	}
}

func TestListUnguardedParts_OrdersByUseAndDropsGuardedParts(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	shared, orphan, consequence := seedSharedThreat(t, mgr, ctx, o.DraftID)
	reader := mgr.(libraryUsageReader)

	parts, err := reader.ListUnguardedParts(ctx, o.DraftID)
	if err != nil {
		t.Fatalf("ListUnguardedParts: %v", err)
	}
	if len(parts) != 3 || parts[0].Code != shared.Code || parts[1].Code != consequence.Code || parts[2].Code != orphan.Code {
		t.Fatalf("expected shared threat, consequence, orphan in that order, got %+v", parts)
	}
	if len(parts[0].Bowties) != 2 || parts[0].Bowties[0].Title != "First risk" || parts[0].Uses != 2 {
		t.Fatalf("shared threat should name both bowties once, got %+v", parts[0])
	}
	if len(parts[2].Bowties) != 0 {
		t.Fatalf("an unused part has no bowties, got %+v", parts[2])
	}

	control, err := mgr.CreateControl(ctx, models.Control{DraftID: o.DraftID, Text: "Two suppliers per order"})
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	if _, err := mgr.CreateControlAttachment(ctx, control.ID, models.TargetTypeThreat, shared.ID); err != nil {
		t.Fatalf("CreateControlAttachment: %v", err)
	}
	parts, err = reader.ListUnguardedParts(ctx, o.DraftID)
	if err != nil {
		t.Fatalf("ListUnguardedParts: %v", err)
	}
	if len(parts) != 2 || parts[0].Code != consequence.Code || parts[1].Code != orphan.Code {
		t.Fatalf("one control on the threat guards it in every bowtie, got %+v", parts)
	}
}
