package mwanachamaagency

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

type recordPermissionCodes struct {
	given   map[string]bool
	records []string
}

type recordCodeFault struct {
	code   string
	record string
	field  string
	plain  bool
}

func recordPermissionCodesOf(records []models.Record) recordPermissionCodes {
	codes := recordPermissionCodes{given: make(map[string]bool, len(records)*8)}
	for _, r := range records {
		codes.records = append(codes.records, r.Code)
		for _, p := range RecordPermissionsOf(r) {
			codes.given[p.Code] = true
		}
	}
	return codes
}

func isRecordPermission(code string) bool {
	return strings.HasPrefix(code, RecordPermissionPrefix)
}

func (c recordPermissionCodes) recordOf(code string) (record, rest string, found bool) {
	for _, candidate := range c.records {
		prefix := RecordPermissionPrefix + candidate + "."
		if strings.HasPrefix(code, prefix) && len(candidate) >= len(record) {
			record, rest, found = candidate, strings.TrimPrefix(code, prefix), true
		}
	}
	return record, rest, found
}

func (c recordPermissionCodes) faultOf(code string) recordCodeFault {
	record, rest, found := c.recordOf(code)
	if !found {
		return recordCodeFault{code: code}
	}
	action, field, _ := strings.Cut(rest, ".")
	if !slices.Contains(recordFieldActions, action) {
		return recordCodeFault{code: code, record: record}
	}
	if field == "" {
		return recordCodeFault{code: code, record: record, plain: true}
	}
	return recordCodeFault{code: code, record: record, field: field}
}

func (f recordCodeFault) reason(shown, noRecord string) string {
	switch {
	case f.plain:
		return fmt.Sprintf("%s, which is now %s.* for every field or %s.<field> for one", shown, f.code, f.code)
	case f.field != "":
		return fmt.Sprintf("%s, but record %q has no field %q", shown, f.record, f.field)
	case f.record != "":
		return fmt.Sprintf("%s, which record %q does not give", shown, f.record)
	default:
		return fmt.Sprintf("%s, %s", shown, noRecord)
	}
}

func (c recordPermissionCodes) faultsIn(permissions []string) []recordCodeFault {
	var faults []recordCodeFault
	for _, code := range permissions {
		if isRecordPermission(code) && !c.given[code] {
			faults = append(faults, c.faultOf(code))
		}
	}
	return faults
}

func refuseRecordCodesNoRecordGives(roleName string, permissions []string, codes recordPermissionCodes) error {
	faults := codes.faultsIn(permissions)
	if len(faults) == 0 {
		return nil
	}
	reasons := make([]string, 0, len(faults))
	for _, f := range faults {
		reasons = append(reasons, f.reason(f.code, "which no record of its draft gives"))
	}
	return fmt.Errorf("%w: role %q carries %s", ErrUnknownPermission, roleName, strings.Join(reasons, "; "))
}

func (m *agencyManager) recordPermissionCodes(ctx context.Context, tx *gorm.DB, draftID string) (recordPermissionCodes, error) {
	records, err := list[models.Record](m, scopeQuery(tx.WithContext(ctx).Table(m.table(roleRecord)), draftID), roleRecord)
	if err != nil {
		return recordPermissionCodes{}, fmt.Errorf("recordPermissionCodes: %w", err)
	}
	return recordPermissionCodesOf(records), nil
}

func (m *agencyManager) checkRoleRecordPermissions(ctx context.Context, tx *gorm.DB, draftID, roleName string, permissions []string) error {
	given, err := m.recordPermissionCodes(ctx, tx, draftID)
	if err != nil {
		return err
	}
	return refuseRecordCodesNoRecordGives(roleName, permissions, given)
}

func validateDocumentRolePermissions(doc models.AgencyDocument) error {
	records := make([]models.Record, 0, len(doc.Records))
	for _, r := range doc.Records {
		records = append(records, models.Record{Code: r.Code, Name: r.Name, FormName: r.FormName, ItemName: r.ItemName, Questions: keyedFields(r.Code, r.Questions)})
	}
	given := recordPermissionCodesOf(records)
	for _, role := range doc.Roles {
		if err := refuseRecordCodesNoRecordGives(role.Name, role.Permissions, given); err != nil {
			return err
		}
	}
	return nil
}
