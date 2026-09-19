package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IssueRow is the GORM row for a [models.Issue]. DraftID always names a
// real Draft (AGD-006 — there is no live row-scope; a scan always runs
// against a specific Draft, AGD-004).
type IssueRow struct {
	ID         string `gorm:"primaryKey"`
	DraftID    string `gorm:"index;not null"`
	Severity   string
	Message    string
	EntityType string
	EntityID   string
	CreatedAt  string
}

func (r *IssueRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// IssueToRow converts a domain Issue to its row shape.
func IssueToRow(i models.Issue) IssueRow {
	return IssueRow{
		ID:         i.ID,
		DraftID:    i.DraftID,
		Severity:   string(i.Severity),
		Message:    i.Message,
		EntityType: i.EntityType,
		EntityID:   i.EntityID,
		CreatedAt:  i.CreatedAt,
	}
}

// IssueFromRow converts a row back to the domain Issue.
func IssueFromRow(r IssueRow) models.Issue {
	return models.Issue{
		ID:         r.ID,
		DraftID:    r.DraftID,
		Severity:   models.IssueSeverity(r.Severity),
		Message:    r.Message,
		EntityType: r.EntityType,
		EntityID:   r.EntityID,
		CreatedAt:  r.CreatedAt,
	}
}
