package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

// registerOrgChartTools registers Department/Level tools — the org chart:
// a parent/child tree of named departments, each pinned to a named Level.
func registerOrgChartTools(server *mcp.Server, resolve ManagerResolver) {
	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_department", Description: "Create a Department — pass parent_id \"\" for a top-level one."}, mcpCreateDepartment(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_department", Description: "Get a Department by id."}, mcpGetDepartment(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_departments", Description: "List every Department in the org chart."}, mcpListDepartments(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_department_children", Description: "List a Department's direct children."}, mcpListDepartmentChildren(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_rename_department", Description: "Rename a Department."}, mcpRenameDepartment(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_move_department", Description: "Re-parent a Department under a different (or no) parent."}, mcpMoveDepartment(resolve))

	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_level", Description: "Create a Level — a named rung (e.g. \"Division\", \"Team\") Departments are pinned to."}, mcpCreateLevel(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_level", Description: "Get a Level by id."}, mcpGetLevel(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_levels", Description: "List every Level."}, mcpListLevels(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_rename_level", Description: "Rename a Level."}, mcpRenameLevel(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_delete_level", Description: "Delete a Level. Refused while any Department still wears it."}, mcpDeleteLevel(resolve))
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
		return nil, out, err
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
		return nil, out, err
	}
}

func mcpListDepartments(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, instanceOnlyParams) (*mcp.CallToolResult, ListResult[models.Department], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p instanceOnlyParams) (*mcp.CallToolResult, ListResult[models.Department], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.Department]{}, err
		}
		out, err := m.ListDepartments(ctx)
		return nil, ListResult[models.Department]{Items: out}, err
	}
}

type mcpDepartmentParentParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ParentID   string `json:"parent_id" jsonschema:"the parent Department's id; \"\" lists top-level departments"`
}

func mcpListDepartmentChildren(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpDepartmentParentParams) (*mcp.CallToolResult, ListResult[models.Department], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpDepartmentParentParams) (*mcp.CallToolResult, ListResult[models.Department], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.Department]{}, err
		}
		out, err := m.ListDepartmentChildren(ctx, p.ParentID)
		return nil, ListResult[models.Department]{Items: out}, err
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
		return nil, out, err
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
		return nil, out, err
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
		return nil, out, err
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
		return nil, out, err
	}
}

func mcpListLevels(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, instanceOnlyParams) (*mcp.CallToolResult, ListResult[models.Level], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p instanceOnlyParams) (*mcp.CallToolResult, ListResult[models.Level], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.Level]{}, err
		}
		out, err := m.ListLevels(ctx)
		return nil, ListResult[models.Level]{Items: out}, err
	}
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
		return nil, out, err
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
		return nil, DeleteResult{Deleted: true}, nil
	}
}
