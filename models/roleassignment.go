package models

// RACILabel is a Role's responsibility for an Objective it is assigned to.
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

// RoleAssignment assigns a Role to an Objective with a RACILabel. At most
// one Accountable assignment may exist per Objective (within the same
// scope). As of 2026-09-10 (AGD-001), ObjectiveID replaces the former
// WorkItemID — RACI moved from WorkItem-scoped to Objective-scoped.
//
// Same draft-scoping and live-immutability rules as Role: DraftID is ""
// for a live assignment, or the owning Draft's ID for a draft-scoped copy.
type RoleAssignment struct {
	ID          string    `json:"id"`
	DraftID     string    `json:"draft_id,omitempty"`
	RoleID      string    `json:"role_id"`
	ObjectiveID string    `json:"objective_id"`
	Label       RACILabel `json:"label"`
	CreatedAt   string    `json:"created_at"`
	LastUpdated string    `json:"last_updated"`
	Deleted     bool      `json:"deleted"`
}
