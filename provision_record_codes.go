package mwanachamaagency

import (
	"fmt"
	"sort"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func carryRecordKeysForward(db *gorm.DB, s *spec.Spec) error {
	records, roles := tableFor(s, roleRecord), tableFor(s, roleRole)
	held, err := tablesNamed(db, []string{records, roles})
	if err != nil {
		return err
	}
	if !held[records] || !held[roles] {
		return nil
	}
	err = db.Transaction(func(tx *gorm.DB) error { return keyRecordsAndMoveRoles(tx, s, records, roles) })
	if err != nil {
		return fmt.Errorf("agency: give the records readable codes and keys: %w", err)
	}
	return nil
}

func keyRecordsAndMoveRoles(tx *gorm.DB, s *spec.Spec, records, roles string) error {
	recordObject := mustObject(s, roleRecord)
	var stored []map[string]any
	if err := tx.Table(records).Order("created_at, id").Find(&stored).Error; err != nil {
		return err
	}
	byDraft := map[string][]models.Record{}
	var drafts []string
	for _, raw := range stored {
		var record models.Record
		if err := specstore.Decode(recordObject, raw, &record); err != nil {
			return err
		}
		if _, seen := byDraft[record.DraftID]; !seen {
			drafts = append(drafts, record.DraftID)
		}
		byDraft[record.DraftID] = append(byDraft[record.DraftID], record)
	}
	sort.Strings(drafts)
	for _, draftID := range drafts {
		if err := keyRecordsOfDraft(tx, records, roles, s, draftID, byDraft[draftID]); err != nil {
			return err
		}
	}
	return nil
}

func keyRecordsOfDraft(tx *gorm.DB, records, roles string, s *spec.Spec, draftID string, held []models.Record) error {
	older := false
	identities := make([]recordIdentity, len(held))
	for i, r := range held {
		identities[i] = recordIdentity{Key: r.Key, Code: r.Code, Name: r.Name}
		older = older || r.Key == ""
	}
	if !older {
		return nil
	}
	settled, renames, err := settleRecordIdentities(identities)
	if err != nil {
		return err
	}
	for i, r := range held {
		if r.Key != "" {
			continue
		}
		changed := map[string]any{"code": settled[i].Code, "key": settled[i].Key}
		if err := tx.Table(records).Where("id = ?", r.ID).Updates(changed).Error; err != nil {
			return err
		}
	}
	return moveStoredRolePermissions(tx, mustObject(s, roleRole), roles, draftID, renames)
}

func moveStoredRolePermissions(tx *gorm.DB, roleObject spec.Object, roles, draftID string, renames map[string]string) error {
	if len(renames) == 0 {
		return nil
	}
	var stored []map[string]any
	if err := tx.Table(roles).Where("draft_id = ?", draftID).Find(&stored).Error; err != nil {
		return err
	}
	for _, raw := range stored {
		var role models.Role
		if err := specstore.Decode(roleObject, raw, &role); err != nil {
			return err
		}
		moved, changed := movedPermissions(role.Permissions, renames)
		if !changed {
			continue
		}
		role.Permissions = moved
		cells, err := specstore.Encode(roleObject, role)
		if err != nil {
			return err
		}
		if err := tx.Table(roles).Where("id = ?", role.ID).Update("permissions", cells["permissions"]).Error; err != nil {
			return err
		}
	}
	return nil
}
