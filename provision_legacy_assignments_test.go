package mwanachamaagency_test

import (
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

func TestProvisionKeepsAnObjectiveLevelRoleAssignmentTableThatRepeatsARoleAndLabel(t *testing.T) {
	db := openDB(t)
	s, err := mwanachamaagency.SpecFor("agency")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	var table string
	for _, o := range s.Objects {
		if o.Role == "role_assignment" {
			table = s.TableFor(o)
		}
	}
	if err := db.Exec(`create table ` + table + ` (
		id text primary key, draft_id text not null default '', objective_id text not null default '',
		role_id text not null default '', label text not null default '',
		created_at text not null default '', last_updated text not null default '', deleted boolean not null default false)`).Error; err != nil {
		t.Fatalf("create the older table: %v", err)
	}
	for _, id := range []string{"a", "b"} {
		if err := db.Exec(`insert into `+table+` (id, draft_id, objective_id, role_id, label) values (?, 'd', ?, 'r', 'accountable')`, id, "o"+id).Error; err != nil {
			t.Fatalf("insert %s: %v", id, err)
		}
	}

	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("Provision over objective-level rows that repeat a role and label: %v", err)
	}

	var n int64
	if err := db.Raw(`select count(*) from ` + table).Scan(&n).Error; err != nil || n != 2 {
		t.Fatalf("rows = %d, err %v, want the 2 older rows kept", n, err)
	}
}
