package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type composeWorld struct {
	mgr         mwanachamaagency.AgencyManager
	ctx         context.Context
	draftID     string
	otherDraft  string
	hazard      models.Hazard
	risk        models.Risk
	threat      models.Threat
	consequence models.Consequence
}

func newComposeWorld(t *testing.T) composeWorld {
	t.Helper()
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	hazard, err := mgr.CreateHazard(ctx, models.Hazard{DraftID: o.DraftID, Title: "Perishable produce"})
	if err != nil {
		t.Fatalf("CreateHazard: %v", err)
	}
	risk, err := mgr.CreateRisk(ctx, models.Risk{DraftID: o.DraftID, Title: "Unsold harvest", TopEvent: "A harvest goes unsold", HazardCode: hazard.Code})
	if err != nil {
		t.Fatalf("CreateRisk: %v", err)
	}
	threat, err := mgr.CreateThreat(ctx, models.Threat{DraftID: o.DraftID, Title: "No buyer is listed"})
	if err != nil {
		t.Fatalf("CreateThreat: %v", err)
	}
	consequence, err := mgr.CreateConsequence(ctx, models.Consequence{DraftID: o.DraftID, HazardCode: hazard.Code, Title: "Produce spoils"})
	if err != nil {
		t.Fatalf("CreateConsequence: %v", err)
	}
	return composeWorld{mgr: mgr, ctx: ctx, draftID: o.DraftID, hazard: hazard, risk: risk, threat: threat, consequence: consequence}
}

