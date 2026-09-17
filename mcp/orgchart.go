package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// registerOrgChartTools registers Level tools — the named rungs an Agency's
// own design vocabulary uses. The kinds of node an org is built from
// ("Department", "Ward Branch") and the nodes themselves both moved to
// mwanachama-backend-actor on 2026-09-17 (AGD-017) — this repo has no
// dependency on that package, so it exposes no tool for either.
func registerOrgChartTools(server *mcp.Server, resolve ManagerResolver) {
	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_level", Title: "Create Level", Description: "Create a Level — a named rung (e.g. \"Division\", \"Team\", \"Ward\")."}, mcpCreateLevel(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_level", Title: "Get Level", Description: "Get a Level by id."}, mcpGetLevel(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_levels", Title: "List Levels", Description: "List every Level."}, mcpListLevels(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_rename_level", Title: "Rename Level", Description: "Rename a Level."}, mcpRenameLevel(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_delete_level", Title: "Delete Level", Description: "Delete a Level."}, mcpDeleteLevel(resolve))
}

// mcpCreateLevelParams mirrors models.Level, but with Depth marked
// omitempty — a zero Depth is the common case (the first rung), and a
// required field the caller must spell out as 0 reads as a mistake.
type mcpCreateLevelParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	Name       string `json:"name" jsonschema:"the Level's name, e.g. \"Division\", \"Team\""`
	Depth      int    `json:"depth,omitempty" jsonschema:"a pure sort key — levels may share a depth or leave gaps"`
}

func mcpCreateLevel(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpCreateLevelParams) (*mcp.CallToolResult, models.Level, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpCreateLevelParams) (*mcp.CallToolResult, models.Level, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Level{}, err
		}
		out, err := m.CreateLevel(ctx, models.Level{Name: p.Name, Depth: p.Depth})
		return summary("Created Level %s %q (%s)", out.Code, out.Name, out.ID), out, err
	}
}

type mcpLevelIDParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	LevelID    string `json:"level_id" jsonschema:"the Level's id"`
}

func mcpGetLevel(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpLevelIDParams) (*mcp.CallToolResult, models.Level, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpLevelIDParams) (*mcp.CallToolResult, models.Level, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Level{}, err
		}
		out, err := m.GetLevel(ctx, p.LevelID)
		return summary("Level %s %q (%s)", out.Code, out.Name, out.ID), out, err
	}
}

// mcpListLevelsParams extends instanceOnlyParams with LevelFilter's fields.
type mcpListLevelsParams struct {
	instanceOnlyParams
	Code string `json:"code,omitempty" jsonschema:"filter: case-insensitive substring match on Code"`
	Name string `json:"name,omitempty" jsonschema:"filter: case-insensitive substring match on Name"`
}

func mcpListLevels(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpListLevelsParams) (*mcp.CallToolResult, ListResult[models.Level], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpListLevelsParams) (*mcp.CallToolResult, ListResult[models.Level], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.Level]{}, err
		}
		out, err := m.ListLevels(ctx, mwanachamaagency.LevelFilter{Code: p.Code, Name: p.Name})
		return summaryWithContext(fmt.Sprintf("Found %d level(s)", len(out)), levelsContextText(out)), ListResult[models.Level]{Items: out}, err
	}
}

// levelsContextText is what an LLM sees for a Levels list — one line per
// Level, field-by-field rather than a struct dump, so dropping a field
// from an LLM's context later is just deleting it from this Sprintf.
func levelsContextText(levels []models.Level) string {
	if len(levels) == 0 {
		return "There are no levels yet."
	}
	lines := make([]string, 0, len(levels))
	for _, l := range levels {
		lines = append(lines, fmt.Sprintf("- %s: %s, depth %d (level id: %s)", l.Code, l.Name, l.Depth, l.ID))
	}
	return "Current levels:\n" + strings.Join(lines, "\n")
}

type mcpRenameLevelParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	LevelID    string `json:"level_id" jsonschema:"the Level's id"`
	Name       string `json:"name" jsonschema:"the Level's new name; must not be empty"`
}

func mcpRenameLevel(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpRenameLevelParams) (*mcp.CallToolResult, models.Level, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpRenameLevelParams) (*mcp.CallToolResult, models.Level, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Level{}, err
		}
		out, err := m.RenameLevel(ctx, p.LevelID, p.Name)
		return summary("Renamed Level %s to %q", out.Code, out.Name), out, err
	}
}

func mcpDeleteLevel(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpLevelIDParams) (*mcp.CallToolResult, DeleteResult, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpLevelIDParams) (*mcp.CallToolResult, DeleteResult, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, DeleteResult{}, err
		}
		if err := m.DeleteLevel(ctx, p.LevelID); err != nil {
			return nil, DeleteResult{}, err
		}
		return summary("Deleted Level %s", p.LevelID), DeleteResult{Deleted: true}, nil
	}
}
