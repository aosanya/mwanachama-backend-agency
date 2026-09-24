package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

type ImportResult struct {
	Draft       models.Draft `json:"draft"`
	Attachments int          `json:"attachments"`
}

// ImportDocument writes a whole agency document into a draft of its own. It
// is the way a document that was never published here lands — a mined
// template, or an export from somewhere else — where CreateDraft's
// "publication" fork is the way one that was published here comes back.
func (m *agencyManager) ImportDocument(ctx context.Context, description string, doc models.AgencyDocument) (ImportResult, error) {
	var out ImportResult
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := m.ensureAgency(ctx, tx, doc.Agency); err != nil {
			return err
		}

		now := models.NowRFC3339()
		draft := models.Draft{
			ID:          newID(),
			Description: description,
			Status:      models.DraftStatusOpen,
			CreatedAt:   now,
			LastUpdated: now,
		}
		if err := m.copyRow(ctx, tx, roleDraft, draft); err != nil {
			return err
		}
		if err := applyDocumentToDraft(ctx, m, tx, draft.ID, doc, now); err != nil {
			return err
		}

		attached, err := m.applyAttachments(ctx, tx, doc, now)
		if err != nil {
			return err
		}
		if err := m.advanceCodeSequences(ctx, tx, draft.ID); err != nil {
			return err
		}
		out = ImportResult{Draft: draft, Attachments: attached}
		return nil
	})
	if err != nil {
		return ImportResult{}, fmt.Errorf("ImportDocument: %w", err)
	}
	return out, nil
}

func (m *agencyManager) ensureAgency(ctx context.Context, tx *gorm.DB, from models.DocumentAgency) error {
	_, err := m.findAgency(ctx, tx)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrAgencyNotFound) {
		return err
	}

	now := models.NowRFC3339()
	return m.copyRow(ctx, tx, roleAgency, models.Agency{
		ID:                           newID(),
		Name:                         from.Name,
		Description:                  from.Description,
		Enabled:                      from.Enabled,
		DefaultFailurePipelineBudget: from.DefaultFailurePipelineBudget,
		InactivityTimeoutSeconds:     from.InactivityTimeoutSeconds,
		EventFlows:                   from.EventFlows,
		CreatedAt:                    now,
		LastUpdated:                  now,
	})
}

// applyAttachments writes the value and rule attachments the document
// carries. applyDocumentToDraft leaves them alone because an attachment is
// not draft-scoped — it names both ends by their stable code, so a draft's
// copies are already covered by whatever is attached. A document arriving
// from outside has nothing covering it yet, which is what this writes.
func (m *agencyManager) applyAttachments(ctx context.Context, tx *gorm.DB, doc models.AgencyDocument, now string) (int, error) {
	written := 0

	for _, a := range doc.ValueAttachments {
		held, err := m.attachmentHeld(ctx, tx, roleValueAttachment, "value_code", a.ValueCode, a.TargetType, a.TargetCode)
		if err != nil || held {
			if err != nil {
				return written, err
			}
			continue
		}
		row := models.ValueAttachment{
			ID:         newID(),
			ValueCode:  a.ValueCode,
			TargetType: models.TargetType(a.TargetType),
			TargetCode: a.TargetCode,
			CreatedAt:  now,
		}
		if err := m.copyRow(ctx, tx, roleValueAttachment, row); err != nil {
			return written, err
		}
		written++
	}

	for _, a := range doc.RuleAttachments {
		held, err := m.attachmentHeld(ctx, tx, roleRuleAttachment, "rule_code", a.RuleCode, a.TargetType, a.TargetCode)
		if err != nil || held {
			if err != nil {
				return written, err
			}
			continue
		}
		row := models.RuleAttachment{
			ID:         newID(),
			RuleCode:   a.RuleCode,
			TargetType: models.TargetType(a.TargetType),
			TargetCode: a.TargetCode,
			CreatedAt:  now,
		}
		if err := m.copyRow(ctx, tx, roleRuleAttachment, row); err != nil {
			return written, err
		}
		written++
	}

	return written, nil
}

// advanceCodeSequences moves each counter past the codes the document
// brought with it. An imported row keeps the code it was exported with, so
// a counter left at zero would mint a G-1 that already exists — and a code
// is what every attachment resolves through.
func (m *agencyManager) advanceCodeSequences(ctx context.Context, tx *gorm.DB, draftID string) error {
	sequences := m.table(roleCodeSequence)
	for _, c := range codedRoles {
		var codes []string
		if err := scopeQuery(tx.WithContext(ctx).Table(m.table(c.role)), draftID).
			Pluck("code", &codes).Error; err != nil {
			return fmt.Errorf("advanceCodeSequences: %w", err)
		}
		highest := 0
		for _, code := range codes {
			if n := numberIn(code, c.prefix); n > highest {
				highest = n
			}
		}
		if highest == 0 {
			continue
		}
		if err := raiseSequence(ctx, tx, sequences, c.entityType, highest+1); err != nil {
			return err
		}
	}
	return nil
}

func numberIn(code, prefix string) int {
	rest, found := strings.CutPrefix(code, prefix+"-")
	if !found {
		return 0
	}
	n, err := strconv.Atoi(rest)
	if err != nil {
		return 0
	}
	return n
}

func raiseSequence(ctx context.Context, tx *gorm.DB, table, entityType string, next int) error {
	var held []int
	if err := tx.WithContext(ctx).Table(table).
		Where("entity_type = ?", entityType).Pluck("next_number", &held).Error; err != nil {
		return fmt.Errorf("raiseSequence: %w", err)
	}
	if len(held) == 0 {
		err := tx.WithContext(ctx).Table(table).
			Create(map[string]any{"entity_type": entityType, "next_number": next}).Error
		if err != nil {
			return fmt.Errorf("raiseSequence: %w", err)
		}
		return nil
	}
	if held[0] >= next {
		return nil
	}
	err := tx.WithContext(ctx).Table(table).
		Where("entity_type = ?", entityType).Update("next_number", next).Error
	if err != nil {
		return fmt.Errorf("raiseSequence: %w", err)
	}
	return nil
}

func (m *agencyManager) attachmentHeld(ctx context.Context, tx *gorm.DB, role, codeColumn, code, targetType, targetCode string) (bool, error) {
	var count int64
	err := tx.WithContext(ctx).Table(m.table(role)).
		Where(codeColumn+" = ? AND target_type = ? AND target_code = ?", code, targetType, targetCode).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("attachmentHeld: %w", err)
	}
	return count > 0, nil
}
