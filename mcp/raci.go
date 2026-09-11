package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

// registerRACITools registers Role/RoleAssignment tools — the RACI matrix:
// personas (Role) assigned to Objectives with a
// Responsible/Accountable/Consulted/Informed label. A live Role or
// RoleAssignment cannot be edited or deleted directly — fork a Draft, edit
// the draft-scoped copy, promote (agency_create_draft/agency_promote_draft
// in agency.go).
func registerRACITools(server *mcp.Server, resolve ManagerResolver) {
	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_role", Description: "Create a Role (a persona, e.g. \"bug-hunter\")."}, mcpCreateRole(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_role", Description: "Get a Role by id."}, mcpGetRole(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_roles", Description: "List Roles — live, or a Draft's scoped copies."}, mcpListRoles(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_update_role", Description: "Update a Role. Refused on a live Role — only a Draft-scoped copy may be edited."}, mcpUpdateRole(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_delete_role", Description: "Delete a Role. Refused on a live Role — only a Draft-scoped copy may be deleted."}, mcpDeleteRole(resolve))

	mcp.AddTool(server, &mcp.Tool{Name: "agency_set_raci", Description: "Assign a Role to an Objective with a RACI label (responsible/accountable/consulted/informed). At most one accountable assignment may exist per Objective."}, mcpCreateRoleAssignment(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_role_assignment", Description: "Get a RoleAssignment by id."}, mcpGetRoleAssignment(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_role_assignments", Description: "List RoleAssignments — live, or a Draft's scoped copies."}, mcpListRoleAssignments(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_role_assignments_for_objective", Description: "List who is assigned to an Objective, and with what RACI label."}, mcpListRoleAssignmentsForObjective(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_update_role_assignment", Description: "Update a RoleAssignment. Refused on a live one — only a Draft-scoped copy may be edited."}, mcpUpdateRoleAssignment(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_delete_role_assignment", Description: "Delete a RoleAssignment. Refused on a live one — only a Draft-scoped copy may be deleted."}, mcpDeleteRoleAssignment(resolve))
}

// mcpRoleParams mirrors models.Role, but with every field but Name marked
// omitempty — the go-sdk's schema inference (google/jsonschema-go) treats
// any field lacking omitempty/omitzero as required, and models.Role's own
// json tags (decision #1: imported unmodified) mostly don't mark it so.
// ID is required for update, ignored for create.
type mcpRoleParams struct {
	InstanceID  string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ID          string `json:"id,omitempty" jsonschema:"the Role's id — required for update, ignored for create"`
	DraftID     string `json:"draft_id,omitempty" jsonschema:"\"\" for a live Role, or the owning Draft's id for a draft-scoped copy"`
	Name        string `json:"name" jsonschema:"the Role's name, e.g. \"bug-hunter\""`
	Description string `json:"description,omitempty" jsonschema:"what this Role is responsible for"`
}

func (p mcpRoleParams) toModel() models.Role {
	return models.Role{ID: p.ID, DraftID: p.DraftID, Name: p.Name, Description: p.Description}
}

func mcpCreateRole(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpRoleParams) (*mcp.CallToolResult, models.Role, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpRoleParams) (*mcp.CallToolResult, models.Role, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Role{}, err
		}
		out, err := m.CreateRole(ctx, p.toModel())
		return summary("Created Role %q (%s)", out.Name, out.ID), out, err
	}
}

type mcpRoleIDParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	RoleID     string `json:"role_id" jsonschema:"the Role's id"`
}

func mcpGetRole(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpRoleIDParams) (*mcp.CallToolResult, models.Role, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpRoleIDParams) (*mcp.CallToolResult, models.Role, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Role{}, err
		}
		out, err := m.GetRole(ctx, p.RoleID)
		return summary("Role %q (%s)", out.Name, out.ID), out, err
	}
}

func mcpListRoles(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, instanceAndDraftParams) (*mcp.CallToolResult, ListResult[models.Role], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p instanceAndDraftParams) (*mcp.CallToolResult, ListResult[models.Role], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.Role]{}, err
		}
		out, err := m.ListRoles(ctx, p.DraftID)
		return summary("Found %d role(s)", len(out)), ListResult[models.Role]{Items: out}, err
	}
}

func mcpUpdateRole(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpRoleParams) (*mcp.CallToolResult, models.Role, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpRoleParams) (*mcp.CallToolResult, models.Role, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Role{}, err
		}
		out, err := m.UpdateRole(ctx, p.toModel())
		return summary("Updated Role %q (%s)", out.Name, out.ID), out, err
	}
}

func mcpDeleteRole(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpRoleIDParams) (*mcp.CallToolResult, DeleteResult, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpRoleIDParams) (*mcp.CallToolResult, DeleteResult, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, DeleteResult{}, err
		}
		if err := m.DeleteRole(ctx, p.RoleID); err != nil {
			return nil, DeleteResult{}, err
		}
		return summary("Deleted Role %s", p.RoleID), DeleteResult{Deleted: true}, nil
	}
}

