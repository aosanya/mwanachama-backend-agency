package models

// Objective is the single measurable operationalization of a Goal —
// exactly one Objective per Goal (1:1) — and the unit WorkItem, RoleAssignment
// (RACI), and Objective-scoped Value/Rule all attach to. Net new, added
// 2026-09-10 (AGD-001).
type Objective struct {
	ID     string `json:"id"`
	GoalID string `json:"goal_id"`
	// DraftID is "" for a live Objective, or the owning Draft's ID for a
	// draft-scoped copy — same Draft-scoping as Goal/Workflow/WorkItem
	// (a live Objective can still be edited directly; there is no
	// live-immutability check here, matching those three's existing
	// behavior rather than Role/RoleAssignment's stricter one).
	DraftID     string `json:"draft_id,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Ordinality  int    `json:"ordinality"`
	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
	Deleted     bool   `json:"deleted"`
}
