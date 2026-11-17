package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

var ErrFunctionNotFound = errors.New("mwanachamaagency: function not found")

type FunctionInput struct {
	Name string
	Type string
}

type FunctionCatalog interface {
	InputsOf(ctx context.Context, name string, version int) ([]FunctionInput, bool, error)
}

func WithFunctionCatalog(catalog FunctionCatalog) Option {
	return func(m *agencyManager) { m.functions = catalog }
}

func init() {
	RegisterStepShapeCheck(computeFieldsOnTheRightKind)
	RegisterStepKind(models.StepKindCompute, StepKindHooks{
		Shape:      computeShape,
		References: computeReferences,
	})
	RegisterIssueCheck(computeIssueCheck)
}

const sourceTypesSentence = "answer, record_field, ledger_balance and fixed"

var wholeNumber = regexp.MustCompile(`^[0-9]+$`)
var decimalNumber = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)

func hasComputeFields(wi models.WorkItem) bool {
	return wi.Function != "" || len(wi.Bindings) > 0 || wi.ResultName != ""
}

func computeFieldsOnTheRightKind(wi models.WorkItem) error {
	if wi.Kind != models.StepKindCompute {
		if !hasComputeFields(wi) {
			return nil
		}
		if wi.Kind == "" {
			return invalidStep("a step with no kind takes no function, bindings or result name")
		}
		return invalidStep("a %s step takes no function, bindings or result name", wi.Kind)
	}
	if wi.FormName != "" || len(wi.Questions) > 0 || wi.RecipientRoleID != "" || wi.SentStepID != "" ||
		len(wi.Routes) > 0 || wi.OtherwiseStepID != "" || wi.RunWorkflowID != "" || wi.HasChatFields() || wi.Tool != "" {
		return invalidStep("a compute step takes no form, recipient, sent step, routes, workflow to run, chat fields or tool")
	}
	return nil
}

func parseFunction(text string) (string, int, error) {
	if strings.TrimSpace(text) == "" {
		return "", 0, invalidStep("a compute step names the function it calls, written library.function@version")
	}
	at := strings.LastIndex(text, "@")
	if at < 0 {
		return "", 0, invalidStep("the function %q has no version, write it as library.function@version", text)
	}
	name, digits := text[:at], text[at+1:]
	if strings.TrimSpace(name) == "" || strings.ContainsAny(name, " \t\r\n") {
		return "", 0, invalidStep("the function %q has no name before its version", text)
	}
	version, err := strconv.Atoi(digits)
	if !wholeNumber.MatchString(digits) || err != nil {
		return "", 0, invalidStep("the version of the function %q is not a whole number", text)
	}
	return name, version, nil
}

func computeShape(wi models.WorkItem) error {
	if _, _, err := parseFunction(wi.Function); err != nil {
		return err
	}
	if strings.TrimSpace(wi.ResultName) == "" {
		return invalidStep("a compute step names the result it keeps, in result_name")
	}
	return bindingShapes(wi.Bindings)
}

func bindingShapes(bindings []models.StepBinding) error {
	seen := map[string]bool{}
	for i, b := range bindings {
		if strings.TrimSpace(b.Input) == "" {
			return invalidStep("binding %d names no input", i+1)
		}
		if seen[b.Input] {
			return invalidStep("input %q has two bindings, and only one source is allowed", b.Input)
		}
		seen[b.Input] = true
		if err := bindingSourceShape(b); err != nil {
			return err
		}
	}
	return nil
}

func bindingSourceShape(b models.StepBinding) error {
	switch b.SourceType {
	case models.SourceAnswer:
		if b.StepID == "" {
			return invalidStep("the answer for input %q names no step", b.Input)
		}
		if b.Question == "" {
			return invalidStep("the answer for input %q names no question", b.Input)
		}
	case models.SourceRecordField:
		if b.RecordID == "" {
			return invalidStep("the record field for input %q names no record", b.Input)
		}
		if b.Question == "" {
			return invalidStep("the record field for input %q names no question", b.Input)
		}
	case models.SourceLedgerBalance:
	case models.SourceFixed:
		if strings.TrimSpace(b.Value) == "" {
			return invalidStep("the fixed value for input %q is empty", b.Input)
		}
	default:
		return invalidStep("the binding for input %q has source type %q, which is none of %s", b.Input, b.SourceType, sourceTypesSentence)
	}
	return nil
}

