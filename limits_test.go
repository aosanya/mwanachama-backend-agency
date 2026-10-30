package mwanachamaagency_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCheckDesignSpecSize_RefusesOnlyPastTheCap(t *testing.T) {
	if err := mwanachamaagency.CheckDesignSpecSize(mwanachamaagency.MaxDesignSpecEntities); err != nil {
		t.Fatalf("a spec at the cap was refused: %v", err)
	}
	err := mwanachamaagency.CheckDesignSpecSize(mwanachamaagency.MaxDesignSpecEntities + 1)
	if !errors.Is(err, mwanachamaagency.ErrLimitExceeded) {
		t.Fatalf("a spec past the cap answered %v, want ErrLimitExceeded", err)
	}
}

func TestCreateDraft_AnAgencyHoldsAtMostTheCapOfDraftsOpenOrArchived(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "Drafts"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	held, err := mgr.ListDrafts(ctx, mwanachamaagency.DraftFilter{})
	if err != nil {
		t.Fatalf("ListDrafts: %v", err)
	}
	for i := len(held); i < mwanachamaagency.MaxDraftsPerAgency; i++ {
		draft, err := mgr.CreateDraft(ctx, fmt.Sprint("d", i), "", "")
		if err != nil {
			t.Fatalf("CreateDraft %d under the cap: %v", i, err)
		}
		if _, err := mgr.ArchiveDraft(ctx, draft.ID); err != nil {
			t.Fatalf("ArchiveDraft: %v", err)
		}
	}
	if _, err := mgr.CreateDraft(ctx, "one too many", "", ""); !errors.Is(err, mwanachamaagency.ErrLimitExceeded) {
		t.Fatalf("CreateDraft past the cap answered %v, want ErrLimitExceeded", err)
	}
}
