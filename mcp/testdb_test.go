package mcp_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

// newTestManager builds an AgencyManager backed by a fresh in-memory
// sqlite database, migrated the same way a real deployment would — same
// pattern as routes_test's own helper.
func newTestManager(t *testing.T) mwanachamaagency.AgencyManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}

	tables := mwanachamaagency.DefaultTableNames("mcp_test")
	if err := mwanachamaagency.Migrate(db, tables); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	mgr, err := mwanachamaagency.NewAgencyManager(db, tables)
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	return mgr
}
