package mwanachamaagency_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// seedPublishableAgency is seedLiveAgency plus the one thing publish-time
// validation demands that it leaves out: an Accountable Role on its
// Objective. Every test here needs a live Agency that actually passes the
// gate before it can assert anything about versions, tags or statuses.
func seedPublishableAgency(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context) (models.Agency, models.Objective, models.Role) {
	t.Helper()
	ag, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	role, err := mgr.CreateRole(ctx, models.Role{Name: "owner"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		RoleID: role.ID, ObjectiveID: o.ID, Label: models.RACILabelAccountable,
	}); err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}
	return ag, o, role
}

func TestPublishAgency_NoDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedPublishableAgency(t, mgr, ctx)

	p1, err := mgr.PublishAgency(ctx, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if p1.Version != 1 || p1.Tag != "v1" || p1.Status != "active" || p1.ContentHash == "" {
		t.Fatalf("unexpected first publication: %+v", p1)
	}

	if _, err := mgr.PublishAgency(ctx, "", ""); !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("expected ErrNoChangesDetected republishing unchanged content, got %v", err)
	}

	// An Agency-wide Value changes the content hash without introducing a
	// blocking Issue, unlike a bare new Goal (which would have no
	// Objectives).
	if _, err := mgr.CreateValue(ctx, models.Value{Text: "correctness over speed"}); err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	p2, err := mgr.PublishAgency(ctx, "", "")
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
	_, o, _ := seedPublishableAgency(t, mgr, ctx)

	p1, err := mgr.PublishAgency(ctx, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}

	consulted, err := mgr.CreateRole(ctx, models.Role{Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		RoleID: consulted.ID, ObjectiveID: o.ID, Label: models.RACILabelConsulted,
	}); err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	p2, err := mgr.PublishAgency(ctx, "", "")
	if err != nil {
		t.Fatalf("expected a RACI-only change to require republishing, got %v", err)
	}
	if p2.Version != p1.Version+1 {
		t.Fatalf("expected version %d, got %d", p1.Version+1, p2.Version)
	}
}

// TestPublishAgency_ModuleOnlyChangeIsAChange mirrors
// TestPublishAgency_RACIOnlyChangeIsAChange: a Module/ObjectiveModule-only
// edit must require republishing too, now that liveContentHash covers them.
func TestPublishAgency_ModuleOnlyChangeIsAChange(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)

	p1, err := mgr.PublishAgency(ctx, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}

	mod, err := mgr.CreateModule(ctx, models.Module{Code: "widgets", Name: "Widgets", Enabled: true})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	if _, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{ObjectiveID: o.ID, ModuleID: mod.ID}); err != nil {
		t.Fatalf("CreateObjectiveModule: %v", err)
	}

	p2, err := mgr.PublishAgency(ctx, "", "")
	if err != nil {
		t.Fatalf("expected a Module-only change to require republishing, got %v", err)
	}
	if p2.Version != p1.Version+1 {
		t.Fatalf("expected version %d, got %d", p1.Version+1, p2.Version)
	}
}

func TestPublishAgency_RequiresPromotedDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, _, _ := seedPublishableAgency(t, mgr, ctx)

	d, err := mgr.CreateDraft(ctx, "not promoted yet", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	if _, err := mgr.PublishAgency(ctx, d.ID, ""); !errors.Is(err, mwanachamaagency.ErrDraftNotPromoted) {
		t.Fatalf("expected ErrDraftNotPromoted, got %v", err)
	}

	if _, err := mgr.PromoteDraft(ctx, d.ID); err != nil {
		t.Fatalf("PromoteDraft: %v", err)
	}
	p, err := mgr.PublishAgency(ctx, d.ID, "")
	if err != nil {
		t.Fatalf("PublishAgency after promote: %v", err)
	}
	if p.DraftID != d.ID {
		t.Fatalf("expected publication to record DraftID %q, got %q", d.ID, p.DraftID)
	}
}

