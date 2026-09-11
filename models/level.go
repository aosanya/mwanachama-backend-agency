package models

// Level is a named rung in an Agency's department tree (e.g. "Division",
// "Team"). Depth is a pure sort key — levels may share a Depth or leave
// gaps, matching mwanachama-backend-actor's Level.
type Level struct {
	ID string `json:"id"`
	// Code is a stable, human-readable identifier (e.g. "L-1") assigned
	// once at creation and never changed — see [Goal.Code]'s doc comment.
	Code        string `json:"code"`
	Name        string `json:"name"`
	Depth       int    `json:"depth"`
	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
	Deleted     bool   `json:"deleted"`
}
