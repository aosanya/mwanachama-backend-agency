package mwanachamaagency_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

func newTestManager(t *testing.T) mwanachamaagency.AgencyManager {
	t.Helper()
	mgr, _, _ := newTestManagerDB(t)
	return mgr
}

func newTestManagerDB(t *testing.T) (mwanachamaagency.AgencyManager, *gorm.DB, *spec.Spec) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}

	s, err := mwanachamaagency.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}

	mgr, err := mwanachamaagency.NewAgencyManager(db, s)
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	return mgr, db, s
}

func TestNewAgencyManager_NilDB(t *testing.T) {
	s, err := mwanachamaagency.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if _, err := mwanachamaagency.NewAgencyManager(nil, s); err == nil {
		t.Fatal("expected error for nil db")
	}
}

func tableOf(t *testing.T, s *spec.Spec, role string) string {
	t.Helper()
	o, ok := s.ByRole(role)
	if !ok {
		t.Fatalf("the spec fills no object for the role %q", role)
	}
	return s.TableFor(o)
}
