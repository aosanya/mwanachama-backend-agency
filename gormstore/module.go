package gormstore

import (
	"fmt"

	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ModuleRow is the GORM row for a [models.Module]. Code uses plain
// gorm:"index", not uniqueIndex — same AG16 precedent as every other
// Draft-scoped Code column: a Draft-scoped copy carries the same Code as
// its live original while both exist side by side between CreateDraft and
// PromoteDraft.
type ModuleRow struct {
	ID          string  `gorm:"primaryKey"`
	Code        string  `gorm:"index"`
	DraftID     *string `gorm:"index"`
	Name        string
	Description string
	Enabled     bool
	Locked      bool
	CreatedAt   string
	UpdatedAt   string
	Deleted     bool
}

func (r *ModuleRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// ModuleToRow converts a domain Module to its row shape, stamping UpdatedAt.
func ModuleToRow(m models.Module) ModuleRow {
	return ModuleRow{
		ID:          m.ID,
		Code:        m.Code,
		DraftID:     StringToNullable(m.DraftID),
		Name:        m.Name,
		Description: m.Description,
		Enabled:     m.Enabled,
		Locked:      m.Locked,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   models.NowRFC3339(),
		Deleted:     m.Deleted,
	}
}

// ModuleFromRow converts a row back to the domain Module.
func ModuleFromRow(r ModuleRow) models.Module {
	return models.Module{
		ID:          r.ID,
		Code:        r.Code,
		DraftID:     nullableToString(r.DraftID),
		Name:        r.Name,
		Description: r.Description,
		Enabled:     r.Enabled,
		Locked:      r.Locked,
		CreatedAt:   r.CreatedAt,
		LastUpdated: r.UpdatedAt,
		Deleted:     r.Deleted,
	}
}

// BackfillModules ensures every code in [models.ModuleCatalog] has a live
// Module row, inserting one from that catalog entry's Name/Description/
// Enabled/Locked for any code that doesn't yet. Idempotent, and only ever
// inserts a missing row — an existing live row (of any Enabled/Locked
// value) is never touched, so an already-migrated deployment keeps
// whatever it already has. Safe to call on every Migrate, mirroring
// [BackfillCodes]'s shape. Without this, a pre-existing deployment's
// Module table starts empty the moment this migration ships, and a
// gateway gating real routes on Module.Enabled would take every one of
// those routes dark rather than leaving them as they were.
func BackfillModules(db *gorm.DB, table string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, entry := range models.ModuleCatalog {
			var count int64
			if err := tx.Table(table).Where("code = ? AND draft_id IS NULL", entry.Code).Count(&count).Error; err != nil {
				return fmt.Errorf("BackfillModules: %w", err)
			}
			if count > 0 {
				continue
			}
			now := models.NowRFC3339()
			row := ModuleRow{
				Code:        entry.Code,
				Name:        entry.Name,
				Description: entry.Description,
				Enabled:     entry.Enabled,
				Locked:      entry.Locked,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if err := tx.Table(table).Create(&row).Error; err != nil {
				return fmt.Errorf("BackfillModules: %w", err)
			}
		}
		return nil
	})
}
