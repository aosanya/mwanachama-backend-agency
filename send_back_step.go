package mwanachamaagency

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func hasSendBackFields(wi models.WorkItem) bool {
	return wi.SendBackDecision != "" || wi.SendBackReason != ""
}

func validateSendBackShape(wi models.WorkItem) error {
	if !hasSendBackFields(wi) {
		return nil
	}
	switch {
	case wi.SendBackDecision == "":
		return invalidStep("a step that can be sent back names the yes or no question that holds the decision")
	case wi.SendBackReason == "":
		return invalidStep("a step that can be sent back names the free text question that holds the reason")
	case wi.Kind != models.StepKindFillForm:
		return invalidStep("only a fill_form step can be sent back, not a step of kind %q", wi.Kind)
	case wi.RoleID == "":
		return invalidStep("a step that can be sent back is acted on by a role")
	case hasRowFields(wi):
		return invalidStep("a step that writes a row of a record cannot be sent back")
	}
	if err := sendBackQuestionFits(wi, wi.SendBackDecision, models.AnswerYesNo, "decision"); err != nil {
		return err
	}
	return sendBackQuestionFits(wi, wi.SendBackReason, models.AnswerFreeText, "reason")
}

func sendBackQuestionFits(wi models.WorkItem, prompt string, want models.AnswerType, part string) error {
	for _, q := range wi.Questions {
		if q.Prompt != prompt {
			continue
		}
		if q.AnswerType != want {
			return invalidStep("the question %q holds the %s of a send back, so it is a %s question, not %s", prompt, part, want, q.AnswerType)
		}
		return nil
	}
	return invalidStep("the %s of a send back is the question %q, which the step's form does not ask", part, prompt)
}
