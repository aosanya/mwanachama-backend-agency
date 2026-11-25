package mwanachamaagency

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func notReturned(q *gorm.DB) *gorm.DB {
	return q.Where("(returned_on = ? OR returned_on IS NULL)", "")
}

func (m *agencyManager) ReturnRecordRow(ctx context.Context, rowID, returnedBy, reason string) (models.RecordRow, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return models.RecordRow{}, fmt.Errorf("ReturnRecordRow: %w", invalidRow("a returned row says why"))
	}
	var out models.RecordRow
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row models.RecordRow
		if err := m.take(tx.WithContext(ctx).Table(m.table(roleRecordRow)).Where("id = ?", rowID), roleRecordRow, &row, ErrRecordNotFound); err != nil {
			return invalidRow("the row to return does not exist")
		}
		if row.Deleted {
			return invalidRow("the row is retired")
		}
		if row.ReturnedOn != "" {
			return invalidRow("the row is already returned")
		}
		replacing, err := list[models.RecordRow](m, tx.WithContext(ctx).Table(m.table(roleRecordRow)).Where("replaces = ?", row.ID), roleRecordRow)
		if err != nil {
			return err
		}
		if len(replacing) > 0 {
			return invalidRow("the row was replaced by a correction, so it is history and not a current row")
		}
		returnedByID := returnedByIDOf(ctx)
		now := rowClock().Format(models.TimeLayout)
		res := tx.WithContext(ctx).Table(m.table(roleRecordRow)).
			Where("id = ? AND deleted = ? AND (returned_on = ? OR returned_on IS NULL)", row.ID, false, "").
			Updates(map[string]any{
				"returned_by": returnedBy, "returned_by_id": returnedByID, "returned_reason": reason, "returned_on": now, "send_key": "", "last_updated": now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return invalidRow("the row is already returned")
		}
		row.ReturnedBy, row.ReturnedByID, row.ReturnedReason, row.ReturnedOn, row.SendKey, row.LastUpdated = returnedBy, returnedByID, reason, now, "", now
		out = row
		return nil
	})
	if err != nil {
		return models.RecordRow{}, fmt.Errorf("ReturnRecordRow: %w", err)
	}
	return out, nil
}
