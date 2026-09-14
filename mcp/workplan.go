package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// registerWorkPlanTools registers WorkPlan/ContextSource tools — an
// execution plan binding an inbound event trigger to an AI agent's
// instructions, plus the context sources it reads before running.
func registerWorkPlanTools(server *mcp.Server, resolve ManagerResolver) {
	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_work_plan", Title: "Create Work Plan", Description: "Create a WorkPlan."}, mcpCreateWorkPlan(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_work_plan", Title: "Get Work Plan", Description: "Get a WorkPlan by id."}, mcpGetWorkPlan(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_work_plans", Title: "List Work Plans", Description: "List every WorkPlan."}, mcpListWorkPlans(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_update_work_plan", Title: "Update Work Plan", Description: "Update a WorkPlan."}, mcpUpdateWorkPlan(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_match_work_plans", Title: "Match Work Plans", Description: "Find every enabled WorkPlan whose trigger topic (and payload condition, if set) matches an event, ordered by ordinality."}, mcpMatchWorkPlans(resolve))

	mcp.AddTool(server, &mcp.Tool{Name: "agency_add_context_source", Title: "Add Context Source", Description: "Attach a ContextSource (git/comm/work) to a WorkPlan."}, mcpAddContextSource(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_context_sources", Title: "List Context Sources", Description: "List the ContextSources attached to a WorkPlan."}, mcpListContextSources(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_remove_context_source", Title: "Remove Context Source", Description: "Detach a ContextSource from a WorkPlan."}, mcpRemoveContextSource(resolve))
}

// mcpWorkPlanParams mirrors models.WorkPlan, but with every field but Name
// marked omitempty — the go-sdk's schema inference (google/jsonschema-go)
// treats any field lacking omitempty/omitzero as required, and
// models.WorkPlan's own json tags (decision #1: imported unmodified) leave
// several (Name, TriggerTopic, Enabled, Ordinality, ...) without one. ID is
// required for update, ignored for create.
type mcpWorkPlanParams struct {
	InstanceID         string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ID                 string `json:"id,omitempty" jsonschema:"the WorkPlan's id — required for update, ignored for create"`
	Code               string `json:"code,omitempty" jsonschema:"the stable slug other WorkPlans reference via on_failure_pipeline"`
	Name               string `json:"name" jsonschema:"the WorkPlan's name"`
	Description        string `json:"description,omitempty"`
	TriggerTopic       string `json:"trigger_topic,omitempty" jsonschema:"the event topic that triggers this WorkPlan"`
	PayloadCondition   string `json:"payload_condition,omitempty" jsonschema:"a regex the event's JSON payload must also match"`
	Instructions       string `json:"instructions,omitempty" jsonschema:"the instructions given to the AI agent handling this WorkPlan"`
	AgentID            string `json:"agent_id,omitempty"`
	AgentCode          string `json:"agent_code,omitempty"`
	FunctionCode       string `json:"function_code,omitempty"`
	FunctionParams     string `json:"function_params,omitempty"`
	HandlerService     string `json:"handler_service,omitempty"`
	Enabled            bool   `json:"enabled,omitempty" jsonschema:"whether agency_match_work_plans may match this WorkPlan"`
	Ordinality         int    `json:"ordinality,omitempty" jsonschema:"sort order among matched WorkPlans"`
	SuccessEvent       string `json:"success_event,omitempty"`
	FailureEvent       string `json:"failure_event,omitempty"`
	OnFailurePipeline  string `json:"on_failure_pipeline,omitempty" jsonschema:"another WorkPlan's code to run on failure"`
	StepTimeout        string `json:"step_timeout,omitempty"`
	ReviewStepType     string `json:"review_step_type,omitempty"`
	ReviewTriggerTopic string `json:"review_trigger_topic,omitempty"`
	ReviewSuccessTopic string `json:"review_success_topic,omitempty"`
	ReviewFailureTopic string `json:"review_failure_topic,omitempty"`
}

