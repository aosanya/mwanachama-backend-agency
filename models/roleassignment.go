package models

// RACILabel is a Role's responsibility for a WorkItem it is assigned to.
type RACILabel string

const (
	RACILabelResponsible RACILabel = "responsible"
	RACILabelAccountable RACILabel = "accountable"
	RACILabelConsulted   RACILabel = "consulted"
	RACILabelInformed    RACILabel = "informed"
)

// Valid reports whether l is one of the four defined labels.
func (l RACILabel) Valid() bool {
	switch l {
	case RACILabelResponsible, RACILabelAccountable, RACILabelConsulted, RACILabelInformed:
		return true
	default:
		return false
	}
}

// RoleAssignment assigns a Role to a WorkItem with a RACILabel. At most one
// Accountable assignment may exist per WorkItem (within the same scope).
//
// Same draft-scoping and live-immutability rules as Role: DraftID is ""
// for a live assignment, or the owning Draft's ID for a draft-scoped copy.
type RoleAssignment struct {
	ID          string    `json:"id"`
	DraftID     string    `json:"draft_id,omitempty"`
	RoleID      string    `json:"role_id"`
	WorkItemID  string    `json:"work_item_id"`
	Label       RACILabel `json:"label"`
	CreatedAt   string    `json:"created_at"`
	LastUpdated string    `json:"last_updated"`
	Deleted     bool      `json:"deleted"`
}
