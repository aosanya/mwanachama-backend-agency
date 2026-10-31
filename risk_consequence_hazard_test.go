package mwanachamaagency_test

import (
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCreateRiskConsequence_RefusesARiskWithNoHazard(t *testing.T) {
	w := newComposeWorld(t)
	bare, err := w.mgr.CreateRisk(w.ctx, models.Risk{DraftID: w.draftID, Title: "Bare", TopEvent: "Nothing named"})
	if err != nil {
		t.Fatalf("CreateRisk: %v", err)
	}
	_, err = w.mgr.CreateRiskConsequence(w.ctx, models.RiskConsequence{DraftID: w.draftID, RiskCode: bare.Code, ConsequenceCode: w.consequence.Code})
	if !errors.Is(err, mwanachamaagency.ErrConsequenceOutsideHazard) {
		t.Fatalf("got %v, want ErrConsequenceOutsideHazard", err)
	}
}

func TestCreateRiskConsequence_RefusesAConsequenceOfAnotherHazard(t *testing.T) {
	w := newComposeWorld(t)
	other, err := w.mgr.CreateHazard(w.ctx, models.Hazard{DraftID: w.draftID, Title: "Cold chain"})
	if err != nil {
		t.Fatalf("CreateHazard: %v", err)
	}
	stranger, err := w.mgr.CreateConsequence(w.ctx, models.Consequence{DraftID: w.draftID, HazardCode: other.Code, Title: "Vaccines warm"})
	if err != nil {
		t.Fatalf("CreateConsequence: %v", err)
	}
	_, err = w.mgr.CreateRiskConsequence(w.ctx, models.RiskConsequence{DraftID: w.draftID, RiskCode: w.risk.Code, ConsequenceCode: stranger.Code})
	if !errors.Is(err, mwanachamaagency.ErrConsequenceOutsideHazard) {
		t.Fatalf("got %v, want ErrConsequenceOutsideHazard", err)
	}
	if _, err := w.mgr.CreateRiskConsequence(w.ctx, models.RiskConsequence{DraftID: w.draftID, RiskCode: w.risk.Code, ConsequenceCode: w.consequence.Code}); err != nil {
		t.Fatalf("same hazard link: %v", err)
	}
}

func TestUpdateRisk_HazardChangeRefusedWhileAConsequenceOfTheOldHazardIsLinked(t *testing.T) {
	w := newComposeWorld(t)
	other, err := w.mgr.CreateHazard(w.ctx, models.Hazard{DraftID: w.draftID, Title: "Cold chain"})
	if err != nil {
		t.Fatalf("CreateHazard: %v", err)
	}
	if _, err := w.mgr.CreateRiskConsequence(w.ctx, models.RiskConsequence{DraftID: w.draftID, RiskCode: w.risk.Code, ConsequenceCode: w.consequence.Code}); err != nil {
		t.Fatalf("link: %v", err)
	}
	moved := w.risk
	moved.HazardCode = other.Code
	if _, err := w.mgr.UpdateRisk(w.ctx, moved); !errors.Is(err, mwanachamaagency.ErrConsequenceOutsideHazard) {
		t.Fatalf("switch hazard: got %v, want ErrConsequenceOutsideHazard", err)
	}
	cleared := w.risk
	cleared.HazardCode = ""
	if _, err := w.mgr.UpdateRisk(w.ctx, cleared); !errors.Is(err, mwanachamaagency.ErrConsequenceOutsideHazard) {
		t.Fatalf("clear hazard: got %v, want ErrConsequenceOutsideHazard", err)
	}
	renamed := w.risk
	renamed.Title = "Renamed"
	if _, err := w.mgr.UpdateRisk(w.ctx, renamed); err != nil {
		t.Fatalf("update keeping hazard: %v", err)
	}
}

func TestUpdateRisk_HazardChangeAllowedWithNoLinkedConsequence(t *testing.T) {
	w := newComposeWorld(t)
	other, err := w.mgr.CreateHazard(w.ctx, models.Hazard{DraftID: w.draftID, Title: "Cold chain"})
	if err != nil {
		t.Fatalf("CreateHazard: %v", err)
	}
	moved := w.risk
	moved.HazardCode = other.Code
	got, err := w.mgr.UpdateRisk(w.ctx, moved)
	if err != nil || got.HazardCode != other.Code {
		t.Fatalf("switch hazard = %+v, %v", got, err)
	}
	moved.HazardCode = ""
	if _, err := w.mgr.UpdateRisk(w.ctx, moved); err != nil {
		t.Fatalf("clear hazard: %v", err)
	}
}
