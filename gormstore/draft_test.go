package gormstore

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestDraftRoundTrip(t *testing.T) {
	d := models.Draft{
		ID:             "d1",
		Description:    "Q3 restructuring",
		ForkedFromID:   "a1",
		ForkedFromType: "live",
		Status:         models.DraftStatusOpen,
		CreatedAt:      "2026-09-09T00:00:00.000000000Z",
	}

	row := DraftToRow(d)
	if row.UpdatedAt == "" {
		t.Fatal("DraftToRow: expected UpdatedAt to be stamped")
	}
	if row.Status != "open" {
		t.Fatalf("expected status column %q, got %q", "open", row.Status)
	}

	got := DraftFromRow(row)
	got.LastUpdated = ""
	d.LastUpdated = ""
	if got != d {
		t.Fatalf("round trip mismatch:\n got  %+v\n want %+v", got, d)
	}
}

func TestDraftRowBeforeCreateMintsID(t *testing.T) {
	row := DraftRow{}
	if err := row.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate: %v", err)
	}
	if row.ID == "" {
		t.Fatal("expected BeforeCreate to mint an ID")
	}
}
