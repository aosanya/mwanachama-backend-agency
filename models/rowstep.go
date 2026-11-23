package models

const (
	RowActionAdd    = "add"
	RowActionEdit   = "edit"
	RowActionDelete = "delete"
)

const DeleteReasonPrompt = "Why is it removed"

func RowActions() []string {
	return []string{RowActionAdd, RowActionEdit, RowActionDelete}
}

func ValidRowAction(action string) bool {
	for _, known := range RowActions() {
		if action == known {
			return true
		}
	}
	return false
}

func (r Record) RoleOfRowAction(action string) string {
	switch {
	case action == RowActionEdit && r.EditRoleID != "":
		return r.EditRoleID
	case action == RowActionDelete && r.DeleteRoleID != "":
		return r.DeleteRoleID
	}
	return r.RoleID
}

func DeleteReasonQuestion() StepQuestion {
	return StepQuestion{Prompt: DeleteReasonPrompt, AnswerType: AnswerFreeText, Required: true}
}
