package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// registerValueRuleTools registers Value/Rule tools — plain-text
// cultural/strategic statements (Value) and constraints (Rule) an AI agent
// acting as a Role needs made explicit, since a human Role-holder would
// just treat them as common knowledge. A Value/Rule with no attachment at
// all is Agency-wide; attach it to a Role/Goal/Objective/Workflow/WorkItem
// to scope it — a single Value/Rule may carry more than one attachment at
// once (AGD-003, 2026-09-15).
//
// Create/Get/List/attach/detach only — no update or delete tool for the
// Value/Rule row itself, for either scope. Agency-wide rows have no edit
// path at all in the library itself (decision #1: "immutable... no edit
// path afterward, ever" — CreateDraft never forks them). Attached rows do
// have a library-level edit path for their Text (fork a Draft, edit the
// draft-scoped copy, promote), but the gate on that path — only a
// target's Accountable may fork/edit/promote its Values/Rules — has no
// implementation anywhere in mwanachama-backend-agency yet (flagged in
// AgencyManager's own doc comment as new auth machinery needing its own
// design). Exposing an Update/Delete tool here would let any caller
// holding valid credentials bypass that not-yet-built gate, so neither is
// built even for the attached path until that design question is
// answered. Attach/detach themselves are live-editable (never Draft-gated
// — AGD-003 decision #3), so they carry no such restriction.
func registerValueRuleTools(server *mcp.Server, resolve ManagerResolver) {
	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_value", Title: "Create Value", Description: "Create a Value: a cultural/strategic statement (e.g. \"we prioritize correctness over speed\"), Agency-wide until attached to a target. Informational only — nothing enforces it at runtime. No update/delete tool: Agency-wide Values have no edit path at all; attached ones need a write-authorization gate (only a target's Accountable may write) that doesn't exist yet."}, mcpCreateValue(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_value", Title: "Get Value", Description: "Get a Value by id."}, mcpGetValue(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_values", Title: "List Values", Description: "List Values — live, or a Draft's scoped copies. Filter fields narrow to Values attached to a given Role/Goal/Objective/Workflow/WorkItem."}, mcpListValues(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_attach_value", Title: "Attach Value", Description: "Attach a Value to a Role/Goal/Objective/Workflow/WorkItem, scoping what would otherwise be an Agency-wide statement. A Value may carry more than one attachment. Live-editable — takes effect immediately, no Draft involved."}, mcpAttachValue(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_value_attachments", Title: "List Value Attachments", Description: "List every target a Value is attached to."}, mcpListValueAttachments(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_detach_value", Title: "Detach Value", Description: "Remove one Value attachment by its own id. Live-editable, same as attach."}, mcpDetachValue(resolve))

	mcp.AddTool(server, &mcp.Tool{Name: "agency_create_rule", Title: "Create Rule", Description: "Create a Rule: a constraint statement (e.g. \"never touch production data\"), Agency-wide until attached to a target. Informational only — nothing enforces it at runtime. No update/delete tool — same reasoning as agency_create_value."}, mcpCreateRule(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_get_rule", Title: "Get Rule", Description: "Get a Rule by id."}, mcpGetRule(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_rules", Title: "List Rules", Description: "List Rules — live, or a Draft's scoped copies. Filter fields narrow to Rules attached to a given Role/Goal/Objective/Workflow/WorkItem."}, mcpListRules(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_attach_rule", Title: "Attach Rule", Description: "Attach a Rule to a Role/Goal/Objective/Workflow/WorkItem — same behavior as agency_attach_value."}, mcpAttachRule(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_list_rule_attachments", Title: "List Rule Attachments", Description: "List every target a Rule is attached to."}, mcpListRuleAttachments(resolve))
	mcp.AddTool(server, &mcp.Tool{Name: "agency_detach_rule", Title: "Detach Rule", Description: "Remove one Rule attachment by its own id."}, mcpDetachRule(resolve))
}

type mcpValueParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	DraftID    string `json:"draft_id" jsonschema:"the Draft this Value belongs to"`
	Text       string `json:"text" jsonschema:"the Value's text, e.g. \"we prioritize correctness over speed\""`
}

func (p mcpValueParams) toModel() models.Value {
	return models.Value{DraftID: p.DraftID, Text: p.Text}
}

func mcpCreateValue(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpValueParams) (*mcp.CallToolResult, models.Value, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpValueParams) (*mcp.CallToolResult, models.Value, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Value{}, err
		}
		out, err := m.CreateValue(ctx, p.toModel())
		return summary("Created Value %s %q (%s)", out.Code, out.Text, out.ID), out, err
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
		return summary("Value %s %q (%s)", out.Code, out.Text, out.ID), out, err
	}
}

