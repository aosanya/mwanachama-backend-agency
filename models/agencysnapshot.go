package models

// AgencySnapshot is an immutable record written each time a Draft is
// promoted.
type AgencySnapshot struct {
	ID string `json:"id"`
	// DraftID is the Draft that produced this snapshot. The earlier
	// service declared this field required but never populated it — fixed here.
	DraftID    string `json:"draft_id"`
	SnapshotAt string `json:"snapshot_at"`
}
