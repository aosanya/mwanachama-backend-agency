package mwanachamaagency

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func validateRecordCodeShape(wi models.WorkItem) error {
	if wi.RecordCode == "" {
		return nil
	}
	if wi.Kind != models.StepKindFillForm {
		return invalidStep("only a fill_form step names the record it writes, not a step of kind %q", wi.Kind)
	}
	return nil
}
