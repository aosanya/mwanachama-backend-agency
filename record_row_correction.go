package mwanachamaagency

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

const duplicateRowWindow = time.Minute

var rowClock = func() time.Time { return time.Now().UTC() }

func (m *agencyManager) settleReplacement(ctx context.Context, row *models.RecordRow, record models.Record) error {
	if row.Replaces == "" {
		return nil
	}
	var old models.RecordRow
	if err := m.take(m.q(ctx, roleRecordRow).Where("id = ?", row.Replaces), roleRecordRow, &old, ErrRecordNotFound); err != nil {
		return invalidRow("the row to replace does not exist")
	}
	if old.Deleted {
		return invalidRow("the row to replace is retired")
	}
	if old.ReturnedOn != "" {
		return invalidRow("the row to replace was returned")
	}
	if old.RecordCode != record.Code {
		return invalidRow("the row to replace belongs to another record")
	}
	if row.OrderID == "" {
		row.OrderID = old.OrderID
	}
	if row.OrderID != old.OrderID {
		return invalidRow("a replacing row stays with the order of the row it replaces")
	}
	if row.SubjectRowID == "" {
		row.SubjectRowID = old.SubjectRowID
	}
	if old.SubjectRowID != "" && row.SubjectRowID != old.SubjectRowID {
		return invalidRow("a replacing row stays with the subject row of the row it replaces")
	}
	taken, err := list[models.RecordRow](m, m.q(ctx, roleRecordRow).Where("replaces = ?", old.ID), roleRecordRow)
	if err != nil {
		return err
	}
	if len(taken) > 0 {
		return invalidRow("the row to replace was already corrected")
	}
	row.Tags = slices.Clone(old.Tags)
	if row.Tags == nil {
		row.Tags = []string{}
	}
	row.CorrectedOn = row.CreatedAt
	return nil
}

func (m *agencyManager) refuseDuplicate(ctx context.Context, row models.RecordRow) error {
	since := rowClock().Add(-duplicateRowWindow).Format(models.TimeLayout)
	recent, err := list[models.RecordRow](m, m.q(ctx, roleRecordRow).
		Where("record_code = ? AND created_at >= ?", row.RecordCode, since).Scopes(notRetired, notReturned), roleRecordRow)
	if err != nil {
		return err
	}
	replaced := map[string]bool{}
	for _, r := range recent {
		if r.Replaces != "" {
			replaced[r.Replaces] = true
		}
	}
	for _, r := range recent {
		if !replaced[r.ID] && r.AddedBy == row.AddedBy && r.OrderID == row.OrderID && maps.Equal(r.Answers, row.Answers) {
			return ErrDuplicateRecordRow
		}
	}
	return nil
}
