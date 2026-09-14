package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// registerGoalWorkflowTools registers Goal/Workflow/WorkItem/EventFlowStep
// tools — the work breakdown: Goals an Agency pursues, Workflows that
// pursue them, WorkItems within a Workflow, and the queryable projection
// of a Workflow's event flow.
//
// Every Create/Update params struct below mirrors its models.X type
// field-for-field but with every field but the genuinely required ones
// marked omitempty — the go-sdk's schema inference (google/jsonschema-go)
// treats any field lacking omitempty/omitzero as required, and
// mwanachama-backend-agency's own json tags (decision #1: imported
// unmodified) carry almost none, so embedding a models.X type directly
// would force a caller to invent values for id/created_at/etc. on every
// call. The shared Create/Update struct's ID is itself omitempty — Create
// ignores it, Update needs it — since the manager's own "not found" error
// already covers a missing one, and jsonschema-go has no way to say
// "required for one caller, not another."
func registerGoalWorkflowTools(server *mcp.Server, resolve ManagerResolver) {
	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_goal", Title: "Create Goal", Description: "Create a Goal for an Agency."}, mcpCreateGoal(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_goal", Title: "Get Goal", Description: "Get a Goal by id."}, mcpGetGoal(resolve))
	mcp.AddTool(server, &mcp.Tool{
		Name: "agency_list_goals", Title: "List Goals", Description: "List Goals — live, or a Draft's scoped copies.",
		Meta: dashboardUIMeta,
	}, mcpListGoals(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_update_goal", Title: "Update Goal", Description: "Update a Goal (full replace — supply every field you want kept, not just the ones changing). id is required."}, mcpUpdateGoal(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_delete_goal", Title: "Delete Goal", Description: "Delete a Goal. Cascades: also deletes every Objective under it, and every RACI assignment on those Objectives — warn the operator about that before calling this."}, mcpDeleteGoal(resolve))

	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_workflow", Title: "Create Workflow", Description: "Create a Workflow for an Agency."}, mcpCreateWorkflow(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_workflow", Title: "Get Workflow", Description: "Get a Workflow by id."}, mcpGetWorkflow(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_workflows", Title: "List Workflows", Description: "List Workflows — live, or a Draft's scoped copies."}, mcpListWorkflows(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_update_workflow", Title: "Update Workflow", Description: "Update a Workflow (full replace). id is required."}, mcpUpdateWorkflow(resolve))

	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_work_item", Title: "Create Work Item", Description: "Create a WorkItem within a Workflow, belonging to an Objective."}, mcpCreateWorkItem(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_work_item", Title: "Get Work Item", Description: "Get a WorkItem by id."}, mcpGetWorkItem(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_work_items", Title: "List Work Items", Description: "List WorkItems — live, or a Draft's scoped copies."}, mcpListWorkItems(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_update_work_item", Title: "Update Work Item", Description: "Update a WorkItem (full replace). id is required."}, mcpUpdateWorkItem(resolve))

	mcp.AddTool(server, &mcp.Tool{Name: "agency_lookup_event_flow_step", Title: "Look Up Event Flow Step", Description: "Look up one EventFlowStep by handler code, or by (trigger topic, consumer), or by its own code — whichever is set, in that precedence order."}, mcpLookupFlowStep(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_event_flow_steps", Title: "List Event Flow Steps", Description: "List the EventFlowSteps for one Workflow, by its code."}, mcpListEventFlowSteps(resolve))
}

type mcpGoalParams struct {
	InstanceID  string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ID          string `json:"id,omitempty" jsonschema:"the Goal's id — required for update, ignored for create"`
	DraftID     string `json:"draft_id,omitempty" jsonschema:"\"\" for a live Goal, or the owning Draft's id for a draft-scoped copy"`
	Title       string `json:"title" jsonschema:"the Goal's title"`
	Description string `json:"description,omitempty" jsonschema:"the Goal's description"`
	Ordinality  int    `json:"ordinality,omitempty" jsonschema:"sort order among sibling Goals"`
}

func (p mcpGoalParams) toModel() models.Goal {
	return models.Goal{ID: p.ID, DraftID: p.DraftID, Title: p.Title, Description: p.Description, Ordinality: p.Ordinality}
}

func mcpCreateGoal(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpGoalParams) (*mcp.CallToolResult, models.Goal, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpGoalParams) (*mcp.CallToolResult, models.Goal, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Goal{}, err
		}
		out, err := m.CreateGoal(ctx, p.toModel())
		return summary("Created Goal %s %q (%s)", out.Code, out.Title, out.ID), out, err
	}
}

type mcpGoalIDParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	GoalID     string `json:"goal_id" jsonschema:"the Goal's id"`
}

func mcpGetGoal(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpGoalIDParams) (*mcp.CallToolResult, models.Goal, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpGoalIDParams) (*mcp.CallToolResult, models.Goal, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Goal{}, err
		}
		out, err := m.GetGoal(ctx, p.GoalID)
		return summary("Goal %s %q (%s)", out.Code, out.Title, out.ID), out, err
	}
}

// mcpListGoalsParams extends instanceAndDraftParams with GoalFilter's
// fields.
type mcpListGoalsParams struct {
	instanceAndDraftParams
	Code        string `json:"code,omitempty" jsonschema:"filter: case-insensitive substring match on Code"`
	Title       string `json:"title,omitempty" jsonschema:"filter: case-insensitive substring match on Title"`
	Description string `json:"description,omitempty" jsonschema:"filter: case-insensitive substring match on Description"`
}

func mcpListGoals(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpListGoalsParams) (*mcp.CallToolResult, ListResult[models.Goal], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpListGoalsParams) (*mcp.CallToolResult, ListResult[models.Goal], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.Goal]{}, err
		}
		out, err := m.ListGoals(ctx, p.DraftID, mwanachamaagency.GoalFilter{Code: p.Code, Title: p.Title, Description: p.Description})
		return summaryWithContext(fmt.Sprintf("Found %d goal(s)", len(out)), goalsContextText(out)), ListResult[models.Goal]{Items: out}, err
	}
}

// goalsContextText is what an LLM sees for a Goals list opened outside its
// own tool loop (mwanachama-wakala-studio's sidebar shortcut) — one line per
// Goal, field-by-field rather than a struct dump, so dropping a field from
// an LLM's context later is just deleting it from this Sprintf.
func goalsContextText(goals []models.Goal) string {
	if len(goals) == 0 {
		return "There are no goals yet."
	}
	lines := make([]string, 0, len(goals))
	for _, g := range goals {
		line := fmt.Sprintf("- %s: %s", g.Code, g.Title)
		if g.Description != "" {
			line += " — " + g.Description
		}
		lines = append(lines, line+fmt.Sprintf(" (goal id: %s)", g.ID))
	}
	return "Current goals:\n" + strings.Join(lines, "\n")
}

func mcpUpdateGoal(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpGoalParams) (*mcp.CallToolResult, models.Goal, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpGoalParams) (*mcp.CallToolResult, models.Goal, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Goal{}, err
		}
		out, err := m.UpdateGoal(ctx, p.toModel())
		return summary("Updated Goal %s %q (%s)", out.Code, out.Title, out.ID), out, err
	}
}

func mcpDeleteGoal(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpGoalIDParams) (*mcp.CallToolResult, DeleteResult, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpGoalIDParams) (*mcp.CallToolResult, DeleteResult, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, DeleteResult{}, err
		}
		if err := m.DeleteGoal(ctx, p.GoalID); err != nil {
			return nil, DeleteResult{}, err
		}
		return summary("Deleted Goal %s", p.GoalID), DeleteResult{Deleted: true}, nil
	}
}

type mcpWorkflowParams struct {
	InstanceID  string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ID          string `json:"id,omitempty" jsonschema:"the Workflow's id — required for update, ignored for create"`
	DraftID     string `json:"draft_id,omitempty" jsonschema:"\"\" for a live Workflow, or the owning Draft's id for a draft-scoped copy"`
	Name        string `json:"name" jsonschema:"the Workflow's name"`
	Description string `json:"description,omitempty" jsonschema:"the Workflow's description"`
	Ordinality  int    `json:"ordinality,omitempty" jsonschema:"sort order among sibling Workflows"`
	EventFlows  string `json:"event_flows,omitempty" jsonschema:"raw JSON event-flow blob; falls back to the owning Agency's when empty"`
}

