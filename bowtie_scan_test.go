package mwanachamaagency_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestScan_ABowtieNamesItsGapsAsWarnings(t *testing.T) {
	w := newHazardScanWorld(t)
	risk, err := w.mgr.CreateRisk(context.Background(), models.Risk{DraftID: w.draftID, Title: "Unsold harvest", TopEvent: "Harvest goes unsold"})
	if err != nil {
		t.Fatalf("CreateRisk: %v", err)
	}
	got := messagesFor(t, w.scenarioWorld, "risk")
	for _, want := range []string{"names no hazard", "has no owner", "has no threat", "has no consequence"} {
		if !anyContains(got, want) {
			t.Fatalf("risk %s issues = %v, want %q", risk.Code, got, want)
		}
	}
}

func TestScan_ACompleteBowtieRaisesNoBowtieWarning(t *testing.T) {
	w := newHazardScanWorld(t)
	hazard := w.hazard(t, "Handling money")
	consequence := w.consequence(t, hazard)
	risk, err := w.mgr.CreateRisk(context.Background(), models.Risk{DraftID: w.draftID, Title: "Payment error", TopEvent: "Wrong payee", HazardCode: hazard.Code, Owner: "Treasurer"})
	if err != nil {
		t.Fatalf("CreateRisk: %v", err)
	}
	threat, err := w.maker.CreateThreat(context.Background(), models.Threat{DraftID: w.draftID, Title: "Typo in account"})
	if err != nil {
		t.Fatalf("CreateThreat: %v", err)
	}
	if _, err := w.mgr.CreateRiskThreat(context.Background(), models.RiskThreat{DraftID: w.draftID, RiskCode: risk.Code, ThreatCode: threat.Code}); err != nil {
		t.Fatalf("CreateRiskThreat: %v", err)
	}
	if _, err := w.mgr.CreateRiskConsequence(context.Background(), models.RiskConsequence{DraftID: w.draftID, RiskCode: risk.Code, ConsequenceCode: consequence.Code}); err != nil {
		t.Fatalf("CreateRiskConsequence: %v", err)
	}
	if got := messagesFor(t, w.scenarioWorld, "risk"); len(got) != 0 {
		t.Fatalf("risk issues = %v, want none", got)
	}
	got := messagesFor(t, w.scenarioWorld, "control_gap")
	if len(got) != 2 || !anyContains(got, "threat "+threat.Code) || !anyContains(got, "consequence "+consequence.Code) {
		t.Fatalf("control gaps = %v, want the threat and the consequence", got)
	}
}
