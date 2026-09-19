package gormstore

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestRoleRoundTrip(t *testing.T) {
	r := models.Role{
		ID:          "role1",
		DraftID:     "d1",
		Name:        "bug-hunter",
		Description: "Finds and reports bugs.",
		CreatedAt:   "2026-09-09T00:00:00.000000000Z",
	}

	row := RoleToRow(r)
	if row.UpdatedAt == "" {
		t.Fatal("RoleToRow: expected UpdatedAt to be stamped")
	}
	if row.DraftID != "d1" {
		t.Fatalf("expected DraftID column %q, got %q", "d1", row.DraftID)
	}

	got := RoleFromRow(row)
	got.LastUpdated = ""
	r.LastUpdated = ""
	if got != r {
		t.Fatalf("round trip mismatch:\n got  %+v\n want %+v", got, r)
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
