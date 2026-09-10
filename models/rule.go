package models

// Rule is a constraint statement an AI agent acting as a Role needs made
// explicit — the kind of thing a human Role-holder would just treat as
// common knowledge (e.g. "never touch production data"). Plain text,
// informational only — nothing in this package enforces it at runtime.
// Same scoping, lifecycle, and open write-authorization question as
// [Value] — see its doc comment. Net new, added 2026-09-10 (AGD-001).
type Rule struct {
	ID          string `json:"id"`
	ObjectiveID string `json:"objective_id,omitempty"`
	// DraftID is "" for a live Rule, or the owning Draft's ID for a
	// draft-scoped copy.
	DraftID     string `json:"draft_id,omitempty"`
	Text        string `json:"text"`
	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
	Deleted     bool   `json:"deleted"`
}
