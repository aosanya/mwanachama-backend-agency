package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) CreateRule(ctx context.Context, r models.Rule) (models.Rule, error) {
	if r.Text == "" {
		return models.Rule{}, fmt.Errorf("%w: Rule.Text is required", ErrInvalidRule)
	}
	if err := m.validateOpenDraftID(ctx, r.DraftID); err != nil {
		return models.Rule{}, err
	}
	r.ID = ""
	r.CreatedAt = models.NowRFC3339()
	var row gormstore.RuleRow
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		code, err := gormstore.NextCode(ctx, tx, m.tables.CodeSequences, "rule", "RU")
		if err != nil {
			return err
		}
		r.Code = code
		row = gormstore.RuleToRow(r)
		return tx.Table(m.tables.Rules).Create(&row).Error
	})
	if err != nil {
		return models.Rule{}, fmt.Errorf("CreateRule: %w", err)
	}
	return gormstore.RuleFromRow(row), nil
}

func (m *agencyManager) GetRule(ctx context.Context, id string) (models.Rule, error) {
	row, err := m.findRuleRow(ctx, m.db, id)
	if err != nil {
		return models.Rule{}, err
	}
	return gormstore.RuleFromRow(row), nil
}

// ListRules returns draftID's Rules, narrowed by filter — including,
// optionally, "only Rules attached to this Role/Goal/Objective/Workflow/
// WorkItem" (AGD-003). A Rule with no attachment at all (Agency-wide) only
// ever shows up in whichever one Draft it was created in — it is never
// forked into any other Draft (see CreateDraft's own doc comment).
func (m *agencyManager) ListRules(ctx context.Context, draftID string, filter RuleFilter) ([]models.Rule, error) {
	q := scopeQuery(m.db.WithContext(ctx).Table(m.tables.Rules), draftID)
	q = likeFilter(q, "code", filter.Code)
	q = likeFilter(q, "text", filter.Text)
	q, matchable, err := m.applyTargetFilters(ctx, q, m.tables.RuleAttachments, "rule_code", targetFilters{
		RoleID: filter.RoleID, GoalID: filter.GoalID, ObjectiveID: filter.ObjectiveID,
		WorkflowID: filter.WorkflowID, WorkItemID: filter.WorkItemID,
	})
	if err != nil {
		return nil, fmt.Errorf("ListRules: %w", err)
	}
	if !matchable {
		return nil, nil
	}
	var rows []gormstore.RuleRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListRules: %w", err)
	}
	out := make([]models.Rule, len(rows))
	for i, row := range rows {
		out[i] = gormstore.RuleFromRow(row)
	}
	return out, nil
}

// UpdateRule replaces r's mutable fields via an explicit column map (see
// UpdateGoal's doc for why). Returns [ErrCannotEditLive] if the existing
// row carries no attachment — permanent for an Agency-wide Rule, since
// nothing ever forks one into another Draft (see [hasAnyAttachment]).
func (m *agencyManager) UpdateRule(ctx context.Context, r models.Rule) (models.Rule, error) {
	existing, err := m.findRuleRow(ctx, m.db, r.ID)
	if err != nil {
		return models.Rule{}, err
	}
	attached, err := hasAnyAttachment(ctx, m.db, m.tables.RuleAttachments, "rule_code", existing.Code)
	if err != nil {
		return models.Rule{}, err
	}
	if !attached {
		return models.Rule{}, ErrCannotEditLive
	}
	if r.Text == "" {
		return models.Rule{}, fmt.Errorf("%w: Rule.Text is required", ErrInvalidRule)
	}
	row := gormstore.RuleToRow(r)
	row.CreatedAt = existing.CreatedAt
	row.Code = existing.Code
	cols := map[string]any{
		"text": row.Text, "updated_at": row.UpdatedAt, "deleted": row.Deleted,
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Rules).Where("id = ?", r.ID).Updates(cols).Error; err != nil {
		return models.Rule{}, fmt.Errorf("UpdateRule: %w", err)
	}
	return gormstore.RuleFromRow(row), nil
}

