package models

import "encoding/json"

type AgencyDocument struct {
	Agency             DocumentAgency              `json:"agency"`
	Roles              []DocumentRole              `json:"roles"`
	Goals              []DocumentGoal              `json:"goals"`
	Objectives         []DocumentObjective         `json:"objectives"`
	Workflows          []DocumentWorkflow          `json:"workflows"`
	WorkflowObjectives []DocumentWorkflowObjective `json:"workflow_objectives"`
	WorkItems          []DocumentWorkItem          `json:"work_items"`
	RoleAssignments    []DocumentRoleAssignment    `json:"role_assignments"`
	Values             []DocumentStatement         `json:"values"`
	Rules              []DocumentStatement         `json:"rules"`
	ValueAttachments   []DocumentAttachment        `json:"value_attachments"`
	RuleAttachments    []DocumentAttachment        `json:"rule_attachments"`
	Modules            []DocumentModule            `json:"modules"`
	EventFlowSteps     []EventFlowStep             `json:"event_flow_steps"`
	Records            []DocumentRecord            `json:"records,omitempty"`
	Measures           []DocumentMeasure           `json:"measures,omitempty"`
	Scenarios          []DocumentScenario          `json:"scenarios,omitempty"`
}

type DocumentAgency struct {
	Name                         string `json:"name"`
	Description                  string `json:"description"`
	Enabled                      bool   `json:"enabled"`
	DefaultFailurePipelineBudget int    `json:"default_failure_pipeline_budget"`
	InactivityTimeoutSeconds     int    `json:"inactivity_timeout_seconds"`
	EventFlows                   string `json:"event_flows"`
}

type DocumentRole struct {
	ID          string   `json:"id"`
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions,omitempty"`
	Staff       bool     `json:"staff,omitempty"`
}

type DocumentGoal struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Ordinality  int    `json:"ordinality"`
}

type DocumentObjective struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	GoalID      string `json:"goal_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Ordinality  int    `json:"ordinality"`
}

type DocumentWorkflow struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Ordinality  int    `json:"ordinality"`
	EventFlows  string `json:"event_flows"`
}

type DocumentWorkItem struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	WorkflowID  string `json:"workflow_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Ordinality  int    `json:"ordinality"`
	Prompt      string `json:"prompt"`

	LegacyObjectiveID string `json:"objective_id,omitempty"`

	Kind          StepKind       `json:"kind,omitempty"`
	RoleID        string         `json:"role_id,omitempty"`
	RunWorkflowID string         `json:"run_workflow_id,omitempty"`
	FormName      string         `json:"form_name,omitempty"`
	Questions     []StepQuestion `json:"questions,omitempty"`

	RecipientRoleID string      `json:"recipient_role_id,omitempty"`
	SentStepID      string      `json:"sent_step_id,omitempty"`
	Routes          []StepRoute `json:"routes,omitempty"`
	OtherwiseStepID string      `json:"otherwise_step_id,omitempty"`

	TimeLimitMinutes int    `json:"time_limit_minutes,omitempty"`
	CancelWorkflowID string `json:"cancel_workflow_id,omitempty"`
}

type DocumentWorkflowObjective struct {
	ID          string `json:"id"`
	WorkflowID  string `json:"workflow_id"`
	ObjectiveID string `json:"objective_id"`
}

type DocumentRoleAssignment struct {
	ID         string `json:"id,omitempty"`
	WorkItemID string `json:"work_item_id"`
	RoleID     string `json:"role_id"`
	Label      string `json:"label"`

	LegacyObjectiveID string `json:"objective_id,omitempty"`
}

type DocumentStatement struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Text string `json:"text"`
}
type DocumentAttachment struct {
	ValueID    string `json:"value_id,omitempty"`
	ValueCode  string `json:"value_code,omitempty"`
	RuleID     string `json:"rule_id,omitempty"`
	RuleCode   string `json:"rule_code,omitempty"`
	TargetType string `json:"target_type"`
	TargetCode string `json:"target_code"`
}

type DocumentModule struct {
	ID       string          `json:"id"`
	Code     string          `json:"code"`
	Enabled  bool            `json:"enabled"`
	Spec     json.RawMessage `json:"spec,omitempty"`
	Settings json.RawMessage `json:"settings,omitempty"`
	Mounts   json.RawMessage `json:"mounts,omitempty"`
}
