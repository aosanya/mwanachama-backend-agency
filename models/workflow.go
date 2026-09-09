package models

// Workflow is an ordered sequence of WorkItems belonging to an Agency.
// Ported from CodeValdAgency's Workflow, dropping AgencyID (single-agency
// deployment, no cross-agency scoping needed yet).
type Workflow struct {
	ID          string `json:"id"`
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
