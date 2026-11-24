package mwanachamaagency

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/aosanya/mwanachama-backend-shared/answerrules"
)

func invalidRecord(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidRecord, fmt.Sprintf(format, args...))
}

func invalidRow(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidRecordRow, fmt.Sprintf(format, args...))
}

func validateRecordShape(r models.Record) error {
	if strings.TrimSpace(r.Name) == "" {
		return invalidRecord("a record has a name")
	}
	if err := refuseGoneRoleFields(r.Code, r.Name, r.CarriedGoneRoleFields()); err != nil {
		return err
	}
	if r.WorkflowID == "" {
		return invalidRecord("a record belongs to a workflow")
	}
	return validateForm(invalidRecord, fmt.Sprintf("the record %q", r.Name), r.FormName, r.Questions)
}

func (m *agencyManager) validateRecord(ctx context.Context, tx *gorm.DB, r models.Record) error {
	if err := validateRecordShape(r); err != nil {
		return err
	}
	var workflow models.Workflow
	workflowQuery := scopeQuery(tx.WithContext(ctx).Table(m.table(roleWorkflow)), r.DraftID).Where("id = ?", r.WorkflowID)
	if err := m.take(workflowQuery, roleWorkflow, &workflow, ErrWorkflowNotFound); err != nil {
		return err
	}
	if workflow.Generated {
		return invalidRecord("a generated workflow keeps no record of its own")
	}
	if err := m.validateStandIns(ctx, tx, r); err != nil {
		return err
	}
	var same int64
	err := scopeQuery(tx.WithContext(ctx).Table(m.table(roleRecord)), r.DraftID).
		Where("workflow_id = ? AND LOWER(name) = ? AND id <> ?", r.WorkflowID, strings.ToLower(strings.TrimSpace(r.Name)), r.ID).
		Count(&same).Error
	if err != nil {
		return err
	}
	if same > 0 {
		return invalidRecord("the workflow %q already keeps a record called %q", workflow.Name, r.Name)
	}
	return nil
}

func (m *agencyManager) CreateRecord(ctx context.Context, r models.Record) (models.Record, error) {
	if err := m.validateOpenDraftID(ctx, r.DraftID); err != nil {
		return models.Record{}, err
	}
	r.Name = strings.TrimSpace(r.Name)
	r.ItemName = strings.TrimSpace(r.ItemName)
	settled, err := settleFields(fieldSettlement{makeKey: randomFieldKey}, r.Questions)
	if err != nil {
		return models.Record{}, fmt.Errorf("CreateRecord: %w", err)
	}
	r.Questions = settled
	if err := m.validateRecord(ctx, m.db, r); err != nil {
		return models.Record{}, fmt.Errorf("CreateRecord: %w", err)
	}
	r.ID = newID()
	r.Deleted = false
	r.CreatedAt = models.NowRFC3339()
	r.LastUpdated = r.CreatedAt

	if r, err = m.insertRecord(ctx, r); err != nil {
		return models.Record{}, fmt.Errorf("CreateRecord: %w", err)
	}
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return m.ensureGenerated(ctx, tx, r.DraftID) })
	if err != nil {
		return models.Record{}, fmt.Errorf("CreateRecord: %w", err)
	}
	return r, nil
}

func (m *agencyManager) findRecord(ctx context.Context, tx *gorm.DB, id string) (models.Record, error) {
	var r models.Record
	if err := m.find(ctx, tx, roleRecord, id, &r, ErrRecordNotFound); err != nil {
		return models.Record{}, err
	}
	return r, nil
}

func (m *agencyManager) GetRecord(ctx context.Context, id string) (models.Record, error) {
	return m.findRecord(ctx, m.db, id)
}

func (m *agencyManager) ListRecords(ctx context.Context, draftID string, filter RecordFilter) ([]models.Record, error) {
	q := scopeQuery(m.q(ctx, roleRecord).Order("created_at"), draftID)
	q = likeFilter(q, "code", filter.Code)
	q = likeFilter(q, "name", filter.Name)
	q = eqFilter(q, "workflow_id", filter.WorkflowID)

	out, err := list[models.Record](m, q, roleRecord)
	if err != nil {
		return nil, fmt.Errorf("ListRecords: %w", err)
	}
	return out, nil
}