// mcpListValuesParams extends instanceAndDraftParams with ValueFilter's
// fields, including the five target-id filters (AGD-003).
type mcpListValuesParams struct {
	instanceAndDraftParams
	Code        string `json:"code,omitempty" jsonschema:"filter: case-insensitive substring match on Code"`
	Text        string `json:"text,omitempty" jsonschema:"filter: case-insensitive substring match on Text"`
	RoleID      string `json:"role_id,omitempty" jsonschema:"filter: only Values attached to this Role"`
	GoalID      string `json:"goal_id,omitempty" jsonschema:"filter: only Values attached to this Goal"`
	ObjectiveID string `json:"objective_id,omitempty" jsonschema:"filter: only Values attached to this Objective"`
	WorkflowID  string `json:"workflow_id,omitempty" jsonschema:"filter: only Values attached to this Workflow"`
	WorkItemID  string `json:"work_item_id,omitempty" jsonschema:"filter: only Values attached to this WorkItem"`
}

func mcpListValues(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpListValuesParams) (*mcp.CallToolResult, ListResult[models.Value], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpListValuesParams) (*mcp.CallToolResult, ListResult[models.Value], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.Value]{}, err
		}
		out, err := m.ListValues(ctx, p.DraftID, mwanachamaagency.ValueFilter{
			Code: p.Code, Text: p.Text,
			RoleID: p.RoleID, GoalID: p.GoalID, ObjectiveID: p.ObjectiveID,
			WorkflowID: p.WorkflowID, WorkItemID: p.WorkItemID,
		})
		return summaryWithContext(fmt.Sprintf("Found %d value(s)", len(out)), valuesContextText(out)), ListResult[models.Value]{Items: out}, err
	}
}

// valuesContextText is what an LLM sees for a Values list — one line per
// Value, field-by-field rather than a struct dump, so dropping a field
// from an LLM's context later is just deleting it from this Sprintf.
func valuesContextText(values []models.Value) string {
	if len(values) == 0 {
		return "There are no values yet."
	}
	lines := make([]string, 0, len(values))
	for _, v := range values {
		lines = append(lines, fmt.Sprintf("- %s: %q (value id: %s)", v.Code, v.Text, v.ID))
	}
	return "Current values:\n" + strings.Join(lines, "\n")
}

type mcpAttachValueParams struct {
	InstanceID string            `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	ValueID    string            `json:"value_id" jsonschema:"the Value's id"`
	TargetType models.TargetType `json:"target_type" jsonschema:"one of: role, goal, objective, workflow, work_item"`
	TargetID   string            `json:"target_id" jsonschema:"the target's id"`
}

func mcpAttachValue(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpAttachValueParams) (*mcp.CallToolResult, models.ValueAttachment, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpAttachValueParams) (*mcp.CallToolResult, models.ValueAttachment, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.ValueAttachment{}, err
		}
		out, err := m.CreateValueAttachment(ctx, p.ValueID, p.TargetType, p.TargetID)
		return summary("Attached Value %s to %s %s", out.ValueCode, out.TargetType, out.TargetCode), out, err
	}
}

func mcpListValueAttachments(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpValueIDParams) (*mcp.CallToolResult, ListResult[models.ValueAttachment], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpValueIDParams) (*mcp.CallToolResult, ListResult[models.ValueAttachment], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.ValueAttachment]{}, err
		}
		out, err := m.ListValueAttachments(ctx, p.ValueID)
		return summaryWithContext(fmt.Sprintf("Found %d attachment(s)", len(out)), valueAttachmentsContextText(out)), ListResult[models.ValueAttachment]{Items: out}, err
	}
}

// valueAttachmentsContextText is what an LLM sees for a Value's attachment
// list. Referenced by Code, not row id, on both sides (AGD-003) — but the
// join row's own id is what agency_detach_value actually takes, so it's
// still surfaced here alongside the Codes.
func valueAttachmentsContextText(attachments []models.ValueAttachment) string {
	if len(attachments) == 0 {
		return "This Value has no attachments — it is Agency-wide."
	}
	lines := make([]string, 0, len(attachments))
	for _, a := range attachments {
		lines = append(lines, fmt.Sprintf("- %s %s (attachment id: %s)", a.TargetType, a.TargetCode, a.ID))
	}
	return "Current attachments:\n" + strings.Join(lines, "\n")
}

type mcpAttachmentIDParams struct {
	InstanceID   string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	AttachmentID string `json:"attachment_id" jsonschema:"the attachment's own id"`
}

func mcpDetachValue(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpAttachmentIDParams) (*mcp.CallToolResult, DeleteResult, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpAttachmentIDParams) (*mcp.CallToolResult, DeleteResult, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, DeleteResult{}, err
		}
		err = m.DeleteValueAttachment(ctx, p.AttachmentID)
		return summary("Detached value attachment %s", p.AttachmentID), DeleteResult{Deleted: err == nil}, err
	}
}

type mcpRuleParams struct {
	InstanceID string `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	DraftID    string `json:"draft_id" jsonschema:"the Draft this Rule belongs to"`
	Text       string `json:"text" jsonschema:"the Rule's text, e.g. \"never touch production data\""`
}

func (p mcpRuleParams) toModel() models.Rule {
	return models.Rule{DraftID: p.DraftID, Text: p.Text}
}

