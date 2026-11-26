package mwanachamaagency

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func hasSubjectSettings(wi models.WorkItem) bool {
	return len(wi.SetsSubject) > 0
}

func validateSubjectSettingShape(wi models.WorkItem) error {
	if !hasSubjectSettings(wi) {
		return nil
	}
	switch {
	case wi.Kind != models.StepKindFillForm:
		return invalidStep("only a fill_form step sets a choice on the subject record, not a step of kind %q", wi.Kind)
	case wi.RoleID == "":
		return invalidStep("a step that sets a choice on the subject record is acted on by a role")
	case hasRowFields(wi):
		return invalidStep("a step that writes a row of a record cannot also set a choice on the subject record")
	}
	asked := make(map[string]bool, len(wi.Questions))
	for _, q := range wi.Questions {
		asked[q.Prompt] = true
	}
	for i, setting := range wi.SetsSubject {
		label := fmt.Sprintf("setting %d", i+1)
		if err := validateRule(label, setting.When); err != nil {
			return err
		}
		if !asked[setting.When.Question] {
			return invalidStep("%s asks %q, which the step's form does not", label, setting.When.Question)
		}
		if strings.TrimSpace(setting.Field) == "" {
			return invalidStep("%s names no field of the subject record", label)
		}
		if strings.TrimSpace(setting.To) == "" {
			return invalidStep("%s names no option to set", label)
		}
	}
	return nil
}

func subjectSettingsFit(settings []models.SubjectSetting, subjectCode string, subject *models.Record) error {
	if subject == nil {
		if subjectCode == "" {
			return invalidStep("the workflow of this step has no subject record, so the step cannot set a choice on one")
		}
		return invalidStep("the subject record %s is not a record of this draft", QuoteIfUnsafe(subjectCode))
	}
	for i, setting := range settings {
		at := slices.IndexFunc(subject.Questions, func(q models.StepQuestion) bool { return q.Prompt == setting.Field })
		if at < 0 {
			return invalidStep("setting %d names the field %q, which the record %s does not ask", i+1, setting.Field, subject.Name)
		}
		field := subject.Questions[at]
		if field.AnswerType != models.AnswerSingleChoice {
			return invalidStep("setting %d names the field %q of the record %s, which is not a choice", i+1, setting.Field, subject.Name)
		}
		if !slices.Contains(field.Options, setting.To) {
			return invalidStep("setting %d sets %q, which is not an option of %q in the record %s", i+1, setting.To, setting.Field, subject.Name)
		}
	}
	return nil
}

func (m *agencyManager) validateSubjectSettingReferences(ctx context.Context, tx *gorm.DB, wi models.WorkItem) error {
	if !hasSubjectSettings(wi) {
		return nil
	}
	var workflow models.Workflow
	q := scopeQuery(tx.WithContext(ctx).Table(m.table(roleWorkflow)), wi.DraftID).Where("id = ?", wi.WorkflowID)
	if err := m.take(q, roleWorkflow, &workflow, ErrWorkflowNotFound); err != nil {
		return err
	}
	var subject *models.Record
	if workflow.SubjectRecord != "" {
		var record models.Record
		rq := scopeQuery(tx.WithContext(ctx).Table(m.table(roleRecord)), wi.DraftID).Where("key = ?", workflow.SubjectRecord)
		if err := m.take(rq, roleRecord, &record, ErrRecordNotFound); err == nil {
			subject = &record
		}
	}
	return subjectSettingsFit(wi.SetsSubject, workflow.SubjectRecord, subject)
}

func (s *issueScan) checkSubjectSettings() error {
	steps, err := list[models.WorkItem](s.m, s.scoped(roleWorkItem), roleWorkItem)
	if err != nil {
		return err
	}
	workflows, err := list[models.Workflow](s.m, s.scoped(roleWorkflow), roleWorkflow)
	if err != nil {
		return err
	}
	records, err := list[models.Record](s.m, s.scoped(roleRecord), roleRecord)
	if err != nil {
		return err
	}
	for _, wi := range steps {
		if !hasSubjectSettings(wi) || wi.Deleted {
			continue
		}
		subjectCode := ""
		for _, w := range workflows {
			if w.ID == wi.WorkflowID {
				subjectCode = w.SubjectRecord
			}
		}
		var subject *models.Record
		for i := range records {
			if subjectCode != "" && records[i].Key == subjectCode {
				subject = &records[i]
			}
		}
		if err := subjectSettingsFit(wi.SetsSubject, subjectCode, subject); err != nil {
			s.add(models.IssueSeverityError, "work_item", wi.ID, "Work item %s %q: %s", wi.Code, wi.Title, strings.TrimPrefix(err.Error(), ErrInvalidWorkItem.Error()+": "))
		}
	}
	return nil
}

func validateDocumentSubjectSettings(doc models.AgencyDocument) error {
	subjectOf := make(map[string]string, len(doc.Workflows))
	for _, w := range doc.Workflows {
		subjectOf[w.ID] = w.SubjectRecord
	}
	for _, wi := range doc.WorkItems {
		if len(wi.SetsSubject) == 0 {
			continue
		}
		code := subjectOf[wi.WorkflowID]
		var subject *models.Record
		for _, r := range doc.Records {
			if code != "" && r.Identity() == code {
				subject = &models.Record{Name: r.Name, Questions: r.Questions}
			}
		}
		if err := subjectSettingsFit(wi.SetsSubject, code, subject); err != nil {
			return fmt.Errorf("the step %s: %w", wi.Code, err)
		}
	}
	return nil
}
