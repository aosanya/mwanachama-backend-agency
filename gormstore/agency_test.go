package gormstore

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAgencyRoundTrip(t *testing.T) {
	a := models.Agency{
		ID:                           "a1",
		Name:                         "NeoLabHQ",
		Enabled:                      true,
		DefaultFailurePipelineBudget: 3,
		InactivityTimeoutSeconds:     600,
		EventFlows:                   `{"steps":[]}`,
		CreatedAt:                    "2026-09-09T00:00:00.000000000Z",
		Deleted:                      false,
	}

	row := AgencyToRow(a)
	if row.UpdatedAt == "" {
		t.Fatal("AgencyToRow: expected UpdatedAt to be stamped")
	}

	got := AgencyFromRow(row)
	got.LastUpdated = "" // stamped value, not part of the round-trip comparison
	a.LastUpdated = ""
	if got != a {
		t.Fatalf("round trip mismatch:\n got  %+v\n want %+v", got, a)
	}
}

func TestAgencyRowBeforeCreateMintsID(t *testing.T) {
	row := AgencyRow{}
	if err := row.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate: %v", err)
	}
	if row.ID == "" {
		t.Fatal("expected BeforeCreate to mint an ID")
	}
}
