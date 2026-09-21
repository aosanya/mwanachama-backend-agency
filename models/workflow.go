package models

// Workflow is an ordered sequence of WorkItems belonging to an Agency.
// It omits AgencyID (single-agency
// deployment, no cross-agency scoping needed yet).
type Workflow struct {
	ID string `json:"id"`
	// Code is a stable, human-readable identifier (e.g. "W-1") assigned
	// once at creation and never changed — see [Goal.Code]'s doc comment.
	Code string `json:"code"`
	// DraftID is "" for a live Workflow, or the owning Draft's ID for a
	// draft-scoped copy.
	DraftID     string `json:"draft_id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Ordinality  int    `json:"ordinality"`
	// EventFlows is a raw JSON blob (name + ordered steps). Falls back to
	// the owning Agency's EventFlows when empty.
	EventFlows  string `json:"event_flows,omitempty"`
	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
	Deleted     bool   `json:"deleted"`
}
