package mwanachamaagency_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

func openDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	return db
}

func liveTables(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var names []string
	if err := db.Raw(`select name from sqlite_master where type='table' and name not like 'sqlite_%'`).
		Scan(&names).Error; err != nil {
		t.Fatalf("list tables: %v", err)
	}
	sort.Strings(names)
	return names
}

func indexesOf(t *testing.T, db *gorm.DB, table string) []string {
	t.Helper()
	var names []string
	if err := db.Raw(`select name from sqlite_master where type='index' and tbl_name = ?`, table).
		Scan(&names).Error; err != nil {
		t.Fatalf("list indexes: %v", err)
	}
	sort.Strings(names)
	return names
}

func TestProvisionCreatesEveryDeclaredTable(t *testing.T) {
	db := openDB(t)
	s, err := mwanachamaagency.SpecFor("agency")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}

	want := []string{
		"agency_agency_agencies", "agency_agency_agency_publications",
		"agency_agency_code_sequences", "agency_agency_drafts",
		"agency_agency_event_flow_steps", "agency_agency_goals",
		"agency_agency_issues", "agency_agency_modules",
		"agency_agency_objective_modules", "agency_agency_objectives",
		"agency_agency_role_assignments", "agency_agency_roles",
		"agency_agency_rule_attachments", "agency_agency_rules",
		"agency_agency_value_attachments", "agency_agency_values",
		"agency_agency_work_items", "agency_agency_workflows",
	}
	if got := liveTables(t, db); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("tables =\n  %v\nwant\n  %v", got, want)
	}
}

func TestProvisionIsIdempotent(t *testing.T) {
	db := openDB(t)
	s, err := mwanachamaagency.SpecFor("agency")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := mwanachamaagency.Provision(db, s); err != nil {
			t.Fatalf("Provision %d: %v", i, err)
		}
	}
	if n := len(liveTables(t, db)); n != 18 {
		t.Errorf("tables = %d, want 18", n)
	}
}

func TestProvisionMovesALegacyDatabaseOntoTheDeclaredNames(t *testing.T) {
	db := openDB(t)
	seedLegacyDatabase(t, db)

	before := indexesOf(t, db, "agency_goals")
	if len(before) == 0 {
		t.Fatal("the legacy fixture created no indexes, so this test cannot show them replaced")
	}

	s, err := mwanachamaagency.SpecFor("agency")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}

	tables := liveTables(t, db)
	if contains(tables, "agency_goals") {
		t.Error("the legacy table is still there, so the rename did not happen")
	}
	if !contains(tables, "agency_agency_goals") {
		t.Fatalf("tables = %v, want the declared name", tables)
	}

	var got struct {
		Title       string
		LastUpdated string
	}
	if err := db.Table("agency_agency_goals").
		Select("title, last_updated").Where("id = ?", "goal-1").Scan(&got).Error; err != nil {
		t.Fatalf("read the moved row: %v", err)
	}
	if got.Title != "Ship the conversion" {
		t.Errorf("title = %q, want the row carried across unchanged", got.Title)
	}
	if got.LastUpdated != "2026-09-24T01:00:00.000000000Z" {
		t.Errorf("last_updated = %q, want what updated_at held", got.LastUpdated)
	}

	for _, name := range indexesOf(t, db, "agency_agency_goals") {
		if strings.HasPrefix(name, "idx_") {
			t.Errorf("index %q survived, and the declared index over the same column is now a second one", name)
		}
	}
	if !contains(indexesOf(t, db, "agency_agency_goals"), "agency_agency_goals_code_idx") {
		t.Errorf("indexes = %v, want the declared ones created", indexesOf(t, db, "agency_agency_goals"))
	}
}

func TestProvisionLeavesTheRetiredSnapshotTableAlone(t *testing.T) {
	db := openDB(t)
	seedLegacyDatabase(t, db)

	s, err := mwanachamaagency.SpecFor("agency")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if !contains(liveTables(t, db), "agency_agency_snapshots") {
		t.Error("agency_agency_snapshots is gone — the conversion declares no snapshot, but it does not drop one either")
	}
}

