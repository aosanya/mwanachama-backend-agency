package mcp

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// typeInfo is a plain-language glossary entry for one domain type — what it
// is, and (if it has one) which tool creates it. Static content, not
// derived from anything at runtime, so it needs no ManagerResolver and
// answers instantly regardless of which Agency the chat is scoped to.
type typeInfo struct {
	Name     string `json:"name"`
	WhatItIs string `json:"what_it_is"`
	HowToAdd string `json:"how_to_add"`
}

// domainTypeCatalog is every domain type this package's tools expose,
// mirroring each type's own doc comment in models/ — kept short and
// operator-facing rather than restating those comments verbatim.
var domainTypeCatalog = []typeInfo{
	{
		Name:     "Agency",
		WhatItIs: "The root of the whole design: the Agency document itself (name, description), everything else hangs off it. Exactly one per Instance.",
		HowToAdd: "agency_create_agency, once, when an Instance is first registered — there is no adding a second one.",
	},
	{
		Name:     "Department",
		WhatItIs: "One node in the org chart: a parent/child tree of named departments (e.g. \"Software\", \"Mechanical\"), each pinned to a Level.",
		HowToAdd: "agency_create_department (name, level_id, and parent_id — empty parent_id makes it top-level).",
	},
	{
		Name:     "Level",
		WhatItIs: "A named rung departments sit at in the org chart (e.g. \"Division\", \"Team\"). Depth is just a sort key — levels can share one or leave gaps.",
		HowToAdd: "agency_create_level (name, optional depth).",
	},
	{
		Name:     "Goal",
		WhatItIs: "A strategic objective the Agency pursues — the top of the work breakdown: Goal → Objective → WorkItem.",
		HowToAdd: "agency_create_goal (title, optional description).",
	},
	{
		Name:     "Objective",
		WhatItIs: "A measurable, granular piece of a Goal — a Goal can have several. Each Objective is the unit one Role gets RACI-assigned to, and the unit WorkItems and Objective-scoped Values/Rules attach to.",
		HowToAdd: "agency_create_objective (goal_id, title) — always created under an existing Goal.",
	},
	{
		Name:     "Role",
		WhatItIs: "A persona that can be assigned to an Objective via RACI (e.g. \"bug-hunter\", \"code-reviewer\"). A live Role can't be edited directly — only a Draft-scoped copy can.",
		HowToAdd: "agency_create_role (name, optional description).",
	},
	{
		Name:     "RoleAssignment (RACI)",
		WhatItIs: "Assigns one Role to one Objective with a RACI label — responsible, accountable, consulted, or informed. At most one accountable assignment may exist per Objective.",
		HowToAdd: "agency_set_raci (role_id, objective_id, label).",
	},
	{
		Name:     "Value",
		WhatItIs: "A cultural/strategic statement made explicit for an AI agent acting as a Role to read directly (e.g. \"we prioritize correctness over speed\") — the kind of thing a human would just treat as common knowledge. Agency-wide, or scoped to one Objective; informational only, nothing enforces it at runtime.",
		HowToAdd: "agency_create_value (text, optional objective_id). No update or delete tool: Agency-wide Values have no edit path at all, and Objective-scoped ones are pending a write-authorization design.",
	},
	{
		Name:     "Rule",
		WhatItIs: "A constraint statement made explicit the same way as a Value (e.g. \"never touch production data\") — Agency-wide or Objective-scoped, informational only.",
		HowToAdd: "agency_create_rule (text, optional objective_id). Same no-update/no-delete reasoning as Value.",
	},
	{
		Name:     "Workflow",
		WhatItIs: "An ordered sequence of WorkItems belonging to the Agency.",
		HowToAdd: "agency_create_workflow (name, optional description).",
	},
	{
		Name:     "WorkItem",
		WhatItIs: "A single unit of work, owned by exactly one Workflow and belonging to exactly one Objective it advances.",
		HowToAdd: "agency_create_work_item (workflow_id, objective_id, title).",
	},
	{
		Name:     "EventFlowStep",
		WhatItIs: "The queryable, structured projection of one step in a Workflow's event-flow diagram: which topic/consumer triggers it, which handler runs it, what it emits on success or error.",
		HowToAdd: "Not directly creatable through chat — populated from a Workflow's EventFlows data. Look one up with agency_lookup_event_flow_step or agency_list_event_flow_steps.",
	},
	{
		Name:     "Draft",
		WhatItIs: "An in-progress, scoped copy of the live Agency's editable data, opened so changes can be made without touching what's live yet.",
		HowToAdd: "agency_create_draft — though in this chat, one is already open and managed by the host; never call it yourself.",
	},
	{
		Name:     "AgencyPublication",
		WhatItIs: "A versioned, content-hashed, tagged snapshot of the Agency's live data.",
		HowToAdd: "Not created directly — agency_publish_agency produces one as its result.",
	},
	{
		Name:     "AgencySnapshot",
		WhatItIs: "An immutable audit record written automatically each time a Draft is promoted to live.",
		HowToAdd: "Never created directly — it's a side effect of agency_promote_draft.",
	},
	{
		Name:     "Issue",
		WhatItIs: "A system-detected problem found by scanning the Agency — e.g. a Work Item whose Objective no longer exists, an Objective with no Accountable role, or a Goal with no Objectives. info/warning/error severity, scoped either to the live Agency or to one Draft. An error-severity Issue blocks publishing.",
		HowToAdd: "Never created directly — agency_list_issues re-scans and regenerates the Issues every time it's called.",
	},
}

// registerCatalogTools registers the domain-type glossary tool. Needs no
// ManagerResolver — the catalog is static content describing the data
// model itself, not any one Agency's data — matching registerDashboardUI's
// same "no manager needed" precedent in dashboard_ui.go.
func registerCatalogTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "agency_list_types", Title: "List Domain Types",
		Description: "Look up what a domain type is and, if it has one, which tool creates it — Agency, Goal, Objective, Role, RACI, Value, Rule, Workflow, WorkItem, EventFlowStep, Draft, Department, Level, AgencyPublication, AgencySnapshot, Issue. Use this to answer an operator's \"what is an Objective\" or \"how do I add a Role\" question rather than guessing. Omit name to list every type; name does a case-insensitive substring match.",
	}, mcpListTypes())
}

type mcpListTypesParams struct {
	Name string `json:"name,omitempty" jsonschema:"filter: case-insensitive substring match on the type's name, e.g. \"objective\"; omit to list every domain type"`
}

func mcpListTypes() func(context.Context, *mcp.CallToolRequest, mcpListTypesParams) (*mcp.CallToolResult, ListResult[typeInfo], error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, p mcpListTypesParams) (*mcp.CallToolResult, ListResult[typeInfo], error) {
		items := domainTypeCatalog
		if p.Name != "" {
			items = nil
			for _, t := range domainTypeCatalog {
				if strings.Contains(strings.ToLower(t.Name), strings.ToLower(p.Name)) {
					items = append(items, t)
				}
			}
		}
		return summary("Found %d domain type(s)", len(items)), ListResult[typeInfo]{Items: items}, nil
	}
}
