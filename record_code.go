package mwanachamaagency

import (
	"context"
	"regexp"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

const (
	recordCodeFallback = "record"
)

var (
	defaultRecordCodeSuffix = regexp.MustCompile(`^_[0-9]+$`)
	recordUUIDKey           = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

func RecordCodeOf(name string) string {
	return codeFromText(name, recordCodeFallback)
}

func isDefaultRecordCode(code, name string) bool {
	proposed := RecordCodeOf(name)
	rest, found := strings.CutPrefix(code, proposed)
	return found && (rest == "" || defaultRecordCodeSuffix.MatchString(rest))
}

func nonDefaultRecordCode(r models.Record) string {
	if isDefaultRecordCode(r.Code, r.Name) {
		return ""
	}
	return r.Code
}

func settleRecordCode(given, name string, taken map[string]bool) (string, error) {
	if given == "" {
		return uniqueFieldCode(RecordCodeOf(name), taken), nil
	}
	if !validFieldCode(given) {
		return "", invalidRecord("the record %q has the code %q; a code is lower case letters, digits and underscores, starting with a letter", name, given)
	}
	if taken[given] {
		return "", invalidRecord("another record of the draft already has the code %q", given)
	}
	return given, nil
}

func (m *agencyManager) recordCodesTaken(ctx context.Context, tx *gorm.DB, draftID, exceptID string) (map[string]bool, error) {
	var codes []string
	q := scopeQuery(tx.WithContext(ctx).Table(m.table(roleRecord)), draftID).Where("id <> ?", exceptID)
	if err := q.Pluck("code", &codes).Error; err != nil {
		return nil, err
	}
	taken := make(map[string]bool, len(codes))
	for _, code := range codes {
		taken[code] = true
	}
	return taken, nil
}

func (m *agencyManager) insertRecord(ctx context.Context, r models.Record) (models.Record, error) {
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		taken, err := m.recordCodesTaken(ctx, tx, r.DraftID, "")
		if err != nil {
			return err
		}
		if r.Code, err = settleRecordCode(r.Code, r.Name, taken); err != nil {
			return err
		}
		r.Key = newID()
		row, err := m.encode(roleRecord, r)
		if err != nil {
			return err
		}
		return tx.Table(m.table(roleRecord)).Create(row).Error
	})
	return r, classifyConflict(err)
}

func (m *agencyManager) settleRecordRename(ctx context.Context, tx *gorm.DB, existing models.Record, r *models.Record) error {
	r.Key = existing.Identity()
	if r.Code == "" || r.Code == existing.Code {
		r.Code = existing.Code
		return nil
	}
	taken, err := m.recordCodesTaken(ctx, tx, existing.DraftID, existing.ID)
	if err != nil {
		return err
	}
	_, err = settleRecordCode(r.Code, r.Name, taken)
	return err
}

type recordIdentity struct {
	Key  string
	Code string
	Name string
}

func settleRecordIdentities(in []recordIdentity) ([]recordIdentity, map[string]string, error) {
	out := append([]recordIdentity(nil), in...)
	taken := map[string]bool{}
	renames := map[string]string{}
	for i, r := range out {
		if r.Key == "" {
			continue
		}
		if r.Code == "" {
			continue
		}
		if _, err := settleRecordCode(r.Code, r.Name, taken); err != nil {
			return nil, nil, err
		}
		taken[out[i].Code] = true
	}
	for i, r := range out {
		if r.Key != "" {
			continue
		}
		if r.Code == "" {
			continue
		}
		out[i].Key = r.Code
		if validFieldCode(r.Code) && !taken[r.Code] {
			taken[r.Code] = true
			continue
		}
		out[i].Code = ""
	}
	for i, r := range out {
		if r.Code != "" {
			continue
		}
		out[i].Code = uniqueFieldCode(RecordCodeOf(r.Name), taken)
		taken[out[i].Code] = true
		if out[i].Key == "" {
			out[i].Key = out[i].Code
		}
		if was := in[i].Code; was != "" && was != out[i].Code {
			renames[was] = out[i].Code
		}
	}
	return out, renames, nil
}

func movedPermissionCode(code string, renames map[string]string) string {
	rest, found := strings.CutPrefix(code, RecordPermissionPrefix)
	if !found {
		return code
	}
	recordCode, tail, _ := strings.Cut(rest, ".")
	moved, renamed := renames[recordCode]
	if !renamed {
		return code
	}
	if !strings.Contains(rest, ".") {
		return RecordPermissionPrefix + moved
	}
	return RecordPermissionPrefix + moved + "." + tail
}