func (m *agencyManager) UpdateRecord(ctx context.Context, r models.Record) (models.Record, error) {
	var out models.Record
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := m.findRecord(ctx, tx, r.ID)
		if err != nil {
			return err
		}
		if err := m.requireOpenDraft(ctx, tx, existing.DraftID); err != nil {
			return err
		}
		if err := m.settleRecordRename(ctx, tx, existing, &r); err != nil {
			return err
		}
		r.DraftID = existing.DraftID
		r.CreatedAt = existing.CreatedAt
		r.LastUpdated = models.NowRFC3339()
		r.Name = strings.TrimSpace(r.Name)
		r.ItemName = strings.TrimSpace(r.ItemName)
		settled, err := settleFields(fieldSettlement{
			recordCode: existing.Identity(), before: keyedFields(existing.Identity(), existing.Questions), known: true, makeKey: randomFieldKey,
		}, r.Questions)
		if err != nil {
			return err
		}
		r.Questions = settled

		if err := m.validateRecord(ctx, tx, r); err != nil {
			return err
		}
		if err := m.writeUpdate(ctx, tx, roleRecord, r.ID, r, ErrRecordNotFound); err != nil {
			return err
		}
		out = r
		if err := m.renameRecordCode(ctx, tx, existing, r.Code); err != nil {
			return err
		}
		return m.ensureGenerated(ctx, tx, existing.DraftID)
	})
	if err != nil {
		return models.Record{}, fmt.Errorf("UpdateRecord: %w", err)
	}
	return out, nil
}

func (m *agencyManager) DeleteRecord(ctx context.Context, id string) error {
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := m.findRecord(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := m.requireOpenDraft(ctx, tx, existing.DraftID); err != nil {
			return err
		}
		if err := m.deleteWhere(ctx, tx, roleRecordRow, "record_id = ?", id); err != nil {
			return err
		}
		if err := m.deleteWhere(ctx, tx, roleRecord, "id = ?", id); err != nil {
			return err
		}
		return m.ensureGenerated(ctx, tx, existing.DraftID)
	})
	if err != nil {
		return fmt.Errorf("DeleteRecord: %w", err)
	}
	return nil
}

const amendmentPromptPrefix = "Changed: "

func validateRowAnswers(ctx context.Context, record models.Record, answers map[string]string) (map[string]string, error) {
	allowance := rowAllowanceOf(ctx)
	fields := recordFieldsOf(record)
	kept := make(map[string]string, len(answers))
	named := map[string]string{}
	for name, answer := range answers {
		if q, ok := fields.named(name); ok {
			if earlier, twice := named[q.Key]; twice {
				return nil, invalidRow("the question %q is answered twice, as %q and as %q", q.Prompt, earlier, name)
			}
			named[q.Key] = name
			kept[q.Key] = answer
			continue
		}
		if _, extra := allowance.extraNamed(name); !extra && !isApprovalStampPrompt(name) && !strings.HasPrefix(name, amendmentPromptPrefix) {
			return nil, invalidRow("%q is not a question of the form", name)
		}
		kept[name] = answer
	}
	asked := make(map[string]bool, len(record.Questions))
	for _, q := range record.Questions {
		asked[q.Prompt] = true
	}
	for _, extra := range allowance.Extra {
		if !asked[extra.Prompt] && extra.Required && strings.TrimSpace(kept[extra.Prompt]) == "" {
			return nil, invalidRow("the detail %q is required", extra.Prompt)
		}
	}
	for _, q := range keyedFields(record.Identity(), record.Questions) {
		answer := strings.TrimSpace(kept[q.Key])
		if answer == "" {
			if q.Required && !allowance.waives(q.Prompt) {
				return nil, invalidRow("the question %q is required", q.Prompt)
			}
			continue
		}
		if err := validateRowAnswer(q, answer); err != nil {
			return nil, err
		}
	}
	return kept, nil
}

func validateRowAnswer(q models.StepQuestion, answer string) error {
	switch q.AnswerType {
	case models.AnswerYesNo:
		if a := strings.ToLower(answer); a != "yes" && a != "no" {
			return invalidRow("the answer to %q is yes or no, got %q", q.Prompt, answer)
		}
	case models.AnswerNumber, models.AnswerCurrency:
		if _, err := strconv.ParseFloat(answer, 64); err != nil {
			return invalidRow("the answer to %q is a number, got %q", q.Prompt, answer)
		}
	case models.AnswerDate:
		if _, err := time.Parse("2006-01-02", answer); err != nil {
			return invalidRow("the answer to %q is a date as 2006-01-02, got %q", q.Prompt, answer)
		}
	case models.AnswerTimeRange:
		from, to, ok := answerrules.TimeRangeOfWords(answer)
		if !ok {
			return invalidRow("the answer to %q is a time range as from HH:MM to HH:MM, got %q", q.Prompt, answer)
		}
		if err := answerrules.CheckTimeRange(from, to); err != nil {
			return invalidRow("the answer to %q: %v", q.Prompt, err)
		}
	case models.AnswerSingleChoice:
		if len(q.Options) > 0 && !slices.Contains(q.Options, answer) {
			return invalidRow("the answer to %q is one of %v, got %q", q.Prompt, q.Options, answer)
		}
	case models.AnswerMultiSelect:
		return validateMultiSelectAnswer(q, answer)
	}
	return nil
}

