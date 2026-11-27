package mwanachamaagency

import (
	"context"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) settleSubjectRow(ctx context.Context, row *models.RecordRow) error {
	if row.SubjectRowID != "" {
		var subject models.RecordRow
		q := m.q(ctx, roleRecordRow).Where("id = ?", row.SubjectRowID).Scopes(notRetired)
		if err := m.take(q, roleRecordRow, &subject, ErrRecordNotFound); err != nil {
			return invalidRow("the subject row %s is not a row of this agency", QuoteIfUnsafe(row.SubjectRowID))
		}
	}
	if row.OrderID == "" {
		return nil
	}
	var sibling models.RecordRow
	q := m.q(ctx, roleRecordRow).
		Where("order_id = ? AND subject_row_id <> ''", row.OrderID).Order("created_at")
	if err := m.take(q, roleRecordRow, &sibling, ErrRecordNotFound); err != nil {
		return nil
	}
	if row.SubjectRowID == "" {
		row.SubjectRowID = sibling.SubjectRowID
	}
	if row.SubjectRowID != sibling.SubjectRowID {
		return invalidRow("a row of a run stays with the subject row the run was started on, %s", QuoteIfUnsafe(sibling.SubjectRowID))
	}
	return nil
}
