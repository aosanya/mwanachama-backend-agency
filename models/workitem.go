package models

// WorkItem is a unit of work owned by exactly one Workflow, and optionally
// tagged to a Goal it advances. Ported from CodeValdAgency's WorkItem.
type WorkItem struct {
	ID         string `json:"id"`
	WorkflowID string `json:"workflow_id"`
	// GoalID is "" when this WorkItem is not tagged to a Goal.
	GoalID      string `json:"goal_id,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Ordinality  int    `json:"ordinality"`
	Prompt      string `json:"prompt"`
	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
	Deleted     bool   `json:"deleted"`
}
