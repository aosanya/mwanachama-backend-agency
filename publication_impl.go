package mwanachamaagency

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// liveContentHash SHA-256-hashes every live Goal/Objective/Workflow/
// WorkItem/EventFlowStep/Role/RoleAssignment/Value/Rule plus the Agency's
// name: one JSON record per row (fields other than timestamps), sorted
// lexically so row order never affects the hash, joined with "\n". A
// change to any of those fields changes the hash; nothing else does.
// Mission/Vision dropped from the Agency record 2026-09-10 (AGD-001).
func (m *agencyManager) liveContentHash(ctx context.Context, tx *gorm.DB) (string, error) {
	var records []string

	var goalRows []gormstore.GoalRow
	if err := tx.WithContext(ctx).Table(m.tables.Goals).Where("draft_id IS NULL").Find(&goalRows).Error; err != nil {
		return "", fmt.Errorf("liveContentHash: %w", err)
	}
	for _, r := range goalRows {
		records = append(records, hashRecord("Goal", map[string]any{
			"id": r.ID, "title": r.Title, "description": r.Description,
			"ordinality": r.Ordinality, "deleted": r.Deleted,
		}))
	}

	var objectiveRows []gormstore.ObjectiveRow
	if err := tx.WithContext(ctx).Table(m.tables.Objectives).Where("draft_id IS NULL").Find(&objectiveRows).Error; err != nil {
		return "", fmt.Errorf("liveContentHash: %w", err)
	}
	for _, r := range objectiveRows {
		records = append(records, hashRecord("Objective", map[string]any{
			"id": r.ID, "goal_id": r.GoalID, "title": r.Title, "description": r.Description,
			"ordinality": r.Ordinality, "deleted": r.Deleted,
		}))
	}

	var workflowRows []gormstore.WorkflowRow
	if err := tx.WithContext(ctx).Table(m.tables.Workflows).Where("draft_id IS NULL").Find(&workflowRows).Error; err != nil {
		return "", fmt.Errorf("liveContentHash: %w", err)
	}
	for _, r := range workflowRows {
		records = append(records, hashRecord("Workflow", map[string]any{
			"id": r.ID, "name": r.Name, "description": r.Description,
			"ordinality": r.Ordinality, "event_flows": r.EventFlows, "deleted": r.Deleted,
		}))
	}

	var workItemRows []gormstore.WorkItemRow
	if err := tx.WithContext(ctx).Table(m.tables.WorkItems).Where("draft_id IS NULL").Find(&workItemRows).Error; err != nil {
		return "", fmt.Errorf("liveContentHash: %w", err)
	}
	for _, r := range workItemRows {
		records = append(records, hashRecord("WorkItem", map[string]any{
			"id": r.ID, "workflow_id": r.WorkflowID, "objective_id": r.ObjectiveID,
			"title": r.Title, "description": r.Description, "ordinality": r.Ordinality,
			"prompt": r.Prompt, "deleted": r.Deleted,
		}))
	}

	var stepRows []gormstore.EventFlowStepRow
	if err := tx.WithContext(ctx).Table(m.tables.EventFlowSteps).Where("draft_id IS NULL").Find(&stepRows).Error; err != nil {
		return "", fmt.Errorf("liveContentHash: %w", err)
	}
	for _, r := range stepRows {
		records = append(records, hashRecord("EventFlowStep", map[string]any{
			"id": r.ID, "code": r.Code, "workflow_code": r.WorkflowCode, "step": r.Step,
			"name": r.Name, "description": r.Description, "step_type": r.StepType,
			"trigger_topic": r.TriggerTopic, "trigger_publisher": r.TriggerPublisher,
			"consumer": r.Consumer, "handler_code": r.HandlerCode,
			"emits_topics": r.EmitsTopics, "on_error_emits_topics": r.OnErrorEmitsTopics,
			"ordinality": r.Ordinality, "deleted": r.Deleted,
		}))
	}

	var roleRows []gormstore.RoleRow
	if err := tx.WithContext(ctx).Table(m.tables.Roles).Where("draft_id IS NULL").Find(&roleRows).Error; err != nil {
		return "", fmt.Errorf("liveContentHash: %w", err)
	}
	for _, r := range roleRows {
		records = append(records, hashRecord("Role", map[string]any{
			"id": r.ID, "name": r.Name, "description": r.Description, "deleted": r.Deleted,
		}))
	}

	var assignmentRows []gormstore.RoleAssignmentRow
	if err := tx.WithContext(ctx).Table(m.tables.RoleAssignments).Where("draft_id IS NULL").Find(&assignmentRows).Error; err != nil {
		return "", fmt.Errorf("liveContentHash: %w", err)
	}
	for _, r := range assignmentRows {
		records = append(records, hashRecord("RoleAssignment", map[string]any{
			"id": r.ID, "role_id": r.RoleID, "objective_id": r.ObjectiveID,
			"label": r.Label, "deleted": r.Deleted,
		}))
	}

	var valueRows []gormstore.ValueRow
	if err := tx.WithContext(ctx).Table(m.tables.Values).Where("draft_id IS NULL").Find(&valueRows).Error; err != nil {
		return "", fmt.Errorf("liveContentHash: %w", err)
	}
	for _, r := range valueRows {
		records = append(records, hashRecord("Value", map[string]any{
			"id": r.ID, "objective_id": r.ObjectiveID, "text": r.Text, "deleted": r.Deleted,
		}))
	}

	var ruleRows []gormstore.RuleRow
	if err := tx.WithContext(ctx).Table(m.tables.Rules).Where("draft_id IS NULL").Find(&ruleRows).Error; err != nil {
		return "", fmt.Errorf("liveContentHash: %w", err)
	}
	for _, r := range ruleRows {
		records = append(records, hashRecord("Rule", map[string]any{
			"id": r.ID, "objective_id": r.ObjectiveID, "text": r.Text, "deleted": r.Deleted,
		}))
	}

	agencyRow, err := m.findAgencyRow(ctx, tx)
	if err != nil {
		return "", err
	}
	records = append(records, hashRecord("Agency", map[string]any{
		"name": agencyRow.Name,
	}))

	sort.Strings(records)
	sum := sha256.Sum256([]byte(strings.Join(records, "\n")))
	return hex.EncodeToString(sum[:]), nil
}

