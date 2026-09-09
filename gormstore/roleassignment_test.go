package gormstore

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestRoleAssignmentRoundTrip(t *testing.T) {
	a := models.RoleAssignment{
		ID:         "ra1",
		RoleID:     "role1",
		WorkItemID: "wi1",
		Label:      models.RACILabelAccountable,
		CreatedAt:  "2026-09-09T00:00:00.000000000Z",
	}

	row := RoleAssignmentToRow(a)
	if row.UpdatedAt == "" {
		t.Fatal("RoleAssignmentToRow: expected UpdatedAt to be stamped")
	}
	if row.Label != "accountable" {
		t.Fatalf("expected label column %q, got %q", "accountable", row.Label)
	}

	got := RoleAssignmentFromRow(row)
	got.LastUpdated = ""
	a.LastUpdated = ""
	if got != a {
		t.Fatalf("round trip mismatch:\n got  %+v\n want %+v", got, a)
	}
}

func TestRoleAssignmentRoundTrip_DraftScoped(t *testing.T) {
	a := models.RoleAssignment{ID: "ra2", DraftID: "d1", RoleID: "role1", WorkItemID: "wi1", Label: models.RACILabelResponsible}

	row := RoleAssignmentToRow(a)
	if row.DraftID == nil || *row.DraftID != "d1" {
		t.Fatalf("expected DraftID column %q, got %v", "d1", row.DraftID)
	}

	got := RoleAssignmentFromRow(row)
	if got.DraftID != "d1" {
		t.Fatalf("expected DraftID %q after round trip, got %q", "d1", got.DraftID)
	}
}

func TestRoleAssignmentRowBeforeCreateMintsID(t *testing.T) {
	row := RoleAssignmentRow{}
	if err := row.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate: %v", err)
	}
	if row.ID == "" {
		t.Fatal("expected BeforeCreate to mint an ID")
	}
}
