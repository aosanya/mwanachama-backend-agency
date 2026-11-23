package mwanachamaagency

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func notRetired(q *gorm.DB) *gorm.DB {
	return q.Where("(deleted = ? OR deleted IS NULL)", false)
}

func (m *agencyManager) RetireRecordRow(ctx context.Context, rowID, deletedBy, reason string) (models.RecordRow, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return models.RecordRow{}, fmt.Errorf("RetireRecordRow: %w", invalidRow("a retired row says why"))
	}
	var out models.RecordRow
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row models.RecordRow
		if err := m.take(tx.WithContext(ctx).Table(m.table(roleRecordRow)).Where("id = ?", rowID), roleRecordRow, &row, ErrRecordNotFound); err != nil {
			return invalidRow("the row to retire does not exist")
		}
		if row.Deleted {
			return invalidRow("the row is already retired")
		}
		replacing, err := list[models.RecordRow](m, tx.WithContext(ctx).Table(m.table(roleRecordRow)).Where("replaces = ?", row.ID), roleRecordRow)
		if err != nil {
			return err
		}
		if len(replacing) > 0 {
			return invalidRow("the row was replaced by a correction, so it is history and not a current row")
		}
		now := rowClock().Format(models.TimeLayout)
		res := tx.WithContext(ctx).Table(m.table(roleRecordRow)).Where("id = ? AND deleted = ?", row.ID, false).Updates(map[string]any{
			"deleted": true, "deleted_by": deletedBy, "deleted_reason": reason, "deleted_on": now, "last_updated": now,
		})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return invalidRow("the row is already retired")
		}
		row.Deleted, row.DeletedBy, row.DeletedReason, row.DeletedOn, row.LastUpdated = true, deletedBy, reason, now, now
		out = row
		return nil
	})
	if err != nil {
		return models.RecordRow{}, fmt.Errorf("RetireRecordRow: %w", err)
	}
	return out, nil
}
