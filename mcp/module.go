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
// mounts a domain's routes at all (AGD-002). Module itself is read-only
// here: which codes exist, their Name/Description and their Locked state
// come from models.ModuleCatalog, not this tool surface — there is no
// create/update/delete tool for Module. Enabling one is
// agency_attach_module: Enabled is derived from the links, never set.
func registerModuleTools(server *mcp.Server, resolve ManagerResolver) {
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_modules", Title: "List Modules", Description: "List Modules — live, or a Draft's scoped copies. Read-only: the catalog of which modules exist comes from a fixed list, not this tool, and Enabled is derived from the Objectives that link each Module."}, mcpListModules(resolve))

	mcp.AddTool(server, &mcp.Tool{Name: "agency_attach_module", Title: "Attach Module", Description: "Link a Module to an Objective — this is how a Module is selected: a Module is enabled exactly while some Objective links it. Both must already exist in the same scope: live-to-live, or the same Draft."}, mcpAttachModule(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_objective_modules", Title: "List Objective Modules", Description: "List the Modules linked to an Objective."}, mcpListObjectiveModules(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_detach_module", Title: "Detach Module", Description: "Remove one Objective↔Module link by its own id. Removing the last link to a Module disables it, unless it is locked."}, mcpDetachModule(resolve))
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
		locked := ""
		if mo.Locked {
			locked = ", locked"
		}
		lines = append(lines, fmt.Sprintf("- %s: %s (%s%s, module id: %s)", mo.Code, mo.Name, state, locked, mo.ID))
	}
	return "Current modules:\n" + strings.Join(lines, "\n")
}

type mcpAttachModuleParams struct {
	InstanceID  string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ObjectiveID string `json:"objective_id" jsonschema:"the Objective's id"`
	ModuleID    string `json:"module_id" jsonschema:"the Module's id"`
	DraftID     string `json:"draft_id" jsonschema:"the Draft to link within — the Objective and Module must both already exist in that same Draft"`
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
	DraftID     string `json:"draft_id" jsonschema:"the Draft to scope to"`
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