func mcpCreateRule(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpRuleParams) (*mcp.CallToolResult, models.Rule, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpRuleParams) (*mcp.CallToolResult, models.Rule, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.Rule{}, err
		}
		out, err := m.CreateRule(ctx, p.toModel())
		return summary("Created Rule %s %q (%s)", out.Code, out.Text, out.ID), out, err
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
		return summary("Rule %s %q (%s)", out.Code, out.Text, out.ID), out, err
	}
}

// mcpListRulesParams extends instanceAndDraftParams with RuleFilter's
// fields, including the five target-id filters (AGD-003).
type mcpListRulesParams struct {
	instanceAndDraftParams
	Code        string `json:"code,omitempty" jsonschema:"filter: case-insensitive substring match on Code"`
	Text        string `json:"text,omitempty" jsonschema:"filter: case-insensitive substring match on Text"`
	RoleID      string `json:"role_id,omitempty" jsonschema:"filter: only Rules attached to this Role"`
	GoalID      string `json:"goal_id,omitempty" jsonschema:"filter: only Rules attached to this Goal"`
	ObjectiveID string `json:"objective_id,omitempty" jsonschema:"filter: only Rules attached to this Objective"`
	WorkflowID  string `json:"workflow_id,omitempty" jsonschema:"filter: only Rules attached to this Workflow"`
	WorkItemID  string `json:"work_item_id,omitempty" jsonschema:"filter: only Rules attached to this WorkItem"`
}

func mcpListRules(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpListRulesParams) (*mcp.CallToolResult, ListResult[models.Rule], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpListRulesParams) (*mcp.CallToolResult, ListResult[models.Rule], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.Rule]{}, err
		}
		out, err := m.ListRules(ctx, p.DraftID, mwanachamaagency.RuleFilter{
			Code: p.Code, Text: p.Text,
			RoleID: p.RoleID, GoalID: p.GoalID, ObjectiveID: p.ObjectiveID,
			WorkflowID: p.WorkflowID, WorkItemID: p.WorkItemID,
		})
		return summaryWithContext(fmt.Sprintf("Found %d rule(s)", len(out)), rulesContextText(out)), ListResult[models.Rule]{Items: out}, err
	}
}

// rulesContextText is what an LLM sees for a Rules list — one line per
// Rule, field-by-field rather than a struct dump, so dropping a field
// from an LLM's context later is just deleting it from this Sprintf.
func rulesContextText(rules []models.Rule) string {
	if len(rules) == 0 {
		return "There are no rules yet."
	}
	lines := make([]string, 0, len(rules))
	for _, r := range rules {
		lines = append(lines, fmt.Sprintf("- %s: %q (rule id: %s)", r.Code, r.Text, r.ID))
	}
	return "Current rules:\n" + strings.Join(lines, "\n")
}

type mcpAttachRuleParams struct {
	InstanceID string            `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	RuleID     string            `json:"rule_id" jsonschema:"the Rule's id"`
	TargetType models.TargetType `json:"target_type" jsonschema:"one of: role, goal, objective, workflow, work_item"`
	TargetID   string            `json:"target_id" jsonschema:"the target's id"`
}

func mcpAttachRule(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpAttachRuleParams) (*mcp.CallToolResult, models.RuleAttachment, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpAttachRuleParams) (*mcp.CallToolResult, models.RuleAttachment, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, models.RuleAttachment{}, err
		}
		out, err := m.CreateRuleAttachment(ctx, p.RuleID, p.TargetType, p.TargetID)
		return summary("Attached Rule %s to %s %s", out.RuleCode, out.TargetType, out.TargetCode), out, err
	}
}

func mcpListRuleAttachments(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpRuleIDParams) (*mcp.CallToolResult, ListResult[models.RuleAttachment], error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpRuleIDParams) (*mcp.CallToolResult, ListResult[models.RuleAttachment], error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, ListResult[models.RuleAttachment]{}, err
		}
		out, err := m.ListRuleAttachments(ctx, p.RuleID)
		return summaryWithContext(fmt.Sprintf("Found %d attachment(s)", len(out)), ruleAttachmentsContextText(out)), ListResult[models.RuleAttachment]{Items: out}, err
	}
}

// ruleAttachmentsContextText mirrors valueAttachmentsContextText — see its
// doc comment.
func ruleAttachmentsContextText(attachments []models.RuleAttachment) string {
	if len(attachments) == 0 {
		return "This Rule has no attachments — it is Agency-wide."
	}
	lines := make([]string, 0, len(attachments))
	for _, a := range attachments {
		lines = append(lines, fmt.Sprintf("- %s %s (attachment id: %s)", a.TargetType, a.TargetCode, a.ID))
	}
	return "Current attachments:\n" + strings.Join(lines, "\n")
}

func mcpDetachRule(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpAttachmentIDParams) (*mcp.CallToolResult, DeleteResult, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpAttachmentIDParams) (*mcp.CallToolResult, DeleteResult, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, DeleteResult{}, err
		}
		err = m.DeleteRuleAttachment(ctx, p.AttachmentID)
		return summary("Detached rule attachment %s", p.AttachmentID), DeleteResult{Deleted: err == nil}, err
	}
}
