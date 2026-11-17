package mwanachamaagency

import "github.com/aosanya/mwanachama-backend-agency/models"

func registerHoldStepCopyRules() {
	hooks := StepKindHooks{
		Remap:        remapHoldFields,
		ToDocument:   holdToDocument,
		FromDocument: holdFromDocument,
		Hash:         holdHashRecords,
	}
	for _, kind := range []models.StepKind{models.StepKindCompute, models.StepKindPost, models.StepKindYesNo} {
		RegisterStepKind(kind, hooks)
	}
}

func remapHoldFields(wi *models.WorkItem, ids StepIDMaps) {
	wi.HeldStepID = remap(ids.Steps, wi.HeldStepID)
}

func copyHold(hold *models.StepHold) *models.StepHold {
	if hold == nil {
		return nil
	}
	copied := *hold
	return &copied
}

func holdToDocument(wi models.WorkItem, out *models.DocumentWorkItem) {
	out.Hold = copyHold(wi.Hold)
	out.HeldStepID = wi.HeldStepID
}

func holdFromDocument(in models.DocumentWorkItem, row *models.WorkItem, ids StepIDMaps) {
	row.Hold = copyHold(in.Hold)
	row.HeldStepID = remap(ids.Steps, in.HeldStepID)
}

func holdHashRecords(wi models.WorkItem) []string {
	if wi.Hold == nil && wi.HeldStepID == "" {
		return nil
	}
	fields := map[string]any{"id": wi.ID, "held_step_id": wi.HeldStepID}
	if wi.Hold != nil {
		fields["hold"] = *wi.Hold
	}
	return []string{hashRecord("WorkItemHold", fields)}
}
