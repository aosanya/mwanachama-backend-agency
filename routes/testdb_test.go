package routes_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// newTestManager builds an AgencyManager backed by a fresh in-memory
// sqlite database, migrated the same way a real deployment would.
func newTestManager(t *testing.T) mwanachamaagency.AgencyManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}

	tables := mwanachamaagency.DefaultTableNames("routes_test")
	if err := mwanachamaagency.Migrate(db, tables); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	mgr, err := mwanachamaagency.NewAgencyManager(db, tables)
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	return mgr
}

// bootstrapDraftID returns the single Draft gormstore.EnsureWorkingDraft
// creates during Migrate — the home for gormstore.BackfillModules' seeded
// catalog rows before any test has forked or created a Draft of its own.
func bootstrapDraftID(t *testing.T, mgr mwanachamaagency.AgencyManager) string {
	t.Helper()
	drafts, err := mgr.ListDrafts(t.Context(), mwanachamaagency.DraftFilter{})
	if err != nil {
		t.Fatalf("ListDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("expected exactly 1 bootstrap draft right after Migrate, got %d: %+v", len(drafts), drafts)
	}
	return drafts[0].ID
}

// newTestManagerWithDraft is newTestManager plus an Agency and one empty
// open Draft — every business-entity row belongs to a real Draft
// (AGD-006), so this is the minimum setup any Create<Type> call needs,
// whether made directly or through an HTTP handler's request body.
func newTestManagerWithDraft(t *testing.T) (mwanachamaagency.AgencyManager, string) {
	t.Helper()
	mgr := newTestManager(t)
	if _, err := mgr.CreateAgency(t.Context(), models.Agency{Name: "SWELab"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	d, err := mgr.CreateDraft(t.Context(), "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	return mgr, d.ID
}
