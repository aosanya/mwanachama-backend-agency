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
	out.Account = wi.Account
	out.CounterAccount = wi.CounterAccount
	out.Direction = wi.Direction
	out.Reason = wi.Reason
}

func postFromDocument(in models.DocumentWorkItem, row *models.WorkItem, ids StepIDMaps) {
	row.AmountStepID = remap(ids.Steps, in.AmountStepID)
	row.Account = in.Account
	row.CounterAccount = in.CounterAccount
	row.Direction = in.Direction
	row.Reason = in.Reason
}

func postHashRecords(wi models.WorkItem) []string {
	if wi.AmountStepID == "" && wi.Account == "" && wi.CounterAccount == "" && wi.Direction == "" && wi.Reason == "" {
		return nil
	}
	return []string{hashRecord("WorkItemPost", map[string]any{
		"id": wi.ID, "amount_step_id": wi.AmountStepID, "account": wi.Account,
		"counter_account": wi.CounterAccount, "direction": wi.Direction, "reason": wi.Reason,
	})}
}
