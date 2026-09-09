package models

// Agency is the root of the declarative document this package models: a
// Mission/Vision statement plus the Goals, Workflows, and WorkItems
// recording how the Agency intends to operate. Ported from
// CodeValdAgency's Agency (models.go), dropping the ArangoDB/entitygraph
// storage fields.
type Agency struct {
	ID                           string `json:"id"`
	Name                         string `json:"name"`
	Mission                      string `json:"mission"`
	Vision                       string `json:"vision"`
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