func (l managerStepLookup) RecordOf(ctx context.Context, draftID, recordID string) (models.Record, bool, error) {
	q := scopeQuery(l.tx.WithContext(ctx).Table(l.m.table(roleRecord)), draftID).Where("id = ?", recordID)
	found, err := list[models.Record](l.m, q, roleRecord)
	if err != nil || len(found) == 0 {
		return models.Record{}, false, err
	}
	return found[0], true, nil
}

func (l managerStepLookup) computeStepsOf(ctx context.Context, draftID, workflowID string) ([]models.WorkItem, error) {
	q := scopeQuery(l.tx.WithContext(ctx).Table(l.m.table(roleWorkItem)), draftID).
		Where("workflow_id = ? AND kind = ?", workflowID, string(models.StepKindCompute))
	return list[models.WorkItem](l.m, q, roleWorkItem)
}

func (m *agencyManager) functionInputs(ctx context.Context, function string) ([]FunctionInput, bool, error) {
	if m.functions == nil {
		return nil, false, nil
	}
	name, version, err := parseFunction(function)
	if err != nil {
		return nil, false, nil
	}
	inputs, found, err := m.functions.InputsOf(ctx, name, version)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, fmt.Errorf("%w: the library has no function %s", ErrFunctionNotFound, function)
	}
	return inputs, true, nil
}

func typesOfInputs(inputs []FunctionInput) map[string]string {
	types := make(map[string]string, len(inputs))
	for _, in := range inputs {
		types[in.Name] = in.Type
	}
	return types
}

func computeReferences(ctx context.Context, lookup StepLookup, wi models.WorkItem) error {
	l, ok := lookup.(managerStepLookup)
	if !ok {
		return nil
	}
	functionInputs, known, err := l.m.functionInputs(ctx, wi.Function)
	if err != nil {
		return err
	}
	inputs := typesOfInputs(functionInputs)
	if err := l.uniqueResultName(ctx, wi); err != nil {
		return err
	}
	for _, b := range wi.Bindings {
		inputType, hasInput := inputs[b.Input]
		if known && !hasInput {
			return invalidStep("the function %s has no input %q", wi.Function, b.Input)
		}
		sourceType, err := l.sourceTypeOf(ctx, wi, b)
		if err != nil {
			return err
		}
		if known && sourceType != "" && !fitsInput(sourceType, inputType) {
			return invalidStep("%s, but input %q asks for %s", sourceDescription(b, sourceType), b.Input, inputType)
		}
		if known && b.SourceType == models.SourceFixed && !fixedFits(b.Value, inputType) {
			return invalidStep("the fixed value %q for input %q is not a %s", b.Value, b.Input, inputType)
		}
	}
	return nil
}

func sourceDescription(b models.StepBinding, sourceType string) string {
	if b.SourceType == models.SourceLedgerBalance {
		return "the ledger balance is " + sourceType
	}
	return fmt.Sprintf("the question %q is %s", b.Question, sourceType)
}

func (l managerStepLookup) uniqueResultName(ctx context.Context, wi models.WorkItem) error {
	steps, err := l.computeStepsOf(ctx, wi.DraftID, wi.WorkflowID)
	if err != nil {
		return err
	}
	for _, other := range steps {
		if other.ID != wi.ID && strings.EqualFold(strings.TrimSpace(other.ResultName), strings.TrimSpace(wi.ResultName)) {
			return invalidStep("the result name %q is already used by another compute step of this workflow", wi.ResultName)
		}
	}
	return nil
}

