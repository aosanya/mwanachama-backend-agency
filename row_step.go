package mwanachamaagency

import (
	"context"
	"reflect"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func hasRowFields(wi models.WorkItem) bool {
	return wi.RowRecord != "" || wi.RowAction != ""
}

func validateRowStepShape(wi models.WorkItem) error {
	if !hasRowFields(wi) {
		return nil
	}
	switch {
	case wi.RowRecord == "":
		return invalidStep("a step with a row action names the record whose row it writes")
	case wi.RowAction == "":
		return invalidStep("a step that names a record to write to says whether it adds, edits or deletes a row")
	case !models.ValidRowAction(wi.RowAction):
		return invalidStep("a row action is one of %v, got %q", models.RowActions(), wi.RowAction)
	case wi.Kind != models.StepKindFillForm:
		return invalidStep("only a fill_form step writes a row of a record, not a step of kind %q", wi.Kind)
	}
	return nil
}

func (m *agencyManager) validateRowStepReferences(ctx context.Context, tx *gorm.DB, wi models.WorkItem) error {
	if !hasRowFields(wi) {
		return m.refuseStepBesideRowStep(ctx, tx, wi)
	}
	var record models.Record
	q := scopeQuery(tx.WithContext(ctx).Table(m.table(roleRecord)), wi.DraftID).Where("key = ?", wi.RowRecord)
	if err := m.take(q, roleRecord, &record, ErrRecordNotFound); err != nil {
		return invalidStep("the record %s is not a record of this draft", QuoteIfUnsafe(wi.RowRecord))
	}
	if want := record.RoleOfRowAction(wi.RowAction); wi.RoleID != want {
		return invalidStep("a step that %ss a row of %s is held by the role that does so, %s", wi.RowAction, record.Identity(), QuoteIfUnsafe(want))
	}
	if err := rowStepFormFits(wi, record); err != nil {
		return err
	}
	return m.refuseStepBesideRowStep(ctx, tx, wi)
}

func rowStepFormFits(wi models.WorkItem, record models.Record) error {
	if wi.RowAction == models.RowActionDelete {
		if len(wi.Questions) != 1 {
			return invalidStep("a step that deletes a row asks one question, the reason")
		}
		reason := wi.Questions[0]
		if reason.AnswerType != models.AnswerFreeText || !reason.Required {
			return invalidStep("the question of a step that deletes a row is a required free_text question")
		}
		return nil
	}
	if wi.FormName != record.FormName || !reflect.DeepEqual(withoutFieldIdentity(wi.Questions), withoutFieldIdentity(record.Questions)) {
		return invalidStep("a step that %ss a row takes the form of the record %s: %q and its questions", wi.RowAction, record.Identity(), record.FormName)
	}
	return nil
}

func (m *agencyManager) refuseStepBesideRowStep(ctx context.Context, tx *gorm.DB, wi models.WorkItem) error {
	others, err := list[models.WorkItem](m, scopeQuery(tx.WithContext(ctx).Table(m.table(roleWorkItem)), wi.DraftID).
		Where("workflow_id = ? AND id <> ?", wi.WorkflowID, wi.ID), roleWorkItem)
	if err != nil {
		return err
	}
	for _, other := range others {
		if hasRowFields(wi) || hasRowFields(other) {
			return invalidStep("a workflow with a row step holds that one step only, and %s has %s", QuoteIfUnsafe(wi.WorkflowID), other.Code)
		}
	}
	return nil
}
