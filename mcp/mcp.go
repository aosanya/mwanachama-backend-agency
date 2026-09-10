// Package mcp exposes mwanachama-backend-agency's manager methods as Model
// Context Protocol tools — mirroring routes/'s "one file per domain group,
// each function takes an already-resolved manager" shape, adapted for MCP:
// mcp.AddTool wires a handler onto the server once, at registration time,
// not per-call the way routes/ rebuilds Routes(m, names) fresh on every
// HTTP request — so every tool here closes over a ManagerResolver instead
// of a concrete manager.
package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

// ManagerResolver turns a caller-supplied id into the AgencyManager that id
// names. How the id is interpreted is entirely the mounting host's
// decision — a single-Agency host can ignore it and always return the same
// manager; a multi-Agency host (e.g. mwanachama-wakala-api's own Registry)
// resolves it per call.
type ManagerResolver func(ctx context.Context, id string) (mwanachamaagency.AgencyManager, error)

// RegisterTools registers every tool this package exposes onto server,
// each one resolving its manager via resolve.
func RegisterTools(server *mcp.Server, resolve ManagerResolver) {
	registerAgencyTools(server, resolve)
	registerGoalWorkflowTools(server, resolve)
	registerObjectiveTools(server, resolve)
	registerOrgChartTools(server, resolve)
	registerRACITools(server, resolve)
	registerWorkPlanTools(server, resolve)
	registerValueRuleTools(server, resolve)
	registerCompositeTools(server, resolve)
}

// ListResult wraps a List/Match tool's slice: MCP's spec defines
// CallToolResult.structuredContent as a JSON object, and the go-sdk
// otherwise serializes a bare slice return type directly into it — a shape
// a schema-strict client rejects even though the go-sdk's own test client
// tolerates it. Every List/Match tool in this package returns one of
// these, never a bare slice.
type ListResult[T any] struct {
	Items []T `json:"items" jsonschema:"the matching items"`
}

// DeleteResult is the result shape for a tool wrapping a manager method
// that returns only an error (Delete*/Remove*) — AddTool needs some result
// type, and a bare bool reads clearly on the wire.
type DeleteResult struct {
	Deleted bool `json:"deleted"`
}

// instanceOnlyParams is the params shape for a tool needing only which
// Agency to act on — e.g. agency_get_agency.
type instanceOnlyParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
}

// instanceAndDraftParams is the params shape for a List call scoped to a
// Draft — Goal/Workflow/WorkItem/Role/RoleAssignment's shared nullable-
// DraftID convention: empty means the live (published) copy.
type instanceAndDraftParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	DraftID    string `json:"draft_id,omitempty" jsonschema:"scope to this Draft's copy; empty means the live (published) copy"`
}