const multiSelectSeparator = ";"

func multiSelectChoices(answer string) []string {
	choices := strings.Split(answer, multiSelectSeparator)
	for i, choice := range choices {
		choices[i] = strings.TrimSpace(choice)
	}
	return choices
}

func validateMultiSelectAnswer(q models.StepQuestion, answer string) error {
	chosen := map[string]bool{}
	for _, choice := range multiSelectChoices(answer) {
		if choice == "" || chosen[choice] {
			return invalidRow("the answer to %q is options joined with %q, each once, got %q", q.Prompt, multiSelectSeparator+" ", answer)
		}
		if len(q.Options) > 0 && !slices.Contains(q.Options, choice) {
			return invalidRow("the answer to %q is options from %v joined with %q, got %q", q.Prompt, q.Options, multiSelectSeparator+" ", choice)
		}
		chosen[choice] = true
	}
	return nil
}

func (m *agencyManager) CreateRecordRow(ctx context.Context, row models.RecordRow) (models.RecordRow, error) {
	if row.DraftID == "" {
		return models.RecordRow{}, ErrDraftRequired
	}
	if _, err := m.findDraft(ctx, m.db, row.DraftID); err != nil {
		return models.RecordRow{}, err
	}
	var record models.Record
	q := scopeQuery(m.q(ctx, roleRecord), row.DraftID).Where("id = ?", row.RecordID)
	if err := m.take(q, roleRecord, &record, ErrRecordNotFound); err != nil {
		return models.RecordRow{}, fmt.Errorf("CreateRecordRow: %w", err)
	}
	if !m.roleAddsRows(ctx, m.db, row.DraftID, row.RoleID, record) {
		return models.RecordRow{}, fmt.Errorf("CreateRecordRow: %w: only a role that carries an add permission of %q may", ErrRecordRowRefused, record.Name)
	}
	kept, err := validateRowAnswers(ctx, record, row.Answers)
	if err != nil {
		return models.RecordRow{}, fmt.Errorf("CreateRecordRow: %w", err)
	}
	row.Answers = kept
	if row.Tags, err = cleanTags(row.Tags); err != nil {
		return models.RecordRow{}, fmt.Errorf("CreateRecordRow: %w", err)
	}
	row.PlaceRefs = recordFieldsOf(record).keyed(row.PlaceRefs)
	row.ID = newID()
	row.Deleted = false
	row.RecordCode = record.Identity()
	row.CreatedAt = rowClock().Format(models.TimeLayout)
	row.LastUpdated = row.CreatedAt
	row.CorrectedOn = ""
	if err := m.settleReplacement(ctx, &row, record); err != nil {
		return models.RecordRow{}, fmt.Errorf("CreateRecordRow: %w", err)
	}
	if err := m.settleSubjectRow(ctx, &row); err != nil {
		return models.RecordRow{}, fmt.Errorf("CreateRecordRow: %w", err)
	}
	if err := m.refuseDuplicate(ctx, row); err != nil {
		return models.RecordRow{}, fmt.Errorf("CreateRecordRow: %w", err)
	}
	if err := duplicateOr(m.st.Insert(ctx, roleRecordRow, row), ErrStepAlreadySent); err != nil {
		return models.RecordRow{}, fmt.Errorf("CreateRecordRow: %w", err)
	}
	return row, nil
}

func (m *agencyManager) ListRecordRows(ctx context.Context, draftID string, filter RecordRowFilter) ([]models.RecordRow, error) {
	q := scopeQuery(m.q(ctx, roleRecordRow).Order("created_at"), draftID)
	q = eqFilter(q, "record_id", filter.RecordID)
	q = eqFilter(q, "record_code", m.recordKeyNamed(ctx, draftID, filter.RecordCode))
	q = eqFilter(q, "order_id", filter.OrderID)
	q = eqFilter(q, "subject_row_id", filter.SubjectRowID)
	q = tagFilter(q, filter.Tags)
	if !filter.IncludeRetired {
		q = notRetired(q)
	}
	if !filter.IncludeReturned {
		q = notReturned(q)
	}

	out, err := list[models.RecordRow](m, q, roleRecordRow)
	if err != nil {
		return nil, fmt.Errorf("ListRecordRows: %w", err)
	}
	kept := make([]models.RecordRow, 0, len(out))
	for _, row := range out {
		if row.Tags == nil {
			row.Tags = []string{}
		}
		if reachesEveryRow(filter.Tags) || carriesAny(row.Tags, filter.Tags) {
			kept = append(kept, row)
		}
	}
	return kept, nil
}
