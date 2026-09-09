package models

// Goal is a strategic objective belonging to an Agency. Ported from
// CodeValdAgency's Goal, dropping AgencyID (single-agency deployment, no
// cross-agency scoping needed yet).
type Goal struct {
	ID string `json:"id"`
	// DraftID is "" for a live Goal, or the owning Draft's ID for a
	// draft-scoped copy.
	DraftID     string `json:"draft_id,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Ordinality  int    `json:"ordinality"`
	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
	Deleted     bool   `json:"deleted"`
}
