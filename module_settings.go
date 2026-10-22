package mwanachamaagency

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

type SettingsChecker func(ctx context.Context, module string, settings json.RawMessage) []string

type Option func(*agencyManager)

func WithSettingsChecker(check SettingsChecker) Option {
	return func(m *agencyManager) { m.settings = check }
}

func normalizedSettings(raw json.RawMessage) (string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return "", nil
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &object); err != nil || object == nil {
		return "", ErrInvalidModuleSettings
	}
	return string(trimmed), nil
}

func (m *agencyManager) SetModuleSettings(ctx context.Context, id string, settings json.RawMessage) (models.Module, error) {
	normalized, err := normalizedSettings(settings)
	if err != nil {
		return models.Module{}, fmt.Errorf("SetModuleSettings: %w", err)
	}
	current, err := m.findModule(ctx, m.db, id)
	if err != nil {
		return models.Module{}, fmt.Errorf("SetModuleSettings: %w", err)
	}
	if err := m.validateOpenDraftID(ctx, current.DraftID); err != nil {
		return models.Module{}, fmt.Errorf("SetModuleSettings: %w", err)
	}
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		mod, err := m.findModule(ctx, tx, id)
		if err != nil {
			return err
		}
		mod.Settings = normalized
		mod.LastUpdated = models.NowRFC3339()
		return m.writeUpdate(ctx, tx, roleModule, id, mod, ErrModuleNotFound, "draft_id", "code", "created_at")
	})
	if err != nil {
		return models.Module{}, fmt.Errorf("SetModuleSettings: %w", err)
	}
	return m.GetModule(ctx, id)
}

func (s *issueScan) checkModuleSettings() error {
	if s.m.settings == nil {
		return nil
	}
	modules, err := list[models.Module](s.m, s.scoped(roleModule).Order("code"), roleModule)
	if err != nil {
		return err
	}
	var linkedIDs []string
	if err := s.scoped(roleObjectiveModule).Pluck("module_id", &linkedIDs).Error; err != nil {
		return err
	}
	linked := make(map[string]bool, len(linkedIDs))
	for _, id := range linkedIDs {
		linked[id] = true
	}
	for _, mod := range deriveEnabled(modules, linked) {
		if !mod.Enabled {
			continue
		}
		for _, problem := range s.m.settings(s.ctx, mod.Code, json.RawMessage(mod.Settings)) {
			s.add(models.IssueSeverityError, "module", mod.ID, "Module %s settings: %s", mod.Code, problem)
		}
	}
	return nil
}
