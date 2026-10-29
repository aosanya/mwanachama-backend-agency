package mwanachamaagency_test

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-shared/spec"
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
	if err := db.Raw(`select name from sqlite_master where type='table' and name not like 'sqlite_%'
		and name not like '%_' || ?`, spec.NameRegistrySuffix).
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

	want := declaredTables(s)
	if got := liveTables(t, db); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("tables =\n  %v\nwant\n  %v", got, want)
	}
	if len(want) != 20 {
		t.Errorf("declared tables = %d, want the 20 objects the spec declares", len(want))
	}
}

func declaredTables(s *spec.Spec) []string {
	out := make([]string, 0, len(s.Objects))
	for _, o := range s.Objects {
		out = append(out, s.TableFor(o))
	}
	sort.Strings(out)
	return out
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
	if n := len(liveTables(t, db)); n != 20 {
		t.Errorf("tables = %d, want 20", n)
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

func TestAG51_OpenBug_ProvisionReseedsAfterAllDraftsAreArchived(t *testing.T) {
	db := openDB(t)
	s, err := mwanachamaagency.SpecFor("agency")
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
	ctx := context.Background()

	drafts, err := mgr.ListDrafts(ctx, mwanachamaagency.DraftFilter{})
	if err != nil {
		t.Fatalf("ListDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("drafts after the first Provision = %d, want exactly the one bootstrap draft", len(drafts))
	}
	if _, err := mgr.ArchiveDraft(ctx, drafts[0].ID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}

	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("second Provision: %v", err)
	}

	after, err := mgr.ListDrafts(ctx, mwanachamaagency.DraftFilter{})
	if err != nil {
		t.Fatalf("ListDrafts after: %v", err)
	}
	if len(after) != 1 {
		t.Fatalf("AG51 open bug: an operator who archives their only Draft (a legitimate move under AGD-006 — any draft may be archived) and whose process later restarts sees Provision mint a second, spurious \"Initial draft\" — drafts after = %d, want the archived one left alone with no new draft created", len(after))
	}
}

func TestASecondMountTakesItsOwnTables(t *testing.T) {
	db := openDB(t)

	main, err := mwanachamaagency.SpecFor("agency")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaagency.Provision(db, main); err != nil {
		t.Fatalf("provision the default mount: %v", err)
	}

	goals, _ := main.ByRole("goal")
	mainGoals := main.TableFor(goals)
	if err := db.Exec(`insert into ` + mainGoals + `
		(id, code, draft_id, title, description, ordinality, created_at, last_updated, deleted)
		values ('goal-1', 'G-1', 'draft-1', 'Belongs to the default mount', '', 1,
		        '2026-09-29T00:00:00.000000000Z', '2026-09-29T00:00:00.000000000Z', 0)`).Error; err != nil {
		t.Fatalf("seed the default mount: %v", err)
	}

	second, err := mwanachamaagency.SpecForMount("agency", "supplier")
	if err != nil {
		t.Fatalf("SpecForMount: %v", err)
	}
	if err := mwanachamaagency.Provision(db, second); err != nil {
		t.Fatalf("provision the second mount: %v", err)
	}

	secondGoals := second.TableFor(mustRole(t, second, "goal"))
	if secondGoals == mainGoals {
		t.Fatalf("both mounts hash to %q", mainGoals)
	}

	tables := liveTables(t, db)
	if !contains(tables, mainGoals) || !contains(tables, secondGoals) {
		t.Fatalf("tables = %v, want both mounts' own goal tables", tables)
	}

	var n int64
	if err := db.Table(secondGoals).Count(&n).Error; err != nil {
		t.Fatalf("count the second mount: %v", err)
	}
	if n != 0 {
		t.Errorf("the second mount holds %d row(s), want an empty table of its own", n)
	}

	var title string
	if err := db.Table(mainGoals).Select("title").Where("id = ?", "goal-1").Scan(&title).Error; err != nil {
		t.Fatalf("read the default mount: %v", err)
	}
	if title != "Belongs to the default mount" {
		t.Errorf("title = %q, want the default mount's row untouched", title)
	}
}

func mustRole(t *testing.T, s *spec.Spec, role string) spec.Object {
	t.Helper()
	o, ok := s.ByRole(role)
	if !ok {
		t.Fatalf("the spec fills no %q", role)
	}
	return o
}
