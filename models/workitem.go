package models

// WorkItem is a unit of work owned by exactly one Workflow, and belongs
// to exactly one Objective it advances. As of 2026-09-10 (AGD-001), ObjectiveID replaces the former
// optional GoalID tag — every WorkItem now has an Objective, there is no
// more "untagged" WorkItem.
type WorkItem struct {
	ID string `json:"id"`
	// Code is a stable, human-readable identifier (e.g. "WI-1") assigned
	// once at creation and never changed — see [Goal.Code]'s doc comment.
	Code        string `json:"code"`
	WorkflowID  string `json:"workflow_id"`
	ObjectiveID string `json:"objective_id"`
	// DraftID is "" for a live WorkItem, or the owning Draft's ID for a
	// draft-scoped copy.
	DraftID     string `json:"draft_id,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Ordinality  int    `json:"ordinality"`
	Prompt      string `json:"prompt"`
	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
	Deleted     bool   `json:"deleted"`
}
