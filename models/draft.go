package models

// DraftStatus is a Draft's position in its edit lifecycle. Ported from
// CodeValdAgency's AgencyDraftStatus.
type DraftStatus string

const (
	DraftStatusOpen     DraftStatus = "open"
	DraftStatusPromoted DraftStatus = "promoted"
	DraftStatusArchived DraftStatus = "archived"
)

// CanTransitionTo reports whether next is a legal transition from s. Open
// is the only non-terminal status: it can promote or be archived; promoted
// and archived are both terminal.
func (s DraftStatus) CanTransitionTo(next DraftStatus) bool {
	if s != DraftStatusOpen {
		return false
	}
	return next == DraftStatusPromoted || next == DraftStatusArchived
}

// Draft is a header/pointer entity for an in-progress edit of an Agency's
// sub-graph — it does not itself carry a copy of that sub-graph. Ported
// from CodeValdAgency's AgencyDraft, dropping AgencyID (single-agency
// deployment, no cross-agency scoping needed yet).
type Draft struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	// ForkedFromID is an Agency.ID when ForkedFromType is "live", or
	// another Draft.ID when it is "draft".
	ForkedFromID   string      `json:"forked_from_id"`
	ForkedFromType string      `json:"forked_from_type"`
	Status         DraftStatus `json:"status"`
	CreatedAt      string      `json:"created_at"`
	LastUpdated    string      `json:"last_updated"`
	Deleted        bool        `json:"deleted"`
}
