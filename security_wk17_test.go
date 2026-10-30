package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCreate_IgnoresCallerSuppliedID_AndCreatedAt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)

	forgedCreatedAt := "1999-01-01T00:00:00Z"

	g, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, ID: "forged-goal-id", Title: "Probe", CreatedAt: forgedCreatedAt})
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

	ag, err := newTestManager(t).CreateAgency(ctx, models.Agency{ID: "forged-agency-id", Name: "Probe Co 2", CreatedAt: forgedCreatedAt})
	if err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	if ag.ID == "forged-agency-id" {
		t.Fatal("CreateAgency honoured a caller-supplied id")
	}
	if ag.CreatedAt == forgedCreatedAt {
		t.Fatal("CreateAgency honoured a caller-supplied created_at")
	}
}

// TestCreate_RefusesUnknownDraftID covers the other half of WK17: a
// caller-supplied draft_id naming no real Draft used to be written
// verbatim, attaching a row to a Draft CreateDraft's own fork never
// produced.
func TestCreate_RefusesUnknownDraftID(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	_, err := mgr.CreateGoal(ctx, models.Goal{Title: "Probe", DraftID: "forged-draft-id"})
	if !errors.Is(err, mwanachamaagency.ErrDraftNotFound) {
		t.Fatalf("got %v, want ErrDraftNotFound", err)
	}
}

// TestCreate_RefusesArchivedDraftID: a Draft that is already archived is
// terminal — a new row attached to it after the fact would never be
// reachable by any future publish.
func TestCreate_RefusesArchivedDraftID(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)

	if _, err := mgr.ArchiveDraft(ctx, draftID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}
	if _, err := mgr.CreateGoal(ctx, models.Goal{Title: "Late arrival", DraftID: draftID}); !errors.Is(err, mwanachamaagency.ErrDraftNotOpen) {
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
	draftID := newDraftAgency(t, mgr, ctx)

	draft, err := mgr.CreateDraft(ctx, "in progress", draftID, "draft")
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
