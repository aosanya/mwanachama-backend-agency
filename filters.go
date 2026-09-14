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

type WorkPlanFilter struct {
	Code        string
	Name        string
	Description string
}

// ContextSourceFilter's SourceType is exact-match against
// [models.ContextSourceType]'s values ("git", "comm", "work").
type ContextSourceFilter struct {
	SourceType string
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

type ValueFilter struct {
	Code string
	Text string
}

type RuleFilter struct {
	Code string
	Text string
}

// IssueFilter's Severity/EntityType are exact-match against
// [models.IssueSeverity]'s three values / the entity noun ("objective",
// "goal", ...) an Issue points at.
type IssueFilter struct {
	Message    string
	Severity   string
	EntityType string
}
