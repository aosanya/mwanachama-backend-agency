package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// registerModuleTools registers Module/ObjectiveModule tools — the catalog
// of capability domains (e.g. "git", "taskmanager") an Agency may enable,
// and the Objective↔Module links that gate whether a mounting gateway
// mounts a domain's routes at all (AGD-002).
func registerModuleTools(server *mcp.Server, resolve ManagerResolver) {
	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_module", Title: "Create Module", Description: "Create a Module catalog entry — a capability domain (e.g. \"git\", \"taskmanager\") this Agency may enable. Enabled gates whether a mounting gateway registers that domain's routes at all."}, mcpCreateModule(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_update_module", Title: "Update Module", Description: "Update a Module (full replace — supply every field you want kept). id is required; code cannot be changed."}, mcpUpdateModule(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_delete_module", Title: "Delete Module", Description: "Delete a Module. Cascades: also deletes every ObjectiveModule link referencing it — warn the operator about that before calling this."}, mcpDeleteModule(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_modules", Title: "List Modules", Description: "List Modules — live, or a Draft's scoped copies."}, mcpListModules(resolve))

	mcp.AddTool(server, &mcp.Tool{Name: "agency_attach_module", Title: "Attach Module", Description: "Link a Module to an Objective. Both must already exist in the same scope: live-to-live, or the same Draft."}, mcpAttachModule(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_objective_modules", Title: "List Objective Modules", Description: "List the Modules linked to an Objective."}, mcpListObjectiveModules(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_detach_module", Title: "Detach Module", Description: "Remove one Objective↔Module link by its own id."}, mcpDetachModule(resolve))
}

// mcpModuleParams mirrors models.Module, but with every field but Name
// marked omitempty — the go-sdk's schema inference (google/jsonschema-go)
// treats any field lacking omitempty/omitzero as required. Code is
// required for create, ignored for update (immutable after create); ID is
// required for update, ignored for create.
type mcpModuleParams struct {
	InstanceID  string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ID          string `json:"id,omitempty" jsonschema:"the Module's id — required for update, ignored for create"`
	DraftID     string `json:"draft_id,omitempty" jsonschema:"\"\" for a live Module, or the owning Draft's id for a draft-scoped copy"`
	Code        string `json:"code,omitempty" jsonschema:"the Module's stable slug, e.g. \"git\" — required for create, ignored for update"`
	Name        string `json:"name" jsonschema:"the Module's name"`
	Description string `json:"description,omitempty" jsonschema:"the Module's description"`
	Enabled     bool   `json:"enabled,omitempty" jsonschema:"whether this Module is enabled — gates whether a mounting gateway registers its routes"`
}

func (p mcpModuleParams) toModel() models.Module {
	return models.Module{ID: p.ID, DraftID: p.DraftID, Code: p.Code, Name: p.Name, Description: p.Description, Enabled: p.Enabled}
}

func mcpCreateModule(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpModuleParams) (*mcp.CallToolResult, models.Module, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpModuleParams) (*mcp.CallToolResult, models.Module, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Module{}, err
		}
		out, err := m.CreateModule(ctx, p.toModel())
		return summary("Created Module %s %q (%s)", out.Code, out.Name, out.ID), out, err
	}
}

type mcpModuleIDParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ModuleID   string `json:"module_id" jsonschema:"the Module's id"`
}

// mcpListModulesParams extends instanceAndDraftParams with ModuleFilter's
// fields.
type mcpListModulesParams struct {
	instanceAndDraftParams
	Code        string `json:"code,omitempty" jsonschema:"filter: case-insensitive substring match on Code"`
	Name        string `json:"name,omitempty" jsonschema:"filter: case-insensitive substring match on Name"`
	Description string `json:"description,omitempty" jsonschema:"filter: case-insensitive substring match on Description"`
	Enabled     *bool  `json:"enabled,omitempty" jsonschema:"filter: exact match on Enabled"`
}

func mcpListModules(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpListModulesParams) (*mcp.CallToolResult, ListResult[models.Module], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpListModulesParams) (*mcp.CallToolResult, ListResult[models.Module], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.Module]{}, err
		}
		out, err := m.ListModules(ctx, p.DraftID, mwanachamaagency.ModuleFilter{
			Code: p.Code, Name: p.Name, Description: p.Description, Enabled: p.Enabled,
		})
		return summaryWithContext(fmt.Sprintf("Found %d module(s)", len(out)), modulesContextText(out)), ListResult[models.Module]{Items: out}, err
	}
}

