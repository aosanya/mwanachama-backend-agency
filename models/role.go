package models

import "encoding/json"

type Role struct {
	ID                 string   `json:"id"`
	Code               string   `json:"code"`
	DraftID            string   `json:"draft_id,omitempty"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	Permissions        []string `json:"permissions,omitempty"`
	Staff              bool     `json:"staff"`
	ApprovesAboveLimit bool     `json:"approves_above_limit"`
	CreatedAt          string   `json:"created_at"`
	LastUpdated        string   `json:"last_updated"`
	Deleted            bool     `json:"deleted"`

	carriedKeys
}

func (r *Role) UnmarshalJSON(data []byte) error {
	type plain Role
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*r = Role(decoded)
	return r.record(data)
}
