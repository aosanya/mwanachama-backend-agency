package models

import "strings"

type StepKind string

const (
	StepKindFillForm    StepKind = "fill_form"
	StepKindSendForm    StepKind = "send_form"
	StepKindNotify      StepKind = "notify"
	StepKindYesNo       StepKind = "yes_no"
	StepKindRoute       StepKind = "route"
	StepKindRunWorkflow StepKind = "run_workflow"
)

var stepKinds = []StepKind{
	StepKindFillForm, StepKindSendForm, StepKindNotify,
	StepKindYesNo, StepKindRoute, StepKindRunWorkflow,
}

func StepKinds() []StepKind { return append([]StepKind(nil), stepKinds...) }

func (k StepKind) Valid() bool {
	for _, known := range stepKinds {
		if k == known {
			return true
		}
	}
	return false
}

func (k StepKind) TakesForm() bool {
	return k == StepKindFillForm || k == StepKindSendForm || k == StepKindNotify || k == StepKindYesNo
}

func (k StepKind) AgencyOnly() bool {
	return k == StepKindSendForm || k == StepKindRoute
}

type AnswerType string

const (
	AnswerYesNo        AnswerType = "yes_no"
	AnswerSingleChoice AnswerType = "single_choice"
	AnswerFreeText     AnswerType = "free_text"
	AnswerDate         AnswerType = "date"
	AnswerNumber       AnswerType = "number"
	AnswerCurrency     AnswerType = "currency"
)

var answerTypes = []AnswerType{
	AnswerYesNo, AnswerSingleChoice, AnswerFreeText,
	AnswerDate, AnswerNumber, AnswerCurrency,
}

func (a AnswerType) Valid() bool {
	for _, known := range answerTypes {
		if a == known {
			return true
		}
	}
	return false
}

type StepQuestion struct {
	Prompt      string     `json:"prompt"`
	AnswerType  AnswerType `json:"answer_type"`
	Required    bool       `json:"required"`
	Helper      string     `json:"helper,omitempty"`
	Options     []string   `json:"options,omitempty"`
	OptionsFrom string     `json:"options_from,omitempty"`
	Fixed       string     `json:"fixed,omitempty"`
}

func lowerFirst(text string) string {
	if text == "" {
		return text
	}
	return strings.ToLower(text[:1]) + text[1:]
}

func ActionOf(kind StepKind, formName, runWorkflowName string) string {
	switch kind {
	case StepKindFillForm:
		return "Fills the " + formName
	case StepKindSendForm:
		return "Sends the " + lowerFirst(formName)
	case StepKindNotify:
		return "Notifies with the " + lowerFirst(formName)
	case StepKindYesNo:
		return "Gives " + lowerFirst(formName)
	case StepKindRoute:
		return "Routes the answers"
	case StepKindRunWorkflow:
		return "Starts " + runWorkflowName
	}
	return ""
}
