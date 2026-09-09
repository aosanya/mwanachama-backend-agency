package gormstore

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAgencySnapshotRoundTrip(t *testing.T) {
	s := models.AgencySnapshot{ID: "snap1", DraftID: "d1", SnapshotAt: "2026-09-09T00:00:00.000000000Z"}

	row := AgencySnapshotToRow(s)
	got := AgencySnapshotFromRow(row)
	if got != s {
		t.Fatalf("round trip mismatch:\n got  %+v\n want %+v", got, s)
	}
}

func TestAgencySnapshotRowBeforeCreateMintsID(t *testing.T) {
	row := AgencySnapshotRow{}
	if err := row.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate: %v", err)
	}
	if row.ID == "" {
		t.Fatal("expected BeforeCreate to mint an ID")
	}
}
