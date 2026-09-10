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
	if r.CreatedAt == "" {
		r.CreatedAt = models.NowRFC3339()
	}
	row := gormstore.RuleToRow(r)
	if err := m.db.WithContext(ctx).Table(m.tables.Rules).Create(&row).Error; err != nil {
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

// ListRules returns live Rules when draftID is "", or draftID's
// draft-scoped copies otherwise. Agency-wide Rules (ObjectiveID == "")
// only ever show up in the "" (live) list — they are never forked.
func (m *agencyManager) ListRules(ctx context.Context, draftID string) ([]models.Rule, error) {
	var rows []gormstore.RuleRow
	if err := scopeQuery(m.db.WithContext(ctx).Table(m.tables.Rules), draftID).Find(&rows).Error; err != nil {
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
// row is live — permanent for an Agency-wide Rule, since nothing ever
// forks one into a Draft.
func (m *agencyManager) UpdateRule(ctx context.Context, r models.Rule) (models.Rule, error) {
	existing, err := m.findRuleRow(ctx, m.db, r.ID)
	if err != nil {
		return models.Rule{}, err
	}
	if existing.DraftID == nil {
		return models.Rule{}, ErrCannotEditLive
	}
	if r.Text == "" {
		return models.Rule{}, fmt.Errorf("%w: Rule.Text is required", ErrInvalidRule)
	}
	row := gormstore.RuleToRow(r)
	row.CreatedAt = existing.CreatedAt
	cols := map[string]any{
		"text": row.Text, "updated_at": row.UpdatedAt, "deleted": row.Deleted,
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Rules).Where("id = ?", r.ID).Updates(cols).Error; err != nil {
		return models.Rule{}, fmt.Errorf("UpdateRule: %w", err)
	}
	return gormstore.RuleFromRow(row), nil
}

// DeleteRule returns [ErrCannotEditLive] if the target row is live.
func (m *agencyManager) DeleteRule(ctx context.Context, id string) error {
	existing, err := m.findRuleRow(ctx, m.db, id)
	if err != nil {
		return err
	}
	if existing.DraftID == nil {
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
