package models

// ObjectiveModule links an Objective to a Module — the many-to-many cross
// table AGD-002 chose over a ModuleID column on Objective, so one
// Objective can reference several Modules and one Module can back many
// Objectives.
//
// Draft-scoped like RoleAssignment, and enforcing the same same-scope
// rule: the Objective and Module it names must both exist in this row's
// own scope (a Draft's Objective can only link a Module already forked
// into that same Draft; a live Objective can only link a live Module —
// AGD-002 decision #6). DraftID is "" for a live link, or the owning
// Draft's id for a draft-scoped copy.
type ObjectiveModule struct {
	ID          string `json:"id"`
	DraftID     string `json:"draft_id,omitempty"`
	ObjectiveID string `json:"objective_id"`
	ModuleID    string `json:"module_id"`
	// Rationale is why this Objective needs this Module; CreateObjectiveModule
	// rejects a blank one.
	Rationale   string `json:"rationale"`
	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
	Deleted     bool   `json:"deleted"`
}
