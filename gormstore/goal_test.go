package gormstore

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestGoalRoundTrip(t *testing.T) {
	g := models.Goal{
		ID:          "g1",
		Title:       "Grow adoption",
		Description: "Get every department onto the new Agency model.",
		Ordinality:  1,
		CreatedAt:   "2026-09-09T00:00:00.000000000Z",
	}

	row := GoalToRow(g)
	if row.UpdatedAt == "" {
		t.Fatal("GoalToRow: expected UpdatedAt to be stamped")
	}

	got := GoalFromRow(row)
	got.LastUpdated = ""
	g.LastUpdated = ""
	if got != g {
		t.Fatalf("round trip mismatch:\n got  %+v\n want %+v", got, g)
	}
}

func TestGoalRoundTrip_DraftScoped(t *testing.T) {
	g := models.Goal{ID: "g2", DraftID: "d1", Title: "Draft-scoped goal"}

	row := GoalToRow(g)
	if row.DraftID != "d1" {
		t.Fatalf("expected DraftID column %q, got %q", "d1", row.DraftID)
	}

	got := GoalFromRow(row)
	if got.DraftID != "d1" {
		t.Fatalf("expected DraftID %q after round trip, got %q", "d1", got.DraftID)
	}
}

func TestGoalRowBeforeCreateMintsID(t *testing.T) {
	row := GoalRow{}
	if err := row.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate: %v", err)
	}
	if row.ID == "" {
		t.Fatal("expected BeforeCreate to mint an ID")
	}
}