func (l managerStepLookup) sourceTypeOf(ctx context.Context, wi models.WorkItem, b models.StepBinding) (string, error) {
	switch b.SourceType {
	case models.SourceAnswer:
		return l.answerSourceType(ctx, wi, b)
	case models.SourceRecordField:
		return l.recordSourceType(ctx, wi, b)
	case models.SourceLedgerBalance:
		return typeCurrency, nil
	}
	return "", nil
}

func (l managerStepLookup) answerSourceType(ctx context.Context, wi models.WorkItem, b models.StepBinding) (string, error) {
	step, found, err := l.StepOf(ctx, wi.DraftID, b.StepID)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("%w: the answer for input %q comes from a step that is not in this draft", ErrWorkItemNotFound, b.Input)
	}
	where := fmt.Sprintf("the answer for input %q comes from the step %q", b.Input, step.Title)
	switch {
	case step.WorkflowID != wi.WorkflowID:
		return "", invalidStep("%s, which belongs to another workflow", where)
	case wi.ID != "" && step.ID == wi.ID:
		return "", invalidStep("the answer for input %q comes from the step itself", b.Input)
	case step.Ordinality >= wi.Ordinality:
		return "", invalidStep("%s, which does not come before this step", where)
	case step.FormName == "" || len(step.Questions) == 0:
		return "", invalidStep("%s, which has no form to read an answer from", where)
	}
	return questionSourceType(where, step.Questions, b.Question)
}

func (l managerStepLookup) recordSourceType(ctx context.Context, wi models.WorkItem, b models.StepBinding) (string, error) {
	record, found, err := l.RecordOf(ctx, wi.DraftID, b.RecordID)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("%w: the record field for input %q comes from a record that is not in this draft", ErrRecordNotFound, b.Input)
	}
	where := fmt.Sprintf("the record field for input %q comes from the record %q", b.Input, record.Name)
	return questionSourceType(where, record.Questions, b.Question)
}

func questionSourceType(where string, questions []models.StepQuestion, prompt string) (string, error) {
	for _, q := range questions {
		if q.Prompt != prompt {
			continue
		}
		sourceType, ok := sourceTypeOfAnswer[q.AnswerType]
		if !ok {
			return "", invalidStep("the question %q is a %s question, which cannot be a source", prompt, q.AnswerType)
		}
		return sourceType, nil
	}
	return "", invalidStep("%s, which does not ask %q", where, prompt)
}

func fixedFits(value, inputType string) bool {
	value = strings.TrimSpace(value)
	switch inputType {
	case "number", typeCurrency:
		return decimalNumber.MatchString(value)
	case "date":
		_, err := time.Parse("2006-01-02", value)
		return err == nil
	case "time":
		_, err := time.Parse("15:04", value)
		return err == nil
	case "yes_no":
		switch strings.ToLower(value) {
		case "yes", "no", "true", "false":
			return true
		}
		return false
	}
	return true
}

func computeIssueCheck(report StepScanReport, steps []models.WorkItem) error {
	r, ok := report.(scanReport)
	if !ok {
		return nil
	}
	var computing []models.WorkItem
	byID := make(map[string]models.WorkItem, len(steps))
	for _, step := range steps {
		byID[step.ID] = step
		if step.Kind == models.StepKindCompute {
			computing = append(computing, step)
		}
	}
	if len(computing) == 0 {
		return nil
	}
	records, err := list[models.Record](r.s.m, r.s.scoped(roleRecord), roleRecord)
	if err != nil {
		return err
	}
	recordByID := make(map[string]models.Record, len(records))
	for _, rec := range records {
		recordByID[rec.ID] = rec
	}
	for _, step := range computing {
		scan := computeScan{s: r.s, step: step, steps: byID, records: recordByID}
		if err := scan.run(); err != nil {
			return err
		}
	}
	return nil
}

type computeScan struct {
	s       *issueScan
	step    models.WorkItem
	steps   map[string]models.WorkItem
	records map[string]models.Record
}

