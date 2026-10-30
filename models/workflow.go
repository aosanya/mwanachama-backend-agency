package models

type Workflow struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	DraftID     string `json:"draft_id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Ordinality  int    `json:"ordinality"`
	EventFlows  string `json:"event_flows,omitempty"`
	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
	Deleted     bool   `json:"deleted"`

	Default        bool  `json:"default,omitempty" spec:"-"`
	InUse          bool  `json:"in_use" spec:"-"`
	Version        int   `json:"version" spec:"-"`
	OrdersUnderWay int   `json:"orders_under_way" spec:"-"`
	OrderVersions  []int `json:"order_versions,omitempty" spec:"-"`
}
