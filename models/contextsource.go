package models

// ContextSourceType identifies which kind of context a ContextSource
// fetches.
type ContextSourceType string

const (
	// ContextSourceGit fetches file-signal context from mwanachama-backend-git.
	ContextSourceGit ContextSourceType = "git"
	// ContextSourceComm fetches conversation-thread context from mwanachama-backend-comm.
	ContextSourceComm ContextSourceType = "comm"
	// ContextSourceWork fetches task-detail context from mwanachama-backend-taskmanager.
	ContextSourceWork ContextSourceType = "work"
)

// GitContextSourceConfig configures a ContextSourceGit source.
type GitContextSourceConfig struct {
	// Signals is a comma-separated list of signal layers, e.g. "authority,contributor".
	Signals string `json:"signals,omitempty"`
	// MaxResults caps the number of files returned (0 means the caller's default).
	MaxResults int `json:"max_results,omitempty"`
	// MatchMode is "AND" or "OR".
	MatchMode string `json:"match_mode,omitempty"`
	// Cascade expands keywords to taxonomy descendants when true.
	Cascade bool `json:"cascade,omitempty"`
	// FileTypes is a comma-separated extension filter, e.g. ".go,.ts".
	FileTypes string `json:"file_types,omitempty"`
}

// CommContextSourceConfig configures a ContextSourceComm source.
type CommContextSourceConfig struct {
	LookbackDays int `json:"lookback_days,omitempty"`
	MaxResults   int `json:"max_results,omitempty"`
}

// WorkContextSourceConfig configures a ContextSourceWork source.
type WorkContextSourceConfig struct {
	IncludeDescription bool `json:"include_description,omitempty"`
	IncludeHistory      bool `json:"include_history,omitempty"`
}

// ContextSource is a polymorphic, typed piece of context configuration
// attached to a WorkPlan. Exactly one of Git, Comm, or Work is non-nil,
// selected by SourceType.
//
// Ported from CodeValdAgency's ContextSource, which stored this as one of
// three heterogeneous entitygraph entity types. GORM has no equivalent
// polymorphic-entity concept, so this port flattens all three configs'
// fields onto one row (see gormstore.ContextSourceRow) rather than
// reintroducing a type-per-kind table.
type ContextSource struct {
	ID         string            `json:"id"`
	WorkPlanID string            `json:"work_plan_id"`
	SourceType ContextSourceType `json:"source_type"`

	Git  *GitContextSourceConfig  `json:"git,omitempty"`
	Comm *CommContextSourceConfig `json:"comm,omitempty"`
	Work *WorkContextSourceConfig `json:"work,omitempty"`

	CreatedAt string `json:"created_at"`
}