// mcpRoleAssignmentParams mirrors models.RoleAssignment, but with ID and
// DraftID marked omitempty — the go-sdk's schema inference
// (google/jsonschema-go) treats any field lacking omitempty/omitzero as
// required, and models.RoleAssignment's own json tags (decision #1:
// imported unmodified) don't mark them so. ID is required for update,
// ignored for create.
type mcpRoleAssignmentParams struct {
	InstanceID  string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ID          string `json:"id,omitempty" jsonschema:"the RoleAssignment's id — required for update, ignored for create"`
	DraftID     string `json:"draft_id,omitempty" jsonschema:"\"\" for a live RoleAssignment, or the owning Draft's id for a draft-scoped copy"`
	RoleID      string `json:"role_id" jsonschema:"the assigned Role's id"`
	ObjectiveID string `json:"objective_id" jsonschema:"the Objective this Role is assigned to"`
	Label       string `json:"label" jsonschema:"the RACI label: responsible, accountable, consulted, or informed — at most one accountable assignment may exist per Objective"`
}

func (p mcpRoleAssignmentParams) toModel() models.RoleAssignment {
	return models.RoleAssignment{ID: p.ID, DraftID: p.DraftID, RoleID: p.RoleID, ObjectiveID: p.ObjectiveID, Label: models.RACILabel(p.Label)}
}

func mcpCreateRoleAssignment(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpRoleAssignmentParams) (*mcp.CallToolResult, models.RoleAssignment, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpRoleAssignmentParams) (*mcp.CallToolResult, models.RoleAssignment, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.RoleAssignment{}, err
		}
		out, err := m.CreateRoleAssignment(ctx, p.toModel())
		return summary("Assigned role %s as %s on objective %s", out.RoleID, out.Label, out.ObjectiveID), out, err
	}
}

type mcpRoleAssignmentIDParams struct {
	InstanceID   string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	AssignmentID string `json:"assignment_id" jsonschema:"the RoleAssignment's id"`
}

func mcpGetRoleAssignment(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpRoleAssignmentIDParams) (*mcp.CallToolResult, models.RoleAssignment, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpRoleAssignmentIDParams) (*mcp.CallToolResult, models.RoleAssignment, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.RoleAssignment{}, err
		}
		out, err := m.GetRoleAssignment(ctx, p.AssignmentID)
		return summary("Role %s is %s on objective %s", out.RoleID, out.Label, out.ObjectiveID), out, err
	}
}

func mcpListRoleAssignments(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, instanceAndDraftParams) (*mcp.CallToolResult, ListResult[models.RoleAssignment], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p instanceAndDraftParams) (*mcp.CallToolResult, ListResult[models.RoleAssignment], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.RoleAssignment]{}, err
		}
		out, err := m.ListRoleAssignments(ctx, p.DraftID)
		return summary("Found %d role assignment(s)", len(out)), ListResult[models.RoleAssignment]{Items: out}, err
	}
}

type mcpRoleAssignmentsForObjectiveParams struct {
	InstanceID  string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ObjectiveID string `json:"objective_id" jsonschema:"the Objective's id"`
	DraftID     string `json:"draft_id,omitempty" jsonschema:"scope to this Draft's copy; empty means the live (published) copy"`
}

func mcpListRoleAssignmentsForObjective(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpRoleAssignmentsForObjectiveParams) (*mcp.CallToolResult, ListResult[models.RoleAssignment], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpRoleAssignmentsForObjectiveParams) (*mcp.CallToolResult, ListResult[models.RoleAssignment], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.RoleAssignment]{}, err
		}
		out, err := m.ListRoleAssignmentsForObjective(ctx, p.ObjectiveID, p.DraftID)
		return summary("Found %d role assignment(s) for objective %s", len(out), p.ObjectiveID), ListResult[models.RoleAssignment]{Items: out}, err
	}
}

func mcpUpdateRoleAssignment(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpRoleAssignmentParams) (*mcp.CallToolResult, models.RoleAssignment, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpRoleAssignmentParams) (*mcp.CallToolResult, models.RoleAssignment, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.RoleAssignment{}, err
		}
		out, err := m.UpdateRoleAssignment(ctx, p.toModel())
		return summary("Updated role assignment %s: role %s is now %s on objective %s", out.ID, out.RoleID, out.Label, out.ObjectiveID), out, err
	}
}

func mcpDeleteRoleAssignment(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpRoleAssignmentIDParams) (*mcp.CallToolResult, DeleteResult, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpRoleAssignmentIDParams) (*mcp.CallToolResult, DeleteResult, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, DeleteResult{}, err
		}
		if err := m.DeleteRoleAssignment(ctx, p.AssignmentID); err != nil {
			return nil, DeleteResult{}, err
		}
		return summary("Deleted role assignment %s", p.AssignmentID), DeleteResult{Deleted: true}, nil
	}
}
