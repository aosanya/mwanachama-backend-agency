package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// registerOrgChartTools registers Department/Level tools — the org chart:
// a parent/child tree of named departments, each pinned to a named Level.
func registerOrgChartTools(server *mcp.Server, resolve ManagerResolver) {
	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_department", Title: "Create Department", Description: "Create a Department — pass parent_id \"\" for a top-level one."}, mcpCreateDepartment(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_department", Title: "Get Department", Description: "Get a Department by id."}, mcpGetDepartment(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_departments", Title: "List Departments", Description: "List every Department in the org chart."}, mcpListDepartments(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_department_children", Title: "List Department Children", Description: "List a Department's direct children."}, mcpListDepartmentChildren(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_rename_department", Title: "Rename Department", Description: "Rename a Department."}, mcpRenameDepartment(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_move_department", Title: "Move Department", Description: "Re-parent a Department under a different (or no) parent."}, mcpMoveDepartment(resolve))

	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_level", Title: "Create Level", Description: "Create a Level — a named rung (e.g. \"Division\", \"Team\") Departments are pinned to."}, mcpCreateLevel(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_level", Title: "Get Level", Description: "Get a Level by id."}, mcpGetLevel(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_levels", Title: "List Levels", Description: "List every Level."}, mcpListLevels(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_rename_level", Title: "Rename Level", Description: "Rename a Level."}, mcpRenameLevel(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_delete_level", Title: "Delete Level", Description: "Delete a Level. Refused while any Department still wears it."}, mcpDeleteLevel(resolve))
}

// mcpCreateDepartmentParams mirrors models.Department, but with ParentID
// marked omitempty — the go-sdk's schema inference (google/jsonschema-go)
// treats any field lacking omitempty/omitzero as required, and
// models.Department's own json tags (decision #1: imported unmodified)
// don't mark it so, which would force a caller to always name a parent
// even for a top-level Department.
type mcpCreateDepartmentParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ParentID   string `json:"parent_id,omitempty" jsonschema:"the parent Department's id; \"\" makes it top-level"`
	LevelID    string `json:"level_id" jsonschema:"the Level this Department is pinned to"`
	Name       string `json:"name" jsonschema:"the Department's name"`
}

func mcpCreateDepartment(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpCreateDepartmentParams) (*mcp.CallToolResult, models.Department, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpCreateDepartmentParams) (*mcp.CallToolResult, models.Department, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Department{}, err
		}
		out, err := m.CreateDepartment(ctx, models.Department{ParentID: p.ParentID, LevelID: p.LevelID, Name: p.Name})
		return summary("Created Department %s %q (%s)", out.Code, out.Name, out.ID), out, err
	}
}

type mcpDepartmentIDParams struct {
	InstanceID   string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	DepartmentID string `json:"department_id" jsonschema:"the Department's id"`
}

func mcpGetDepartment(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpDepartmentIDParams) (*mcp.CallToolResult, models.Department, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpDepartmentIDParams) (*mcp.CallToolResult, models.Department, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Department{}, err
		}
		out, err := m.GetDepartment(ctx, p.DepartmentID)
		return summary("Department %s %q (%s)", out.Code, out.Name, out.ID), out, err
	}
}

// mcpListDepartmentsParams extends instanceOnlyParams with
// DepartmentFilter's fields.
type mcpListDepartmentsParams struct {
	instanceOnlyParams
	Code string `json:"code,omitempty" jsonschema:"filter: case-insensitive substring match on Code"`
	Name string `json:"name,omitempty" jsonschema:"filter: case-insensitive substring match on Name"`
}

func mcpListDepartments(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpListDepartmentsParams) (*mcp.CallToolResult, ListResult[models.Department], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpListDepartmentsParams) (*mcp.CallToolResult, ListResult[models.Department], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.Department]{}, err
		}
		out, err := m.ListDepartments(ctx, mwanachamaagency.DepartmentFilter{Code: p.Code, Name: p.Name})
		return summaryWithContext(fmt.Sprintf("Found %d department(s)", len(out)), departmentsContextText(out)), ListResult[models.Department]{Items: out}, err
	}
}

// departmentsContextText is what an LLM sees for a Departments list — one
// line per Department, field-by-field rather than a struct dump, so
// dropping a field from an LLM's context later is just deleting it from
// this Sprintf. Shared by both agency_list_departments and
// agency_list_department_children — same row shape either way.
func departmentsContextText(departments []models.Department) string {
	if len(departments) == 0 {
		return "There are no departments yet."
	}
	lines := make([]string, 0, len(departments))
	for _, d := range departments {
		lines = append(lines, fmt.Sprintf("- %s: %s (department id: %s, level id: %s, parent id: %s)", d.Code, d.Name, d.ID, d.LevelID, d.ParentID))
	}
	return "Current departments:\n" + strings.Join(lines, "\n")
}

type mcpDepartmentParentParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ParentID   string `json:"parent_id" jsonschema:"the parent Department's id; \"\" lists top-level departments"`
	Code       string `json:"code,omitempty" jsonschema:"filter: case-insensitive substring match on Code"`
	Name       string `json:"name,omitempty" jsonschema:"filter: case-insensitive substring match on Name"`
}

func mcpListDepartmentChildren(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpDepartmentParentParams) (*mcp.CallToolResult, ListResult[models.Department], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpDepartmentParentParams) (*mcp.CallToolResult, ListResult[models.Department], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.Department]{}, err
		}
		out, err := m.ListDepartmentChildren(ctx, p.ParentID, mwanachamaagency.DepartmentFilter{Code: p.Code, Name: p.Name})
		return summaryWithContext(fmt.Sprintf("Found %d child department(s)", len(out)), departmentsContextText(out)), ListResult[models.Department]{Items: out}, err
	}
}

type mcpRenameDepartmentParams struct {
	InstanceID   string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	DepartmentID string `json:"department_id" jsonschema:"the Department's id"`
	Name         string `json:"name" jsonschema:"the Department's new name; must not be empty"`
}

func mcpRenameDepartment(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpRenameDepartmentParams) (*mcp.CallToolResult, models.Department, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpRenameDepartmentParams) (*mcp.CallToolResult, models.Department, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Department{}, err
		}
		out, err := m.RenameDepartment(ctx, p.DepartmentID, p.Name)
		return summary("Renamed Department %s to %q", out.Code, out.Name), out, err
	}
}

type mcpMoveDepartmentParams struct {
	InstanceID   string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	DepartmentID string `json:"department_id" jsonschema:"the Department's id"`
	NewParentID  string `json:"new_parent_id" jsonschema:"the new parent Department's id; \"\" makes it top-level"`
}

func mcpMoveDepartment(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpMoveDepartmentParams) (*mcp.CallToolResult, models.Department, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpMoveDepartmentParams) (*mcp.CallToolResult, models.Department, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Department{}, err
		}
		out, err := m.MoveDepartment(ctx, p.DepartmentID, p.NewParentID)
		return summary("Moved Department %s %q under parent %q", out.Code, out.Name, out.ParentID), out, err
	}
}

// mcpCreateLevelParams mirrors models.Level, but with Depth marked
// omitempty (see mcpCreateDepartmentParams's doc comment for why).
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
