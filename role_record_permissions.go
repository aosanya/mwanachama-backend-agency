package mwanachamaagency

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func recordPermissionCodesOf(records []models.Record) map[string]bool {
	given := make(map[string]bool, len(records)*4)
	for _, r := range records {
		for _, p := range RecordPermissionsOf(r) {
			given[p.Code] = true
		}
	}
	return given
}

func isRecordPermission(code string) bool {
	return strings.HasPrefix(code, RecordPermissionPrefix)
}

func recordCodesNoRecordGives(permissions []string, given map[string]bool) []string {
	var missing []string
	for _, code := range permissions {
		if isRecordPermission(code) && !given[code] {
			missing = append(missing, code)
		}
	}
	return missing
}

func refuseRecordCodesNoRecordGives(roleName string, permissions []string, given map[string]bool) error {
	missing := recordCodesNoRecordGives(permissions, given)
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%w: role %q carries %s, which no record of its draft gives", ErrUnknownPermission, roleName, strings.Join(missing, ", "))
}

func (m *agencyManager) recordPermissionCodes(ctx context.Context, tx *gorm.DB, draftID string) (map[string]bool, error) {
	records, err := list[models.Record](m, scopeQuery(tx.WithContext(ctx).Table(m.table(roleRecord)), draftID), roleRecord)
	if err != nil {
		return nil, fmt.Errorf("recordPermissionCodes: %w", err)
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
		records = append(records, models.Record{Code: r.Code, Name: r.Name, FormName: r.FormName, ItemName: r.ItemName})
	}
	given := recordPermissionCodesOf(records)
	for _, role := range doc.Roles {
		if err := refuseRecordCodesNoRecordGives(role.Name, role.Permissions, given); err != nil {
			return err
		}
	}
	return nil
}
