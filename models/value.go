package models

// Value is a cultural/strategic statement an AI agent acting as a Role
// needs made explicit — the kind of thing a human Role-holder would just
// treat as common knowledge (e.g. "we prioritize correctness over
// speed"). Plain text, informational only — nothing in this package
// enforces it. Net new, added 2026-09-10 (AGD-001), replacing Agency's
// former Mission/Vision fields.
//
// ObjectiveID is "" for an Agency-wide Value, or the Objective it's
// scoped to. Agency-wide Values are set once (at Agency creation/seed
// time) and have no edit path at all, ever: [CreateDraft] deliberately
// never forks them (see draft_impl.go), so their DraftID stays "" for
// the Agency's whole lifetime. Objective-scoped Values follow Role's
// stricter live-immutability rule, not Goal's laxer one — fork a Draft,
// edit the copy, promote. Write authorization (only the Objective's
// Accountable may fork/edit/promote its Values/Rules) is not implemented
// here — flagged in AGD-001 as new machinery needing its own design, not
// built in this pass.
type Value struct {
	ID string `json:"id"`
	// Code is a stable, human-readable identifier (e.g. "V-1") assigned
	// once at creation and never changed — see [Goal.Code]'s doc comment.
	Code        string `json:"code"`
	ObjectiveID string `json:"objective_id,omitempty"`
	// DraftID is "" for a live Value, or the owning Draft's ID for a
	// draft-scoped copy.
	DraftID     string `json:"draft_id,omitempty"`
	Text        string `json:"text"`
	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
	Deleted     bool   `json:"deleted"`
}
