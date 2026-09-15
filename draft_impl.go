package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

// likeFilter chains a case-insensitive substring match on column onto q,
// or returns q unchanged if value is "". LOWER(x) LIKE ? rather than
// Postgres-only ILIKE, matching the existing idiom in roleNotFoundError —
// portable to the SQLite this repo's tests run against.
func likeFilter(q *gorm.DB, column, value string) *gorm.DB {
	if value == "" {
		return q
	}
	return q.Where("LOWER("+column+") LIKE ?", "%"+strings.ToLower(value)+"%")
}

// eqFilter chains an exact match on column onto q, or returns q unchanged
// if value is "" — for enum-shaped columns (Status, Label, SourceType)
// rather than free text.
func eqFilter(q *gorm.DB, column, value string) *gorm.DB {
	if value == "" {
		return q
	}
	return q.Where(column+" = ?", value)
}

// eqBoolFilter chains an exact match on column onto q, or returns q
// unchanged if value is nil. A *bool rather than eqFilter's ""-means-unset
// convention, since a bare bool has no value that can mean "not set."
func eqBoolFilter(q *gorm.DB, column string, value *bool) *gorm.DB {
	if value == nil {
		return q
	}
	return q.Where(column+" = ?", *value)
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

// validateOpenDraftID refuses a Create<Type> call's caller-supplied
// draftID unless it names a real, still-open Draft — draftID == "" (a live
// row) is always allowed and returns nil without a lookup. Closes WK17: an
// unvalidated draftID let a caller attach a new live write to an arbitrary
// or already-promoted/archived Draft, which CreateDraft's own fork never
// produced and PromoteDraft's copy-and-cutover logic does not expect.
// ErrDraftNotFound for an unknown id, ErrDraftNotOpen for one that is
// promoted or archived.
func (m *agencyManager) validateOpenDraftID(ctx context.Context, draftID string) error {
	if draftID == "" {
		return nil
	}
	row, err := m.findDraftRow(ctx, m.db, draftID)
	if err != nil {
		return err
	}
	if row.Status != string(models.DraftStatusOpen) {
		return ErrDraftNotOpen
	}
	return nil
}

// CreateDraft forks the sub-graph scoped to forkedFromID (an Agency.ID when
// forkedFromType=="live", or another open Draft's ID when
// forkedFromType=="draft") into brand-new Goal/Objective/Workflow/
// WorkItem/EventFlowStep/Role/RoleAssignment/Module/ObjectiveModule/Value/
// Rule rows scoped to the new Draft. Objective's GoalID, WorkItem's
// WorkflowID/ObjectiveID, RoleAssignment's RoleID/ObjectiveID, and
// ObjectiveModule's ObjectiveID/ModuleID are remapped to the copies' new
// ids; EventFlowStep needs no remap (it joins Workflow by the
// WorkflowCode business key, not a row id), and nothing points into
// Module, so its own copy loop needs no remap either. A live Role can only
// be edited by forking it into a draft this way, editing the copy, then
// promoting.
//
// Agency-wide Value/Rule rows (no [models.ValueAttachment]/
// [models.RuleAttachment] at all) are deliberately never copied here —
// AGD-001 confirmed they have no edit path at all, ever, and forking them
// into a Draft would be exactly that. Only rows carrying at least one
// attachment fork, same as Goal/WorkItem/Role. Attachments themselves are
// never forked or remapped here — AGD-003 (2026-09-15) made them
// live-editable, addressed by Code rather than row id specifically so
// they need no remap step at fork or promote time.
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

		var objectiveRows []gormstore.ObjectiveRow
		if err := scopeQuery(tx.Table(m.tables.Objectives), sourceScope).Find(&objectiveRows).Error; err != nil {
			return fmt.Errorf("CreateDraft: %w", err)
		}
		objectiveIDMap := make(map[string]string, len(objectiveRows))
		for _, or := range objectiveRows {
			o := gormstore.ObjectiveFromRow(or)
			oldID := o.ID
			o.ID, o.DraftID, o.CreatedAt = "", newDraftID, now
			if mapped, ok := goalIDMap[o.GoalID]; ok {
				o.GoalID = mapped
			}
			newRow := gormstore.ObjectiveToRow(o)
			if err := tx.Table(m.tables.Objectives).Create(&newRow).Error; err != nil {
				return fmt.Errorf("CreateDraft: %w", err)
			}
			objectiveIDMap[oldID] = newRow.ID
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
		workItemIDMap := make(map[string]string, len(workItemRows))
		for _, wir := range workItemRows {
			wi := gormstore.WorkItemFromRow(wir)
			oldID := wi.ID
			wi.ID, wi.DraftID, wi.CreatedAt = "", newDraftID, now
			if mapped, ok := workflowIDMap[wi.WorkflowID]; ok {
				wi.WorkflowID = mapped
			}
			if mapped, ok := objectiveIDMap[wi.ObjectiveID]; ok {
				wi.ObjectiveID = mapped
			}
			newRow := gormstore.WorkItemToRow(wi)
			if err := tx.Table(m.tables.WorkItems).Create(&newRow).Error; err != nil {
				return fmt.Errorf("CreateDraft: %w", err)
			}
			workItemIDMap[oldID] = newRow.ID
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

		var roleRows []gormstore.RoleRow
		if err := scopeQuery(tx.Table(m.tables.Roles), sourceScope).Find(&roleRows).Error; err != nil {
			return fmt.Errorf("CreateDraft: %w", err)
		}
		roleIDMap := make(map[string]string, len(roleRows))
		for _, rr := range roleRows {
			r := gormstore.RoleFromRow(rr)
			oldID := r.ID
			r.ID, r.DraftID, r.CreatedAt = "", newDraftID, now
			newRow := gormstore.RoleToRow(r)
			if err := tx.Table(m.tables.Roles).Create(&newRow).Error; err != nil {
				return fmt.Errorf("CreateDraft: %w", err)
			}
			roleIDMap[oldID] = newRow.ID
		}

		var moduleRows []gormstore.ModuleRow
		if err := scopeQuery(tx.Table(m.tables.Modules), sourceScope).Find(&moduleRows).Error; err != nil {
			return fmt.Errorf("CreateDraft: %w", err)
		}
		moduleIDMap := make(map[string]string, len(moduleRows))
		for _, modr := range moduleRows {
			mod := gormstore.ModuleFromRow(modr)
			oldID := mod.ID
			mod.ID, mod.DraftID, mod.CreatedAt = "", newDraftID, now
			newRow := gormstore.ModuleToRow(mod)
			if err := tx.Table(m.tables.Modules).Create(&newRow).Error; err != nil {
				return fmt.Errorf("CreateDraft: %w", err)
			}
			moduleIDMap[oldID] = newRow.ID
		}

		var assignmentRows []gormstore.RoleAssignmentRow
		if err := scopeQuery(tx.Table(m.tables.RoleAssignments), sourceScope).Find(&assignmentRows).Error; err != nil {
			return fmt.Errorf("CreateDraft: %w", err)
		}
		for _, ar := range assignmentRows {
			a := gormstore.RoleAssignmentFromRow(ar)
			a.ID, a.DraftID, a.CreatedAt = "", newDraftID, now
			if mapped, ok := roleIDMap[a.RoleID]; ok {
				a.RoleID = mapped
			}
			if mapped, ok := objectiveIDMap[a.ObjectiveID]; ok {
				a.ObjectiveID = mapped
			}
			newRow := gormstore.RoleAssignmentToRow(a)
			if err := tx.Table(m.tables.RoleAssignments).Create(&newRow).Error; err != nil {
				return fmt.Errorf("CreateDraft: %w", err)
			}
		}

		var objectiveModuleRows []gormstore.ObjectiveModuleRow
		if err := scopeQuery(tx.Table(m.tables.ObjectiveModules), sourceScope).Find(&objectiveModuleRows).Error; err != nil {
			return fmt.Errorf("CreateDraft: %w", err)
		}
		for _, omr := range objectiveModuleRows {
			om := gormstore.ObjectiveModuleFromRow(omr)
			om.ID, om.DraftID, om.CreatedAt = "", newDraftID, now
			if mapped, ok := objectiveIDMap[om.ObjectiveID]; ok {
				om.ObjectiveID = mapped
			}
			if mapped, ok := moduleIDMap[om.ModuleID]; ok {
				om.ModuleID = mapped
			}
			newRow := gormstore.ObjectiveModuleToRow(om)
			if err := tx.Table(m.tables.ObjectiveModules).Create(&newRow).Error; err != nil {
				return fmt.Errorf("CreateDraft: %w", err)
			}
		}

		// Only Value/Rule rows carrying at least one attachment fork —
		// Agency-wide ones (no attachment at all) are excluded by this
		// query and so can never enter a Draft. See this function's doc
		// comment. No id remap needed — attachments reference Code, not
		// row id.
		var valueRows []gormstore.ValueRow
		if err := scopeQuery(tx.Table(m.tables.Values), sourceScope).
			Where("code IN (SELECT value_code FROM " + m.tables.ValueAttachments + ")").
			Find(&valueRows).Error; err != nil {
			return fmt.Errorf("CreateDraft: %w", err)
		}
		for _, vr := range valueRows {
			v := gormstore.ValueFromRow(vr)
			v.ID, v.DraftID, v.CreatedAt = "", newDraftID, now
			newRow := gormstore.ValueToRow(v)
			if err := tx.Table(m.tables.Values).Create(&newRow).Error; err != nil {
				return fmt.Errorf("CreateDraft: %w", err)
			}
		}

		var ruleRows []gormstore.RuleRow
		if err := scopeQuery(tx.Table(m.tables.Rules), sourceScope).
			Where("code IN (SELECT rule_code FROM " + m.tables.RuleAttachments + ")").
			Find(&ruleRows).Error; err != nil {
			return fmt.Errorf("CreateDraft: %w", err)
		}
		for _, rr := range ruleRows {
			r := gormstore.RuleFromRow(rr)
			r.ID, r.DraftID, r.CreatedAt = "", newDraftID, now
			newRow := gormstore.RuleToRow(r)
			if err := tx.Table(m.tables.Rules).Create(&newRow).Error; err != nil {
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

func (m *agencyManager) ListDrafts(ctx context.Context, filter DraftFilter) ([]models.Draft, error) {
	q := m.db.WithContext(ctx).Table(m.tables.Drafts).Order("created_at")
	q = likeFilter(q, "description", filter.Description)
	q = eqFilter(q, "status", filter.Status)
	var rows []gormstore.DraftRow
	if err := q.Find(&rows).Error; err != nil {
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
// Goal/Objective/Workflow/WorkItem/EventFlowStep/Role/RoleAssignment/
// Module/ObjectiveModule/Value/Rule rows (Value/Rule scoped to rows
// carrying at least one attachment only — Agency-wide ones are never
// draft-scoped, so must never be deleted here), flips the Draft's own
// rows' DraftID to "" (making them
// live — no copy-and-swap needed since a [models.Draft]-scoped row already
// exists and just needs reclassifying), writes an AgencySnapshot, marks
// the Agency enabled, and marks the Draft promoted.
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
		if err := tx.Table(m.tables.Objectives).Where("draft_id IS NULL").Delete(&gormstore.ObjectiveRow{}).Error; err != nil {
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
		if err := tx.Table(m.tables.Roles).Where("draft_id IS NULL").Delete(&gormstore.RoleRow{}).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}
		if err := tx.Table(m.tables.Modules).Where("draft_id IS NULL").Delete(&gormstore.ModuleRow{}).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}
		if err := tx.Table(m.tables.RoleAssignments).Where("draft_id IS NULL").Delete(&gormstore.RoleAssignmentRow{}).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}
		if err := tx.Table(m.tables.ObjectiveModules).Where("draft_id IS NULL").Delete(&gormstore.ObjectiveModuleRow{}).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}
		if err := tx.Table(m.tables.Values).
			Where("draft_id IS NULL AND code IN (SELECT value_code FROM " + m.tables.ValueAttachments + ")").
			Delete(&gormstore.ValueRow{}).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}
		if err := tx.Table(m.tables.Rules).
			Where("draft_id IS NULL AND code IN (SELECT rule_code FROM " + m.tables.RuleAttachments + ")").
			Delete(&gormstore.RuleRow{}).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}

		now := models.NowRFC3339()
		promote := map[string]any{"draft_id": nil, "updated_at": now}
		if err := tx.Table(m.tables.Goals).Where("draft_id = ?", draftID).Updates(promote).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}
		if err := tx.Table(m.tables.Objectives).Where("draft_id = ?", draftID).Updates(promote).Error; err != nil {
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
		if err := tx.Table(m.tables.Roles).Where("draft_id = ?", draftID).Updates(promote).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}
		if err := tx.Table(m.tables.Modules).Where("draft_id = ?", draftID).Updates(promote).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}
		if err := tx.Table(m.tables.RoleAssignments).Where("draft_id = ?", draftID).Updates(promote).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}
		if err := tx.Table(m.tables.ObjectiveModules).Where("draft_id = ?", draftID).Updates(promote).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}
		if err := tx.Table(m.tables.Values).Where("draft_id = ?", draftID).Updates(promote).Error; err != nil {
			return fmt.Errorf("PromoteDraft: %w", err)
		}
		if err := tx.Table(m.tables.Rules).Where("draft_id = ?", draftID).Updates(promote).Error; err != nil {
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
