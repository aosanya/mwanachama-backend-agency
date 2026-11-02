package mcp_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

func newTestManager(t *testing.T, opts ...mwanachamaagency.Option) mwanachamaagency.AgencyManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}

	s, err := mwanachamaagency.SpecFor("mcptest")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}

	mgr, err := mwanachamaagency.NewAgencyManager(db, s, opts...)
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	return mgr
}