func (p mcpWorkPlanParams) toModel() models.WorkPlan {
	return models.WorkPlan{
		ID: p.ID, Code: p.Code, Name: p.Name, Description: p.Description,
		TriggerTopic: p.TriggerTopic, PayloadCondition: p.PayloadCondition, Instructions: p.Instructions,
		AgentID: p.AgentID, AgentCode: p.AgentCode, FunctionCode: p.FunctionCode,
		FunctionParams: p.FunctionParams, HandlerService: p.HandlerService,
		Enabled: p.Enabled, Ordinality: p.Ordinality,
		SuccessEvent: p.SuccessEvent, FailureEvent: p.FailureEvent, OnFailurePipeline: p.OnFailurePipeline, StepTimeout: p.StepTimeout,
		ReviewStepType: p.ReviewStepType, ReviewTriggerTopic: p.ReviewTriggerTopic,
		ReviewSuccessTopic: p.ReviewSuccessTopic, ReviewFailureTopic: p.ReviewFailureTopic,
	}
}

func mcpCreateWorkPlan(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpWorkPlanParams) (*mcp.CallToolResult, models.WorkPlan, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpWorkPlanParams) (*mcp.CallToolResult, models.WorkPlan, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.WorkPlan{}, err
		}
		out, err := m.CreateWorkPlan(ctx, p.toModel())
		return summary("Created WorkPlan %q (%s)", out.Name, out.ID), out, err
	}
}

type mcpWorkPlanIDParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	WorkPlanID string `json:"work_plan_id" jsonschema:"the WorkPlan's id"`
}

func mcpGetWorkPlan(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpWorkPlanIDParams) (*mcp.CallToolResult, models.WorkPlan, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpWorkPlanIDParams) (*mcp.CallToolResult, models.WorkPlan, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.WorkPlan{}, err
		}
		out, err := m.GetWorkPlan(ctx, p.WorkPlanID)
		return summary("WorkPlan %q (%s)", out.Name, out.ID), out, err
	}
}

// mcpListWorkPlansParams extends instanceOnlyParams with WorkPlanFilter's
// fields.
type mcpListWorkPlansParams struct {
	instanceOnlyParams
	Code        string `json:"code,omitempty" jsonschema:"filter: case-insensitive substring match on Code"`
	Name        string `json:"name,omitempty" jsonschema:"filter: case-insensitive substring match on Name"`
	Description string `json:"description,omitempty" jsonschema:"filter: case-insensitive substring match on Description"`
}

func mcpListWorkPlans(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpListWorkPlansParams) (*mcp.CallToolResult, ListResult[models.WorkPlan], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpListWorkPlansParams) (*mcp.CallToolResult, ListResult[models.WorkPlan], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.WorkPlan]{}, err
		}
		out, err := m.ListWorkPlans(ctx, mwanachamaagency.WorkPlanFilter{Code: p.Code, Name: p.Name, Description: p.Description})
		return summary("Found %d work plan(s)", len(out)), ListResult[models.WorkPlan]{Items: out}, err
	}
}

func mcpUpdateWorkPlan(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpWorkPlanParams) (*mcp.CallToolResult, models.WorkPlan, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpWorkPlanParams) (*mcp.CallToolResult, models.WorkPlan, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.WorkPlan{}, err
		}
		out, err := m.UpdateWorkPlan(ctx, p.toModel())
		return summary("Updated WorkPlan %q (%s)", out.Name, out.ID), out, err
	}
}

type mcpMatchWorkPlansParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	Topic      string `json:"topic" jsonschema:"the event's topic"`
	Payload    string `json:"payload,omitempty" jsonschema:"the event's JSON payload, checked against each candidate's payload_condition"`
}

