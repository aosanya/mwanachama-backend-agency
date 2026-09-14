package mwanachamaagency

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// scanDraftIssues re-derives every Issue for draftID from that Draft's
// current Goal/Objective/RoleAssignment rows. Called inside ListIssues'
// transaction, after the Draft's stale rows have already been deleted.
//
// First batch of checks (AG20): an Objective with no Accountable
// RoleAssignment (warning), and a Goal with no Objectives (info). Add
// further checks here as this list grows — each just appends more
// []models.Issue built the same way, no other plumbing needed.
func scanDraftIssues(ctx context.Context, tx *gorm.DB, tables TableNames, draftID string) ([]models.Issue, error) {
	var goalRows []gormstore.GoalRow
	if err := scopeQuery(tx.WithContext(ctx).Table(tables.Goals), draftID).Find(&goalRows).Error; err != nil {
		return nil, fmt.Errorf("scanDraftIssues: %w", err)
	}
	var objectiveRows []gormstore.ObjectiveRow
	if err := scopeQuery(tx.WithContext(ctx).Table(tables.Objectives), draftID).Find(&objectiveRows).Error; err != nil {
		return nil, fmt.Errorf("scanDraftIssues: %w", err)
	}
	var accountableRows []gormstore.RoleAssignmentRow
	err := scopeQuery(tx.WithContext(ctx).Table(tables.RoleAssignments), draftID).
		Where("label = ?", string(models.RACILabelAccountable)).Find(&accountableRows).Error
	if err != nil {
		return nil, fmt.Errorf("scanDraftIssues: %w", err)
	}

	now := models.NowRFC3339()
	var found []models.Issue

	objectivesWithAccountable := make(map[string]bool, len(accountableRows))
	for _, ar := range accountableRows {
		objectivesWithAccountable[gormstore.RoleAssignmentFromRow(ar).ObjectiveID] = true
	}
	goalsWithObjective := make(map[string]bool, len(objectiveRows))
	for _, or := range objectiveRows {
		o := gormstore.ObjectiveFromRow(or)
		goalsWithObjective[o.GoalID] = true
		if !objectivesWithAccountable[o.ID] {
			found = append(found, models.Issue{
				DraftID:    draftID,
				Severity:   models.IssueSeverityWarning,
				Message:    fmt.Sprintf("Objective %s %q has no Accountable role assigned", o.Code, o.Title),
				EntityType: "objective",
				EntityID:   o.ID,
				CreatedAt:  now,
			})
		}
	}

	for _, gr := range goalRows {
		g := gormstore.GoalFromRow(gr)
		if !goalsWithObjective[g.ID] {
			found = append(found, models.Issue{
				DraftID:    draftID,
				Severity:   models.IssueSeverityInfo,
				Message:    fmt.Sprintf("Goal %s %q has no Objectives yet", g.Code, g.Title),
				EntityType: "goal",
				EntityID:   g.ID,
				CreatedAt:  now,
			})
		}
	}

	return found, nil
}

// ListIssues re-scans draftID before returning: it deletes the Draft's
// existing Issue rows, re-runs [scanDraftIssues] against that Draft's
// current data, re-inserts the result, then reads it back through filter.
// Returns [ErrDraftIDRequired] if draftID is "".
func (m *agencyManager) ListIssues(ctx context.Context, draftID string, filter IssueFilter) ([]models.Issue, error) {
	if draftID == "" {
		return nil, ErrDraftIDRequired
	}

	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table(m.tables.Issues).Where("draft_id = ?", draftID).Delete(&gormstore.IssueRow{}).Error; err != nil {
			return fmt.Errorf("ListIssues: %w", err)
		}
		found, err := scanDraftIssues(ctx, tx, m.tables, draftID)
		if err != nil {
			return err
		}
		for _, issue := range found {
			row := gormstore.IssueToRow(issue)
			if err := tx.Table(m.tables.Issues).Create(&row).Error; err != nil {
				return fmt.Errorf("ListIssues: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	q := m.db.WithContext(ctx).Table(m.tables.Issues).Where("draft_id = ?", draftID).Order("created_at")
	q = likeFilter(q, "message", filter.Message)
	q = eqFilter(q, "severity", filter.Severity)
	q = eqFilter(q, "entity_type", filter.EntityType)
	var rows []gormstore.IssueRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListIssues: %w", err)
	}
	out := make([]models.Issue, len(rows))
	for i, r := range rows {
		out[i] = gormstore.IssueFromRow(r)
	}
	return out, nil
}
