package gormstore

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestWorkflowRoundTrip(t *testing.T) {
	w := models.Workflow{
		ID:          "w1",
		Name:        "Onboard a new department",
		Description: "Steps to stand up a department under this Agency.",
		Ordinality:  1,
		EventFlows:  `{"name":"onboard","steps":[]}`,
		CreatedAt:   "2026-09-09T00:00:00.000000000Z",
	}

	row := WorkflowToRow(w)
	if row.UpdatedAt == "" {
		t.Fatal("WorkflowToRow: expected UpdatedAt to be stamped")
	}

	got := WorkflowFromRow(row)
	got.LastUpdated = ""
	w.LastUpdated = ""
	if got != w {
		t.Fatalf("round trip mismatch:\n got  %+v\n want %+v", got, w)
	}
}

func TestWorkflowRoundTrip_DraftScoped(t *testing.T) {
	w := models.Workflow{ID: "w2", DraftID: "d1", Name: "Draft-scoped workflow"}

	row := WorkflowToRow(w)
	if row.DraftID != "d1" {
		t.Fatalf("expected DraftID column %q, got %q", "d1", row.DraftID)
	}

	got := WorkflowFromRow(row)
	if got.DraftID != "d1" {
		t.Fatalf("expected DraftID %q after round trip, got %q", "d1", got.DraftID)
	}
}

func TestWorkflowRowBeforeCreateMintsID(t *testing.T) {
	row := WorkflowRow{}
	if err := row.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate: %v", err)
	}
	if row.ID == "" {
		t.Fatal("expected BeforeCreate to mint an ID")
	}
}
