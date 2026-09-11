package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

// registerValueRuleTools registers Value/Rule tools — plain-text
// cultural/strategic statements (Value) and constraints (Rule) an AI agent
// acting as a Role needs made explicit, since a human Role-holder would
// just treat them as common knowledge. ObjectiveID "" means Agency-wide;
// otherwise scoped to one Objective.
//
// Create/Get/List only — no update or delete tool, for either scope.
// Agency-wide rows have no edit path at all in the library itself
// (decision #1: "immutable... no edit path afterward, ever" — CreateDraft
// never forks them). Objective-scoped rows do have a library-level edit
// path (fork a Draft, edit the draft-scoped copy, promote), but the gate
// on that path — only the Objective's Accountable may fork/edit/promote
// its Values/Rules — has no implementation anywhere in
// mwanachama-backend-agency yet (flagged in AgencyManager's own doc
// comment as new auth machinery needing its own design). Exposing
// Update/Delete tools here would let any caller holding valid credentials
// bypass that not-yet-built gate, so neither is built even for the
// Draft-scoped path until that design question is answered.
func registerValueRuleTools(server *mcp.Server, resolve ManagerResolver) {
	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_value", Description: "Create a Value: a cultural/strategic statement (e.g. \"we prioritize correctness over speed\"), Agency-wide or scoped to one Objective. Informational only — nothing enforces it at runtime. No update/delete tool: Agency-wide Values have no edit path at all; Objective-scoped ones need a write-authorization gate (only the Objective's Accountable may write) that doesn't exist yet."}, mcpCreateValue(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_value", Description: "Get a Value by id."}, mcpGetValue(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_values", Description: "List Values — live, or a Draft's scoped copies."}, mcpListValues(resolve))

	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_rule", Description: "Create a Rule: a constraint statement (e.g. \"never touch production data\"), Agency-wide or scoped to one Objective. Informational only — nothing enforces it at runtime. No update/delete tool — same reasoning as agency_create_value."}, mcpCreateRule(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_rule", Description: "Get a Rule by id."}, mcpGetRule(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_rules", Description: "List Rules — live, or a Draft's scoped copies."}, mcpListRules(resolve))
}

type mcpValueParams struct {
	InstanceID  string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ObjectiveID string `json:"objective_id,omitempty" jsonschema:"\"\" for an Agency-wide Value, or the Objective it's scoped to"`
	DraftID     string `json:"draft_id,omitempty" jsonschema:"\"\" for a live Value, or the owning Draft's id for a draft-scoped copy"`
	Text        string `json:"text" jsonschema:"the Value's text, e.g. \"we prioritize correctness over speed\""`
}

func (p mcpValueParams) toModel() models.Value {
	return models.Value{ObjectiveID: p.ObjectiveID, DraftID: p.DraftID, Text: p.Text}
}

func mcpCreateValue(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpValueParams) (*mcp.CallToolResult, models.Value, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpValueParams) (*mcp.CallToolResult, models.Value, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Value{}, err
		}
		out, err := m.CreateValue(ctx, p.toModel())
		return summary("Created Value %q (%s)", out.Text, out.ID), out, err
	}
}

type mcpValueIDParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ValueID    string `json:"value_id" jsonschema:"the Value's id"`
}

func mcpGetValue(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpValueIDParams) (*mcp.CallToolResult, models.Value, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpValueIDParams) (*mcp.CallToolResult, models.Value, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Value{}, err
		}
		out, err := m.GetValue(ctx, p.ValueID)
		return summary("Value %q (%s)", out.Text, out.ID), out, err
	}
}

func mcpListValues(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, instanceAndDraftParams) (*mcp.CallToolResult, ListResult[models.Value], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p instanceAndDraftParams) (*mcp.CallToolResult, ListResult[models.Value], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.Value]{}, err
		}
		out, err := m.ListValues(ctx, p.DraftID)
		return summary("Found %d value(s)", len(out)), ListResult[models.Value]{Items: out}, err
	}
}

type mcpRuleParams struct {
	InstanceID  string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ObjectiveID string `json:"objective_id,omitempty" jsonschema:"\"\" for an Agency-wide Rule, or the Objective it's scoped to"`
	DraftID     string `json:"draft_id,omitempty" jsonschema:"\"\" for a live Rule, or the owning Draft's id for a draft-scoped copy"`
	Text        string `json:"text" jsonschema:"the Rule's text, e.g. \"never touch production data\""`
}

func (p mcpRuleParams) toModel() models.Rule {
	return models.Rule{ObjectiveID: p.ObjectiveID, DraftID: p.DraftID, Text: p.Text}
}

func mcpCreateRule(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpRuleParams) (*mcp.CallToolResult, models.Rule, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpRuleParams) (*mcp.CallToolResult, models.Rule, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Rule{}, err
		}
		out, err := m.CreateRule(ctx, p.toModel())
		return summary("Created Rule %q (%s)", out.Text, out.ID), out, err
	}
}

type mcpRuleIDParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	RuleID     string `json:"rule_id" jsonschema:"the Rule's id"`
}

func mcpGetRule(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpRuleIDParams) (*mcp.CallToolResult, models.Rule, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpRuleIDParams) (*mcp.CallToolResult, models.Rule, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Rule{}, err
		}
		out, err := m.GetRule(ctx, p.RuleID)
		return summary("Rule %q (%s)", out.Text, out.ID), out, err
	}
}

func mcpListRules(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, instanceAndDraftParams) (*mcp.CallToolResult, ListResult[models.Rule], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p instanceAndDraftParams) (*mcp.CallToolResult, ListResult[models.Rule], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.Rule]{}, err
		}
		out, err := m.ListRules(ctx, p.DraftID)
		return summary("Found %d rule(s)", len(out)), ListResult[models.Rule]{Items: out}, err
	}
}
