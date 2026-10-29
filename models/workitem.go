package models

type WorkItem struct {
	ID            string         `json:"id"`
	Code          string         `json:"code"`
	WorkflowID    string         `json:"workflow_id"`
	ObjectiveID   string         `json:"objective_id"`
	DraftID       string         `json:"draft_id,omitempty"`
	Title         string         `json:"title"`
	Description   string         `json:"description"`
	Ordinality    int            `json:"ordinality"`
	Prompt        string         `json:"prompt"`
	CreatedAt     string         `json:"created_at"`
	LastUpdated   string         `json:"last_updated"`
	Deleted       bool           `json:"deleted"`
	Kind          StepKind       `json:"kind"`
	RoleID        string         `json:"role_id"`
	RunWorkflowID string         `json:"run_workflow_id"`
	FormName      string         `json:"form_name"`
	Questions     []StepQuestion `json:"questions,omitempty"`
	Action        string         `json:"action,omitempty" spec:"-"`
}