func movedPermissions(permissions []string, renames map[string]string) ([]string, bool) {
	moved := make([]string, len(permissions))
	changed := false
	for i, code := range permissions {
		moved[i] = movedPermissionCode(code, renames)
		changed = changed || moved[i] != code
	}
	return moved, changed
}

func (m *agencyManager) moveRolePermissions(ctx context.Context, tx *gorm.DB, draftID string, renames map[string]string) error {
	roles, err := list[models.Role](m, scopeQuery(tx.WithContext(ctx).Table(m.table(roleRole)), draftID), roleRole)
	if err != nil {
		return err
	}
	now := models.NowRFC3339()
	for _, role := range roles {
		moved, changed := movedPermissions(role.Permissions, renames)
		if !changed {
			continue
		}
		role.Permissions, role.LastUpdated = moved, now
		if err := m.writeUpdate(ctx, tx, roleRole, role.ID, role, ErrRoleNotFound); err != nil {
			return err
		}
	}
	return nil
}

func settleDocumentRecords(doc models.AgencyDocument) (models.AgencyDocument, error) {
	identities := make([]recordIdentity, len(doc.Records))
	for i, r := range doc.Records {
		identities[i] = recordIdentity{Key: r.Key, Code: r.Code, Name: r.Name}
	}
	settled, renames, err := settleRecordIdentities(identities)
	if err != nil {
		return doc, err
	}
	doc.Records = append([]models.DocumentRecord(nil), doc.Records...)
	keyOfCode := make(map[string]string, len(settled))
	for i := range doc.Records {
		doc.Records[i].Key, doc.Records[i].Code = settled[i].Key, settled[i].Code
		keyOfCode[settled[i].Code] = settled[i].Key
	}
	doc.Workflows = append([]models.DocumentWorkflow(nil), doc.Workflows...)
	for i := range doc.Workflows {
		doc.Workflows[i].SubjectRecord = keyedRecordRef(keyOfCode, doc.Workflows[i].SubjectRecord)
	}
	doc.WorkItems = append([]models.DocumentWorkItem(nil), doc.WorkItems...)
	for i := range doc.WorkItems {
		doc.WorkItems[i].RowRecord = keyedRecordRef(keyOfCode, doc.WorkItems[i].RowRecord)
	}
	if len(renames) == 0 {
		return doc, nil
	}
	doc.Roles = append([]models.DocumentRole(nil), doc.Roles...)
	for i := range doc.Roles {
		doc.Roles[i].Permissions, _ = movedPermissions(doc.Roles[i].Permissions, renames)
	}
	return doc, nil
}

func (m *agencyManager) recordKeyNamed(ctx context.Context, draftID, name string) string {
	if name == "" {
		return ""
	}
	var keys []string
	q := scopeQuery(m.q(ctx, roleRecord), draftID).Where("code = ?", name)
	if err := q.Pluck("key", &keys).Error; err != nil || len(keys) == 0 || keys[0] == "" {
		return name
	}
	return keys[0]
}

func (m *agencyManager) recordKeyOfRef(ctx context.Context, tx *gorm.DB, draftID, ref string) string {
	if ref == "" {
		return ref
	}
	var held []models.Record
	err := scopeQuery(tx.WithContext(ctx).Table(m.table(roleRecord)), draftID).Where("key = ? OR code = ?", ref, ref).Find(&held).Error
	if err != nil {
		return ref
	}
	for _, r := range held {
		if r.Key == ref {
			return r.Key
		}
	}
	if len(held) == 1 && held[0].Key != "" {
		return held[0].Key
	}
	return ref
}

func keyedRecordRef(keyOfCode map[string]string, ref string) string {
	if key, found := keyOfCode[ref]; found {
		return key
	}
	return ref
}

func (m *agencyManager) renameRecordCode(ctx context.Context, tx *gorm.DB, existing models.Record, code string) error {
	if code == existing.Code {
		return nil
	}
	if err := tx.WithContext(ctx).Table(m.table(roleRecord)).Where("id = ?", existing.ID).Update("code", code).Error; err != nil {
		return err
	}
	return m.moveRolePermissions(ctx, tx, existing.DraftID, map[string]string{existing.Code: code})
}
