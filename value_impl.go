package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) CreateValue(ctx context.Context, v models.Value) (models.Value, error) {
	if v.Text == "" {
		return models.Value{}, fmt.Errorf("%w: Value.Text is required", ErrInvalidValue)
	}
	if err := m.validateOpenDraftID(ctx, v.DraftID); err != nil {
		return models.Value{}, err
	}
	v.ID = ""
	v.CreatedAt = models.NowRFC3339()
	var row gormstore.ValueRow
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		code, err := gormstore.NextCode(ctx, tx, m.tables.CodeSequences, "value", "V")
		if err != nil {
			return err
		}
		v.Code = code
		row = gormstore.ValueToRow(v)
		return tx.Table(m.tables.Values).Create(&row).Error
	})
	if err != nil {
		return models.Value{}, fmt.Errorf("CreateValue: %w", err)
	}
	return gormstore.ValueFromRow(row), nil
}

func (m *agencyManager) GetValue(ctx context.Context, id string) (models.Value, error) {
	row, err := m.findValueRow(ctx, m.db, id)
	if err != nil {
		return models.Value{}, err
	}
	return gormstore.ValueFromRow(row), nil
}

// ListValues returns live Values when draftID is "", or draftID's
// draft-scoped copies otherwise, narrowed by filter. Agency-wide Values
// (ObjectiveID == "") only ever show up in the "" (live) list — they are
// never forked.
func (m *agencyManager) ListValues(ctx context.Context, draftID string, filter ValueFilter) ([]models.Value, error) {
	q := scopeQuery(m.db.WithContext(ctx).Table(m.tables.Values), draftID)
	q = likeFilter(q, "code", filter.Code)
	q = likeFilter(q, "text", filter.Text)
	var rows []gormstore.ValueRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListValues: %w", err)
	}
	out := make([]models.Value, len(rows))
	for i, row := range rows {
		out[i] = gormstore.ValueFromRow(row)
	}
	return out, nil
}

// UpdateValue replaces v's mutable fields via an explicit column map (see
// UpdateGoal's doc for why). Returns [ErrCannotEditLive] if the existing
// row is live — permanent for an Agency-wide Value, since nothing ever
// forks one into a Draft.
func (m *agencyManager) UpdateValue(ctx context.Context, v models.Value) (models.Value, error) {
	existing, err := m.findValueRow(ctx, m.db, v.ID)
	if err != nil {
		return models.Value{}, err
	}
	if existing.DraftID == nil {
		return models.Value{}, ErrCannotEditLive
	}
	if v.Text == "" {
		return models.Value{}, fmt.Errorf("%w: Value.Text is required", ErrInvalidValue)
	}
	row := gormstore.ValueToRow(v)
	row.CreatedAt = existing.CreatedAt
	row.Code = existing.Code
	cols := map[string]any{
		"text": row.Text, "updated_at": row.UpdatedAt, "deleted": row.Deleted,
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Values).Where("id = ?", v.ID).Updates(cols).Error; err != nil {
		return models.Value{}, fmt.Errorf("UpdateValue: %w", err)
	}
	return gormstore.ValueFromRow(row), nil
}

// DeleteValue returns [ErrCannotEditLive] if the target row is live.
func (m *agencyManager) DeleteValue(ctx context.Context, id string) error {
	existing, err := m.findValueRow(ctx, m.db, id)
	if err != nil {
		return err
	}
	if existing.DraftID == nil {
		return ErrCannotEditLive
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Values).Where("id = ?", id).
		Delete(&gormstore.ValueRow{}).Error; err != nil {
		return fmt.Errorf("DeleteValue: %w", err)
	}
	return nil
}

func (m *agencyManager) findValueRow(ctx context.Context, tx *gorm.DB, id string) (gormstore.ValueRow, error) {
	var row gormstore.ValueRow
	err := tx.WithContext(ctx).Table(m.tables.Values).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.ValueRow{}, ErrValueNotFound
	}
	if err != nil {
		return gormstore.ValueRow{}, fmt.Errorf("findValueRow: %w", err)
	}
	return row, nil
}
