package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// scopeQuery filters a Goal/Workflow/WorkItem/EventFlowStep query to live
// rows (draftID == "") or one Draft's copies (draftID != "").
func scopeQuery(q *gorm.DB, draftID string) *gorm.DB {
	if draftID == "" {
		return q.Where("draft_id IS NULL")
	}
	return q.Where("draft_id = ?", draftID)
}

func (m *agencyManager) findDraftRow(ctx context.Context, tx *gorm.DB, id string) (gormstore.DraftRow, error) {
	var row gormstore.DraftRow
	err := tx.WithContext(ctx).Table(m.tables.Drafts).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.DraftRow{}, ErrDraftNotFound
	}
	if err != nil {
		return gormstore.DraftRow{}, fmt.Errorf("findDraftRow: %w", err)
	}
	return row, nil
}

// CreateDraft forks the sub-graph scoped to forkedFromID (an Agency.ID when
// forkedFromType=="live", or another open Draft's ID when
// forkedFromType=="draft") into brand-new Goal/Workflow/WorkItem/
// EventFlowStep rows scoped to the new Draft. WorkItem's WorkflowID/GoalID
// are remapped to the copies' new ids; EventFlowStep needs no remap (it
// joins Workflow by the WorkflowCode business key, not a row id).
func (m *agencyManager) CreateDraft(ctx context.Context, description, forkedFromID, forkedFromType string) (models.Draft, error) {
	var out models.Draft
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := m.findAgencyRow(ctx, tx); err != nil {
			return err
		}

		sourceScope := "" // "" forks from live
		if forkedFromType == "draft" {
			srcRow, err := m.findDraftRow(ctx, tx, forkedFromID)
			if err != nil {
				return err
			}
			if gormstore.DraftFromRow(srcRow).Status != models.DraftStatusOpen {
				return ErrDraftNotOpen
			}
			sourceScope = forkedFromID
		}

		now := models.NowRFC3339()
		draftRow := gormstore.DraftToRow(models.Draft{
			Description:    description,
			ForkedFromID:   forkedFromID,
			ForkedFromType: forkedFromType,
			Status:         models.DraftStatusOpen,
			CreatedAt:      now,
		})
		if err := tx.Table(m.tables.Drafts).Create(&draftRow).Error; err != nil {
			return fmt.Errorf("CreateDraft: %w", err)
		}
		newDraftID := draftRow.ID

		var goalRows []gormstore.GoalRow
		if err := scopeQuery(tx.Table(m.tables.Goals), sourceScope).Find(&goalRows).Error; err != nil {
			return fmt.Errorf("CreateDraft: %w", err)
		}
		goalIDMap := make(map[string]string, len(goalRows))
		for _, gr := range goalRows {
			g := gormstore.GoalFromRow(gr)
			oldID := g.ID
			g.ID, g.DraftID, g.CreatedAt = "", newDraftID, now
			newRow := gormstore.GoalToRow(g)
			if err := tx.Table(m.tables.Goals).Create(&newRow).Error; err != nil {
				return fmt.Errorf("CreateDraft: %w", err)
			}
			goalIDMap[oldID] = newRow.ID
		}

		var workflowRows []gormstore.WorkflowRow
		if err := scopeQuery(tx.Table(m.tables.Workflows), sourceScope).Find(&workflowRows).Error; err != nil {
			return fmt.Errorf("CreateDraft: %w", err)
		}
		workflowIDMap := make(map[string]string, len(workflowRows))
		for _, wr := range workflowRows {
			w := gormstore.WorkflowFromRow(wr)
			oldID := w.ID
			w.ID, w.DraftID, w.CreatedAt = "", newDraftID, now
			newRow := gormstore.WorkflowToRow(w)
			if err := tx.Table(m.tables.Workflows).Create(&newRow).Error; err != nil {
				return fmt.Errorf("CreateDraft: %w", err)
			}
			workflowIDMap[oldID] = newRow.ID
		}

		var workItemRows []gormstore.WorkItemRow
		if err := scopeQuery(tx.Table(m.tables.WorkItems), sourceScope).Find(&workItemRows).Error; err != nil {
			return fmt.Errorf("CreateDraft: %w", err)
		}
		for _, wir := range workItemRows {
			wi := gormstore.WorkItemFromRow(wir)
			wi.ID, wi.DraftID, wi.CreatedAt = "", newDraftID, now
			if mapped, ok := workflowIDMap[wi.WorkflowID]; ok {
				wi.WorkflowID = mapped
			}
			if wi.GoalID != "" {
				if mapped, ok := goalIDMap[wi.GoalID]; ok {
					wi.GoalID = mapped
				}
			}
			newRow := gormstore.WorkItemToRow(wi)
			if err := tx.Table(m.tables.WorkItems).Create(&newRow).Error; err != nil {
				return fmt.Errorf("CreateDraft: %w", err)
			}
		}

		var stepRows []gormstore.EventFlowStepRow
		if err := scopeQuery(tx.Table(m.tables.EventFlowSteps), sourceScope).Find(&stepRows).Error; err != nil {
			return fmt.Errorf("CreateDraft: %w", err)
		}
		for _, sr := range stepRows {
			s := gormstore.EventFlowStepFromRow(sr)
			s.ID, s.DraftID, s.CreatedAt = "", newDraftID, now
			newRow := gormstore.EventFlowStepToRow(s)
			if err := tx.Table(m.tables.EventFlowSteps).Create(&newRow).Error; err != nil {
				return fmt.Errorf("CreateDraft: %w", err)
			}
		}

		out = gormstore.DraftFromRow(draftRow)
		return nil
	})
	if err != nil {
		return models.Draft{}, err
	}
	return out, nil
}

