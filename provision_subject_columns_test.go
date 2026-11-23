package mwanachamaagency_test

import (
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

func TestProvisionAddsTheSubjectColumnsToTablesThatPredateThem(t *testing.T) {
	for _, c := range []struct{ role, column string }{
		{"workflow", "subject_record"},
		{"record_row", "subject_row_id"},
	} {
		t.Run(c.role+"."+c.column, func(t *testing.T) {
			_, db, s := newTestManagerDB(t)
			table := tableOf(t, s, c.role)
			if err := db.Exec("alter table " + table + " drop column " + c.column).Error; err != nil {
				t.Fatalf("drop column: %v", err)
			}
			if err := mwanachamaagency.Provision(db, s); err != nil {
				t.Fatalf("Provision: %v", err)
			}
			var held []string
			if err := db.Raw("select name from pragma_table_info(?) where name = ?", table, c.column).Scan(&held).Error; err != nil {
				t.Fatal(err)
			}
			if len(held) != 1 {
				t.Fatalf("%s was not restored on the %s table", c.column, c.role)
			}
		})
	}
}
