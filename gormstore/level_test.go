package gormstore

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestLevelRoundTrip(t *testing.T) {
	l := models.Level{
		ID:        "lvl1",
		Name:      "Division",
		Depth:     0,
		CreatedAt: "2026-09-09T00:00:00.000000000Z",
	}

	row := LevelToRow(l)
	if row.UpdatedAt == "" {
		t.Fatal("LevelToRow: expected UpdatedAt to be stamped")
	}

	got := LevelFromRow(row)
	got.LastUpdated = ""
	l.LastUpdated = ""
	if got != l {
		t.Fatalf("round trip mismatch:\n got  %+v\n want %+v", got, l)
	}
}

func TestLevelRowBeforeCreateMintsID(t *testing.T) {
	row := LevelRow{}
	if err := row.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate: %v", err)
	}
	if row.ID == "" {
		t.Fatal("expected BeforeCreate to mint an ID")
	}
}
