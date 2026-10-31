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
	got := warningsFor(t, w, "risk")
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
	if got := warningsFor(t, w, "risk"); len(got) != 0 {
		t.Fatalf("risk issues = %v, want none", got)
	}
	got := warningsFor(t, w, "control_gap")
	if len(got) != 2 || !anyContains(got, "threat "+threat.Code) || !anyContains(got, "consequence "+consequence.Code) {
		t.Fatalf("control gaps = %v, want the threat and the consequence", got)
	}
}

func warningsFor(t *testing.T, w hazardScanWorld, entity string) []string {
	t.Helper()
	var out []string
	for _, issue := range issuesFor(t, w.mgr, context.Background(), w.draftID, entity) {
		if issue.Severity != models.IssueSeverityWarning {
			t.Fatalf("issue %+v is not a warning", issue)
		}
		out = append(out, issue.Message)
	}
	return out
}

func TestScan_AControlOnTwoPartsIsAWarning(t *testing.T) {
	w := newHazardScanWorld(t)
	ctx := context.Background()
	control, err := w.mgr.CreateControl(ctx, models.Control{DraftID: w.draftID, Text: "Grade every listing"})
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	risk, err := w.mgr.CreateRisk(ctx, models.Risk{DraftID: w.draftID, Title: "Unsold harvest", TopEvent: "Harvest goes unsold"})
	if err != nil {
		t.Fatalf("CreateRisk: %v", err)
	}
	for _, title := range []string{"No bids", "Low offers"} {
		threat, err := w.maker.CreateThreat(ctx, models.Threat{DraftID: w.draftID, Title: title})
		if err != nil {
			t.Fatalf("CreateThreat: %v", err)
		}
		if _, err := w.mgr.CreateRiskThreat(ctx, models.RiskThreat{DraftID: w.draftID, RiskCode: risk.Code, ThreatCode: threat.Code}); err != nil {
			t.Fatalf("CreateRiskThreat: %v", err)
		}
		if _, err := w.mgr.CreateControlAttachment(ctx, control.ID, models.TargetTypeThreat, threat.ID); err != nil {
			t.Fatalf("CreateControlAttachment: %v", err)
		}
	}
	got := warningsFor(t, w, "control_attachment")
	if len(got) != 1 || !anyContains(got, "stands on 2 threats or consequences") {
		t.Fatalf("control attachment warnings = %v, want one naming the two", got)
	}
	if gaps := warningsFor(t, w, "control_gap"); len(gaps) != 0 {
		t.Fatalf("control gaps = %v, want none once both threats have a control", gaps)
	}
}
