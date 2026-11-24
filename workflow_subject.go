package mwanachamaagency

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func unknownSubjectRecord(code string) error {
	return invalidWorkflow("the subject record %s is not a record of this draft", QuoteIfUnsafe(code))
}

func (m *agencyManager) validateWorkflowSubject(ctx context.Context, tx *gorm.DB, draftID, subject string) error {
	if subject == "" {
		return nil
	}
	var record models.Record
	q := scopeQuery(tx.WithContext(ctx).Table(m.table(roleRecord)), draftID).Where("key = ?", subject)
	err := m.take(q, roleRecord, &record, ErrRecordNotFound)
	if errors.Is(err, ErrRecordNotFound) {
		return unknownSubjectRecord(subject)
	}
	return err
}

func validateDocumentSubjects(doc models.AgencyDocument) error {
	held := make(map[string]bool, len(doc.Records))
	for _, r := range doc.Records {
		held[r.Identity()] = true
	}
	for _, w := range doc.Workflows {
		if w.SubjectRecord != "" && !held[w.SubjectRecord] {
			return unknownSubjectRecord(w.SubjectRecord)
		}
	}
	return nil
}

func (s *issueScan) checkWorkflowSubjects() error {
	workflows, err := list[models.Workflow](s.m, s.scoped(roleWorkflow), roleWorkflow)
	if err != nil {
		return err
	}
	codes, err := s.keySet(roleRecord)
	if err != nil {
		return err
	}
	for _, w := range workflows {
		if w.SubjectRecord != "" && !codes[w.SubjectRecord] {
			s.add(models.IssueSeverityError, "workflow", w.ID,
				"Workflow %s %q acts on rows of Record %s, which does not exist", w.Code, w.Name, QuoteIfUnsafe(w.SubjectRecord))
		}
	}
	return nil
}
