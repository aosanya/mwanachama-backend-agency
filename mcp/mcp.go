package mcp

import (
	"context"
	"fmt"
	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-shared/dispatch"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ManagerResolver func(ctx context.Context, id string) (mwanachamaagency.AgencyManager, error)

func RegisterTools(server *mcp.Server, resolve ManagerResolver) {
	RegisterToolsAs(server, resolve, nil)
}
func RegisterToolsAs(server *mcp.Server, resolve ManagerResolver, caller dispatch.Caller) {
	if err := registerDeclaredTools(server, resolve, caller); err != nil {
		panic(fmt.Sprintf("agency tools: %v", err))
	}
	registerCompositeTools(server, resolve)
	registerCatalogTools(server)
	registerDashboardUI(server)
}

type ListResult[T any] struct {
	Items []T `json:"items" jsonschema:"the matching items"`
}
type DeleteResult struct {
	Deleted bool `json:"deleted"`
}
type instanceOnlyParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
}
type instanceAndDraftParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	DraftID    string `json:"draft_id" jsonschema:"the Draft to scope to"`
}

func summary(format string, args ...any) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(format, args...)}}}
}
func summaryWithContext(summaryText, contextText string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: summaryText}, &mcp.TextContent{Text: contextText}}}
}
