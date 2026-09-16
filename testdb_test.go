package mwanachamaagency_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

// newTestManager builds a mwanachamaagency.AgencyManager backed by a fresh
// in-memory sqlite database, migrated the same way a real deployment would.
func newTestManager(t *testing.T) mwanachamaagency.AgencyManager {
	t.Helper()
	mgr, _, _ := newTestManagerDB(t)
	return mgr
}

// newTestManagerDB also hands back the store beneath the manager, for the
// few tests that need to plant a row no manager method can produce — the
// data predating a cascade fix, which the Issue scan still has to catch.
func newTestManagerDB(t *testing.T) (mwanachamaagency.AgencyManager, *gorm.DB, mwanachamaagency.TableNames) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}

	tables := mwanachamaagency.DefaultTableNames("test")
	if err := mwanachamaagency.Migrate(db, tables); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	mgr, err := mwanachamaagency.NewAgencyManager(db, tables)
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	return mgr, db, tables
}

func TestNewAgencyManager_NilDB(t *testing.T) {
	if _, err := mwanachamaagency.NewAgencyManager(nil, mwanachamaagency.DefaultTableNames("test")); err == nil {
		t.Fatal("expected error for nil db")
	}
}
