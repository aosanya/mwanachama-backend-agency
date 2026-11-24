package mwanachamaagency

import (
	"context"
	"fmt"
	"maps"
	"sort"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func carryFieldKeysForward(db *gorm.DB, s *spec.Spec) error {
	records, rows := tableFor(s, roleRecord), tableFor(s, roleRecordRow)
	held, err := tablesNamed(db, []string{records, rows})
	if err != nil {
		return err
	}
	if !held[records] || !held[rows] {
		return nil
	}
	var drafts []string
	err = db.Transaction(func(tx *gorm.DB) error {
		var keyErr error
		drafts, keyErr = keyRecordsAndRows(tx, s, records, rows)
		return keyErr
	})
	if err != nil {
		return fmt.Errorf("agency: give the fields of the records keys: %w", err)
	}
	return regenerateFormsOf(db, s, drafts)
}

func keyRecordsAndRows(tx *gorm.DB, s *spec.Spec, records, rows string) ([]string, error) {
	recordObject, rowObject := mustObject(s, roleRecord), mustObject(s, roleRecordRow)
	var stored []map[string]any
	if err := tx.Table(records).Find(&stored).Error; err != nil {
		return nil, err
	}
	keyed := map[string]models.Record{}
	drafts := map[string]bool{}
	for _, raw := range stored {
		var record models.Record
		if err := specstore.Decode(recordObject, raw, &record); err != nil {
			return nil, err
		}
		settled := keyedFields(record.Code, record.Questions)
		if equalFieldIdentity(settled, record.Questions) {
			continue
		}
		record.Questions = settled
		cells, err := specstore.Encode(recordObject, record)
		if err != nil {
			return nil, err
		}
		if err := tx.Table(records).Where("id = ?", record.ID).Update("questions", cells["questions"]).Error; err != nil {
			return nil, err
		}
		keyed[record.ID] = record
		drafts[record.DraftID] = true
	}
	if len(keyed) == 0 {
		return nil, nil
	}
	ids := make([]string, 0, len(keyed))
	for id := range keyed {
		ids = append(ids, id)
	}
	var rowCells []map[string]any
	if err := tx.Table(rows).Where("record_id IN ?", ids).Find(&rowCells).Error; err != nil {
		return nil, err
	}
	for _, raw := range rowCells {
		var row models.RecordRow
		if err := specstore.Decode(rowObject, raw, &row); err != nil {
			return nil, err
		}
		fields := recordFieldsOf(keyed[row.RecordID])
		answers, placeRefs := fields.keyed(row.Answers), fields.keyed(row.PlaceRefs)
		if maps.Equal(answers, row.Answers) && maps.Equal(placeRefs, row.PlaceRefs) {
			continue
		}
		row.Answers, row.PlaceRefs = answers, placeRefs
		cells, err := specstore.Encode(rowObject, row)
		if err != nil {
			return nil, err
		}
		changed := map[string]any{"answers": cells["answers"], "place_refs": cells["place_refs"]}
		if err := tx.Table(rows).Where("id = ?", row.ID).Updates(changed).Error; err != nil {
			return nil, err
		}
	}
	out := make([]string, 0, len(drafts))
	for id := range drafts {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

func equalFieldIdentity(a, b []models.StepQuestion) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Key != b[i].Key || a[i].Code != b[i].Code {
			return false
		}
	}
	return true
}

func regenerateFormsOf(db *gorm.DB, s *spec.Spec, drafts []string) error {
	if len(drafts) == 0 {
		return nil
	}
	manager, err := NewAgencyManager(db, s)
	if err != nil {
		return err
	}
	for _, draftID := range drafts {
		if err := manager.EnsureGeneratedWorkflows(context.Background(), draftID); err != nil {
			return fmt.Errorf("agency: regenerate the workflows of the draft %s: %w", draftID, err)
		}
	}
	return nil
}
