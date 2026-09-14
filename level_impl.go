package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) CreateLevel(ctx context.Context, l models.Level) (models.Level, error) {
	if l.CreatedAt == "" {
		l.CreatedAt = models.NowRFC3339()
	}
	var row gormstore.LevelRow
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		code, err := gormstore.NextCode(ctx, tx, m.tables.CodeSequences, "level", "L")
		if err != nil {
			return err
		}
		l.Code = code
		row = gormstore.LevelToRow(l)
		return tx.Table(m.tables.Levels).Create(&row).Error
	})
	if err != nil {
		return models.Level{}, fmt.Errorf("CreateLevel: %w", err)
	}
	return gormstore.LevelFromRow(row), nil
}

func (m *agencyManager) GetLevel(ctx context.Context, id string) (models.Level, error) {
	row, err := m.findLevelRow(ctx, m.db, id)
	if err != nil {
		return models.Level{}, err
	}
	return gormstore.LevelFromRow(row), nil
}

// ListLevels returns every Level matching filter, ordered by (Depth, ID).
func (m *agencyManager) ListLevels(ctx context.Context, filter LevelFilter) ([]models.Level, error) {
	q := m.db.WithContext(ctx).Table(m.tables.Levels)
	q = likeFilter(q, "code", filter.Code)
	q = likeFilter(q, "name", filter.Name)
	var rows []gormstore.LevelRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListLevels: %w", err)
	}
	out := make([]models.Level, len(rows))
	for i, r := range rows {
		out[i] = gormstore.LevelFromRow(r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Depth != out[j].Depth {
			return out[i].Depth < out[j].Depth
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// RenameLevel writes a Level's name and nothing else — Depth stays
// reachable only through CreateLevel, matching mwanachama-backend-actor's
// RenameLevel.
func (m *agencyManager) RenameLevel(ctx context.Context, id, name string) (models.Level, error) {
	current, err := m.GetLevel(ctx, id)
	if err != nil {
		return models.Level{}, err
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Levels).Where("id = ?", id).
		Update("name", name).Error; err != nil {
		return models.Level{}, fmt.Errorf("RenameLevel: %w", err)
	}
	current.Name = name
	return current, nil
}

// DeleteLevel refuses while any Department still references id
// (ErrLevelWornByDepartments). Unlike actor's DeleteLevel, there is no
// second RoleKind-style guard yet — no Role model exists in this repo.
func (m *agencyManager) DeleteLevel(ctx context.Context, id string) error {
	if _, err := m.findLevelRow(ctx, m.db, id); err != nil {
		return err
	}
	var wornCount int64
	if err := m.db.WithContext(ctx).Table(m.tables.Departments).Where("level_id = ?", id).
		Count(&wornCount).Error; err != nil {
		return fmt.Errorf("DeleteLevel: %w", err)
	}
	if wornCount > 0 {
		return ErrLevelWornByDepartments
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Levels).Where("id = ?", id).
		Delete(&gormstore.LevelRow{}).Error; err != nil {
		return fmt.Errorf("DeleteLevel: %w", err)
	}
	return nil
}

func (m *agencyManager) findLevelRow(ctx context.Context, tx *gorm.DB, id string) (gormstore.LevelRow, error) {
	var row gormstore.LevelRow
	err := tx.WithContext(ctx).Table(m.tables.Levels).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.LevelRow{}, ErrLevelNotFound
	}
	if err != nil {
		return gormstore.LevelRow{}, fmt.Errorf("findLevelRow: %w", err)
	}
	return row, nil
}