// modulesContextText is what an LLM sees for a Modules list — one line per
// Module, field-by-field rather than a struct dump, so dropping a field
// from an LLM's context later is just deleting it from this Sprintf.
func modulesContextText(modules []models.Module) string {
	if len(modules) == 0 {
		return "There are no modules yet."
	}
	lines := make([]string, 0, len(modules))
	for _, mo := range modules {
		state := "disabled"
		if mo.Enabled {
			state = "enabled"
		}
		lines = append(lines, fmt.Sprintf("- %s: %s (%s, module id: %s)", mo.Code, mo.Name, state, mo.ID))
	}
	return "Current modules:\n" + strings.Join(lines, "\n")
}

func mcpUpdateModule(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpModuleParams) (*mcp.CallToolResult, models.Module, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpModuleParams) (*mcp.CallToolResult, models.Module, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Module{}, err
		}
		out, err := m.UpdateModule(ctx, p.toModel())
		return summary("Updated Module %s %q (%s)", out.Code, out.Name, out.ID), out, err
	}
}

func mcpDeleteModule(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpModuleIDParams) (*mcp.CallToolResult, DeleteResult, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpModuleIDParams) (*mcp.CallToolResult, DeleteResult, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, DeleteResult{}, err
		}
		if err := m.DeleteModule(ctx, p.ModuleID); err != nil {
			return nil, DeleteResult{}, err
		}
		return summary("Deleted Module %s", p.ModuleID), DeleteResult{Deleted: true}, nil
	}
}

type mcpAttachModuleParams struct {
	InstanceID  string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ObjectiveID string `json:"objective_id" jsonschema:"the Objective's id"`
	ModuleID    string `json:"module_id" jsonschema:"the Module's id"`
	DraftID     string `json:"draft_id,omitempty" jsonschema:"\"\" to link within live data, or the owning Draft's id to link within that Draft's scope — the Objective and Module must both already exist in that same scope"`
}

func mcpAttachModule(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpAttachModuleParams) (*mcp.CallToolResult, models.ObjectiveModule, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpAttachModuleParams) (*mcp.CallToolResult, models.ObjectiveModule, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.ObjectiveModule{}, err
		}
		out, err := m.CreateObjectiveModule(ctx, models.ObjectiveModule{DraftID: p.DraftID, ObjectiveID: p.ObjectiveID, ModuleID: p.ModuleID})
		return summary("Linked module %s to objective %s", out.ModuleID, out.ObjectiveID), out, err
	}
}

type mcpObjectiveModulesParams struct {
	InstanceID  string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ObjectiveID string `json:"objective_id" jsonschema:"the Objective's id"`
	DraftID     string `json:"draft_id,omitempty" jsonschema:"scope to this Draft's copy; empty means the live (published) copy"`
	ModuleID    string `json:"module_id,omitempty" jsonschema:"filter: exact match on the linked Module's id"`
}

func mcpListObjectiveModules(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpObjectiveModulesParams) (*mcp.CallToolResult, ListResult[models.ObjectiveModule], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpObjectiveModulesParams) (*mcp.CallToolResult, ListResult[models.ObjectiveModule], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.ObjectiveModule]{}, err
		}
		out, err := m.ListObjectiveModulesForObjective(ctx, p.ObjectiveID, p.DraftID, mwanachamaagency.ObjectiveModuleFilter{ModuleID: p.ModuleID})
		return summaryWithContext(fmt.Sprintf("Found %d module(s) linked to objective %s", len(out), p.ObjectiveID), objectiveModulesContextText(out)), ListResult[models.ObjectiveModule]{Items: out}, err
	}
}

// objectiveModulesContextText is what an LLM sees for an Objective's linked
// Modules — one line per link, field-by-field rather than a struct dump.
func objectiveModulesContextText(links []models.ObjectiveModule) string {
	if len(links) == 0 {
		return "This objective has no modules linked."
	}
	lines := make([]string, 0, len(links))
	for _, l := range links {
		lines = append(lines, fmt.Sprintf("- module %s (link id: %s)", l.ModuleID, l.ID))
	}
	return "Linked modules:\n" + strings.Join(lines, "\n")
}

type mcpObjectiveModuleIDParams struct {
	InstanceID        string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ObjectiveModuleID string `json:"objective_module_id" jsonschema:"the Objective↔Module link's own id"`
}

func mcpDetachModule(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpObjectiveModuleIDParams) (*mcp.CallToolResult, DeleteResult, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpObjectiveModuleIDParams) (*mcp.CallToolResult, DeleteResult, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, DeleteResult{}, err
		}
		err = m.DeleteObjectiveModule(ctx, p.ObjectiveModuleID)
		return summary("Detached module link %s", p.ObjectiveModuleID), DeleteResult{Deleted: err == nil}, err
	}
}
