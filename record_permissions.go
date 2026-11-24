package mwanachamaagency

import (
	"strings"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

const (
	RecordPermissionPrefix = "record."
	RecordViewAction       = "view"
)

func RecordPermissionCode(recordCode, action string) string {
	return RecordPermissionPrefix + recordCode + "." + action
}

func RecordPermissionsOf(r models.Record) []models.Permission {
	item := strings.ToLower(itemNameOf(r))
	article := withArticle(item)
	names := []struct{ action, name string }{
		{models.RowActionAdd, "Add " + article},
		{models.RowActionEdit, "Edit " + article},
		{models.RowActionDelete, "Delete " + article},
		{RecordViewAction, "See every " + item},
	}
	permissions := make([]models.Permission, 0, len(names))
	for _, n := range names {
		permissions = append(permissions, models.Permission{
			Code:    RecordPermissionCode(r.Code, n.action),
			Name:    n.name,
			Actions: []string{},
		})
	}
	return permissions
}
