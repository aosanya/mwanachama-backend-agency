package mwanachamaagency_test

import (
	"context"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCreateDraft_ForksControlsAndTheirAttachments(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	_, threat, _ := seedRiskWithBothArms(t, mgr, ctx, o.DraftID)
	workflows, err := mgr.ListWorkflows(ctx, o.DraftID, mwanachamaagency.WorkflowFilter{})
	if err != nil || len(workflows) == 0 {
		t.Fatalf("ListWorkflows: %v (%d)", err, len(workflows))
	}

	enforced, err := mgr.CreateControl(ctx, models.Control{
		DraftID: o.DraftID, Text: "Payments over 5,000 need a second approver", Action: "payments.release",
		CondField: "amount", CondOperator: models.OperatorGt, CondValue: "5000",
		OnBreach: models.OnBreachHoldForApproval, Severity: models.SeverityHigh,
	})
	if err != nil {
		t.Fatalf("CreateControl(enforced): %v", err)
	}
	running, err := mgr.CreateControl(ctx, models.Control{
		DraftID: o.DraftID, Text: "A leak starts the incident workflow", Action: "data.export",
		OnBreach: models.OnBreachRunWorkflow, WorkflowID: workflows[0].ID,
	})
	if err != nil {
		t.Fatalf("CreateControl(run_workflow): %v", err)
	}
	if _, err := mgr.CreateControlAttachment(ctx, enforced.ID, models.TargetTypeRiskPathway, threat.ID); err != nil {
		t.Fatalf("CreateControlAttachment: %v", err)
	}

	fork, err := mgr.CreateDraft(ctx, "fork", o.DraftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	forkedWorkflows, err := mgr.ListWorkflows(ctx, fork.ID, mwanachamaagency.WorkflowFilter{})
	if err != nil || len(forkedWorkflows) == 0 {
		t.Fatalf("ListWorkflows(fork): %v (%d)", err, len(forkedWorkflows))
	}
	forked, err := mgr.ListControls(ctx, fork.ID, mwanachamaagency.ControlFilter{})
	if err != nil {
		t.Fatalf("ListControls(fork): %v", err)
	}
	if len(forked) != 2 {
		t.Fatalf("expected both controls forked, got %d", len(forked))
	}
	byCode := map[string]models.Control{}
	for _, c := range forked {
		byCode[c.Code] = c
	}

	got := byCode[enforced.Code]
	if got.ID == enforced.ID || got.DraftID != fork.ID {
		t.Fatalf("fork must mint a new row in the new draft: %+v", got)
	}
	if got.Action != enforced.Action || got.CondValue != "5000" || got.OnBreach != models.OnBreachHoldForApproval || got.Severity != models.SeverityHigh {
		t.Fatalf("enforcement fields lost in the fork: %+v", got)
	}
	if byCode[running.Code].WorkflowID != forkedWorkflows[0].ID {
		t.Fatalf("workflow_id must point at the forked workflow, got %q want %q", byCode[running.Code].WorkflowID, forkedWorkflows[0].ID)
	}

	gaps, err := mgr.UnguardedPathways(ctx, fork.ID)
	if err != nil {
		t.Fatalf("UnguardedPathways(fork): %v", err)
	}
	if len(gaps) != 1 || gaps[0].Side != models.RiskSideConsequence {
		t.Fatalf("the threat attachment must survive the fork, got %+v", gaps)
	}

	attachments, err := mgr.ListControlAttachments(ctx, got.ID)
	if err != nil {
		t.Fatalf("ListControlAttachments(fork): %v", err)
	}
	if len(attachments) != 1 || attachments[0].DraftID != fork.ID || attachments[0].ControlCode != enforced.Code {
		t.Fatalf("attachment not forked into the new draft: %+v", attachments)
	}
}

func TestCreateDraft_ForkLeavesTheSourceControlsAlone(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	if _, err := mgr.CreateControl(ctx, models.Control{DraftID: o.DraftID, Text: "We keep our word"}); err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	fork, err := mgr.CreateDraft(ctx, "fork", o.DraftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	forked, _ := mgr.ListControls(ctx, fork.ID, mwanachamaagency.ControlFilter{})
	if _, err := mgr.UpdateControl(ctx, models.Control{ID: forked[0].ID, DraftID: fork.ID, Text: "We keep our promises"}); err != nil {
		t.Fatalf("UpdateControl(fork): %v", err)
	}
	source, err := mgr.ListControls(ctx, o.DraftID, mwanachamaagency.ControlFilter{})
	if err != nil || len(source) != 1 || source[0].Text != "We keep our word" {
		t.Fatalf("editing the fork changed the source: %+v (%v)", source, err)
	}
}
