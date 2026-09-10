package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestPublishAgency_NoDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedLiveAgency(t, mgr, ctx)

	p1, err := mgr.PublishAgency(ctx, "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if p1.Version != 1 || p1.Tag != "v1" || p1.Status != "active" || p1.ContentHash == "" {
		t.Fatalf("unexpected first publication: %+v", p1)
	}

	if _, err := mgr.PublishAgency(ctx, ""); !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("expected ErrNoChangesDetected republishing unchanged content, got %v", err)
	}

	if _, err := mgr.CreateGoal(ctx, models.Goal{Title: "New goal"}); err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	p2, err := mgr.PublishAgency(ctx, "")
	if err != nil {
		t.Fatalf("PublishAgency after change: %v", err)
	}
	if p2.Version != 2 || p2.Tag != "v2" {
		t.Fatalf("expected version 2, got %+v", p2)
	}

	gotP1, err := mgr.GetPublication(ctx, 1)
	if err != nil {
		t.Fatalf("GetPublication(1): %v", err)
	}
	if gotP1.Status != "archived" {
		t.Fatalf("expected publishing v2 to archive v1, got %q", gotP1.Status)
	}
}

func TestPublishAgency_RACIOnlyChangeIsAChange(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	p1, err := mgr.PublishAgency(ctx, "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}

	role, err := mgr.CreateRole(ctx, models.Role{Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		RoleID: role.ID, ObjectiveID: o.ID, Label: models.RACILabelAccountable,
	}); err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	p2, err := mgr.PublishAgency(ctx, "")
	if err != nil {
		t.Fatalf("expected a RACI-only change to require republishing, got %v", err)
	}
	if p2.Version != p1.Version+1 {
		t.Fatalf("expected version %d, got %d", p1.Version+1, p2.Version)
	}
}

func TestPublishAgency_RequiresPromotedDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, _, _, _, _ := seedLiveAgency(t, mgr, ctx)

	d, err := mgr.CreateDraft(ctx, "not promoted yet", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	if _, err := mgr.PublishAgency(ctx, d.ID); !errors.Is(err, mwanachamaagency.ErrDraftNotPromoted) {
		t.Fatalf("expected ErrDraftNotPromoted, got %v", err)
	}

	if _, err := mgr.PromoteDraft(ctx, d.ID); err != nil {
		t.Fatalf("PromoteDraft: %v", err)
	}
	p, err := mgr.PublishAgency(ctx, d.ID)
	if err != nil {
		t.Fatalf("PublishAgency after promote: %v", err)
	}
	if p.DraftID != d.ID {
		t.Fatalf("expected publication to record DraftID %q, got %q", d.ID, p.DraftID)
	}
}

func TestUpdatePublicationStatus(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedLiveAgency(t, mgr, ctx)

	p, err := mgr.PublishAgency(ctx, "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}

	updated, err := mgr.UpdatePublicationStatus(ctx, p.Version, "archived")
	if err != nil {
		t.Fatalf("UpdatePublicationStatus: %v", err)
	}
	if updated.Status != "archived" {
		t.Fatalf("expected archived, got %q", updated.Status)
	}

	if _, err := mgr.UpdatePublicationStatus(ctx, p.Version, "active"); !errors.Is(err, mwanachamaagency.ErrInvalidPublicationStatus) {
		t.Fatalf("expected ErrInvalidPublicationStatus reactivating an archived publication, got %v", err)
	}
}

func TestGetPublication_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.GetPublication(context.Background(), 99); !errors.Is(err, mwanachamaagency.ErrPublicationNotFound) {
		t.Fatalf("expected ErrPublicationNotFound, got %v", err)
	}
}
