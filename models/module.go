package models

// Module is a catalog entry for a capability domain (e.g. "git",
// "taskmanager") this Agency may enable. Enabled is a real gate on a
// mounting gateway's route registration, not a passive record: an
// unselected Module's routes are absent entirely, not just
// capability-denied (AGD-002 decision #1).
//
// Code is the module's stable slug, matching a sibling backend repo's own
// name — supplied by the caller at creation, not minted via the
// [gormstore.NextCode] sequence every other business-entity Code goes
// through (AGD-002 decision #2/#3; see [KnownModuleCodes]).
//
// Same lax draft-scoping as Goal, not RoleAssignment's stricter rule: a
// live Module can be toggled directly (AGD-002 decision #5). DraftID is ""
// for a live Module, or the owning Draft's id for a draft-scoped copy.
type Module struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	DraftID     string `json:"draft_id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
	Deleted     bool   `json:"deleted"`
}

// KnownModuleCodes is the single source of truth for which Module codes
// exist — the sibling backend repos AGD-002 confirmed ship their own
// routes/ package (git, taskmanager, agency, actor, auth, comm, custody,
// forms) plus digitaltwin/accounting, listed even though neither has a
// routes/ package yet (AGD-002's own open item). Reused by
// [gormstore.BackfillModules] and [SeedNeoLabHQ] so both stay in sync with
// this one list.
var KnownModuleCodes = []string{
	"git", "taskmanager", "agency", "actor", "auth", "comm", "custody",
	"forms", "digitaltwin", "accounting",
}