// TestPublishAgency_RefusedOnValidationError asserts the gate: the
// unassigned Objective seedLiveAgency leaves behind is error-severity, so
// publishing is refused until it is fixed (AGD-004).
func TestPublishAgency_RefusedOnValidationError(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	_, err := mgr.PublishAgency(ctx, "", "")
	if !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) {
		t.Fatalf("expected ErrPublishValidationFailed, got %v", err)
	}
	if !strings.Contains(err.Error(), "Accountable") {
		t.Errorf("expected the error to name what is wrong, got %q", err)
	}

	pubs, err := mgr.ListPublications(ctx, mwanachamaagency.PublicationFilter{})
	if err != nil {
		t.Fatalf("ListPublications: %v", err)
	}
	if len(pubs) != 0 {
		t.Fatalf("a refused publish must not write a publication, got %+v", pubs)
	}

	// The same findings are readable as rows through the live scan.
	issues, err := mgr.ListIssues(ctx, "", mwanachamaagency.IssueFilter{Severity: "error"})
	if err != nil {
		t.Fatalf("ListIssues(live, error): %v", err)
	}
	if len(issues) != 1 || issues[0].EntityID != o.ID {
		t.Fatalf("expected the same single finding for objective %s, got %+v", o.ID, issues)
	}

	role, err := mgr.CreateRole(ctx, models.Role{Name: "owner"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		RoleID: role.ID, ObjectiveID: o.ID, Label: models.RACILabelAccountable,
	}); err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, "", ""); err != nil {
		t.Fatalf("expected the publish to succeed once fixed, got %v", err)
	}
}

// TestPublishAgency_WarningsDoNotBlock asserts severity is what the gate
// keys on, not issue count: a warning-severity finding publishes through.
func TestPublishAgency_WarningsDoNotBlock(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedPublishableAgency(t, mgr, ctx)

	issues, err := mgr.ListIssues(ctx, "", mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues(live): %v", err)
	}
	for _, i := range issues {
		if i.Severity == models.IssueSeverityError {
			t.Fatalf("seed should be clean of errors, got %+v", i)
		}
	}
	if _, err := mgr.PublishAgency(ctx, "", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
}

func TestPublishAgency_Tag(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedPublishableAgency(t, mgr, ctx)

	p1, err := mgr.PublishAgency(ctx, "", "2026-Q4 restructure")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if p1.Tag != "2026-Q4 restructure" || p1.Version != 1 {
		t.Fatalf("expected the operator's tag on version 1, got %+v", p1)
	}

	if _, err := mgr.CreateValue(ctx, models.Value{Text: "correctness over speed"}); err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, "", "2026-Q4 restructure"); !errors.Is(err, mwanachamaagency.ErrDuplicateTag) {
		t.Fatalf("expected ErrDuplicateTag, got %v", err)
	}

	p2, err := mgr.PublishAgency(ctx, "", "")
	if err != nil {
		t.Fatalf("PublishAgency (fallback tag): %v", err)
	}
	if p2.Tag != "v2" || p2.Version != 2 {
		t.Fatalf("expected an empty tag to fall back to v2, got %+v", p2)
	}
}

// TestPublishAgency_TagFallbackCollision covers the fallback's own
// uniqueness: an operator is free to name version 1 "v2" by hand, and
// version 2's generated tag must not silently duplicate it.
func TestPublishAgency_TagFallbackCollision(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedPublishableAgency(t, mgr, ctx)

	if _, err := mgr.PublishAgency(ctx, "", "v2"); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if _, err := mgr.CreateValue(ctx, models.Value{Text: "correctness over speed"}); err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, "", ""); !errors.Is(err, mwanachamaagency.ErrDuplicateTag) {
		t.Fatalf("expected ErrDuplicateTag from the generated tag, got %v", err)
	}
}

func TestUpdatePublicationStatus(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedPublishableAgency(t, mgr, ctx)

	p, err := mgr.PublishAgency(ctx, "", "")
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