// DeleteRule returns [ErrCannotEditLive] if the target row carries no
// attachment (see [UpdateRule]'s doc comment).
func (m *agencyManager) DeleteRule(ctx context.Context, id string) error {
	existing, err := m.findRuleRow(ctx, m.db, id)
	if err != nil {
		return err
	}
	attached, err := hasAnyAttachment(ctx, m.db, m.tables.RuleAttachments, "rule_code", existing.Code)
	if err != nil {
		return err
	}
	if !attached {
		return ErrCannotEditLive
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Rules).Where("id = ?", id).
		Delete(&gormstore.RuleRow{}).Error; err != nil {
		return fmt.Errorf("DeleteRule: %w", err)
	}
	return nil
}

func (m *agencyManager) findRuleRow(ctx context.Context, tx *gorm.DB, id string) (gormstore.RuleRow, error) {
	var row gormstore.RuleRow
	err := tx.WithContext(ctx).Table(m.tables.Rules).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.RuleRow{}, ErrRuleNotFound
	}
	if err != nil {
		return gormstore.RuleRow{}, fmt.Errorf("findRuleRow: %w", err)
	}
	return row, nil
}

// CreateRuleAttachment attaches ruleID to one target (targetID, of
// targetType) — live-editable, no Draft involved. See
// [AgencyManager.CreateValueAttachment]'s doc comment; identical behavior.
func (m *agencyManager) CreateRuleAttachment(ctx context.Context, ruleID string, targetType models.TargetType, targetID string) (models.RuleAttachment, error) {
	var out models.RuleAttachment
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ruleCode, err := resolveCodeByID(ctx, tx, m.tables.Rules, ruleID, ErrRuleNotFound)
		if err != nil {
			return err
		}
		targetCode, err := m.resolveTargetCode(ctx, tx, targetType, targetID)
		if err != nil {
			return err
		}
		var count int64
		if err := tx.WithContext(ctx).Table(m.tables.RuleAttachments).
			Where("rule_code = ? AND target_type = ? AND target_code = ?", ruleCode, string(targetType), targetCode).
			Count(&count).Error; err != nil {
			return fmt.Errorf("CreateRuleAttachment: %w", err)
		}
		if count > 0 {
			return ErrDuplicateAttachment
		}
		row := gormstore.RuleAttachmentToRow(models.RuleAttachment{
			RuleCode: ruleCode, TargetType: targetType, TargetCode: targetCode,
			CreatedAt: models.NowRFC3339(),
		})
		if err := tx.Table(m.tables.RuleAttachments).Create(&row).Error; err != nil {
			return fmt.Errorf("CreateRuleAttachment: %w", err)
		}
		out = gormstore.RuleAttachmentFromRow(row)
		return nil
	})
	if err != nil {
		return models.RuleAttachment{}, err
	}
	return out, nil
}

// ListRuleAttachments lists every target ruleID is attached to.
func (m *agencyManager) ListRuleAttachments(ctx context.Context, ruleID string) ([]models.RuleAttachment, error) {
	ruleCode, err := resolveCodeByID(ctx, m.db.WithContext(ctx), m.tables.Rules, ruleID, ErrRuleNotFound)
	if err != nil {
		return nil, err
	}
	var rows []gormstore.RuleAttachmentRow
	if err := m.db.WithContext(ctx).Table(m.tables.RuleAttachments).
		Where("rule_code = ?", ruleCode).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListRuleAttachments: %w", err)
	}
	out := make([]models.RuleAttachment, len(rows))
	for i, row := range rows {
		out[i] = gormstore.RuleAttachmentFromRow(row)
	}
	return out, nil
}

// DeleteRuleAttachment removes one attachment by its own id.
func (m *agencyManager) DeleteRuleAttachment(ctx context.Context, id string) error {
	res := m.db.WithContext(ctx).Table(m.tables.RuleAttachments).Where("id = ?", id).
		Delete(&gormstore.RuleAttachmentRow{})
	if res.Error != nil {
		return fmt.Errorf("DeleteRuleAttachment: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrRuleAttachmentNotFound
	}
	return nil
}