func hashRecord(typeName string, fields map[string]any) string {
	b, _ := json.Marshal(map[string]any{"type": typeName, "props": fields})
	return string(b)
}

// PublishAgency versions and content-hashes whatever is currently live. If
// draftID is non-"", that Draft must already be promoted
// (PublishAgency does not redo PromoteDraft's copy-and-swap).
func (m *agencyManager) PublishAgency(ctx context.Context, draftID string) (models.AgencyPublication, error) {
	var out models.AgencyPublication
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if draftID != "" {
			draftRow, err := m.findDraftRow(ctx, tx, draftID)
			if err != nil {
				return err
			}
			if gormstore.DraftFromRow(draftRow).Status != models.DraftStatusPromoted {
				return ErrDraftNotPromoted
			}
		}

		hash, err := m.liveContentHash(ctx, tx)
		if err != nil {
			return err
		}

		var existing []gormstore.AgencyPublicationRow
		if err := tx.Table(m.tables.AgencyPublications).Find(&existing).Error; err != nil {
			return fmt.Errorf("PublishAgency: %w", err)
		}
		maxVersion := 0
		for _, p := range existing {
			if p.ContentHash == hash {
				return ErrNoChangesDetected
			}
			if p.Version > maxVersion {
				maxVersion = p.Version
			}
		}

		now := models.NowRFC3339()
		if err := tx.Table(m.tables.AgencyPublications).Where("status = ?", "active").
			Updates(map[string]any{"status": "archived", "updated_at": now}).Error; err != nil {
			return fmt.Errorf("PublishAgency: %w", err)
		}

		version := maxVersion + 1
		row := gormstore.AgencyPublicationToRow(models.AgencyPublication{
			DraftID:     draftID,
			ContentHash: hash,
			Version:     version,
			Tag:         fmt.Sprintf("v%d", version),
			Status:      "active",
			PublishedAt: now,
		})
		if err := tx.Table(m.tables.AgencyPublications).Create(&row).Error; err != nil {
			return fmt.Errorf("PublishAgency: %w", err)
		}
		out = gormstore.AgencyPublicationFromRow(row)
		return nil
	})
	if err != nil {
		return models.AgencyPublication{}, err
	}
	return out, nil
}

func (m *agencyManager) findPublicationRow(ctx context.Context, tx *gorm.DB, version int) (gormstore.AgencyPublicationRow, error) {
	var row gormstore.AgencyPublicationRow
	err := tx.WithContext(ctx).Table(m.tables.AgencyPublications).Where("version = ?", version).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.AgencyPublicationRow{}, ErrPublicationNotFound
	}
	if err != nil {
		return gormstore.AgencyPublicationRow{}, fmt.Errorf("findPublicationRow: %w", err)
	}
	return row, nil
}

func (m *agencyManager) GetPublication(ctx context.Context, version int) (models.AgencyPublication, error) {
	row, err := m.findPublicationRow(ctx, m.db, version)
	if err != nil {
		return models.AgencyPublication{}, err
	}
	return gormstore.AgencyPublicationFromRow(row), nil
}

func (m *agencyManager) ListPublications(ctx context.Context) ([]models.AgencyPublication, error) {
	var rows []gormstore.AgencyPublicationRow
	if err := m.db.WithContext(ctx).Table(m.tables.AgencyPublications).Order("version").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListPublications: %w", err)
	}
	out := make([]models.AgencyPublication, len(rows))
	for i, r := range rows {
		out[i] = gormstore.AgencyPublicationFromRow(r)
	}
	return out, nil
}

// UpdatePublicationStatus only allows the active -> archived transition.
func (m *agencyManager) UpdatePublicationStatus(ctx context.Context, version int, status string) (models.AgencyPublication, error) {
	row, err := m.findPublicationRow(ctx, m.db, version)
	if err != nil {
		return models.AgencyPublication{}, err
	}
	if row.Status != "active" || status != "archived" {
		return models.AgencyPublication{}, ErrInvalidPublicationStatus
	}
	now := models.NowRFC3339()
	if err := m.db.WithContext(ctx).Table(m.tables.AgencyPublications).Where("version = ?", version).
		Updates(map[string]any{"status": status, "updated_at": now}).Error; err != nil {
		return models.AgencyPublication{}, fmt.Errorf("UpdatePublicationStatus: %w", err)
	}
	row.Status, row.UpdatedAt = status, now
	return gormstore.AgencyPublicationFromRow(row), nil
}
