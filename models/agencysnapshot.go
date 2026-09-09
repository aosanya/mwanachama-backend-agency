package models

// AgencySnapshot is an immutable record written each time a Draft is
// promoted. Ported from CodeValdAgency's AgencySnapshot.
type AgencySnapshot struct {
	ID string `json:"id"`
	// DraftID is the Draft that produced this snapshot. CodeValdAgency
	// declared this field required but never populated it — fixed here.
	DraftID    string `json:"draft_id"`
	SnapshotAt string `json:"snapshot_at"`
}
