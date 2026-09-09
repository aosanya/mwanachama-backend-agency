package models

// Department is one node in an Agency's org chart: a parent/child tree of
// named departments (e.g. "software", "mechanical"), each pinned to a
// Level. Net new — modeled on mwanachama-backend-actor's Group, but local
// and self-contained (no go.mod dependency on actor, no live sync).
type Department struct {
	ID string `json:"id"`
	// ParentID is "" for a top-level department.
	ParentID string `json:"parent_id,omitempty"`
	LevelID  string `json:"level_id"`
	Name     string `json:"name"`

	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
	Deleted     bool   `json:"deleted"`
}
