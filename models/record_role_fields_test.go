package models

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestRecordRoleFields_ADocumentThatCarriesThemStillReadsAndNamesWhatItCarried(t *testing.T) {
	raw := `{"name":"Tender","role_id":"r1","edit_role_id":"r2","viewer_role_id":"","delete_role_id":"r3"}`
	var record Record
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		t.Fatalf("a record that carries the old fields does not read: %v", err)
	}
	var document DocumentRecord
	if err := json.Unmarshal([]byte(raw), &document); err != nil {
		t.Fatalf("a document record that carries the old fields does not read: %v", err)
	}
	want := []string{RecordFieldRoleID, RecordFieldEditRoleID, RecordFieldDeleteRoleID}
	if !reflect.DeepEqual(record.CarriedGoneRoleFields(), want) || !reflect.DeepEqual(document.CarriedGoneRoleFields(), want) {
		t.Errorf("carried = %v and %v, want %v", record.CarriedGoneRoleFields(), document.CarriedGoneRoleFields(), want)
	}
}

func TestRecordRoleFields_NothingWritesThemBack(t *testing.T) {
	var record Record
	if err := json.Unmarshal([]byte(`{"name":"Tender","role_id":"r1"}`), &record); err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(out, &fields); err != nil {
		t.Fatal(err)
	}
	for _, name := range GoneRecordRoleFields() {
		if _, found := fields[name]; found {
			t.Errorf("a record writes %s back", name)
		}
	}
}