func mcpMatchWorkPlans(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpMatchWorkPlansParams) (*mcp.CallToolResult, ListResult[models.WorkPlanMatch], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpMatchWorkPlansParams) (*mcp.CallToolResult, ListResult[models.WorkPlanMatch], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.WorkPlanMatch]{}, err
		}
		out, err := m.MatchWorkPlans(ctx, p.Topic, p.Payload)
		return summary("Found %d work plan(s) matching topic %q", len(out), p.Topic), ListResult[models.WorkPlanMatch]{Items: out}, err
	}
}

// mcpAddContextSourceParams mirrors models.ContextSource, but with
// SourceType's own ID/WorkPlanID/CreatedAt fields dropped — those are
// either supplied separately (WorkPlanID) or server-managed (ID,
// CreatedAt), and models.ContextSource's own json tags (decision #1:
// imported unmodified) don't mark them omitempty. Git/Comm/Work are
// already omitempty on the model itself — exactly one should be set,
// matching source_type.
type mcpAddContextSourceParams struct {
	InstanceID string                          `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	WorkPlanID string                          `json:"work_plan_id" jsonschema:"the owning WorkPlan's id"`
	SourceType string                          `json:"source_type" jsonschema:"git, comm, or work — selects which of git/comm/work below applies"`
	Git        *models.GitContextSourceConfig  `json:"git,omitempty"`
	Comm       *models.CommContextSourceConfig `json:"comm,omitempty"`
	Work       *models.WorkContextSourceConfig `json:"work,omitempty"`
}

func mcpAddContextSource(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpAddContextSourceParams) (*mcp.CallToolResult, models.ContextSource, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpAddContextSourceParams) (*mcp.CallToolResult, models.ContextSource, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.ContextSource{}, err
		}
		out, err := m.AddContextSource(ctx, p.WorkPlanID, models.ContextSource{
			SourceType: models.ContextSourceType(p.SourceType),
			Git:        p.Git,
			Comm:       p.Comm,
			Work:       p.Work,
		})
		return summary("Added %s ContextSource %s to WorkPlan %s", out.SourceType, out.ID, out.WorkPlanID), out, err
	}
}

// mcpListContextSourcesParams is its own params struct (not
// mcpWorkPlanIDParams, shared with mcpGetWorkPlan) so SourceType's filter
// field doesn't leak onto the Get tool.
type mcpListContextSourcesParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	WorkPlanID string `json:"work_plan_id" jsonschema:"the owning WorkPlan's id"`
	SourceType string `json:"source_type,omitempty" jsonschema:"filter: exact match on SourceType (\"git\", \"comm\", or \"work\")"`
}

func mcpListContextSources(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpListContextSourcesParams) (*mcp.CallToolResult, ListResult[models.ContextSource], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpListContextSourcesParams) (*mcp.CallToolResult, ListResult[models.ContextSource], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.ContextSource]{}, err
		}
		out, err := m.ListContextSources(ctx, p.WorkPlanID, mwanachamaagency.ContextSourceFilter{SourceType: p.SourceType})
		return summary("Found %d context source(s) on WorkPlan %s", len(out), p.WorkPlanID), ListResult[models.ContextSource]{Items: out}, err
	}
}

type mcpRemoveContextSourceParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	WorkPlanID string `json:"work_plan_id" jsonschema:"the owning WorkPlan's id"`
	SourceID   string `json:"source_id" jsonschema:"the ContextSource's id"`
}

func mcpRemoveContextSource(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpRemoveContextSourceParams) (*mcp.CallToolResult, DeleteResult, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpRemoveContextSourceParams) (*mcp.CallToolResult, DeleteResult, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, DeleteResult{}, err
		}
		if err := m.RemoveContextSource(ctx, p.WorkPlanID, p.SourceID); err != nil {
			return nil, DeleteResult{}, err
		}
		return summary("Removed ContextSource %s from WorkPlan %s", p.SourceID, p.WorkPlanID), DeleteResult{Deleted: true}, nil
	}
}
