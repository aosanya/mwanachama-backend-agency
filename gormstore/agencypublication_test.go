package gormstore

import (
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
		Status:      "active",
		PublishedAt: "2026-09-09T00:00:00.000000000Z",
	}

	row := AgencyPublicationToRow(p)
	if row.UpdatedAt == "" {
		t.Fatal("AgencyPublicationToRow: expected UpdatedAt to be stamped")
	}

	got := AgencyPublicationFromRow(row)
	got.LastUpdated = ""
	if got != p {
		t.Fatalf("round trip mismatch:\n got  %+v\n want %+v", got, p)
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
