package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

type mcpListModuleObjectivesParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ModuleID   string `json:"module_id" jsonschema:"the Module's id"`
	DraftID    string `json:"draft_id" jsonschema:"the Draft to scope to"`
}

func registerModuleObjectivesTool(server *mcp.Server, resolve ManagerResolver) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agency_list_module_objectives",
		Title:       "List Module Objectives",
		Description: "The inverse of agency_list_objective_modules: the Objectives a Module backs, each with its Goal and the rationale on that link.",
	}, mcpListModuleObjectives(resolve))
}

type ModuleObjective struct {
	LinkID         string `json:"link_id"`
	ObjectiveID    string `json:"objective_id"`
	ObjectiveTitle string `json:"objective_title"`
	GoalID         string `json:"goal_id"`
	GoalTitle      string `json:"goal_title"`
	Rationale      string `json:"rationale"`
}

func mcpListModuleObjectives(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpListModuleObjectivesParams) (*mcp.CallToolResult, ListResult[ModuleObjective], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpListModuleObjectivesParams) (*mcp.CallToolResult, ListResult[ModuleObjective], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[ModuleObjective]{}, err
		}
		links, err := m.ListObjectiveModules(ctx, p.DraftID, mwanachamaagency.ObjectiveModuleFilter{ModuleID: p.ModuleID})
		if err != nil {
			return nil, ListResult[ModuleObjective]{}, err
		}
		out := make([]ModuleObjective, 0, len(links))
		lines := make([]string, 0, len(links))
		for _, l := range links {
			o, err := m.GetObjective(ctx, l.ObjectiveID)
			if err != nil {
				continue // a deleted Objective leaves its link dangling; skip it
			}
			g, _ := m.GetGoal(ctx, o.GoalID)
			out = append(out, ModuleObjective{LinkID: l.ID, ObjectiveID: o.ID, ObjectiveTitle: o.Title, GoalID: g.ID, GoalTitle: g.Title, Rationale: l.Rationale})
			lines = append(lines, fmt.Sprintf("- %s (goal: %s): %s", o.Title, g.Title, l.Rationale))
		}
		text := "This module backs no objectives."
		if len(lines) > 0 {
			text = "Objectives this module backs:\n" + strings.Join(lines, "\n")
		}
		return summaryWithContext(fmt.Sprintf("Found %d objective(s) backed by module %s", len(out), p.ModuleID), text), ListResult[ModuleObjective]{Items: out}, nil
	}
}
