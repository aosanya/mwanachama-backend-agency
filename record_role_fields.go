package mwanachamaagency

import (
	"fmt"
	"strings"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

var goneRoleFieldActions = map[string]string{
	models.RecordFieldRoleID:       models.RowActionAdd,
	models.RecordFieldEditRoleID:   models.RowActionEdit,
	models.RecordFieldViewerRoleID: RecordViewAction,
	models.RecordFieldDeleteRoleID: models.RowActionDelete,
}

func replacementCodesOf(recordCode, field string) []string {
	action := goneRoleFieldActions[field]
	if action == models.RowActionDelete {
		return []string{RecordPermissionCode(recordCode, action)}
	}
	return []string{RecordPermissionCode(recordCode, action, RecordFieldStar), RecordPermissionCode(recordCode, action, "<field>")}
}

func refuseGoneRoleFields(code, name string, carried []string) error {
	if len(carried) == 0 {
		return nil
	}
	recordCode := code
	if !validFieldCode(recordCode) {
		recordCode = RecordCodeOf(name)
	}
	named := make([]string, 0, len(carried))
	for _, field := range carried {
		named = append(named, fmt.Sprintf("%s is gone from a record: give a role %s instead",
			field, strings.Join(replacementCodesOf(recordCode, field), " or ")))
	}
	return invalidRecord("%s", strings.Join(named, "; "))
}

func refuseGoneRoleFieldsInDocument(doc models.AgencyDocument) error {
	for _, r := range doc.Records {
		if err := refuseGoneRoleFields(r.Code, r.Name, r.CarriedGoneRoleFields()); err != nil {
			return fmt.Errorf("the record %s: %w", r.Name, err)
		}
	}
	return nil
}
