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

// The two AGD-005 retirements, each against a database that already has
// what is being retired — the only case that exercises the drops at all,
// since a fresh database never creates either.
func TestMigrate_DropsRetiredWorkPlanTables(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	tables := gormstore.DefaultTableNames("retired")

	type legacyWorkPlanRow struct {
		ID string `gorm:"primaryKey"`
	}
	for _, name := range []string{"retired_work_plans", "retired_context_sources"} {
		if err := db.Table(name).AutoMigrate(&legacyWorkPlanRow{}); err != nil {
			t.Fatalf("seeding %s: %v", name, err)
		}
		if !db.Migrator().HasTable(name) {
			t.Fatalf("expected %s to exist before Migrate", name)
		}
	}

	if err := gormstore.Migrate(db, tables); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	for _, name := range []string{"retired_work_plans", "retired_context_sources"} {
		if db.Migrator().HasTable(name) {
			t.Errorf("expected Migrate to drop %s", name)
		}
	}
}

func TestMigrate_DropsRetiredStatusColumn(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	tables := gormstore.DefaultTableNames("retired")

	type legacyPublicationRow struct {
		ID     string `gorm:"primaryKey"`
		Status string
	}
	if err := db.Table(tables.AgencyPublications).AutoMigrate(&legacyPublicationRow{}); err != nil {
		t.Fatalf("seeding the legacy publications table: %v", err)
	}

	if err := gormstore.Migrate(db, tables); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	publications := db.Table(tables.AgencyPublications)
	if publications.Migrator().HasColumn(&gormstore.AgencyPublicationRow{}, "status") {
		t.Error("expected Migrate to drop the retired status column")
	}
	if !publications.Migrator().HasColumn(&gormstore.AgencyPublicationRow{}, "channel") {
		t.Error("expected Migrate to add the channel column")
	}
}
