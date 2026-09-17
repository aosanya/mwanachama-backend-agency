package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// registerOrgChartTools registers GroupType/Level tools — the org's
// vocabulary: the named rungs it uses, and the kinds of node that sit at
// them. The nodes themselves are operational and live in
// mwanachama-backend-actor's Group tables, not here (AGD-017).
func registerOrgChartTools(server *mcp.Server, resolve ManagerResolver) {
	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_group_type", Title: "Create Group Type", Description: "Create a GroupType — a kind of node the org is built from, e.g. \"Department\", \"Committee\", \"Ward\". This declares that the org HAS wards; it does not create a ward. Pass level_id to pin the type to one rung, or \"\" if it may sit at any."}, mcpCreateGroupType(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_group_type", Title: "Get Group Type", Description: "Get a GroupType by id."}, mcpGetGroupType(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_group_types", Title: "List Group Types", Description: "List every GroupType the org uses."}, mcpListGroupTypes(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_update_group_type", Title: "Update Group Type", Description: "Update a GroupType's name, singular/plural labels, or the Level it is pinned to."}, mcpUpdateGroupType(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_delete_group_type", Title: "Delete Group Type", Description: "Delete a GroupType."}, mcpDeleteGroupType(resolve))

	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_level", Title: "Create Level", Description: "Create a Level — a named rung (e.g. \"Division\", \"Team\", \"Ward\") GroupTypes are pinned to."}, mcpCreateLevel(resolve))
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

// mcpCreateGroupTypeParams mirrors models.GroupType. Singular/Plural are
// omitempty — CreateGroupType falls back to Name for both, so a caller with
// only one word for the type need not spell them out.
type mcpCreateGroupTypeParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	Name       string `json:"name" jsonschema:"the GroupType's name, e.g. \"Department\", \"Committee\", \"Ward\""`
	Singular   string `json:"singular,omitempty" jsonschema:"what one of these is called; defaults to name"`
	Plural     string `json:"plural,omitempty" jsonschema:"what several are called; defaults to name + \"s\""`
	LevelID    string `json:"level_id,omitempty" jsonschema:"the Level this type sits at; \"\" means any rung"`
}

func mcpCreateGroupType(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpCreateGroupTypeParams) (*mcp.CallToolResult, models.GroupType, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpCreateGroupTypeParams) (*mcp.CallToolResult, models.GroupType, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.GroupType{}, err
		}
		out, err := m.CreateGroupType(ctx, models.GroupType{
			Name: p.Name, Singular: p.Singular, Plural: p.Plural, LevelID: p.LevelID,
		})
		return summary("Created Group Type %s %q (%s)", out.Code, out.Name, out.ID), out, err
	}
}

type mcpGroupTypeIDParams struct {
	InstanceID  string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	GroupTypeID string `json:"group_type_id" jsonschema:"the GroupType's id"`
}

func mcpGetGroupType(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpGroupTypeIDParams) (*mcp.CallToolResult, models.GroupType, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpGroupTypeIDParams) (*mcp.CallToolResult, models.GroupType, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.GroupType{}, err
		}
		out, err := m.GetGroupType(ctx, p.GroupTypeID)
		return summary("Group Type %s %q (%s)", out.Code, out.Name, out.ID), out, err
	}
}

// mcpListGroupTypesParams extends instanceOnlyParams with GroupTypeFilter's
// fields.
type mcpListGroupTypesParams struct {
	instanceOnlyParams
	Code    string `json:"code,omitempty" jsonschema:"filter: case-insensitive substring match on Code"`
	Name    string `json:"name,omitempty" jsonschema:"filter: case-insensitive substring match on Name"`
	LevelID string `json:"level_id,omitempty" jsonschema:"filter: exact match on the Level the type sits at"`
}

func mcpListGroupTypes(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpListGroupTypesParams) (*mcp.CallToolResult, ListResult[models.GroupType], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpListGroupTypesParams) (*mcp.CallToolResult, ListResult[models.GroupType], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.GroupType]{}, err
		}
		out, err := m.ListGroupTypes(ctx, mwanachamaagency.GroupTypeFilter{Code: p.Code, Name: p.Name, LevelID: p.LevelID})
		return summaryWithContext(fmt.Sprintf("Found %d group type(s)", len(out)), groupTypesContextText(out)), ListResult[models.GroupType]{Items: out}, err
	}
}

// groupTypesContextText is what an LLM sees for a GroupTypes list — same
// one-line-per-row, field-by-field shape as levelsContextText.
func groupTypesContextText(types []models.GroupType) string {
	if len(types) == 0 {
		return "There are no group types yet."
	}
	lines := make([]string, 0, len(types))
	for _, t := range types {
		level := t.LevelID
		if level == "" {
			level = "any rung"
		}
		lines = append(lines, fmt.Sprintf("- %s: %s (one: %s, many: %s, level: %s, group type id: %s)", t.Code, t.Name, t.Singular, t.Plural, level, t.ID))
	}
	return "Current group types:\n" + strings.Join(lines, "\n")
}

type mcpUpdateGroupTypeParams struct {
	InstanceID  string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	GroupTypeID string `json:"group_type_id" jsonschema:"the GroupType's id"`
	Name        string `json:"name" jsonschema:"the GroupType's name; must not be empty"`
	Singular    string `json:"singular,omitempty" jsonschema:"what one of these is called; defaults to name"`
	Plural      string `json:"plural,omitempty" jsonschema:"what several are called; defaults to name + \"s\""`
	LevelID     string `json:"level_id,omitempty" jsonschema:"the Level this type sits at; \"\" means any rung"`
}

func mcpUpdateGroupType(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpUpdateGroupTypeParams) (*mcp.CallToolResult, models.GroupType, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpUpdateGroupTypeParams) (*mcp.CallToolResult, models.GroupType, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.GroupType{}, err
		}
		out, err := m.UpdateGroupType(ctx, models.GroupType{
			ID: p.GroupTypeID, Name: p.Name, Singular: p.Singular, Plural: p.Plural, LevelID: p.LevelID,
		})
		return summary("Updated Group Type %s %q (%s)", out.Code, out.Name, out.ID), out, err
	}
}

func mcpDeleteGroupType(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpGroupTypeIDParams) (*mcp.CallToolResult, struct{}, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpGroupTypeIDParams) (*mcp.CallToolResult, struct{}, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, struct{}{}, err
		}
		if err := m.DeleteGroupType(ctx, p.GroupTypeID); err != nil {
			return nil, struct{}{}, err
		}
		return summary("Deleted Group Type %s", p.GroupTypeID), struct{}{}, nil
	}
}
