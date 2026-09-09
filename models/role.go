package models

// Role is a persona that can be assigned to WorkItems via a RoleAssignment
// (e.g. "bug-hunter", "code-reviewer"). Net new — nothing to port.
//
// A live Role cannot be edited directly: DraftID is "" for a live Role,
// or the owning Draft's ID for a draft-scoped copy. Changing a Role means
// forking a Draft, editing the draft-scoped copy, then promoting.
type Role struct {
	ID          string `json:"id"`
	DraftID     string `json:"draft_id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
	Deleted     bool   `json:"deleted"`
}
