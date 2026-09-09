package gormstore

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestRoleRoundTrip(t *testing.T) {
	r := models.Role{
		ID:          "role1",
		Name:        "bug-hunter",
		Description: "Finds and reports bugs.",
		CreatedAt:   "2026-09-09T00:00:00.000000000Z",
	}

	row := RoleToRow(r)
	if row.UpdatedAt == "" {
		t.Fatal("RoleToRow: expected UpdatedAt to be stamped")
	}
	if row.DraftID != nil {
		t.Fatalf("expected nil DraftID column for a live role, got %v", *row.DraftID)
	}

	got := RoleFromRow(row)
	got.LastUpdated = ""
	r.LastUpdated = ""
	if got != r {
		t.Fatalf("round trip mismatch:\n got  %+v\n want %+v", got, r)
	}
}

func TestRoleRoundTrip_DraftScoped(t *testing.T) {
	r := models.Role{ID: "role2", DraftID: "d1", Name: "code-reviewer"}

	row := RoleToRow(r)
	if row.DraftID == nil || *row.DraftID != "d1" {
		t.Fatalf("expected DraftID column %q, got %v", "d1", row.DraftID)
	}

	got := RoleFromRow(row)
	if got.DraftID != "d1" {
		t.Fatalf("expected DraftID %q after round trip, got %q", "d1", got.DraftID)
	}
}

func TestRoleRowBeforeCreateMintsID(t *testing.T) {
	row := RoleRow{}
	if err := row.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate: %v", err)
	}
	if row.ID == "" {
		t.Fatal("expected BeforeCreate to mint an ID")
	}
}
