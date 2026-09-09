package models

// AgencyPublication is a versioned, content-hashed, tagged snapshot of the
// Agency's live data, created by PublishAgency. Ported from CodeValdAgency's
// AgencyPublication, folding in what CodeValdAgency modeled as a separate
// mutable AgencyPublicationStatus entity — a GORM row is already mutable,
// so Status is just a field here, updated in place.
type AgencyPublication struct {
	ID string `json:"id"`
	// DraftID is "" when this publication was produced with no draft
	// (a re-tag of whatever was already live).
	DraftID     string `json:"draft_id,omitempty"`
	ContentHash string `json:"content_hash"`
	Version     int    `json:"version"`
	// Tag is "v<version>".
	Tag string `json:"tag"`
	// Status is "active" or "archived". Only one publication is active at
	// a time — publishing a new one archives whichever was active before.
	Status      string `json:"status"`
	PublishedAt string `json:"published_at"`
	LastUpdated string `json:"last_updated"`
}