func (p mcpWorkflowParams) toModel() models.Workflow {
	return models.Workflow{ID: p.ID, DraftID: p.DraftID, Name: p.Name, Description: p.Description, Ordinality: p.Ordinality, EventFlows: p.EventFlows}
}

func mcpCreateWorkflow(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpWorkflowParams) (*mcp.CallToolResult, models.Workflow, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpWorkflowParams) (*mcp.CallToolResult, models.Workflow, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Workflow{}, err
		}
		out, err := m.CreateWorkflow(ctx, p.toModel())
		return summary("Created Workflow %s %q (%s)", out.Code, out.Name, out.ID), out, err
	}
}

type mcpWorkflowIDParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	WorkflowID string `json:"workflow_id" jsonschema:"the Workflow's id"`
}

func mcpGetWorkflow(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpWorkflowIDParams) (*mcp.CallToolResult, models.Workflow, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpWorkflowIDParams) (*mcp.CallToolResult, models.Workflow, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Workflow{}, err
		}
		out, err := m.GetWorkflow(ctx, p.WorkflowID)
		return summary("Workflow %s %q (%s)", out.Code, out.Name, out.ID), out, err
	}
}

// mcpListWorkflowsParams extends instanceAndDraftParams with
// WorkflowFilter's fields.
type mcpListWorkflowsParams struct {
	instanceAndDraftParams
	Code        string `json:"code,omitempty" jsonschema:"filter: case-insensitive substring match on Code"`
	Name        string `json:"name,omitempty" jsonschema:"filter: case-insensitive substring match on Name"`
	Description string `json:"description,omitempty" jsonschema:"filter: case-insensitive substring match on Description"`
}

func mcpListWorkflows(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpListWorkflowsParams) (*mcp.CallToolResult, ListResult[models.Workflow], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpListWorkflowsParams) (*mcp.CallToolResult, ListResult[models.Workflow], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.Workflow]{}, err
		}
		out, err := m.ListWorkflows(ctx, p.DraftID, mwanachamaagency.WorkflowFilter{Code: p.Code, Name: p.Name, Description: p.Description})
		return summary("Found %d workflow(s)", len(out)), ListResult[models.Workflow]{Items: out}, err
	}
}

func mcpUpdateWorkflow(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpWorkflowParams) (*mcp.CallToolResult, models.Workflow, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpWorkflowParams) (*mcp.CallToolResult, models.Workflow, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Workflow{}, err
		}
		out, err := m.UpdateWorkflow(ctx, p.toModel())
		return summary("Updated Workflow %s %q (%s)", out.Code, out.Name, out.ID), out, err
	}
}

type mcpWorkItemParams struct {
	InstanceID  string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ID          string `json:"id,omitempty" jsonschema:"the WorkItem's id — required for update, ignored for create"`
	WorkflowID  string `json:"workflow_id" jsonschema:"the owning Workflow's id"`
	ObjectiveID string `json:"objective_id" jsonschema:"the Objective this WorkItem belongs to"`
	DraftID     string `json:"draft_id,omitempty" jsonschema:"\"\" for a live WorkItem, or the owning Draft's id for a draft-scoped copy"`
	Title       string `json:"title" jsonschema:"the WorkItem's title"`
	Description string `json:"description,omitempty" jsonschema:"the WorkItem's description"`
	Ordinality  int    `json:"ordinality,omitempty" jsonschema:"sort order among sibling WorkItems"`
	Prompt      string `json:"prompt,omitempty" jsonschema:"the instructions given to whoever/whatever executes this WorkItem"`
}

func (p mcpWorkItemParams) toModel() models.WorkItem {
	return models.WorkItem{
		ID: p.ID, WorkflowID: p.WorkflowID, ObjectiveID: p.ObjectiveID, DraftID: p.DraftID,
		Title: p.Title, Description: p.Description, Ordinality: p.Ordinality, Prompt: p.Prompt,
	}
}

func mcpCreateWorkItem(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpWorkItemParams) (*mcp.CallToolResult, models.WorkItem, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpWorkItemParams) (*mcp.CallToolResult, models.WorkItem, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.WorkItem{}, err
		}
		out, err := m.CreateWorkItem(ctx, p.toModel())
		return summary("Created WorkItem %s %q (%s)", out.Code, out.Title, out.ID), out, err
	}
}

type mcpWorkItemIDParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	WorkItemID string `json:"work_item_id" jsonschema:"the WorkItem's id"`
}

