package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type hazardScanWorld struct {
	scenarioWorld
	maker libraryPartMaker
}

func newHazardScanWorld(t *testing.T) hazardScanWorld {
	t.Helper()
	w := newScenarioWorld(t)
	return hazardScanWorld{scenarioWorld: w, maker: w.mgr.(libraryPartMaker)}
}

func (h hazardScanWorld) hazard(t *testing.T, title string) models.Hazard {
	t.Helper()
	out, err := h.maker.CreateHazard(context.Background(), models.Hazard{DraftID: h.draftID, Title: title})
	if err != nil {
		t.Fatalf("CreateHazard: %v", err)
	}
	return out
}

func (h hazardScanWorld) consequence(t *testing.T, hazard models.Hazard) models.Consequence {
	t.Helper()
	out, err := h.maker.CreateConsequence(context.Background(), models.Consequence{DraftID: h.draftID, HazardCode: hazard.Code, Title: "Money lost"})
	if err != nil {
		t.Fatalf("CreateConsequence: %v", err)
	}
	return out
}

func (h hazardScanWorld) risk(t *testing.T, hazard models.Hazard) models.Risk {
	t.Helper()
	out, err := h.mgr.CreateRisk(context.Background(), models.Risk{DraftID: h.draftID, Title: "Payment error", TopEvent: "Wrong payee", HazardCode: hazard.Code})
	if err != nil {
		t.Fatalf("CreateRisk: %v", err)
	}
	return out
}

func (h hazardScanWorld) setColumn(t *testing.T, role, id, column string, value any) {
	t.Helper()
	s, err := mwanachamaagency.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := h.db.Table(tableOf(t, s, role)).Where("id = ?", id).Update(column, value).Error; err != nil {
		t.Fatalf("update %s.%s: %v", role, column, err)
	}
}

func TestScan_ACleanHazardConsequenceLinkRaisesNothing(t *testing.T) {
	w := newHazardScanWorld(t)
	hazard := w.hazard(t, "Handling money")
	c := w.consequence(t, hazard)
	r := w.risk(t, hazard)
	if _, err := w.mgr.CreateRiskConsequence(context.Background(), models.RiskConsequence{DraftID: w.draftID, RiskCode: r.Code, ConsequenceCode: c.Code}); err != nil {
		t.Fatalf("CreateRiskConsequence: %v", err)
	}
	for _, entity := range []string{"consequence", "risk_consequence"} {
		if got := messagesFor(t, w.scenarioWorld, entity); len(got) != 0 {
			t.Fatalf("%s issues = %v, want none", entity, got)
		}
	}
}

func TestScan_AConsequenceWithNoHazardIsAnError(t *testing.T) {
	w := newHazardScanWorld(t)
	c := w.consequence(t, w.hazard(t, "Handling money"))
	w.setColumn(t, "consequence", c.ID, "hazard_code", "")
	got := messagesFor(t, w.scenarioWorld, "consequence")
	if !anyContains(got, "Consequence "+c.Code) || !anyContains(got, "names no hazard") {
		t.Fatalf("issues = %v, want the hazardless consequence named", got)
	}
}

func TestScan_AConsequenceNamingAMissingHazardIsAnError(t *testing.T) {
	w := newHazardScanWorld(t)
	c := w.consequence(t, w.hazard(t, "Handling money"))
	w.setColumn(t, "consequence", c.ID, "hazard_code", "H-99")
	got := messagesFor(t, w.scenarioWorld, "consequence")
	if !anyContains(got, "names hazard H-99, which does not exist") {
		t.Fatalf("issues = %v, want the missing hazard named", got)
	}
}

func TestScan_AConsequenceOfADeletedHazardIsAnError(t *testing.T) {
	w := newHazardScanWorld(t)
	hazard := w.hazard(t, "Handling money")
	w.consequence(t, hazard)
	w.setColumn(t, "hazard", hazard.ID, "deleted", true)
	got := messagesFor(t, w.scenarioWorld, "consequence")
	if !anyContains(got, "names hazard "+hazard.Code+", which does not exist") {
		t.Fatalf("issues = %v, want the deleted hazard flagged", got)
	}
}

func TestScan_ALinkAcrossHazardsIsAnError(t *testing.T) {
	w := newHazardScanWorld(t)
	money := w.hazard(t, "Handling money")
	stock := w.hazard(t, "Storing stock")
	c := w.consequence(t, money)
	r := w.risk(t, money)
	if _, err := w.mgr.CreateRiskConsequence(context.Background(), models.RiskConsequence{DraftID: w.draftID, RiskCode: r.Code, ConsequenceCode: c.Code}); err != nil {
		t.Fatalf("CreateRiskConsequence: %v", err)
	}
	w.setColumn(t, "risk", r.ID, "hazard_code", stock.Code)
	got := messagesFor(t, w.scenarioWorld, "risk_consequence")
	if !anyContains(got, "is under hazard "+stock.Code+" but links consequence "+c.Code+" of hazard "+money.Code) {
		t.Fatalf("issues = %v, want the mismatch named", got)
	}
}

func TestScan_ALinkFromARiskWithNoHazardIsAnError(t *testing.T) {
	w := newHazardScanWorld(t)
	hazard := w.hazard(t, "Handling money")
	c := w.consequence(t, hazard)
	r := w.risk(t, hazard)
	if _, err := w.mgr.CreateRiskConsequence(context.Background(), models.RiskConsequence{DraftID: w.draftID, RiskCode: r.Code, ConsequenceCode: c.Code}); err != nil {
		t.Fatalf("CreateRiskConsequence: %v", err)
	}
	w.setColumn(t, "risk", r.ID, "hazard_code", "")
	got := messagesFor(t, w.scenarioWorld, "risk_consequence")
	if !anyContains(got, "Risk "+r.Code+" links consequence "+c.Code+" but names no hazard") {
		t.Fatalf("issues = %v, want the hazardless risk named", got)
	}
}

func TestPublish_RefusedWhileAConsequenceNamesNoHazard(t *testing.T) {
	w := newHazardScanWorld(t)
	c := w.consequence(t, w.hazard(t, "Handling money"))
	w.setColumn(t, "consequence", c.ID, "hazard_code", "")
	if _, err := w.mgr.PublishAgency(context.Background(), w.draftID, "", ""); !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) {
		t.Fatalf("PublishAgency err = %v, want ErrPublishValidationFailed", err)
	}
}
