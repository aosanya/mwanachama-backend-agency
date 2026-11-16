package mwanachamaagency_test

import (
	"strings"
	"testing"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

const sendKeyIndexName = "send_key_unique"

func recordRowTable(t *testing.T, s *spec.Spec) string {
	t.Helper()
	return tableOf(t, s, "record_row")
}

func indexNamesOn(t *testing.T, db *gorm.DB, table string) []string {
	t.Helper()
	var names []string
	if err := db.Raw(`select name from sqlite_master where type = 'index' and tbl_name = ?`, table).Scan(&names).Error; err != nil {
		t.Fatalf("read indexes: %v", err)
	}
	return names
}

func holdsIndex(names []string, fragment string) bool {
	for _, name := range names {
		if strings.Contains(name, fragment) {
			return true
		}
	}
	return false
}

func provisionedWithoutSendKeyIndex(t *testing.T) (*gorm.DB, *spec.Spec, string) {
	t.Helper()
	db := openDB(t)
	s, err := mwanachamaagency.SpecFor("agency")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("first provision: %v", err)
	}
	table := recordRowTable(t, s)
	for _, name := range indexNamesOn(t, db, table) {
		if strings.Contains(name, sendKeyIndexName) {
			if err := db.Exec("drop index " + name).Error; err != nil {
				t.Fatalf("drop index: %v", err)
			}
		}
	}
	if holdsIndex(indexNamesOn(t, db, table), sendKeyIndexName) {
		t.Fatalf("the send key index is still there after the drop")
	}
	return db, s, table
}

func TestProvision_GivesBackTheSendKeyIndexAnExistingTableLacks(t *testing.T) {
	db, s, table := provisionedWithoutSendKeyIndex(t)
	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("second provision: %v", err)
	}
	if !holdsIndex(indexNamesOn(t, db, table), sendKeyIndexName) {
		t.Fatalf("second provision left the table without its send key index")
	}
}

func insertRowWithSendKey(t *testing.T, db *gorm.DB, table, id, key string) {
	t.Helper()
	var required []string
	if err := db.Raw(`select name from pragma_table_info(?) where "notnull" = 1 and dflt_value is null and pk = 0`, table).Scan(&required).Error; err != nil {
		t.Fatalf("read required columns: %v", err)
	}
	columns := append([]string{"id", "send_key"}, required...)
	values := []any{id, key}
	for range required {
		values = append(values, "x")
	}
	marks := strings.TrimSuffix(strings.Repeat("?, ", len(columns)), ", ")
	if err := db.Exec("insert into "+table+" ("+strings.Join(columns, ", ")+") values ("+marks+")", values...).Error; err != nil {
		t.Fatalf("insert %s: %v", id, err)
	}
}

func TestProvision_AnExistingTableThatCannotTakeTheSendKeyIndexIsRefusedEveryTime(t *testing.T) {
	db, s, table := provisionedWithoutSendKeyIndex(t)
	for _, id := range []string{"row-1", "row-2"} {
		insertRowWithSendKey(t, db, table, id, "step-1:run-1")
	}
	for attempt := 1; attempt <= 2; attempt++ {
		if err := mwanachamaagency.Provision(db, s); err == nil {
			t.Fatalf("provision %d answered nil with duplicate send keys and no unique index", attempt)
		}
	}
}

func TestProvision_ASecondProvisionOfAnUpToDateDatabaseRunsNoMigration(t *testing.T) {
	db := openDB(t)
	s, err := mwanachamaagency.SpecFor("agency")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("first provision: %v", err)
	}
	var created []string
	err = db.Callback().Raw().Before("gorm:raw").Register("count_ddl", func(tx *gorm.DB) {
		if sql := strings.ToLower(tx.Statement.SQL.String()); strings.HasPrefix(sql, "create ") {
			created = append(created, sql)
		}
	})
	if err != nil {
		t.Fatalf("register callback: %v", err)
	}
	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("second provision: %v", err)
	}
	if len(created) != 0 {
		t.Fatalf("second provision ran %d create statements, first: %s", len(created), created[0])
	}
}
