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

func DeleteReasonQuestion() StepQuestion {
	return StepQuestion{Prompt: DeleteReasonPrompt, AnswerType: AnswerFreeText, Required: true}
}
