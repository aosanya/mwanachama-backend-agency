package models

import "encoding/json"

type Record struct {
	ID          string         `json:"id"`
	Code        string         `json:"code"`
	Key         string         `json:"key"`
	DraftID     string         `json:"draft_id,omitempty"`
	WorkflowID  string         `json:"workflow_id"`
	Name        string         `json:"name"`
	FormName    string         `json:"form_name"`
	Questions   []StepQuestion `json:"questions,omitempty"`
	CreatedAt   string         `json:"created_at"`
	LastUpdated string         `json:"last_updated"`
	Deleted     bool           `json:"deleted"`

	ItemName         string `json:"item_name"`
	AddReplacedBy    string `json:"add_replaced_by"`
	EditReplacedBy   string `json:"edit_replaced_by"`
	DeleteReplacedBy string `json:"delete_replaced_by"`

	carriedRoleFields []string
}

func (r Record) Identity() string {
	if r.Key != "" {
		return r.Key
	}
	return r.Code
}

func (r DocumentRecord) Identity() string {
	if r.Key != "" {
		return r.Key
	}
	return r.Code
}

type RecordRow struct {
	ID           string            `json:"id"`
	DraftID      string            `json:"draft_id,omitempty"`
	RecordID     string            `json:"record_id"`
	RecordCode   string            `json:"record_code"`
	RoleID       string            `json:"role_id"`
	AddedBy      string            `json:"added_by"`
	Answers      map[string]string `json:"answers,omitempty"`
	PlaceRefs    map[string]string `json:"place_refs,omitempty"`
	OrderID      string            `json:"order_id,omitempty"`
	SubjectRowID string            `json:"subject_row_id,omitempty"`
	Tags         []string          `json:"tags"`
	Replaces     string            `json:"replaces,omitempty"`
	CorrectedOn  string            `json:"corrected_on,omitempty"`
	SendKey      string            `json:"-"`
	CreatedAt    string            `json:"created_at"`
	LastUpdated  string            `json:"last_updated"`
	Deleted      bool              `json:"deleted"`

	DeletedBy     string `json:"deleted_by,omitempty"`
	DeletedReason string `json:"deleted_reason,omitempty"`
	DeletedOn     string `json:"deleted_on,omitempty"`

	ReturnedBy     string `json:"returned_by,omitempty"`
	ReturnedByID   string `json:"returned_by_id,omitempty"`
	ReturnedReason string `json:"returned_reason,omitempty"`
	ReturnedOn     string `json:"returned_on,omitempty"`
}

type DocumentRecord struct {
	ID         string         `json:"id"`
	Code       string         `json:"code"`
	Key        string         `json:"key,omitempty"`
	WorkflowID string         `json:"workflow_id"`
	Name       string         `json:"name"`
	FormName   string         `json:"form_name"`
	Questions  []StepQuestion `json:"questions,omitempty"`
	Deleted    bool           `json:"deleted,omitempty"`

	ItemName         string `json:"item_name,omitempty"`
	AddReplacedBy    string `json:"add_replaced_by,omitempty"`
	EditReplacedBy   string `json:"edit_replaced_by,omitempty"`
	DeleteReplacedBy string `json:"delete_replaced_by,omitempty"`

	carriedRoleFields []string
}

func (r RecordRow) MarshalJSON() ([]byte, error) {
	type plain RecordRow
	if r.Tags == nil {
		r.Tags = []string{}
	}
	return json.Marshal(plain(r))
}
