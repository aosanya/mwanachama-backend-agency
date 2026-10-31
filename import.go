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
		doc = convertLegacyStatements(doc)
		if err := applyDocumentToDraft(ctx, m, tx, draft.ID, doc, now); err != nil {
			return err
		}
		if err := m.addRequiredModules(ctx, tx, draft.ID, now); err != nil {
			return err
		}

		if err := m.advanceCodeSequences(ctx, tx, draft.ID); err != nil {
			return err
		}
		out = ImportResult{Draft: draft, Attachments: len(doc.ControlAttachments)}
		return nil
	})
	if err != nil {
		return ImportResult{}, fmt.Errorf("ImportDocument: %w", err)
	}
	return out, nil
}

func (m *agencyManager) addRequiredModules(ctx context.Context, tx *gorm.DB, draftID, now string) error {
	held, err := list[models.Module](m, scopeQuery(tx.WithContext(ctx).Table(m.table(roleModule)), draftID), roleModule)
	if err != nil {
		return fmt.Errorf("addRequiredModules: %w", err)
	}
	kinds, err := m.draftStepKinds(ctx, tx, draftID)
	if err != nil {
		return fmt.Errorf("addRequiredModules: %w", err)
	}

	present := make(map[string]bool, len(held))
	switchedOn := make(map[string]bool, len(held))
	for _, mod := range withEffectiveEnabled(held, kinds) {
		present[mod.Code] = true
		switchedOn[mod.Code] = mod.Enabled
	}

	for _, entry := range models.ModuleCatalog {
		if present[entry.Code] || !models.WithRequiredModules(switchedOn)[entry.Code] {
			continue
		}
		row := models.Module{
			ID: newID(), Code: entry.Code, DraftID: draftID, Name: entry.Name,
			Description: entry.Description, Enabled: true, Locked: entry.Locked,
			CreatedAt: now, LastUpdated: now,
		}
		if err := m.copyRow(ctx, tx, roleModule, row); err != nil {
			return fmt.Errorf("addRequiredModules: %w", err)
		}
	}
	return nil
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

func convertLegacyStatements(doc models.AgencyDocument) models.AgencyDocument {
	held := make(map[string]bool, len(doc.Controls))
	for _, c := range doc.Controls {
		held[c.Code] = true
	}
	for _, statements := range [][]models.DocumentStatement{doc.Values, doc.Rules} {
		for _, st := range statements {
			if held[st.Code] {
				continue
			}
			held[st.Code] = true
			doc.Controls = append(doc.Controls, models.DocumentControl{ID: st.ID, Code: st.Code, Text: st.Text})
		}
	}
	for _, a := range doc.ValueAttachments {
		doc.ControlAttachments = append(doc.ControlAttachments, models.DocumentControlAttachment{
			ControlCode: a.ValueCode, TargetType: a.TargetType, TargetCode: a.TargetCode,
		})
	}
	for _, a := range doc.RuleAttachments {
		doc.ControlAttachments = append(doc.ControlAttachments, models.DocumentControlAttachment{
			ControlCode: a.RuleCode, TargetType: a.TargetType, TargetCode: a.TargetCode,
		})
	}
	doc.Values, doc.Rules, doc.ValueAttachments, doc.RuleAttachments = nil, nil, nil, nil
	doc.ControlAttachments = distinctControlAttachments(doc.ControlAttachments)
	return doc
}

func distinctControlAttachments(in []models.DocumentControlAttachment) []models.DocumentControlAttachment {
	seen := make(map[models.DocumentControlAttachment]bool, len(in))
	var out []models.DocumentControlAttachment
	for _, a := range in {
		if seen[a] {
			continue
		}
		seen[a] = true
		out = append(out, a)
	}
	return out
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
