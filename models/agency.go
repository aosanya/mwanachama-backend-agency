package models

// Agency is the root of the declarative document this package models: the
// Goals/Objectives and Workflows/WorkItems recording how the Agency
// intends to operate. Ported from CodeValdAgency's Agency (models.go),
// dropping the ArangoDB/entitygraph storage fields. Mission/Vision were
// dropped 2026-09-10 (AGD-001) — confirmed with the user that Values/Rules
// (see [Value], [Rule]) do the job they were meant to, and better, since
// they're explicit enough for an AI agent acting as a Role to read
// directly.
type Agency struct {
	ID                           string `json:"id"`
	Name                         string `json:"name"`
	Enabled                      bool   `json:"enabled"`
	DefaultFailurePipelineBudget int    `json:"default_failure_pipeline_budget"`
	InactivityTimeoutSeconds     int    `json:"inactivity_timeout_seconds"`
	// EventFlows is a raw JSON blob, ported as-is from CodeValdAgency —
	// Workflow.EventFlows falls back to this value when its own is empty.
	EventFlows  string `json:"event_flows,omitempty"`
	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
	Deleted     bool   `json:"deleted"`
}
