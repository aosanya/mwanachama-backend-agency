package models

import "encoding/json"

type WorkItem struct {
	ID               string         `json:"id"`
	Code             string         `json:"code"`
	WorkflowID       string         `json:"workflow_id"`
	DraftID          string         `json:"draft_id,omitempty"`
	Title            string         `json:"title"`
	Description      string         `json:"description"`
	Ordinality       int            `json:"ordinality"`
	Prompt           string         `json:"prompt"`
	CreatedAt        string         `json:"created_at"`
	LastUpdated      string         `json:"last_updated"`
	Deleted          bool           `json:"deleted"`
	Kind             StepKind       `json:"kind"`
	RoleID           string         `json:"role_id"`
	RunWorkflowID    string         `json:"run_workflow_id"`
	FormName         string         `json:"form_name"`
	Questions        []StepQuestion `json:"questions,omitempty"`
	Tool             string         `json:"tool"`
	RecipientRoleID  string         `json:"recipient_role_id"`
	SentStepID       string         `json:"sent_step_id"`
	Routes           []StepRoute    `json:"routes,omitempty"`
	OtherwiseStepID  string         `json:"otherwise_step_id"`
	TimeLimitMinutes int            `json:"time_limit_minutes"`
	CancelWorkflowID string         `json:"cancel_workflow_id"`
	AgentRoleID      string         `json:"agent_role_id"`
	Topic            string         `json:"topic"`
	OpeningLine      string         `json:"opening_line"`
	Prompts          []string       `json:"prompts,omitempty"`
	Tools            []string       `json:"tools,omitempty"`
	StartWorkflowIDs []string       `json:"start_workflow_ids,omitempty"`
	Function         string         `json:"function"`
	Bindings         []StepBinding  `json:"bindings,omitempty"`
	ResultName       string         `json:"result_name"`
	OnFailure        *StepFailure   `json:"on_failure,omitempty"`
	AmountStepID     string         `json:"amount_step_id"`
	Account          string         `json:"account"`
	CounterAccount   string         `json:"counter_account"`
	Direction        string         `json:"direction"`
	Reason           string         `json:"reason"`
	Hold             *StepHold      `json:"hold,omitempty"`
	HeldStepID       string         `json:"held_step_id"`
	Action           string         `json:"action,omitempty" spec:"-"`

	carriedKeys
}

func (wi *WorkItem) UnmarshalJSON(data []byte) error {
	type plain WorkItem
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*wi = WorkItem(decoded)
	return wi.record(data)
}
