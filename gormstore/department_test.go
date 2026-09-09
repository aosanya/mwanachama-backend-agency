package gormstore

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestDepartmentRoundTrip(t *testing.T) {
	d := models.Department{
		ID:        "dep1",
		LevelID:   "lvl1",
		Name:      "Software",
		CreatedAt: "2026-09-09T00:00:00.000000000Z",
	}

	row := DepartmentToRow(d)
	if row.UpdatedAt == "" {
		t.Fatal("DepartmentToRow: expected UpdatedAt to be stamped")
	}
	if row.ParentID != nil {
		t.Fatalf("expected nil ParentID column for a top-level department, got %v", *row.ParentID)
	}

	got := DepartmentFromRow(row)
	got.LastUpdated = ""
	d.LastUpdated = ""
	if got != d {
		t.Fatalf("round trip mismatch:\n got  %+v\n want %+v", got, d)
	}
}

func TestDepartmentRoundTrip_WithParent(t *testing.T) {
	d := models.Department{ID: "dep2", ParentID: "dep1", LevelID: "lvl2", Name: "Backend"}

	row := DepartmentToRow(d)
	if row.ParentID == nil || *row.ParentID != "dep1" {
		t.Fatalf("expected ParentID column %q, got %v", "dep1", row.ParentID)
	}

	got := DepartmentFromRow(row)
	if got.ParentID != "dep1" {
		t.Fatalf("expected ParentID %q after round trip, got %q", "dep1", got.ParentID)
	}
}

func TestDepartmentRowBeforeCreateMintsID(t *testing.T) {
	row := DepartmentRow{}
	if err := row.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate: %v", err)
	}
	if row.ID == "" {
		t.Fatal("expected BeforeCreate to mint an ID")
	}
}
