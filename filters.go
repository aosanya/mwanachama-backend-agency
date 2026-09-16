package mwanachamaagency

// Every Filter type below narrows its matching List method: each non-empty
// field must match, ANDed together. String fields are case-insensitive
// substring matches (see [likeFilter]) except where noted as exact-match
// (see [eqFilter]) — those name an enum column (Status, Label, SourceType)
// rather than free text.

type GoalFilter struct {
	Code        string
	Title       string
	Description string
}

type ObjectiveFilter struct {
	Code        string
	Title       string
	Description string
}

type WorkflowFilter struct {
	Code        string
	Name        string
	Description string
}

type WorkItemFilter struct {
	Code        string
	Title       string
	Description string
}

// DraftFilter's Status is exact-match against [models.DraftStatus]'s
// values ("open", "promoted", "archived").
type DraftFilter struct {
	Description string
	Status      string
}

// PublicationFilter's Status is exact-match ("active" or "archived").
type PublicationFilter struct {
	Tag    string
	Status string
}

type EventFlowStepFilter struct {
	Code        string
	Name        string
	Description string
}

type DepartmentFilter struct {
	Code string
	Name string
}

type LevelFilter struct {
	Code string
	Name string
}

type RoleFilter struct {
	Code        string
	Name        string
	Description string
}

// RoleAssignmentFilter's Label is exact-match against
// [models.RACILabel]'s four values.
type RoleAssignmentFilter struct {
	Label string
}

// ModuleFilter's Code/Name/Description are case-insensitive substring
// matches; Enabled, when non-nil, is an exact match — a *bool rather than
// a bare bool so "not filtering on Enabled" is distinguishable from
// "filtering on Enabled == false".
type ModuleFilter struct {
	Code        string
	Name        string
	Description string
	Enabled     *bool
}

// ObjectiveModuleFilter's ObjectiveID/ModuleID are exact-match narrows.
type ObjectiveModuleFilter struct {
	ObjectiveID string
	ModuleID    string
}

// ValueFilter's Role/Goal/Objective/Workflow/WorkItem ID fields each
// narrow to Values attached to that specific target (AGD-003) — exact
// match against a resolved id, not a substring search like Code/Text.
// Set more than one and a Value must be attached to every target named,
// not just one of them (same AND-all-fields rule as everything else).
type ValueFilter struct {
	Code        string
	Text        string
	RoleID      string
	GoalID      string
	ObjectiveID string
	WorkflowID  string
	WorkItemID  string
}

// RuleFilter's target-id fields work exactly like [ValueFilter]'s.
type RuleFilter struct {
	Code        string
	Text        string
	RoleID      string
	GoalID      string
	ObjectiveID string
	WorkflowID  string
	WorkItemID  string
}

// IssueFilter's Severity/EntityType are exact-match against
// [models.IssueSeverity]'s three values / the entity noun ("objective",
// "goal", ...) an Issue points at.
type IssueFilter struct {
	Message    string
	Severity   string
	EntityType string
}
