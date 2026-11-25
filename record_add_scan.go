package mwanachamaagency

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (s *issueScan) checkRecordAdds() error {
	records, err := list[models.Record](s.m, s.scoped(roleRecord), roleRecord)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}
	for _, r := range records {
		s.checkRecordRequiredField(r)
	}
	roles, err := list[models.Role](s.m, s.scoped(roleRole), roleRole)
	if err != nil {
		return err
	}
	carried := carriedCodesOf(roles)
	for _, r := range records {
		s.checkRecordAdd(r, carried)
	}
	return nil
}

func (s *issueScan) checkRecordAdd(r models.Record, carried map[string]bool) {
	if !carried[RecordPermissionCode(r.Code, models.RowActionAdd, RecordFieldStar)] && !anyFieldAddCarried(r, carried) {
		s.add(models.IssueSeverityError, "record", r.ID,
			"Record %s %q has no role that may add to it: no role carries %s or an add code of one of its fields",
			r.Code, r.Name, RecordPermissionCode(r.Code, models.RowActionAdd, RecordFieldStar))
		return
	}
	for _, field := range r.Questions {
		if field.Required && field.Code != "" && !fieldAddCarried(r, field, carried) {
			s.add(models.IssueSeverityError, "record", r.ID,
				"Record %s %q has a required field %s that no role may add: no role carries %s or %s",
				r.Code, r.Name, field.Code,
				RecordPermissionCode(r.Code, models.RowActionAdd, RecordFieldStar),
				RecordPermissionCode(r.Code, models.RowActionAdd, field.Code))
		}
	}
}

func (s *issueScan) checkRecordRequiredField(r models.Record) {
	for _, field := range r.Questions {
		if field.Required {
			return
		}
	}
	s.add(models.IssueSeverityError, "record", r.ID,
		"Record %s %q has no required field: mark at least one field required", r.Code, r.Name)
}

func carriedCodesOf(roles []models.Role) map[string]bool {
	carried := make(map[string]bool)
	for _, role := range roles {
		for _, code := range role.Permissions {
			carried[code] = true
		}
	}
	return carried
}

func anyFieldAddCarried(r models.Record, carried map[string]bool) bool {
	for _, field := range r.Questions {
		if field.Code != "" && carried[RecordPermissionCode(r.Code, models.RowActionAdd, field.Code)] {
			return true
		}
	}
	return false
}

func fieldAddCarried(r models.Record, field models.StepQuestion, carried map[string]bool) bool {
	return carried[RecordPermissionCode(r.Code, models.RowActionAdd, RecordFieldStar)] ||
		carried[RecordPermissionCode(r.Code, models.RowActionAdd, field.Code)]
}
