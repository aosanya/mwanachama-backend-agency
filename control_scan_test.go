package mwanachamaagency_test

import (
	"context"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func messagesFor(t *testing.T, w scenarioWorld, entityType string) []string {
	t.Helper()
	var out []string
	for _, i := range issuesFor(t, w.mgr, context.Background(), w.draftID, entityType) {
		if i.Severity != models.IssueSeverityError {
			t.Errorf("issue %q is %s, want error", i.Message, i.Severity)
		}
		out = append(out, i.Message)
	}
	return out
}

func anyContains(messages []string, part string) bool {
	for _, m := range messages {
		if strings.Contains(m, part) {
			return true
		}
	}
	return false
}

func TestScan_AnAdvisoryControlRaisesNothing(t *testing.T) {
	w := newScenarioWorld(t)
	if _, err := w.mgr.CreateControl(context.Background(), models.Control{DraftID: w.draftID, Text: "Never touch production data"}); err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	if got := messagesFor(t, w, "control"); len(got) != 0 {
		t.Fatalf("issues = %v, want none", got)
	}
}

func TestScan_AnActionOutsideThePermissionCatalogueIsAnError(t *testing.T) {
	w := newScenarioWorld(t)
	c, err := w.mgr.CreateControl(context.Background(), models.Control{
		DraftID: w.draftID, Text: "Second approver", Action: "payments.release", OnBreach: models.OnBreachBlock,
	})
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	got := messagesFor(t, w, "control")
	if !anyContains(got, c.Code+" names action \"payments.release\"") {
		t.Fatalf("issues = %v, want the unknown action named", got)
	}
}

func TestScan_AnEnforcedControlNoScenarioHasPassedIsAnError(t *testing.T) {
	w := newScenarioWorld(t)
	c := w.enforced(t, "Only the owner publishes results")
	got := messagesFor(t, w, "control")
	if !anyContains(got, c.Code+" is enforced and no scenario") {
		t.Fatalf("issues = %v, want an untested enforced control", got)
	}
}

func TestScan_AnEnforcedControlWithAPassedScenarioIsClean(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	c := w.enforced(t, "Only the owner publishes results")
	w.attachToWorkflow(t, c, w.buyNow)
	s := w.scenario(t, "Shop says yes", w.buyNow, w.actor("Farmer 1", farmer))
	if _, err := w.runToEnd(ctx, s.ID); err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	if got := messagesFor(t, w, "control"); len(got) != 0 {
		t.Fatalf("issues = %v, want none", got)
	}
}

func riskPathways(t *testing.T, w scenarioWorld) (threat, consequence models.RiskPathway) {
	t.Helper()
	ctx := context.Background()
	risk, err := w.mgr.CreateRisk(ctx, models.Risk{DraftID: w.draftID, Title: "Payment error", TopEvent: "A payment goes to the wrong person"})
	if err != nil {
		t.Fatalf("CreateRisk: %v", err)
	}
	threat, err = w.mgr.CreateRiskPathway(ctx, models.RiskPathway{DraftID: w.draftID, RiskID: risk.ID, Side: models.RiskSideThreat, Title: "Typed wrongly"})
	if err != nil {
		t.Fatalf("CreateRiskPathway threat: %v", err)
	}
	consequence, err = w.mgr.CreateRiskPathway(ctx, models.RiskPathway{DraftID: w.draftID, RiskID: risk.ID, Side: models.RiskSideConsequence, Title: "Money lost"})
	if err != nil {
		t.Fatalf("CreateRiskPathway consequence: %v", err)
	}
	return threat, consequence
}

func runWorkflowControl(t *testing.T, w scenarioWorld) models.Control {
	t.Helper()
	c, err := w.mgr.CreateControl(context.Background(), models.Control{
		DraftID: w.draftID, Text: "Open the incident workflow", Action: catalogueAction,
		OnBreach: models.OnBreachRunWorkflow, WorkflowID: w.groupOrd.ID,
	})
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	return c
}

func TestScan_ARunWorkflowControlOnAThreatPathwayIsAnError(t *testing.T) {
	w := newScenarioWorld(t)
	threat, _ := riskPathways(t, w)
	c := runWorkflowControl(t, w)
	if _, err := w.mgr.CreateControlAttachment(context.Background(), c.ID, models.TargetTypeRiskPathway, threat.ID); err != nil {
		t.Fatalf("CreateControlAttachment: %v", err)
	}
	got := messagesFor(t, w, "control_attachment")
	if !anyContains(got, c.Code+" runs a workflow after the event") {
		t.Fatalf("issues = %v, want the threat pathway refused", got)
	}
}

func TestScan_ARunWorkflowControlOnAConsequencePathwayIsAccepted(t *testing.T) {
	w := newScenarioWorld(t)
	_, consequence := riskPathways(t, w)
	c := runWorkflowControl(t, w)
	if _, err := w.mgr.CreateControlAttachment(context.Background(), c.ID, models.TargetTypeRiskPathway, consequence.ID); err != nil {
		t.Fatalf("CreateControlAttachment: %v", err)
	}
	if got := messagesFor(t, w, "control_attachment"); len(got) != 0 {
		t.Fatalf("issues = %v, want none", got)
	}
}

func libraryParts(t *testing.T, w scenarioWorld) (models.Threat, models.Consequence) {
	t.Helper()
	ctx := context.Background()
	maker := w.mgr.(libraryPartMaker)
	threat, err := maker.CreateThreat(ctx, models.Threat{DraftID: w.draftID, Title: "Typed wrongly"})
	if err != nil {
		t.Fatalf("CreateThreat: %v", err)
	}
	hazard, err := maker.CreateHazard(ctx, models.Hazard{DraftID: w.draftID, Title: "Handling money"})
	if err != nil {
		t.Fatalf("CreateHazard: %v", err)
	}
	consequence, err := maker.CreateConsequence(ctx, models.Consequence{DraftID: w.draftID, HazardCode: hazard.Code, Title: "Money lost"})
	if err != nil {
		t.Fatalf("CreateConsequence: %v", err)
	}
	return threat, consequence
}

func TestScan_ARunWorkflowControlOnAThreatIsAnError(t *testing.T) {
	w := newScenarioWorld(t)
	threat, _ := libraryParts(t, w)
	c := runWorkflowControl(t, w)
	if _, err := w.mgr.CreateControlAttachment(context.Background(), c.ID, models.TargetTypeThreat, threat.ID); err != nil {
		t.Fatalf("CreateControlAttachment: %v", err)
	}
	got := messagesFor(t, w, "control_attachment")
	if !anyContains(got, c.Code+" runs a workflow after the event") {
		t.Fatalf("issues = %v, want the threat refused", got)
	}
}

func TestScan_ARunWorkflowControlOnAConsequenceIsAccepted(t *testing.T) {
	w := newScenarioWorld(t)
	_, consequence := libraryParts(t, w)
	c := runWorkflowControl(t, w)
	if _, err := w.mgr.CreateControlAttachment(context.Background(), c.ID, models.TargetTypeConsequence, consequence.ID); err != nil {
		t.Fatalf("CreateControlAttachment: %v", err)
	}
	if got := messagesFor(t, w, "control_attachment"); len(got) != 0 {
		t.Fatalf("issues = %v, want none", got)
	}
}