func TestCreateRiskThreat_LinksByCodeAndRefusesAStrangerOrAFourthCopy(t *testing.T) {
	w := newComposeWorld(t)
	link, err := w.mgr.CreateRiskThreat(w.ctx, models.RiskThreat{DraftID: w.draftID, RiskCode: w.risk.Code, ThreatCode: w.threat.Code})
	if err != nil {
		t.Fatalf("CreateRiskThreat: %v", err)
	}
	if link.ID == "" || link.RiskCode != w.risk.Code || link.ThreatCode != w.threat.Code || link.CreatedAt == "" {
		t.Fatalf("link = %+v", link)
	}
	cases := []struct {
		name string
		in   models.RiskThreat
		want error
	}{
		{"duplicate", models.RiskThreat{DraftID: w.draftID, RiskCode: w.risk.Code, ThreatCode: w.threat.Code}, mwanachamaagency.ErrDuplicateRiskThreat},
		{"unknown risk", models.RiskThreat{DraftID: w.draftID, RiskCode: "RK-99", ThreatCode: w.threat.Code}, mwanachamaagency.ErrRiskNotFound},
		{"unknown threat", models.RiskThreat{DraftID: w.draftID, RiskCode: w.risk.Code, ThreatCode: "T-99"}, mwanachamaagency.ErrThreatNotFound},
		{"no draft", models.RiskThreat{RiskCode: w.risk.Code, ThreatCode: w.threat.Code}, mwanachamaagency.ErrDraftRequired},
	}
	for _, c := range cases {
		if _, err := w.mgr.CreateRiskThreat(w.ctx, c.in); !errors.Is(err, c.want) {
			t.Fatalf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
}

func TestCreateRiskConsequence_LinksByCodeAndRefusesAStrangerOrAFourthCopy(t *testing.T) {
	w := newComposeWorld(t)
	link, err := w.mgr.CreateRiskConsequence(w.ctx, models.RiskConsequence{DraftID: w.draftID, RiskCode: w.risk.Code, ConsequenceCode: w.consequence.Code})
	if err != nil {
		t.Fatalf("CreateRiskConsequence: %v", err)
	}
	if link.ID == "" || link.ConsequenceCode != w.consequence.Code {
		t.Fatalf("link = %+v", link)
	}
	cases := []struct {
		name string
		in   models.RiskConsequence
		want error
	}{
		{"duplicate", models.RiskConsequence{DraftID: w.draftID, RiskCode: w.risk.Code, ConsequenceCode: w.consequence.Code}, mwanachamaagency.ErrDuplicateRiskConsequence},
		{"unknown risk", models.RiskConsequence{DraftID: w.draftID, RiskCode: "RK-99", ConsequenceCode: w.consequence.Code}, mwanachamaagency.ErrRiskNotFound},
		{"unknown consequence", models.RiskConsequence{DraftID: w.draftID, RiskCode: w.risk.Code, ConsequenceCode: "Q-99"}, mwanachamaagency.ErrConsequenceNotFound},
	}
	for _, c := range cases {
		if _, err := w.mgr.CreateRiskConsequence(w.ctx, c.in); !errors.Is(err, c.want) {
			t.Fatalf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
}

func TestRiskLinks_AreScopedToTheirDraft(t *testing.T) {
	w := newComposeWorld(t)
	other, err := w.mgr.CreateDraft(w.ctx, "second", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if _, err := w.mgr.CreateRiskThreat(w.ctx, models.RiskThreat{DraftID: w.draftID, RiskCode: w.risk.Code, ThreatCode: w.threat.Code}); err != nil {
		t.Fatalf("CreateRiskThreat: %v", err)
	}
	if _, err := w.mgr.CreateRiskConsequence(w.ctx, models.RiskConsequence{DraftID: w.draftID, RiskCode: w.risk.Code, ConsequenceCode: w.consequence.Code}); err != nil {
		t.Fatalf("CreateRiskConsequence: %v", err)
	}
	threats, err := w.mgr.ListRiskThreats(w.ctx, other.ID, mwanachamaagency.RiskThreatFilter{})
	if err != nil || len(threats) != 0 {
		t.Fatalf("other draft threats = %v, %v", threats, err)
	}
	consequences, err := w.mgr.ListRiskConsequences(w.ctx, other.ID, mwanachamaagency.RiskConsequenceFilter{})
	if err != nil || len(consequences) != 0 {
		t.Fatalf("other draft consequences = %v, %v", consequences, err)
	}
}

func TestListAndDeleteRiskLinks(t *testing.T) {
	w := newComposeWorld(t)
	second, err := w.mgr.CreateThreat(w.ctx, models.Threat{DraftID: w.draftID, Title: "Buyer pays late"})
	if err != nil {
		t.Fatalf("CreateThreat: %v", err)
	}
	first, _ := w.mgr.CreateRiskThreat(w.ctx, models.RiskThreat{DraftID: w.draftID, RiskCode: w.risk.Code, ThreatCode: w.threat.Code})
	if _, err := w.mgr.CreateRiskThreat(w.ctx, models.RiskThreat{DraftID: w.draftID, RiskCode: w.risk.Code, ThreatCode: second.Code}); err != nil {
		t.Fatalf("CreateRiskThreat: %v", err)
	}
	all, err := w.mgr.ListRiskThreats(w.ctx, w.draftID, mwanachamaagency.RiskThreatFilter{RiskCode: w.risk.Code})
	if err != nil || len(all) != 2 {
		t.Fatalf("list = %v, %v", all, err)
	}
	one, err := w.mgr.ListRiskThreats(w.ctx, w.draftID, mwanachamaagency.RiskThreatFilter{ThreatCode: second.Code})
	if err != nil || len(one) != 1 || one[0].ThreatCode != second.Code {
		t.Fatalf("filtered list = %v, %v", one, err)
	}
	if err := w.mgr.DeleteRiskThreat(w.ctx, first.ID); err != nil {
		t.Fatalf("DeleteRiskThreat: %v", err)
	}
	if err := w.mgr.DeleteRiskThreat(w.ctx, first.ID); !errors.Is(err, mwanachamaagency.ErrRiskThreatNotFound) {
		t.Fatalf("second delete = %v, want ErrRiskThreatNotFound", err)
	}
	if err := w.mgr.DeleteRiskConsequence(w.ctx, "nope"); !errors.Is(err, mwanachamaagency.ErrRiskConsequenceNotFound) {
		t.Fatalf("delete unknown = %v, want ErrRiskConsequenceNotFound", err)
	}
	link, _ := w.mgr.CreateRiskConsequence(w.ctx, models.RiskConsequence{DraftID: w.draftID, RiskCode: w.risk.Code, ConsequenceCode: w.consequence.Code})
	if err := w.mgr.DeleteRiskConsequence(w.ctx, link.ID); err != nil {
		t.Fatalf("DeleteRiskConsequence: %v", err)
	}
	left, err := w.mgr.ListRiskConsequences(w.ctx, w.draftID, mwanachamaagency.RiskConsequenceFilter{})
	if err != nil || len(left) != 0 {
		t.Fatalf("consequences left = %v, %v", left, err)
	}
}

func TestRisk_HazardCodeMustNameAHazardInTheDraft(t *testing.T) {
	w := newComposeWorld(t)
	hazard := w.hazard
	if _, err := w.mgr.CreateRisk(w.ctx, models.Risk{DraftID: w.draftID, Title: "A", TopEvent: "B", HazardCode: "H-99"}); !errors.Is(err, mwanachamaagency.ErrHazardNotFound) {
		t.Fatalf("CreateRisk with unknown hazard = %v, want ErrHazardNotFound", err)
	}
	named, err := w.mgr.CreateRisk(w.ctx, models.Risk{DraftID: w.draftID, Title: "A", TopEvent: "B", HazardCode: hazard.Code})
	if err != nil || named.HazardCode != hazard.Code {
		t.Fatalf("CreateRisk with hazard = %+v, %v", named, err)
	}
	named.HazardCode = "H-99"
	if _, err := w.mgr.UpdateRisk(w.ctx, named); !errors.Is(err, mwanachamaagency.ErrHazardNotFound) {
		t.Fatalf("UpdateRisk with unknown hazard = %v, want ErrHazardNotFound", err)
	}
	named.HazardCode = ""
	if _, err := w.mgr.UpdateRisk(w.ctx, named); err != nil {
		t.Fatalf("UpdateRisk clearing hazard: %v", err)
	}
}
