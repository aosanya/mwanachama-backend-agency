package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

// registerObjectiveTools registers Objective tools — a measurable
// operationalization of a Goal (a Goal may have several, each a granular
// unit one Role is RACI-assigned to), and the unit WorkItem/RACI attach
// to. Mirrors goal.go's Goal tools exactly.
func registerObjectiveTools(server *mcp.Server, resolve ManagerResolver) {
	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_objective", Description: "Create an Objective for a Goal. A Goal may have several Objectives, each a granular unit one Role can be RACI-assigned to."}, mcpCreateObjective(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_objective", Description: "Get an Objective by id."}, mcpGetObjective(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_objectives", Description: "List Objectives — live, or a Draft's scoped copies."}, mcpListObjectives(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_update_objective", Description: "Update an Objective (full replace — supply every field you want kept, not just the ones changing). id is required."}, mcpUpdateObjective(resolve))
}

type mcpObjectiveParams struct {
	InstanceID  string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ID          string `json:"id,omitempty" jsonschema:"the Objective's id — required for update, ignored for create"`
	GoalID      string `json:"goal_id" jsonschema:"the Goal this Objective operationalizes"`
	DraftID     string `json:"draft_id,omitempty" jsonschema:"\"\" for a live Objective, or the owning Draft's id for a draft-scoped copy"`
	Title       string `json:"title" jsonschema:"the Objective's title"`
	Description string `json:"description,omitempty" jsonschema:"the Objective's description"`
	Ordinality  int    `json:"ordinality,omitempty" jsonschema:"sort order among sibling Objectives"`
}

func (p mcpObjectiveParams) toModel() models.Objective {
	return models.Objective{ID: p.ID, GoalID: p.GoalID, DraftID: p.DraftID, Title: p.Title, Description: p.Description, Ordinality: p.Ordinality}
}

func mcpCreateObjective(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpObjectiveParams) (*mcp.CallToolResult, models.Objective, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpObjectiveParams) (*mcp.CallToolResult, models.Objective, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Objective{}, err
		}
		out, err := m.CreateObjective(ctx, p.toModel())
		return summary("Created Objective %q (%s) for Goal %s", out.Title, out.ID, out.GoalID), out, err
	}
}

type mcpObjectiveIDParams struct {
	InstanceID  string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ObjectiveID string `json:"objective_id" jsonschema:"the Objective's id"`
}

func mcpGetObjective(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpObjectiveIDParams) (*mcp.CallToolResult, models.Objective, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpObjectiveIDParams) (*mcp.CallToolResult, models.Objective, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Objective{}, err
		}
		out, err := m.GetObjective(ctx, p.ObjectiveID)
		return summary("Objective %q (%s)", out.Title, out.ID), out, err
	}
}

func mcpListObjectives(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, instanceAndDraftParams) (*mcp.CallToolResult, ListResult[models.Objective], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p instanceAndDraftParams) (*mcp.CallToolResult, ListResult[models.Objective], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.Objective]{}, err
		}
		out, err := m.ListObjectives(ctx, p.DraftID)
		return summary("Found %d objective(s)", len(out)), ListResult[models.Objective]{Items: out}, err
	}
}

func mcpUpdateObjective(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpObjectiveParams) (*mcp.CallToolResult, models.Objective, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpObjectiveParams) (*mcp.CallToolResult, models.Objective, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Objective{}, err
		}
		out, err := m.UpdateObjective(ctx, p.toModel())
		return summary("Updated Objective %q (%s)", out.Title, out.ID), out, err
	}
}