func (c computeScan) add(format string, args ...any) {
	c.s.add(models.IssueSeverityError, "work_item", c.step.ID,
		fmt.Sprintf("Work item %s %q ", c.step.Code, c.step.Title)+format, args...)
}

func (c computeScan) run() error {
	if c.step.Function == "" {
		c.add("is a compute step that names no function")
	}
	if strings.TrimSpace(c.step.ResultName) == "" {
		c.add("is a compute step that names no result name")
	}
	inputs, known, err := c.functionInputs()
	if err != nil {
		return err
	}
	if known {
		c.inputsWithNoSource(inputs)
	}
	types := typesOfInputs(inputs)
	for _, b := range c.step.Bindings {
		c.binding(b, types, known)
	}
	return nil
}

func (c computeScan) functionInputs() ([]FunctionInput, bool, error) {
	if c.step.Function == "" {
		return nil, false, nil
	}
	inputs, known, err := c.s.m.functionInputs(c.s.ctx, c.step.Function)
	if errors.Is(err, ErrFunctionNotFound) {
		return nil, false, nil
	}
	return inputs, known, err
}

func (c computeScan) inputsWithNoSource(inputs []FunctionInput) {
	bound := make(map[string]bool, len(c.step.Bindings))
	for _, b := range c.step.Bindings {
		bound[b.Input] = true
	}
	for _, in := range inputs {
		if !bound[in.Name] {
			c.add("has no source for input %q", in.Name)
		}
	}
}

func (c computeScan) binding(b models.StepBinding, inputs map[string]string, known bool) {
	var sourceType string
	switch b.SourceType {
	case models.SourceAnswer:
		sourceType = c.answerSource(b)
	case models.SourceRecordField:
		sourceType = c.recordSource(b)
	case models.SourceLedgerBalance:
		sourceType = typeCurrency
	case models.SourceFixed:
		return
	default:
		c.add("has a binding for input %q with source type %q, which is none of %s", b.Input, b.SourceType, sourceTypesSentence)
		return
	}
	inputType, hasInput := inputs[b.Input]
	if sourceType == "" || !known || !hasInput || fitsInput(sourceType, inputType) {
		return
	}
	if b.SourceType == models.SourceLedgerBalance {
		c.add("reads input %q, which asks for %s, from the ledger balance, which is %s", b.Input, inputType, sourceType)
		return
	}
	c.add("reads input %q, which asks for %s, from the question %q, which is %s", b.Input, inputType, b.Question, sourceType)
}

func (c computeScan) answerSource(b models.StepBinding) string {
	src, ok := c.steps[b.StepID]
	switch {
	case !ok:
		c.add("reads input %q from a step that does not exist", b.Input)
		return ""
	case src.WorkflowID != c.step.WorkflowID:
		c.add("reads input %q from the step %q, which belongs to another workflow", b.Input, src.Title)
		return ""
	case src.ID == c.step.ID || src.Ordinality >= c.step.Ordinality:
		c.add("reads input %q from the step %q, which no longer comes before it", b.Input, src.Title)
		return ""
	}
	return c.questionSource(b, src.Questions, fmt.Sprintf("the step %q", src.Title))
}

func (c computeScan) recordSource(b models.StepBinding) string {
	rec, ok := c.records[b.RecordID]
	if !ok {
		c.add("reads input %q from a record that does not exist", b.Input)
		return ""
	}
	return c.questionSource(b, rec.Questions, fmt.Sprintf("the record %q", rec.Name))
}

func (c computeScan) questionSource(b models.StepBinding, questions []models.StepQuestion, owner string) string {
	for _, q := range questions {
		if q.Prompt != b.Question {
			continue
		}
		sourceType, ok := sourceTypeOfAnswer[q.AnswerType]
		if !ok {
			c.add("reads input %q from the question %q, which is a %s question and cannot be a source", b.Input, b.Question, q.AnswerType)
		}
		return sourceType
	}
	c.add("reads input %q from the question %q, which %s does not ask any more", b.Input, b.Question, owner)
	return ""
}
