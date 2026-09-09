package models

// WorkPlan is an execution plan that binds an inbound event trigger to an AI
// agent's instructions. Ported from CodeValdAgency's WorkPlan (stranded
// under the misleading filename raci.go — that file implements no RACI
// logic at all; see this repo's requirements.md), dropping AgencyID
// (single-agency deployment, no cross-agency scoping needed yet).
//
// A caller (e.g. mwanachama-backend-api-kazi) matches an inbound event's
// topic and JSON payload against every enabled WorkPlan's TriggerTopic/
// PayloadCondition regex to decide which plan's Instructions to run.
type WorkPlan struct {
	ID string `json:"id"`

	// Code is the stable slug other WorkPlans reference via
	// OnFailurePipeline. Unlike the source's CreateWorkPlanRequest (which
	// had no Code field at all, leaving every WorkPlan's Code permanently
	// empty — a latent bug), this port takes the domain type directly on
	// Create, so a caller can set Code from the start.
	Code string `json:"code,omitempty"`

	Name             string `json:"name"`
	Description      string `json:"description,omitempty"`
	TriggerTopic     string `json:"trigger_topic"`
	PayloadCondition string `json:"payload_condition,omitempty"`
	Instructions     string `json:"instructions,omitempty"`

	AgentID        string `json:"agent_id,omitempty"`
	AgentCode      string `json:"agent_code,omitempty"`
	FunctionCode   string `json:"function_code,omitempty"`
	FunctionParams string `json:"function_params,omitempty"`
	HandlerService string `json:"handler_service,omitempty"`

	Enabled    bool `json:"enabled"`
	Ordinality int  `json:"ordinality"`

	SuccessEvent      string `json:"success_event,omitempty"`
	FailureEvent      string `json:"failure_event,omitempty"`
	OnFailurePipeline string `json:"on_failure_pipeline,omitempty"`
	StepTimeout       string `json:"step_timeout,omitempty"`

	ReviewStepType     string `json:"review_step_type,omitempty"`
	ReviewTriggerTopic string `json:"review_trigger_topic,omitempty"`
	ReviewSuccessTopic string `json:"review_success_topic,omitempty"`
	ReviewFailureTopic string `json:"review_failure_topic,omitempty"`

	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
	Deleted     bool   `json:"deleted"`
}

// WorkPlanMatch pairs a matched WorkPlan with every ContextSource linked to
// it.
type WorkPlanMatch struct {
	WorkPlan       WorkPlan        `json:"work_plan"`
	ContextSources []ContextSource `json:"context_sources,omitempty"`
}
