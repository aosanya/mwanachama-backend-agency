package mwanachamaagency

import "github.com/aosanya/mwanachama-backend-agency/models"

func registerPostStepCopyRules() {
	RegisterStepKind(models.StepKindPost, StepKindHooks{
		Remap:        remapPostFields,
		ToDocument:   postToDocument,
		FromDocument: postFromDocument,
		Hash:         postHashRecords,
	})
}

func remapPostFields(wi *models.WorkItem, ids StepIDMaps) {
	wi.AmountStepID = remap(ids.Steps, wi.AmountStepID)
}

func postToDocument(wi models.WorkItem, out *models.DocumentWorkItem) {
	out.AmountStepID = wi.AmountStepID
	out.DebitAccount = wi.DebitAccount
	out.CreditAccount = wi.CreditAccount
	out.Reason = wi.Reason
}

func postFromDocument(in models.DocumentWorkItem, row *models.WorkItem, ids StepIDMaps) {
	row.AmountStepID = remap(ids.Steps, in.AmountStepID)
	row.DebitAccount = in.DebitAccount
	row.CreditAccount = in.CreditAccount
	row.Reason = in.Reason
}

func postHashRecords(wi models.WorkItem) []string {
	if wi.AmountStepID == "" && wi.DebitAccount == "" && wi.CreditAccount == "" && wi.Reason == "" {
		return nil
	}
	return []string{hashRecord("WorkItemPost", map[string]any{
		"id": wi.ID, "amount_step_id": wi.AmountStepID, "debit_account": wi.DebitAccount,
		"credit_account": wi.CreditAccount, "reason": wi.Reason,
	})}
}
