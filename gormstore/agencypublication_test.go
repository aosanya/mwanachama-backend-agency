package gormstore

import (
	"reflect"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAgencyPublicationRoundTrip(t *testing.T) {
	p := models.AgencyPublication{
		ID:          "pub1",
		DraftID:     "d1",
		ContentHash: "deadbeef",
		Version:     1,
		Tag:         "v1",
		Channel:     "production",
		PublishedAt: "2026-09-09T00:00:00.000000000Z",
		Document: models.AgencyDocument{
			Agency: models.DocumentAgency{Name: "SWELab", Enabled: true},
			Goals:  []models.DocumentGoal{{ID: "g1", Code: "G-1", Title: "Ship reliable software"}},
			Roles:  []models.DocumentRole{{ID: "r1", Code: "R-1", Name: "owner"}},
		},
	}

	row := AgencyPublicationToRow(p)
	if row.UpdatedAt == "" {
		t.Fatal("AgencyPublicationToRow: expected UpdatedAt to be stamped")
	}
	if row.Document == "" {
		t.Fatal("AgencyPublicationToRow: expected the document to be marshalled onto the row")
	}

	got := AgencyPublicationFromRow(row)
	got.LastUpdated = ""
	if !reflect.DeepEqual(got, p) {
		t.Fatalf("round trip mismatch:\n got  %+v\n want %+v", got, p)
	}
}

// A publication written before AGD-005 carries no document text at all —
// it must still read back rather than failing to unmarshal.
func TestAgencyPublicationFromRow_NoDocument(t *testing.T) {
	got := AgencyPublicationFromRow(AgencyPublicationRow{ID: "pub1", Version: 1, Tag: "v1"})
	if got.ID != "pub1" || got.Tag != "v1" {
		t.Fatalf("expected the row's own fields to survive, got %+v", got)
	}
	if !reflect.DeepEqual(got.Document, models.AgencyDocument{}) {
		t.Fatalf("expected an empty document, got %+v", got.Document)
	}
}

func TestAgencyPublicationRowBeforeCreateMintsID(t *testing.T) {
	row := AgencyPublicationRow{}
	if err := row.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate: %v", err)
	}
	if row.ID == "" {
		t.Fatal("expected BeforeCreate to mint an ID")
	}
}
