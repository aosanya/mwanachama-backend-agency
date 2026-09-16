package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IssueRow is the GORM row for a [models.Issue]. DraftID is nullable
// (NULL for live), the same convention as GoalRow's — the scan runs
// against the live Agency as well as against a Draft (AGD-004).
type IssueRow struct {
	ID         string  `gorm:"primaryKey"`
	DraftID    *string `gorm:"index"`
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
		DraftID:    StringToNullable(i.DraftID),
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
		DraftID:    nullableToString(r.DraftID),
		Severity:   models.IssueSeverity(r.Severity),
		Message:    r.Message,
		EntityType: r.EntityType,
		EntityID:   r.EntityID,
		CreatedAt:  r.CreatedAt,
	}
}
