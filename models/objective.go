package models

// Objective is a measurable operationalization of a Goal — a Goal may
// have several — and the unit Objective-scoped Value/Rule attach to.
type Objective struct {
	ID string `json:"id"`
	// Code is a stable, human-readable identifier (e.g. "O-1") assigned
	// once at creation and never changed — see [Goal.Code]'s doc comment.
	Code        string `json:"code"`
	GoalID      string `json:"goal_id"`
	DraftID     string `json:"draft_id,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Ordinality  int    `json:"ordinality"`
	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
	Deleted     bool   `json:"deleted"`
}
