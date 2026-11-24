package mwanachamaagency

import (
	"strings"
	"unicode"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

const (
	RecordPermissionPrefix = "record."
	RecordViewAction       = "view"
	RecordFieldStar        = "*"
)

var recordFieldActions = []string{models.RowActionAdd, models.RowActionEdit, RecordViewAction}

func RecordPermissionCode(recordCode, action string, field ...string) string {
	code := RecordPermissionPrefix + recordCode + "." + action
	if action == models.RowActionDelete || len(field) == 0 || field[0] == "" {
		return code
	}
	return code + "." + field[0]
}

func recordPermissionName(item, action string, field *models.StepQuestion) string {
	article := withArticle(item)
	switch {
	case action == models.RowActionDelete:
		return "Delete " + article
	case field == nil:
		return recordActionVerb(action) + " every field of " + article
	default:
		return recordActionVerb(action) + " the " + fieldWordingOf(*field) + " of " + article
	}
}

func recordActionVerb(action string) string {
	switch action {
	case models.RowActionAdd:
		return "Add"
	case models.RowActionEdit:
		return "Edit"
	default:
		return "See"
	}
}

func fieldWordingOf(field models.StepQuestion) string {
	prompt := strings.TrimSpace(field.Prompt)
	if prompt == "" {
		return field.Code
	}
	runes := []rune(prompt)
	if len(runes) > 1 && unicode.IsUpper(runes[1]) {
		return prompt
	}
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func RecordPermissionsOf(r models.Record) []models.Permission {
	item := strings.ToLower(itemNameOf(r))
	permissions := make([]models.Permission, 0, len(recordFieldActions)*(len(r.Questions)+1)+1)
	give := func(action string, field *models.StepQuestion) {
		code := RecordPermissionCode(r.Code, action, RecordFieldStar)
		if field != nil {
			code = RecordPermissionCode(r.Code, action, field.Code)
		}
		permissions = append(permissions, models.Permission{
			Code:    code,
			Name:    recordPermissionName(item, action, field),
			Actions: []string{},
		})
	}
	for _, action := range recordFieldActions {
		for i := range r.Questions {
			if r.Questions[i].Code != "" {
				give(action, &r.Questions[i])
			}
		}
		give(action, nil)
	}
	give(models.RowActionDelete, nil)
	return permissions
}