func (m *agencyManager) GetDraft(ctx context.Context, id string) (models.Draft, error) {
	row, err := m.findDraftRow(ctx, m.db, id)
	if err != nil {
		return models.Draft{}, err
	}
	return gormstore.DraftFromRow(row), nil
}

func (m *agencyManager) ListDrafts(ctx context.Context) ([]models.Draft, error) {
	var rows []gormstore.DraftRow
	if err := m.db.WithContext(ctx).Table(m.tables.Drafts).Order("created_at").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListDrafts: %w", err)
	}
	out := make([]models.Draft, len(rows))
	for i, r := range rows {
		out[i] = gormstore.DraftFromRow(r)
	}
	return out, nil
}

func (m *agencyManager) UpdateDraftDescription(ctx context.Context, id, description string) (models.Draft, error) {
	row, err := m.findDraftRow(ctx, m.db, id)
	if err != nil {
		return models.Draft{}, err
	}
	draft := gormstore.DraftFromRow(row)
	if draft.Status != models.DraftStatusOpen {
		return models.Draft{}, ErrDraftNotOpen
	}
	now := models.NowRFC3339()
	cols := map[string]any{"description": description, "updated_at": now}
	if err := m.db.WithContext(ctx).Table(m.tables.Drafts).Where("id = ?", id).Updates(cols).Error; err != nil {
		return models.Draft{}, fmt.Errorf("UpdateDraftDescription: %w", err)
	}
	draft.Description, draft.LastUpdated = description, now
	return draft, nil
}

// PromoteDraft requires the Draft to be open. It deletes the current live
// Goal/Workflow/WorkItem/EventFlowStep rows, flips the Draft's own rows'
// DraftID to "" (making them live — no copy-and-swap needed since a
// [models.Draft]-scoped row already exists and just needs reclassifying),
// writes an AgencySnapshot, marks the Agency enabled, and marks the Draft
// promoted.
func (m *agencyManager) PromoteDraft(ctx context.Context, draftID string) (models.Agency, error) {
	var out models.Agency
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		draftRow, err := m.findDraftRow(ctx, tx, draftID)
		if err != nil {
			return err
		}
		draft := gormstore.DraftFromRow(draftRow)
		if !draft.Status.CanTransitionTo(models.DraftStatusPromoted) {
			return ErrDraftNotOpen
		}

		if err := tx.Table(m.tables.Goals).Where("draft_id IS NULL").Delete(&gormstore.GoalRow{}).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}
		if err := tx.Table(m.tables.Workflows).Where("draft_id IS NULL").Delete(&gormstore.WorkflowRow{}).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}
		if err := tx.Table(m.tables.WorkItems).Where("draft_id IS NULL").Delete(&gormstore.WorkItemRow{}).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}
		if err := tx.Table(m.tables.EventFlowSteps).Where("draft_id IS NULL").Delete(&gormstore.EventFlowStepRow{}).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}

		now := models.NowRFC3339()
		promote := map[string]any{"draft_id": nil, "updated_at": now}
		if err := tx.Table(m.tables.Goals).Where("draft_id = ?", draftID).Updates(promote).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}
		if err := tx.Table(m.tables.Workflows).Where("draft_id = ?", draftID).Updates(promote).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}
		if err := tx.Table(m.tables.WorkItems).Where("draft_id = ?", draftID).Updates(promote).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}
		if err := tx.Table(m.tables.EventFlowSteps).Where("draft_id = ?", draftID).Updates(promote).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}

		snapRow := gormstore.AgencySnapshotToRow(models.AgencySnapshot{DraftID: draftID, SnapshotAt: now})
		if err := tx.Table(m.tables.AgencySnapshots).Create(&snapRow).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}

		agencyRow, err := m.findAgencyRow(ctx, tx)
		if err != nil {
			return err
		}
		if err := tx.Table(m.tables.Agencies).Where("id = ?", agencyRow.ID).
			Updates(map[string]any{"enabled": true, "updated_at": now}).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}
		if err := tx.Table(m.tables.Drafts).Where("id = ?", draftID).
			Updates(map[string]any{"status": string(models.DraftStatusPromoted), "updated_at": now}).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}

		agencyRow.Enabled = true
		out = gormstore.AgencyFromRow(agencyRow)
		return nil
	})
	if err != nil {
		return models.Agency{}, err
	}
	return out, nil
}

// ArchiveDraft requires the Draft to be open. Its now-orphaned draft-scoped
// rows are left in place as harmless history, matching source.
func (m *agencyManager) ArchiveDraft(ctx context.Context, id string) (models.Draft, error) {
	row, err := m.findDraftRow(ctx, m.db, id)
	if err != nil {
		return models.Draft{}, err
	}
	draft := gormstore.DraftFromRow(row)
	if !draft.Status.CanTransitionTo(models.DraftStatusArchived) {
		return models.Draft{}, ErrDraftNotOpen
	}
	now := models.NowRFC3339()
	cols := map[string]any{"status": string(models.DraftStatusArchived), "updated_at": now}
	if err := m.db.WithContext(ctx).Table(m.tables.Drafts).Where("id = ?", id).Updates(cols).Error; err != nil {
		return models.Draft{}, fmt.Errorf("ArchiveDraft: %w", err)
	}
	draft.Status, draft.LastUpdated = models.DraftStatusArchived, now
	return draft, nil
}
