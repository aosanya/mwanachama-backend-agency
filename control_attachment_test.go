package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func seedPathway(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID string, side models.RiskSide) models.RiskPathway {
	t.Helper()
	risk, err := mgr.CreateRisk(ctx, models.Risk{DraftID: draftID, Title: "Unsold harvest", TopEvent: "A harvest goes unsold"})
	if err != nil {
		t.Fatalf("CreateRisk: %v", err)
	}
	p, err := mgr.CreateRiskPathway(ctx, models.RiskPathway{DraftID: draftID, RiskID: risk.ID, Side: side, Title: "No buyer is listed"})
	if err != nil {
		t.Fatalf("CreateRiskPathway: %v", err)
	}
	return p
}

func TestControlAttachment_StandsOnARiskPathwayAndOnTheOtherTargets(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, o, w, wi := seedLiveAgency(t, mgr, ctx)
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "owner"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	pathway := seedPathway(t, mgr, ctx, o.DraftID, models.RiskSideThreat)
	c, err := mgr.CreateControl(ctx, paymentHoldControl(o.DraftID))
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}

	targets := map[models.TargetType]string{
		models.TargetTypeRole:        role.ID,
		models.TargetTypeGoal:        g.ID,
		models.TargetTypeObjective:   o.ID,
		models.TargetTypeWorkflow:    w.ID,
		models.TargetTypeWorkItem:    wi.ID,
		models.TargetTypeRiskPathway: pathway.ID,
	}
	for targetType, id := range targets {
		a, err := mgr.CreateControlAttachment(ctx, c.ID, targetType, id)
		if err != nil {
			t.Fatalf("CreateControlAttachment(%s): %v", targetType, err)
		}
		if a.ControlCode != c.Code || a.DraftID != o.DraftID || a.TargetType != targetType || a.ID == "" {
			t.Fatalf("attachment = %+v", a)
		}
	}
	held, err := mgr.ListControlAttachments(ctx, c.ID)
	if err != nil || len(held) != len(targets) {
		t.Fatalf("ListControlAttachments = %+v, %v", held, err)
	}
	var onPathway models.ControlAttachment
	for _, a := range held {
		if a.TargetType == models.TargetTypeRiskPathway {
			onPathway = a
		}
	}
	if onPathway.TargetCode != pathway.Code {
		t.Fatalf("pathway attachment target = %q, want %q", onPathway.TargetCode, pathway.Code)
	}
}

func TestControlAttachment_RefusesDuplicatesUnknownTypesAndTargetsOfAnotherDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, w, _ := seedLiveAgency(t, mgr, ctx)
	c, err := mgr.CreateControl(ctx, models.Control{DraftID: o.DraftID, Text: "Review before release"})
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	if _, err := mgr.CreateControlAttachment(ctx, c.ID, models.TargetTypeWorkflow, w.ID); err != nil {
		t.Fatalf("CreateControlAttachment: %v", err)
	}
	if _, err := mgr.CreateControlAttachment(ctx, c.ID, models.TargetTypeWorkflow, w.ID); !errors.Is(err, mwanachamaagency.ErrDuplicateControlAttachment) {
		t.Fatalf("duplicate = %v, want ErrDuplicateControlAttachment", err)
	}
	if _, err := mgr.CreateControlAttachment(ctx, c.ID, "planet", w.ID); !errors.Is(err, mwanachamaagency.ErrInvalidTargetType) {
		t.Fatalf("unknown type = %v, want ErrInvalidTargetType", err)
	}
	if _, err := mgr.CreateControlAttachment(ctx, c.ID, models.TargetTypeGoal, "missing"); !errors.Is(err, mwanachamaagency.ErrGoalNotFound) {
		t.Fatalf("missing goal = %v, want ErrGoalNotFound", err)
	}
	if _, err := mgr.CreateControlAttachment(ctx, c.ID, models.TargetTypeRiskPathway, "missing"); !errors.Is(err, mwanachamaagency.ErrRiskPathwayNotFound) {
		t.Fatalf("missing pathway = %v, want ErrRiskPathwayNotFound", err)
	}
	if _, err := mgr.CreateControlAttachment(ctx, "missing", models.TargetTypeWorkflow, w.ID); !errors.Is(err, mwanachamaagency.ErrControlNotFound) {
		t.Fatalf("missing control = %v, want ErrControlNotFound", err)
	}

	other, err := mgr.CreateDraft(ctx, "other", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	foreign, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: other.ID, Name: "Elsewhere"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	if _, err := mgr.CreateControlAttachment(ctx, c.ID, models.TargetTypeWorkflow, foreign.ID); !errors.Is(err, mwanachamaagency.ErrWorkflowNotFound) {
		t.Fatalf("target in another draft = %v, want ErrWorkflowNotFound", err)
	}
}

func TestControlAttachment_DetachLeavesTheControl(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	pathway := seedPathway(t, mgr, ctx, o.DraftID, models.RiskSideConsequence)
	c, err := mgr.CreateControl(ctx, models.Control{DraftID: o.DraftID, Text: "Start the incident workflow"})
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	a, err := mgr.CreateControlAttachment(ctx, c.ID, models.TargetTypeRiskPathway, pathway.ID)
	if err != nil {
		t.Fatalf("CreateControlAttachment: %v", err)
	}

	if err := mgr.DeleteControlAttachment(ctx, a.ID); err != nil {
		t.Fatalf("DeleteControlAttachment: %v", err)
	}
	held, err := mgr.ListControlAttachments(ctx, c.ID)
	if err != nil || len(held) != 0 {
		t.Fatalf("ListControlAttachments = %+v, %v", held, err)
	}
	if _, err := mgr.GetControl(ctx, c.ID); err != nil {
		t.Fatalf("GetControl after detach: %v", err)
	}
	if err := mgr.DeleteControlAttachment(ctx, a.ID); !errors.Is(err, mwanachamaagency.ErrControlAttachmentNotFound) {
		t.Fatalf("second detach = %v, want ErrControlAttachmentNotFound", err)
	}
}
