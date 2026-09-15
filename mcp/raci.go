package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// registerRACITools registers Role/RoleAssignment tools — the RACI matrix:
// personas (Role) assigned to Objectives with a
// Responsible/Accountable/Consulted/Informed label. A live Role or
// RoleAssignment cannot be edited or deleted directly — fork a Draft, edit
// the draft-scoped copy, promote (agency_create_draft/agency_promote_draft
// in agency.go).
func registerRACITools(server *mcp.Server, resolve ManagerResolver) {
	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_role", Title: "Create Role", Description: "Create a Role (a persona, e.g. \"bug-hunter\")."}, mcpCreateRole(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_role", Title: "Get Role", Description: "Get a Role by id."}, mcpGetRole(resolve))
	mcp.AddTool(server, &mcp.Tool{
		Name: "agency_list_roles", Title: "List Roles", Description: "List Roles — live, or a Draft's scoped copies.",
		Meta: dashboardUIMeta,
	}, mcpListRoles(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_update_role", Title: "Update Role", Description: "Update a Role. Refused on a live Role — only a Draft-scoped copy may be edited."}, mcpUpdateRole(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_delete_role", Title: "Delete Role", Description: "Delete a Role. Refused on a live Role — only a Draft-scoped copy may be deleted."}, mcpDeleteRole(resolve))

	mcp.AddTool(server, &mcp.Tool{Name: "agency_set_raci", Title: "Assign RACI", Description: "Assign a Role to an Objective with a RACI label (responsible/accountable/consulted/informed). At most one accountable assignment may exist per Objective."}, mcpCreateRoleAssignment(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_role_assignment", Title: "Get RACI Assignment", Description: "Get a RoleAssignment by id."}, mcpGetRoleAssignment(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_role_assignments", Title: "List RACI Assignments", Description: "List RoleAssignments — live, or a Draft's scoped copies."}, mcpListRoleAssignments(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_role_assignments_for_objective", Title: "List RACI for Objective", Description: "List who is assigned to an Objective, and with what RACI label."}, mcpListRoleAssignmentsForObjective(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_update_role_assignment", Title: "Update RACI Assignment", Description: "Update a RoleAssignment. Refused on a live one — only a Draft-scoped copy may be edited."}, mcpUpdateRoleAssignment(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_delete_role_assignment", Title: "Delete RACI Assignment", Description: "Delete a RoleAssignment. Refused on a live one — only a Draft-scoped copy may be deleted."}, mcpDeleteRoleAssignment(resolve))
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
		return summary("Created Role %s %q (%s)", out.Code, out.Name, out.ID), out, err
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
		return summary("Role %s %q (%s)", out.Code, out.Name, out.ID), out, err
	}
}

// mcpListRolesParams extends instanceAndDraftParams with RoleFilter's
// fields.
type mcpListRolesParams struct {
	instanceAndDraftParams
	Code        string `json:"code,omitempty" jsonschema:"filter: case-insensitive substring match on Code"`
	Name        string `json:"name,omitempty" jsonschema:"filter: case-insensitive substring match on Name"`
	Description string `json:"description,omitempty" jsonschema:"filter: case-insensitive substring match on Description"`
}

func mcpListRoles(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpListRolesParams) (*mcp.CallToolResult, ListResult[models.Role], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpListRolesParams) (*mcp.CallToolResult, ListResult[models.Role], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.Role]{}, err
		}
		out, err := m.ListRoles(ctx, p.DraftID, mwanachamaagency.RoleFilter{Code: p.Code, Name: p.Name, Description: p.Description})
		return summaryWithContext(fmt.Sprintf("Found %d role(s)", len(out)), rolesContextText(out)), ListResult[models.Role]{Items: out}, err
	}
}

// rolesContextText is what an LLM sees for a Roles list opened outside its
// own tool loop (mwanachama-wakala-studio's sidebar shortcut) — one line per
// Role, field-by-field rather than a struct dump, so dropping a field from
// an LLM's context later is just deleting it from this Sprintf.
func rolesContextText(roles []models.Role) string {
	if len(roles) == 0 {
		return "There are no roles yet."
	}
	lines := make([]string, 0, len(roles))
	for _, r := range roles {
		lines = append(lines, fmt.Sprintf("- %s: %s (role id: %s)", r.Code, r.Name, r.ID))
	}
	return "Current roles:\n" + strings.Join(lines, "\n")
}

func mcpUpdateRole(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpRoleParams) (*mcp.CallToolResult, models.Role, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpRoleParams) (*mcp.CallToolResult, models.Role, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Role{}, err
		}
		out, err := m.UpdateRole(ctx, p.toModel())
		return summary("Updated Role %s %q (%s)", out.Code, out.Name, out.ID), out, err
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

// mcpListRoleAssignmentsParams extends instanceAndDraftParams with
// RoleAssignmentFilter's fields.
type mcpListRoleAssignmentsParams struct {
	instanceAndDraftParams
	Label string `json:"label,omitempty" jsonschema:"filter: exact match on the RACI label (responsible, accountable, consulted, or informed)"`
}

func mcpListRoleAssignments(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpListRoleAssignmentsParams) (*mcp.CallToolResult, ListResult[models.RoleAssignment], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpListRoleAssignmentsParams) (*mcp.CallToolResult, ListResult[models.RoleAssignment], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.RoleAssignment]{}, err
		}
		out, err := m.ListRoleAssignments(ctx, p.DraftID, mwanachamaagency.RoleAssignmentFilter{Label: p.Label})
		return summaryWithContext(fmt.Sprintf("Found %d role assignment(s)", len(out)), roleAssignmentsContextText(out)), ListResult[models.RoleAssignment]{Items: out}, err
	}
}

// roleAssignmentsContextText is what an LLM sees for a RoleAssignments
// list — one line per assignment, field-by-field rather than a struct
// dump, so dropping a field from an LLM's context later is just deleting
// it from this Sprintf. Shared by agency_list_role_assignments and
// agency_list_role_assignments_for_objective — same row shape either way.
func roleAssignmentsContextText(assignments []models.RoleAssignment) string {
	if len(assignments) == 0 {
		return "There are no role assignments yet."
	}
	lines := make([]string, 0, len(assignments))
	for _, a := range assignments {
		lines = append(lines, fmt.Sprintf("- role %s is %s on objective %s (assignment id: %s)", a.RoleID, a.Label, a.ObjectiveID, a.ID))
	}
	return "Current role assignments:\n" + strings.Join(lines, "\n")
}

type mcpRoleAssignmentsForObjectiveParams struct {
	InstanceID  string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ObjectiveID string `json:"objective_id" jsonschema:"the Objective's id"`
	DraftID     string `json:"draft_id,omitempty" jsonschema:"scope to this Draft's copy; empty means the live (published) copy"`
	Label       string `json:"label,omitempty" jsonschema:"filter: exact match on the RACI label (responsible, accountable, consulted, or informed)"`
}

func mcpListRoleAssignmentsForObjective(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpRoleAssignmentsForObjectiveParams) (*mcp.CallToolResult, ListResult[models.RoleAssignment], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpRoleAssignmentsForObjectiveParams) (*mcp.CallToolResult, ListResult[models.RoleAssignment], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.RoleAssignment]{}, err
		}
		out, err := m.ListRoleAssignmentsForObjective(ctx, p.ObjectiveID, p.DraftID, mwanachamaagency.RoleAssignmentFilter{Label: p.Label})
		return summaryWithContext(fmt.Sprintf("Found %d role assignment(s) for objective %s", len(out), p.ObjectiveID), roleAssignmentsContextText(out)), ListResult[models.RoleAssignment]{Items: out}, err
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
