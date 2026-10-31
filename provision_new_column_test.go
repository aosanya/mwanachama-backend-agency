package mwanachamaagency_test

import (
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

func TestProvisionAddsAColumnADeclaredObjectGainedAfterItsTableExisted(t *testing.T) {
	db := openDB(t)
	s, err := mwanachamaagency.SpecFor("agency")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	var table string
	for _, o := range s.Objects {
		if o.Role == "role" {
			table = s.TableFor(o)
		}
	}
	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("first provision: %v", err)
	}
	if err := db.Exec("alter table " + table + " drop column permissions").Error; err != nil {
		t.Fatalf("drop column: %v", err)
	}
	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("second provision: %v", err)
	}
	var held []string
	if err := db.Raw("select name from pragma_table_info(?) where name = 'permissions'", table).Scan(&held).Error; err != nil {
		t.Fatal(err)
	}
	if len(held) != 1 {
		t.Fatalf("permissions column not restored on the role table")
	}
}
