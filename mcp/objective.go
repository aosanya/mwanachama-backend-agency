package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// registerObjectiveTools registers Objective tools — a measurable
// operationalization of a Goal (a Goal may have several, each a granular
// unit one Role is RACI-assigned to), and the unit WorkItem/RACI attach
// to. Mirrors goal.go's Goal tools exactly.
func registerObjectiveTools(server *mcp.Server, resolve ManagerResolver) {
	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_objective", Title: "Create Objective", Description: "Create an Objective for a Goal. A Goal may have several Objectives, each a granular unit one Role can be RACI-assigned to."}, mcpCreateObjective(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_objective", Title: "Get Objective", Description: "Get an Objective by id."}, mcpGetObjective(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_objectives", Title: "List Objectives", Description: "List Objectives — live, or a Draft's scoped copies."}, mcpListObjectives(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_update_objective", Title: "Update Objective", Description: "Update an Objective (full replace — supply every field you want kept, not just the ones changing). id is required."}, mcpUpdateObjective(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_delete_objective", Title: "Delete Objective", Description: "Delete an Objective. Cascades: also deletes every RACI assignment on it — warn the operator about that before calling this."}, mcpDeleteObjective(resolve))
}

type mcpObjectiveParams struct {
	InstanceID  string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ID          string `json:"id,omitempty" jsonschema:"the Objective's id — required for update, ignored for create"`
	GoalID      string `json:"goal_id" jsonschema:"the Goal this Objective operationalizes"`
	DraftID     string `json:"draft_id" jsonschema:"the Draft this Objective belongs to"`
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
		return summary("Created Objective %s %q (%s) for Goal %s", out.Code, out.Title, out.ID, out.GoalID), out, err
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
		return summary("Objective %s %q (%s)", out.Code, out.Title, out.ID), out, err
	}
}

// mcpListObjectivesParams extends instanceAndDraftParams with
// ObjectiveFilter's fields.
type mcpListObjectivesParams struct {
	instanceAndDraftParams
	Code        string `json:"code,omitempty" jsonschema:"filter: case-insensitive substring match on Code"`
	Title       string `json:"title,omitempty" jsonschema:"filter: case-insensitive substring match on Title"`
	Description string `json:"description,omitempty" jsonschema:"filter: case-insensitive substring match on Description"`
}

func mcpListObjectives(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpListObjectivesParams) (*mcp.CallToolResult, ListResult[models.Objective], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpListObjectivesParams) (*mcp.CallToolResult, ListResult[models.Objective], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.Objective]{}, err
		}
		out, err := m.ListObjectives(ctx, p.DraftID, mwanachamaagency.ObjectiveFilter{Code: p.Code, Title: p.Title, Description: p.Description})
		return summaryWithContext(fmt.Sprintf("Found %d objective(s)", len(out)), objectivesContextText(out)), ListResult[models.Objective]{Items: out}, err
	}
}

// objectivesContextText is what an LLM sees for an Objectives list opened
// outside its own tool loop (mwanachama-wakala-studio's sidebar shortcut) —
// one line per Objective, field-by-field rather than a struct dump, so
// dropping a field from an LLM's context later is just deleting it from
// this Sprintf. Mirrors goal.go's goalsContextText, plus the owning
// Goal's id — an Objective's own Edit lets an operator reassign it to a
// different Goal, so the model needs that id in context too, not just the
// Objective's own.
func objectivesContextText(objectives []models.Objective) string {
	if len(objectives) == 0 {
		return "There are no objectives yet."
	}
	lines := make([]string, 0, len(objectives))
	for _, o := range objectives {
		line := fmt.Sprintf("- %s: %s", o.Code, o.Title)
		if o.Description != "" {
			line += " — " + o.Description
		}
		lines = append(lines, line+fmt.Sprintf(" (objective id: %s, goal id: %s)", o.ID, o.GoalID))
	}
	return "Current objectives:\n" + strings.Join(lines, "\n")
}

func mcpUpdateObjective(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpObjectiveParams) (*mcp.CallToolResult, models.Objective, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpObjectiveParams) (*mcp.CallToolResult, models.Objective, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Objective{}, err
		}
		out, err := m.UpdateObjective(ctx, p.toModel())
		return summary("Updated Objective %s %q (%s)", out.Code, out.Title, out.ID), out, err
	}
}

func mcpDeleteObjective(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpObjectiveIDParams) (*mcp.CallToolResult, DeleteResult, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpObjectiveIDParams) (*mcp.CallToolResult, DeleteResult, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, DeleteResult{}, err
		}
		if err := m.DeleteObjective(ctx, p.ObjectiveID); err != nil {
			return nil, DeleteResult{}, err
		}
		return summary("Deleted Objective %s", p.ObjectiveID), DeleteResult{Deleted: true}, nil
	}
}
