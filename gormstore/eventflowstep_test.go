package gormstore

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestEventFlowStepRoundTrip(t *testing.T) {
	s := models.EventFlowStep{
		ID:           "s1",
		Code:         "planning:1.1",
		WorkflowCode: "planning",
		Step:         "1.1",
		Name:         "Start planning",
		StepType:     "start",
		Consumer:     "codevaldwork",
		HandlerCode:  "wp-planning-start",
		EmitsTopics:  "planning.started",
		Ordinality:   1,
		CreatedAt:    "2026-09-09T00:00:00.000000000Z",
	}

	row := EventFlowStepToRow(s)
	if row.UpdatedAt == "" {
		t.Fatal("EventFlowStepToRow: expected UpdatedAt to be stamped")
	}

	got := EventFlowStepFromRow(row)
	got.LastUpdated = ""
	s.LastUpdated = ""
	if got != s {
		t.Fatalf("round trip mismatch:\n got  %+v\n want %+v", got, s)
	}
}

func TestEventFlowStepRoundTrip_DraftScoped(t *testing.T) {
	s := models.EventFlowStep{ID: "s2", DraftID: "d1", Code: "planning:1.2"}

	row := EventFlowStepToRow(s)
	if row.DraftID == nil || *row.DraftID != "d1" {
		t.Fatalf("expected DraftID column %q, got %v", "d1", row.DraftID)
	}

	got := EventFlowStepFromRow(row)
	if got.DraftID != "d1" {
		t.Fatalf("expected DraftID %q after round trip, got %q", "d1", got.DraftID)
	}
}

func TestEventFlowStepRowBeforeCreateMintsID(t *testing.T) {
	row := EventFlowStepRow{}
	if err := row.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate: %v", err)
	}
	if row.ID == "" {
		t.Fatal("expected BeforeCreate to mint an ID")
	}
}