func TestSpecForRefusesAnInstanceTheDatabaseWouldTruncate(t *testing.T) {
	_, err := mwanachamaagency.SpecFor("agriculturalcommodityexchangewarehousereceiptsystem")
	if err == nil {
		t.Fatal("an instance too long for the database was accepted")
	}
	if !strings.Contains(err.Error(), "truncates at 63") {
		t.Errorf("err = %v, want it to name the limit", err)
	}
}

func contains(in []string, want string) bool {
	for _, s := range in {
		if s == want {
			return true
		}
	}
	return false
}

func TestProvisionRefusesToStrandRowsWhenBothTablesExist(t *testing.T) {
	db := openDB(t)
	s, err := mwanachamaagency.SpecFor("agency")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if err := db.Exec(`create table agency_goals (id text primary key, title text)`).Error; err != nil {
		t.Fatalf("plant a legacy table: %v", err)
	}
	if err := db.Exec(`insert into agency_goals (id, title) values ('g', 'stranded')`).Error; err != nil {
		t.Fatalf("plant a stranded row: %v", err)
	}

	err = mwanachamaagency.Provision(db, s)
	if err == nil {
		t.Fatal("provisioned over a legacy table still holding rows")
	}
	if !strings.Contains(err.Error(), "agency_goals") || !strings.Contains(err.Error(), "1 row") {
		t.Errorf("err = %v, want it to name the table and what it holds", err)
	}
}

func seedLegacyDatabase(t *testing.T, db *gorm.DB) string {
	t.Helper()
	stmts := []string{
		`create table agency_agencies (id text primary key, name text, description text,
			enabled numeric, default_failure_pipeline_budget integer,
			inactivity_timeout_seconds integer, event_flows text,
			created_at text, updated_at text, deleted numeric)`,
		`create table agency_drafts (id text primary key, description text, forked_from_id text,
			forked_from_type text, status text, created_at text, updated_at text, deleted numeric)`,
		`create table agency_goals (id text primary key, code text, draft_id text not null,
			title text, description text, ordinality integer,
			created_at text, updated_at text, deleted numeric)`,
		`create index idx_agency_goals_code on agency_goals(code)`,
		`create index idx_agency_goals_draft_id on agency_goals(draft_id)`,
		`create table agency_agency_snapshots (id text primary key, draft_id text, snapshot_at text)`,
		`insert into agency_drafts (id, description, status, created_at, updated_at, deleted)
			values ('draft-1', 'Initial draft', 'open', '2026-09-24T00:00:00.000000000Z',
			        '2026-09-24T00:00:00.000000000Z', 0)`,
		`insert into agency_goals (id, code, draft_id, title, description, ordinality,
			created_at, updated_at, deleted)
			values ('goal-1', 'G-1', 'draft-1', 'Ship the conversion', 'A row written before the rename.',
			        1, '2026-09-24T00:00:00.000000000Z', '2026-09-24T01:00:00.000000000Z', 0)`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("seed the legacy database: %v\n  in: %s", err, stmt)
		}
	}
	return "draft-1"
}

// Provisioning runs at every mount, and a mount can be every request: a
// remote database turns 70-odd statements into ten seconds of page load.
// A database that is already provisioned must therefore cost a handful of
// reads and no writes at all.
func TestProvisionIsCheapWhenThereIsNothingToDo(t *testing.T) {
	db := openDB(t)
	s, err := mwanachamaagency.SpecFor("agency")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}

	var statements int
	db = db.Session(&gorm.Session{})
	if err := db.Callback().Raw().After("gorm:raw").Register("count_raw", func(*gorm.DB) { statements++ }); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	if err := db.Callback().Query().After("gorm:query").Register("count_query", func(*gorm.DB) { statements++ }); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	for _, name := range []string{"create", "update", "delete"} {
		writes := name
		if err := db.Callback().Create().After("gorm:create").Register("no_"+writes, func(tx *gorm.DB) {
			t.Errorf("a provisioned database was written to by %s", writes)
		}); err != nil {
			t.Fatalf("register callback: %v", err)
		}
	}

	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("second Provision: %v", err)
	}
	if statements > 5 {
		t.Errorf("a second Provision ran %d statements, want a handful", statements)
	}
	t.Logf("a second Provision ran %d statements", statements)
}
