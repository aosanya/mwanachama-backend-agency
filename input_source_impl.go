package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

var ErrInvalidInputType = errors.New("mwanachamaagency: input_type is not a type a source can have")

type InputSourceFilter struct {
	InputType string `query:"input_type"`
}

type InputSource struct {
	Source     string            `json:"source"`
	Type       string            `json:"type"`
	StepID     string            `json:"step_id,omitempty"`
	StepTitle  string            `json:"step_title,omitempty"`
	RecordID   string            `json:"record_id,omitempty"`
	RecordName string            `json:"record_name,omitempty"`
	Question   string            `json:"question,omitempty"`
	AnswerType models.AnswerType `json:"answer_type,omitempty"`
}

type InputSources struct {
	Answer        []InputSource `json:"answer"`
	RecordField   []InputSource `json:"record_field"`
	LedgerBalance []InputSource `json:"ledger_balance"`
	Fixed         []InputSource `json:"fixed"`
}

const (
	sourceAnswer        = "answer"
	sourceRecordField   = "record_field"
	sourceLedgerBalance = "ledger_balance"
	sourceFixed         = "fixed"
	typeCurrency        = "currency"
)

var sourceTypeOfAnswer = map[models.AnswerType]string{
	models.AnswerNumber:       "number",
	models.AnswerCurrency:     typeCurrency,
	models.AnswerDate:         "date",
	models.AnswerTime:         "time",
	models.AnswerYesNo:        "yes_no",
	models.AnswerFreeText:     "text",
	models.AnswerSingleChoice: "text",
}

func knownSourceType(t string) bool {
	for _, known := range sourceTypeOfAnswer {
		if known == t {
			return true
		}
	}
	return false
}

func fitsInput(sourceType, inputType string) bool {
	return inputType == "" || sourceType == inputType
}

func (m *agencyManager) ListInputSources(ctx context.Context, draftID, workItemID string, filter InputSourceFilter) (InputSources, error) {
	if filter.InputType != "" && !knownSourceType(filter.InputType) {
		return InputSources{}, fmt.Errorf("ListInputSources: %w: %q", ErrInvalidInputType, filter.InputType)
	}
	target, err := m.findWorkItem(ctx, m.db, workItemID)
	if err != nil || target.DraftID != draftID {
		return InputSources{}, fmt.Errorf("ListInputSources: %w", ErrWorkItemNotFound)
	}
	steps, err := m.ListWorkItems(ctx, draftID, WorkItemFilter{})
	if err != nil {
		return InputSources{}, fmt.Errorf("ListInputSources: %w", err)
	}
	records, err := m.ListRecords(ctx, draftID, RecordFilter{})
	if err != nil {
		return InputSources{}, fmt.Errorf("ListInputSources: %w", err)
	}
	out := InputSources{
		Answer:        []InputSource{},
		RecordField:   []InputSource{},
		LedgerBalance: []InputSource{},
		Fixed:         []InputSource{{Source: sourceFixed, Type: filter.InputType}},
	}
	for _, step := range steps {
		if step.WorkflowID != target.WorkflowID || step.Ordinality >= target.Ordinality {
			continue
		}
		for _, q := range step.Questions {
			t, ok := sourceTypeOfAnswer[q.AnswerType]
			if ok && fitsInput(t, filter.InputType) {
				out.Answer = append(out.Answer, InputSource{
					Source: sourceAnswer, Type: t, StepID: step.ID, StepTitle: step.Title,
					Question: q.Prompt, AnswerType: q.AnswerType,
				})
			}
		}
	}
	for _, record := range records {
		for _, q := range record.Questions {
			t, ok := sourceTypeOfAnswer[q.AnswerType]
			if ok && fitsInput(t, filter.InputType) {
				out.RecordField = append(out.RecordField, InputSource{
					Source: sourceRecordField, Type: t, RecordID: record.ID, RecordName: record.Name,
					Question: q.Prompt, AnswerType: q.AnswerType,
				})
			}
		}
	}
	if fitsInput(typeCurrency, filter.InputType) {
		out.LedgerBalance = append(out.LedgerBalance, InputSource{Source: sourceLedgerBalance, Type: typeCurrency})
	}
	return out, nil
}
