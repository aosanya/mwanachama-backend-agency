package gormstore_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
)

func TestMigrate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}

	tables := gormstore.DefaultTableNames("test")
	if err := gormstore.Migrate(db, tables); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	for _, table := range []string{
		tables.Agencies, tables.Drafts, tables.Goals, tables.Workflows, tables.WorkItems,
		tables.EventFlowSteps, tables.AgencySnapshots, tables.AgencyPublications,
	} {
		if !db.Migrator().HasTable(table) {
			t.Errorf("expected table %q to exist after Migrate", table)
		}
	}
}
