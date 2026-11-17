package mwanachamaagency

import (
	"sort"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func registerStepCopyRules() {
	RegisterStepKind(models.StepKindCompute, StepKindHooks{
		Remap:        remapComputeFields,
		ToDocument:   computeToDocument,
		FromDocument: computeFromDocument,
		Hash:         computeHashRecords,
	})
	registerPostStepCopyRules()
	registerHoldStepCopyRules()
}

func init() { registerStepCopyRules() }

func remapComputeFields(wi *models.WorkItem, ids StepIDMaps) {
	wi.Bindings = remapBindings(wi.Bindings, ids)
	wi.OnFailure = remapFailure(wi.OnFailure, ids.Steps)
}

func computeToDocument(wi models.WorkItem, out *models.DocumentWorkItem) {
	out.Function = wi.Function
	out.Bindings = copyBindings(wi.Bindings)
	out.ResultName = wi.ResultName
	out.OnFailure = copyFailure(wi.OnFailure)
}

func computeFromDocument(in models.DocumentWorkItem, row *models.WorkItem, ids StepIDMaps) {
	row.Function = in.Function
	row.Bindings = remapBindings(in.Bindings, ids)
	row.ResultName = in.ResultName
	row.OnFailure = remapFailure(in.OnFailure, ids.Steps)
}

func computeHashRecords(wi models.WorkItem) []string {
	if wi.Function == "" && len(wi.Bindings) == 0 && wi.ResultName == "" && wi.OnFailure == nil {
		return nil
	}
	fields := map[string]any{
		"id": wi.ID, "function": wi.Function, "result_name": wi.ResultName,
		"bindings": sortedBindings(wi.Bindings),
	}
	if wi.OnFailure != nil {
		fields["on_failure"] = *wi.OnFailure
	}
	return []string{hashRecord("WorkItemCompute", fields)}
}

func copyBindings(bindings []models.StepBinding) []models.StepBinding {
	if len(bindings) == 0 {
		return nil
	}
	return append([]models.StepBinding(nil), bindings...)
}

func copyFailure(failure *models.StepFailure) *models.StepFailure {
	if failure == nil {
		return nil
	}
	copied := *failure
	return &copied
}

func remapBindings(bindings []models.StepBinding, ids StepIDMaps) []models.StepBinding {
	out := copyBindings(bindings)
	for i := range out {
		switch out[i].SourceType {
		case models.SourceAnswer:
			out[i].StepID = remap(ids.Steps, out[i].StepID)
		case models.SourceRecordField:
			out[i].RecordID = remap(ids.Records, out[i].RecordID)
		}
	}
	return out
}

func remapFailure(failure *models.StepFailure, steps map[string]string) *models.StepFailure {
	out := copyFailure(failure)
	if out != nil && out.Choice == models.FailureGoToStep {
		out.StepID = remap(steps, out.StepID)
	}
	return out
}

func sortedBindings(bindings []models.StepBinding) []models.StepBinding {
	out := copyBindings(bindings)
	sort.SliceStable(out, func(i, j int) bool { return bindingLess(out[i], out[j]) })
	return out
}

func bindingLess(a, b models.StepBinding) bool {
	if a.Input != b.Input {
		return a.Input < b.Input
	}
	if a.SourceType != b.SourceType {
		return a.SourceType < b.SourceType
	}
	if a.StepID != b.StepID {
		return a.StepID < b.StepID
	}
	if a.RecordID != b.RecordID {
		return a.RecordID < b.RecordID
	}
	if a.Question != b.Question {
		return a.Question < b.Question
	}
	return a.Value < b.Value
}
