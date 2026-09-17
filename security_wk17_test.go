package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// TestCreate_IgnoresCallerSuppliedID_AndCreatedAt covers WK17: a caller
// used to be able to set id/created_at directly on a Create<Type> call and
// have the server write them verbatim. Spot-checks one Draft-scoped type
// (Goal) and one non-Draft-scoped type (GroupType) — every other
// Create<Type> shares the identical fix.
func TestCreate_IgnoresCallerSuppliedID_AndCreatedAt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	forgedCreatedAt := "1999-01-01T00:00:00Z"

	g, err := mgr.CreateGoal(ctx, models.Goal{ID: "forged-goal-id", Title: "Probe", CreatedAt: forgedCreatedAt})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if g.ID == "forged-goal-id" {
		t.Fatal("CreateGoal honoured a caller-supplied id")
	}
	if g.CreatedAt == forgedCreatedAt {
		t.Fatal("CreateGoal honoured a caller-supplied created_at")
	}
	if _, err := time.Parse(time.RFC3339, g.CreatedAt); err != nil {
		t.Fatalf("CreateGoal's stamped created_at %q does not parse as RFC3339: %v", g.CreatedAt, err)
	}

	gt, err := mgr.CreateGroupType(ctx, models.GroupType{ID: "forged-group-type-id", Name: "Department", CreatedAt: forgedCreatedAt})
	if err != nil {
		t.Fatalf("CreateGroupType: %v", err)
	}
	if gt.ID == "forged-group-type-id" {
		t.Fatal("CreateGroupType honoured a caller-supplied id")
	}
	if gt.CreatedAt == forgedCreatedAt {
		t.Fatal("CreateGroupType honoured a caller-supplied created_at")
	}
}

// TestCreate_RefusesUnknownDraftID covers the other half of WK17: a
// caller-supplied draft_id naming no real Draft used to be written
// verbatim, attaching a live row to a Draft CreateDraft's own fork never
// produced.
func TestCreate_RefusesUnknownDraftID(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	_, err := mgr.CreateGoal(ctx, models.Goal{Title: "Probe", DraftID: "forged-draft-id"})
	if !errors.Is(err, mwanachamaagency.ErrDraftNotFound) {
		t.Fatalf("got %v, want ErrDraftNotFound", err)
	}
}

// TestCreate_RefusesPromotedOrArchivedDraftID: a Draft that already
// finished its lifecycle (promoted or archived) is terminal — a new row
// attached to it after the fact would never be reachable by any future
// PromoteDraft/ArchiveDraft call.
func TestCreate_RefusesPromotedOrArchivedDraftID(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	seedLiveAgency(t, mgr, ctx)

	promoted, err := mgr.CreateDraft(ctx, "will be promoted", "", "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if _, err := mgr.PromoteDraft(ctx, promoted.ID); err != nil {
		t.Fatalf("PromoteDraft: %v", err)
	}
	if _, err := mgr.CreateGoal(ctx, models.Goal{Title: "Late arrival", DraftID: promoted.ID}); !errors.Is(err, mwanachamaagency.ErrDraftNotOpen) {
		t.Fatalf("CreateGoal into a promoted draft: got %v, want ErrDraftNotOpen", err)
	}

	archived, err := mgr.CreateDraft(ctx, "will be archived", "", "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if _, err := mgr.ArchiveDraft(ctx, archived.ID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}
	if _, err := mgr.CreateGoal(ctx, models.Goal{Title: "Late arrival", DraftID: archived.ID}); !errors.Is(err, mwanachamaagency.ErrDraftNotOpen) {
		t.Fatalf("CreateGoal into an archived draft: got %v, want ErrDraftNotOpen", err)
	}
}

// TestCreate_AllowsOpenDraftID confirms the fix didn't break the
// legitimate, already-tested capability (see issue_test.go) of creating a
// brand-new row directly scoped into a still-open Draft, not only via
// CreateDraft's own fork-copy.
func TestCreate_AllowsOpenDraftID(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, _, _, _, _ = seedLiveAgency(t, mgr, ctx); true {
	}

	draft, err := mgr.CreateDraft(ctx, "in progress", "", "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	g, err := mgr.CreateGoal(ctx, models.Goal{Title: "New in this draft", DraftID: draft.ID})
	if err != nil {
		t.Fatalf("CreateGoal into an open draft: %v", err)
	}
	if g.DraftID != draft.ID {
		t.Fatalf("got draft_id %q, want %q", g.DraftID, draft.ID)
	}
}
