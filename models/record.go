package models

type Record struct {
	ID           string         `json:"id"`
	Code         string         `json:"code"`
	DraftID      string         `json:"draft_id,omitempty"`
	WorkflowID   string         `json:"workflow_id"`
	Name         string         `json:"name"`
	RoleID       string         `json:"role_id"`
	ViewerRoleID string         `json:"viewer_role_id"`
	FormName     string         `json:"form_name"`
	Questions    []StepQuestion `json:"questions,omitempty"`
	CreatedAt    string         `json:"created_at"`
	LastUpdated  string         `json:"last_updated"`
	Deleted      bool           `json:"deleted"`

	ItemName         string `json:"item_name"`
	EditRoleID       string `json:"edit_role_id"`
	DeleteRoleID     string `json:"delete_role_id"`
	AddReplacedBy    string `json:"add_replaced_by"`
	EditReplacedBy   string `json:"edit_replaced_by"`
	DeleteReplacedBy string `json:"delete_replaced_by"`
}

type RecordRow struct {
	ID          string            `json:"id"`
	DraftID     string            `json:"draft_id,omitempty"`
	RecordID    string            `json:"record_id"`
	RecordCode  string            `json:"record_code"`
	RoleID      string            `json:"role_id"`
	AddedBy     string            `json:"added_by"`
	Answers     map[string]string `json:"answers,omitempty"`
	PlaceRefs   map[string]string `json:"place_refs,omitempty"`
	OrderID     string            `json:"order_id,omitempty"`
	Replaces    string            `json:"replaces,omitempty"`
	CorrectedOn string            `json:"corrected_on,omitempty"`
	SendKey     string            `json:"-"`
	CreatedAt   string            `json:"created_at"`
	LastUpdated string            `json:"last_updated"`
	Deleted     bool              `json:"deleted"`

	DeletedBy     string `json:"deleted_by,omitempty"`
	DeletedReason string `json:"deleted_reason,omitempty"`
	DeletedOn     string `json:"deleted_on,omitempty"`
}

type DocumentRecord struct {
	ID           string         `json:"id"`
	Code         string         `json:"code"`
	WorkflowID   string         `json:"workflow_id"`
	Name         string         `json:"name"`
	RoleID       string         `json:"role_id"`
	ViewerRoleID string         `json:"viewer_role_id,omitempty"`
	FormName     string         `json:"form_name"`
	Questions    []StepQuestion `json:"questions,omitempty"`
	Deleted      bool           `json:"deleted,omitempty"`

	ItemName         string `json:"item_name,omitempty"`
	EditRoleID       string `json:"edit_role_id,omitempty"`
	DeleteRoleID     string `json:"delete_role_id,omitempty"`
	AddReplacedBy    string `json:"add_replaced_by,omitempty"`
	EditReplacedBy   string `json:"edit_replaced_by,omitempty"`
	DeleteReplacedBy string `json:"delete_replaced_by,omitempty"`
}
