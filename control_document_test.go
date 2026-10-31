package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func publishableDraftWithGuardedPathway(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context) (string, models.Control) {
	t.Helper()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	_, threat, _ := seedRiskWithBothArms(t, mgr, ctx, o.DraftID)
	control, err := mgr.CreateControl(ctx, models.Control{DraftID: o.DraftID, Text: "Two approvers on payments", Severity: models.SeverityMedium})
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	if _, err := mgr.CreateControlAttachment(ctx, control.ID, models.TargetTypeRiskPathway, threat.ID); err != nil {
		t.Fatalf("CreateControlAttachment: %v", err)
	}
	return o.DraftID, control
}

func TestPublishAgency_DocumentCarriesControlsAndAttachments(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	draftID, control := publishableDraftWithGuardedPathway(t, mgr, ctx)

	pub, err := mgr.PublishAgency(ctx, draftID, "release-1", "production")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if len(pub.Document.Controls) != 1 || pub.Document.Controls[0].Code != control.Code || pub.Document.Controls[0].Severity != models.SeverityMedium {
		t.Fatalf("document controls: %+v", pub.Document.Controls)
	}
	if len(pub.Document.ControlAttachments) != 1 {
		t.Fatalf("document control attachments: %+v", pub.Document.ControlAttachments)
	}
	a := pub.Document.ControlAttachments[0]
	if a.ControlCode != control.Code || a.TargetType != string(models.TargetTypeRiskPathway) || a.TargetCode == "" {
		t.Fatalf("attachment addressed by code expected, got %+v", a)
	}
}

func TestCreateDraft_FromATagRestoresControlsAndAttachments(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	draftID, control := publishableDraftWithGuardedPathway(t, mgr, ctx)
	if _, err := mgr.PublishAgency(ctx, draftID, "release-1", "production"); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}

	revisit, err := mgr.CreateDraft(ctx, "revisit release-1", "release-1", "publication")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	controls, err := mgr.ListControls(ctx, revisit.ID, mwanachamaagency.ControlFilter{})
	if err != nil || len(controls) != 1 {
		t.Fatalf("ListControls(revisit): %v (%d)", err, len(controls))
	}
	if controls[0].Code != control.Code || controls[0].ID == control.ID || controls[0].Text != control.Text {
		t.Fatalf("control not restored from the tag: %+v", controls[0])
	}
	attachments, err := mgr.ListControlAttachments(ctx, controls[0].ID)
	if err != nil || len(attachments) != 1 || attachments[0].DraftID != revisit.ID {
		t.Fatalf("attachment not restored: %+v (%v)", attachments, err)
	}
	gaps, err := mgr.UnguardedPathways(ctx, revisit.ID)
	if err != nil || len(gaps) != 1 || gaps[0].Side != models.RiskSideConsequence {
		t.Fatalf("pathway guard not restored: %+v (%v)", gaps, err)
	}
}

func TestPublishAgency_ControlChangeIsAChange(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	draftID, control := publishableDraftWithGuardedPathway(t, mgr, ctx)
	if _, err := mgr.PublishAgency(ctx, draftID, "release-1", "production"); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, draftID, "release-2", "production"); !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("expected ErrNoChangesDetected, got %v", err)
	}

	if _, err := mgr.UpdateControl(ctx, models.Control{ID: control.ID, DraftID: draftID, Text: "Two approvers on every payment"}); err != nil {
		t.Fatalf("UpdateControl: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, draftID, "release-2", "production"); err != nil {
		t.Fatalf("a changed control must be publishable: %v", err)
	}
}

func TestPublishAgency_AttachmentChangeIsAChange(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	draftID, control := publishableDraftWithGuardedPathway(t, mgr, ctx)
	if _, err := mgr.PublishAgency(ctx, draftID, "release-1", "production"); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	attachments, err := mgr.ListControlAttachments(ctx, control.ID)
	if err != nil || len(attachments) != 1 {
		t.Fatalf("ListControlAttachments: %v (%d)", err, len(attachments))
	}
	if err := mgr.DeleteControlAttachment(ctx, attachments[0].ID); err != nil {
		t.Fatalf("DeleteControlAttachment: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, draftID, "release-2", "production"); err != nil {
		t.Fatalf("detaching a control must be a publishable change: %v", err)
	}
}
