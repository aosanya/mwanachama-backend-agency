package gormstore

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestWorkItemRoundTrip(t *testing.T) {
	w := models.WorkItem{
		ID:          "wi1",
		WorkflowID:  "w1",
		ObjectiveID: "o1",
		Title:       "Draft the department charter",
		Description: "Write the founding charter for the new department.",
		Ordinality:  1,
		Prompt:      "Draft a charter covering scope and reporting lines.",
		CreatedAt:   "2026-09-09T00:00:00.000000000Z",
	}

	row := WorkItemToRow(w)
	if row.UpdatedAt == "" {
		t.Fatal("WorkItemToRow: expected UpdatedAt to be stamped")
	}
	if row.ObjectiveID != "o1" {
		t.Fatalf("expected ObjectiveID column %q, got %q", "o1", row.ObjectiveID)
	}

	got := WorkItemFromRow(row)
	got.LastUpdated = ""
	w.LastUpdated = ""
	if got != w {
		t.Fatalf("round trip mismatch:\n got  %+v\n want %+v", got, w)
	}
}

func TestWorkItemRoundTrip_DraftScoped(t *testing.T) {
	w := models.WorkItem{ID: "wi3", WorkflowID: "w1", ObjectiveID: "o1", DraftID: "d1", Title: "Draft-scoped item"}

	row := WorkItemToRow(w)
	if row.DraftID != "d1" {
		t.Fatalf("expected DraftID column %q, got %q", "d1", row.DraftID)
	}

	got := WorkItemFromRow(row)
	if got.DraftID != "d1" {
		t.Fatalf("expected DraftID %q after round trip, got %q", "d1", got.DraftID)
	}
}

func TestWorkItemRowBeforeCreateMintsID(t *testing.T) {
	row := WorkItemRow{}
	if err := row.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate: %v", err)
	}
	if row.ID == "" {
		t.Fatal("expected BeforeCreate to mint an ID")
	}
}
