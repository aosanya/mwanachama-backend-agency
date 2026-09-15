package gormstore_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// TestBackfillModules_FreshTable asserts a fresh (empty) Module table
// backfills "auth" as Enabled: true, Locked: true, and every other known
// catalog code as Enabled: false, Locked: false — AGD-002's reversed
// default (previously every code defaulted Enabled: true).
func TestBackfillModules_FreshTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	tables := gormstore.DefaultTableNames("test")
	if err := gormstore.Migrate(db, tables); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var rows []gormstore.ModuleRow
	if err := db.Table(tables.Modules).Find(&rows).Error; err != nil {
		t.Fatalf("find modules: %v", err)
	}
	if len(rows) != len(models.ModuleCatalog) {
		t.Fatalf("expected %d rows, got %d: %+v", len(models.ModuleCatalog), len(rows), rows)
	}
	byCode := make(map[string]gormstore.ModuleRow, len(rows))
	for _, r := range rows {
		byCode[r.Code] = r
	}

	auth, ok := byCode["auth"]
	if !ok {
		t.Fatal("expected an auth row")
	}
	if !auth.Enabled || !auth.Locked {
		t.Fatalf("expected auth to backfill Enabled: true, Locked: true, got %+v", auth)
	}

	for _, entry := range models.ModuleCatalog {
		if entry.Code == "auth" {
			continue
		}
		r, ok := byCode[entry.Code]
		if !ok {
			t.Fatalf("expected a row for code %q", entry.Code)
		}
		if r.Enabled || r.Locked {
			t.Fatalf("expected %q to backfill Enabled: false, Locked: false, got %+v", entry.Code, r)
		}
	}
}

// TestBackfillModules_LeavesExistingRowsAlone asserts an already-migrated
// instance's live Module rows (e.g. every code Enabled: true, from before
// AGD-002's default reversal) are never modified by a later Migrate/
// BackfillModules call — it only ever inserts a missing code, never
// updates one that's already there.
func TestBackfillModules_LeavesExistingRowsAlone(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	tables := gormstore.DefaultTableNames("test")
	if err := gormstore.Migrate(db, tables); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Simulate a pre-existing deployment that already backfilled every
	// known code as live and enabled, under the old (pre-reversal) rule.
	now := models.NowRFC3339()
	for _, entry := range models.ModuleCatalog {
		if err := db.Table(tables.Modules).Where("code = ? AND draft_id IS NULL", entry.Code).
			Updates(map[string]any{"enabled": true, "locked": false, "updated_at": now}).Error; err != nil {
			t.Fatalf("seed existing row %q: %v", entry.Code, err)
		}
	}

	// Re-running Migrate (and so BackfillModules) must leave every row's
	// Enabled/Locked exactly as set above.
	if err := gormstore.Migrate(db, tables); err != nil {
		t.Fatalf("Migrate (again): %v", err)
	}

	var rows []gormstore.ModuleRow
	if err := db.Table(tables.Modules).Find(&rows).Error; err != nil {
		t.Fatalf("find modules: %v", err)
	}
	if len(rows) != len(models.ModuleCatalog) {
		t.Fatalf("expected %d rows (no new inserts), got %d: %+v", len(models.ModuleCatalog), len(rows), rows)
	}
	for _, r := range rows {
		if !r.Enabled {
			t.Fatalf("expected pre-existing row %q to remain Enabled: true, got %+v", r.Code, r)
		}
		if r.Locked {
			t.Fatalf("expected pre-existing row %q to remain Locked: false, got %+v", r.Code, r)
		}
	}
}