func mcpGetWorkItem(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpWorkItemIDParams) (*mcp.CallToolResult, models.WorkItem, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpWorkItemIDParams) (*mcp.CallToolResult, models.WorkItem, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.WorkItem{}, err
		}
		out, err := m.GetWorkItem(ctx, p.WorkItemID)
		return summary("WorkItem %s %q (%s)", out.Code, out.Title, out.ID), out, err
	}
}

// mcpListWorkItemsParams extends instanceAndDraftParams with
// WorkItemFilter's fields.
type mcpListWorkItemsParams struct {
	instanceAndDraftParams
	Code        string `json:"code,omitempty" jsonschema:"filter: case-insensitive substring match on Code"`
	Title       string `json:"title,omitempty" jsonschema:"filter: case-insensitive substring match on Title"`
	Description string `json:"description,omitempty" jsonschema:"filter: case-insensitive substring match on Description"`
}

func mcpListWorkItems(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpListWorkItemsParams) (*mcp.CallToolResult, ListResult[models.WorkItem], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpListWorkItemsParams) (*mcp.CallToolResult, ListResult[models.WorkItem], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.WorkItem]{}, err
		}
		out, err := m.ListWorkItems(ctx, p.DraftID, mwanachamaagency.WorkItemFilter{Code: p.Code, Title: p.Title, Description: p.Description})
		return summary("Found %d work item(s)", len(out)), ListResult[models.WorkItem]{Items: out}, err
	}
}

func mcpUpdateWorkItem(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpWorkItemParams) (*mcp.CallToolResult, models.WorkItem, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpWorkItemParams) (*mcp.CallToolResult, models.WorkItem, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.WorkItem{}, err
		}
		out, err := m.UpdateWorkItem(ctx, p.toModel())
		return summary("Updated WorkItem %s %q (%s)", out.Code, out.Title, out.ID), out, err
	}
}

type mcpLookupFlowStepParams struct {
	InstanceID   string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	HandlerCode  string `json:"handler_code,omitempty" jsonschema:"highest-precedence lookup key: the WorkPlan code that executes the step"`
	TriggerTopic string `json:"trigger_topic,omitempty" jsonschema:"used with consumer as the second-precedence lookup key"`
	Consumer     string `json:"consumer,omitempty" jsonschema:"used with trigger_topic as the second-precedence lookup key"`
	Code         string `json:"code,omitempty" jsonschema:"lowest-precedence lookup key: the step's own \"<workflow_code>:<step>\" code"`
}

func mcpLookupFlowStep(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpLookupFlowStepParams) (*mcp.CallToolResult, models.EventFlowStep, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpLookupFlowStepParams) (*mcp.CallToolResult, models.EventFlowStep, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.EventFlowStep{}, err
		}
		out, err := m.LookupFlowStep(ctx, models.EventFlowStepLookup{
			HandlerCode:  p.HandlerCode,
			TriggerTopic: p.TriggerTopic,
			Consumer:     p.Consumer,
			Code:         p.Code,
		})
		return summary("EventFlowStep %s: %s → %s", out.Code, out.TriggerTopic, out.HandlerCode), out, err
	}
}

type mcpListEventFlowStepsParams struct {
	InstanceID   string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	WorkflowCode string `json:"workflow_code" jsonschema:"the owning Workflow's code"`
	Code         string `json:"code,omitempty" jsonschema:"filter: case-insensitive substring match on Code"`
	Name         string `json:"name,omitempty" jsonschema:"filter: case-insensitive substring match on Name"`
	Description  string `json:"description,omitempty" jsonschema:"filter: case-insensitive substring match on Description"`
}

func mcpListEventFlowSteps(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpListEventFlowStepsParams) (*mcp.CallToolResult, ListResult[models.EventFlowStep], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpListEventFlowStepsParams) (*mcp.CallToolResult, ListResult[models.EventFlowStep], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.EventFlowStep]{}, err
		}
		out, err := m.ListEventFlowSteps(ctx, p.WorkflowCode, mwanachamaagency.EventFlowStepFilter{Code: p.Code, Name: p.Name, Description: p.Description})
		return summary("Found %d event flow step(s) for workflow %q", len(out), p.WorkflowCode), ListResult[models.EventFlowStep]{Items: out}, err
	}
}
