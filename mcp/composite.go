// composite.go is this package's higher-level MCP surface: one tool
// composing the granular ones above, for an agent handed a rough brief
// rather than one that wants to place each department/goal/RACI assignment
// with its own call.
//
// The spec's entities reference each other by a caller-chosen "key" (e.g.
// a department's level_key, a work item's workflow_key) rather than a real
// id, since real ids don't exist until this tool creates the rows — the
// same problem a UI form composing several backend calls in one submit has
// to solve. Departments/RoleAssignments/etc. that reference a key must
// have it defined earlier in the same spec (a department's parent_key, for
// instance, must name a department listed before it) — this tool applies
// each section in array order, in the dependency order the domain itself
// imposes (levels before departments, roles/goals/workflows before work
// items, work items before RACI), and does not reorder or validate a
// spec's internal ordering beyond that.
//
// Scoped to live creation only — no Draft support. One entity failing does
// not abort the rest: everything created so far, plus every failure with
// which spec entry caused it, comes back in the result — there is no
// cross-entity transaction in mwanachama-backend-agency to roll back to
// anyway.
package mcp

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func registerCompositeTools(server *mcp.Server, resolve ManagerResolver) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "agency_design_from_spec",
		Description: "Build out an Agency's org chart, goals/objectives/workflows/work items, and RACI assignments from one structured spec, composing the granular agency_* tools internally. " +
			"Entities reference each other by a caller-chosen \"key\" (not a real id); a reference must name a key defined earlier in that section or in a section this tool applies first. " +
			"Order applied: agency (only if none exists yet), levels, departments (parent_key must precede its child), roles, goals, objectives (goal_key), workflows, work_items (objective_key), raci (objective_key). " +
			"One entity failing does not stop the rest — check the result's errors field.",
	}, mcpDesignAgencyFromSpec(resolve))
}

type specAgency struct {
	Name string `json:"name,omitempty" jsonschema:"the Agency's name"`
}

type specLevel struct {
	Key   string `json:"key" jsonschema:"a caller-chosen name other spec entries reference this Level by"`
	Name  string `json:"name" jsonschema:"e.g. \"Division\", \"Team\""`
	Depth int    `json:"depth,omitempty"`
}

type specDepartment struct {
	Key       string `json:"key" jsonschema:"a caller-chosen name other spec entries reference this Department by"`
	Name      string `json:"name"`
	LevelKey  string `json:"level_key" jsonschema:"a key from this spec's levels list"`
	ParentKey string `json:"parent_key,omitempty" jsonschema:"a key from earlier in this spec's departments list; empty makes it top-level"`
}

type specRole struct {
	Key         string `json:"key" jsonschema:"a caller-chosen name other spec entries reference this Role by"`
	Name        string `json:"name" jsonschema:"e.g. \"bug-hunter\""`
	Description string `json:"description,omitempty"`
}

type specGoal struct {
	Key         string `json:"key" jsonschema:"a caller-chosen name other spec entries reference this Goal by"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type specObjective struct {
	Key         string `json:"key" jsonschema:"a caller-chosen name other spec entries reference this Objective by"`
	GoalKey     string `json:"goal_key" jsonschema:"a key from this spec's goals list — a Goal may have several Objectives"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type specWorkflow struct {
	Key         string `json:"key" jsonschema:"a caller-chosen name other spec entries reference this Workflow by"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type specWorkItem struct {
	Key          string `json:"key" jsonschema:"a caller-chosen name other spec entries reference this WorkItem by"`
	WorkflowKey  string `json:"workflow_key" jsonschema:"a key from this spec's workflows list"`
	ObjectiveKey string `json:"objective_key" jsonschema:"a key from this spec's objectives list — every WorkItem belongs to exactly one Objective"`
	Title        string `json:"title"`
	Description  string `json:"description,omitempty"`
	Prompt       string `json:"prompt,omitempty" jsonschema:"instructions for whoever/whatever executes this WorkItem"`
}

type specRACI struct {
	RoleKey      string `json:"role_key" jsonschema:"a key from this spec's roles list"`
	ObjectiveKey string `json:"objective_key" jsonschema:"a key from this spec's objectives list"`
	Label        string `json:"label" jsonschema:"responsible, accountable, consulted, or informed"`
}

type mcpDesignFromSpecParams struct {
	InstanceID  string           `json:"instance_id" jsonschema:"which Agency this operates on, as resolved by the mounting host"`
	Agency      *specAgency      `json:"agency,omitempty" jsonschema:"created only if this Instance has no Agency yet; omit or leave unset otherwise"`
	Levels      []specLevel      `json:"levels,omitempty"`
	Departments []specDepartment `json:"departments,omitempty"`
	Roles       []specRole       `json:"roles,omitempty"`
	Goals       []specGoal       `json:"goals,omitempty"`
	Objectives  []specObjective  `json:"objectives,omitempty"`
	Workflows   []specWorkflow   `json:"workflows,omitempty"`
	WorkItems   []specWorkItem   `json:"work_items,omitempty"`
	RACI        []specRACI       `json:"raci,omitempty"`
}

// mcpDesignFromSpecResult reports every key this call actually resolved to
// a real id, plus every failure — never a Go error for a partial
// application, since a caller needs to see what succeeded even when
// something later in the spec did not (see the package doc comment).
type mcpDesignFromSpecResult struct {
	AgencyCreated   bool              `json:"agency_created"`
	Levels          map[string]string `json:"levels,omitempty"`
	Departments     map[string]string `json:"departments,omitempty"`
	Roles           map[string]string `json:"roles,omitempty"`
	Goals           map[string]string `json:"goals,omitempty"`
	Objectives      map[string]string `json:"objectives,omitempty"`
	Workflows       map[string]string `json:"workflows,omitempty"`
	WorkItems       map[string]string `json:"work_items,omitempty"`
	RoleAssignments []string          `json:"role_assignments,omitempty"`
	Errors          []string          `json:"errors,omitempty"`
}

func mcpDesignAgencyFromSpec(resolve ManagerResolver) func(context.Context, *mcp.CallToolRequest, mcpDesignFromSpecParams) (*mcp.CallToolResult, mcpDesignFromSpecResult, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, p mcpDesignFromSpecParams) (*mcp.CallToolResult, mcpDesignFromSpecResult, error) {
		m, err := resolve(ctx, p.InstanceID)
		if err != nil {
			return nil, mcpDesignFromSpecResult{}, err
		}

		out := mcpDesignFromSpecResult{
			Levels:      map[string]string{},
			Departments: map[string]string{},
			Roles:       map[string]string{},
			Goals:       map[string]string{},
			Objectives:  map[string]string{},
			Workflows:   map[string]string{},
		}

		if p.Agency != nil {
			if _, err := m.GetAgency(ctx); errors.Is(err, mwanachamaagency.ErrAgencyNotFound) {
				if _, err := m.CreateAgency(ctx, models.Agency{Name: p.Agency.Name}); err != nil {
					out.Errors = append(out.Errors, "agency: "+err.Error())
				} else {
					out.AgencyCreated = true
				}
			}
		}

		for _, lvl := range p.Levels {
			row, err := m.CreateLevel(ctx, models.Level{Name: lvl.Name, Depth: lvl.Depth})
			if err != nil {
				out.Errors = append(out.Errors, fmt.Sprintf("level %q: %v", lvl.Key, err))
				continue
			}
			out.Levels[lvl.Key] = row.ID
		}

		for _, dep := range p.Departments {
			levelID, ok := out.Levels[dep.LevelKey]
			if !ok {
				out.Errors = append(out.Errors, fmt.Sprintf("department %q: unknown level_key %q", dep.Key, dep.LevelKey))
				continue
			}
			parentID := ""
			if dep.ParentKey != "" {
				parentID, ok = out.Departments[dep.ParentKey]
				if !ok {
					out.Errors = append(out.Errors, fmt.Sprintf("department %q: unknown parent_key %q (must appear earlier in departments)", dep.Key, dep.ParentKey))
					continue
				}
			}
			row, err := m.CreateDepartment(ctx, models.Department{ParentID: parentID, LevelID: levelID, Name: dep.Name})
			if err != nil {
				out.Errors = append(out.Errors, fmt.Sprintf("department %q: %v", dep.Key, err))
				continue
			}
			out.Departments[dep.Key] = row.ID
		}

		for _, r := range p.Roles {
			row, err := m.CreateRole(ctx, models.Role{Name: r.Name, Description: r.Description})
			if err != nil {
				out.Errors = append(out.Errors, fmt.Sprintf("role %q: %v", r.Key, err))
				continue
			}
			out.Roles[r.Key] = row.ID
		}

		for _, g := range p.Goals {
			row, err := m.CreateGoal(ctx, models.Goal{Title: g.Title, Description: g.Description})
			if err != nil {
				out.Errors = append(out.Errors, fmt.Sprintf("goal %q: %v", g.Key, err))
				continue
			}
			out.Goals[g.Key] = row.ID
		}

		for _, o := range p.Objectives {
			goalID, ok := out.Goals[o.GoalKey]
			if !ok {
				out.Errors = append(out.Errors, fmt.Sprintf("objective %q: unknown goal_key %q", o.Key, o.GoalKey))
				continue
			}
			row, err := m.CreateObjective(ctx, models.Objective{GoalID: goalID, Title: o.Title, Description: o.Description})
			if err != nil {
				out.Errors = append(out.Errors, fmt.Sprintf("objective %q: %v", o.Key, err))
				continue
			}
			out.Objectives[o.Key] = row.ID
		}

		for _, w := range p.Workflows {
			row, err := m.CreateWorkflow(ctx, models.Workflow{Name: w.Name, Description: w.Description})
			if err != nil {
				out.Errors = append(out.Errors, fmt.Sprintf("workflow %q: %v", w.Key, err))
				continue
			}
			out.Workflows[w.Key] = row.ID
		}

		workItemIDs := map[string]string{}
		for _, wi := range p.WorkItems {
			workflowID, ok := out.Workflows[wi.WorkflowKey]
			if !ok {
				out.Errors = append(out.Errors, fmt.Sprintf("work_item %q: unknown workflow_key %q", wi.Key, wi.WorkflowKey))
				continue
			}
			objectiveID, ok := out.Objectives[wi.ObjectiveKey]
			if !ok {
				out.Errors = append(out.Errors, fmt.Sprintf("work_item %q: unknown objective_key %q", wi.Key, wi.ObjectiveKey))
				continue
			}
			row, err := m.CreateWorkItem(ctx, models.WorkItem{WorkflowID: workflowID, ObjectiveID: objectiveID, Title: wi.Title, Description: wi.Description, Prompt: wi.Prompt})
			if err != nil {
				out.Errors = append(out.Errors, fmt.Sprintf("work_item %q: %v", wi.Key, err))
				continue
			}
			workItemIDs[wi.Key] = row.ID
		}
		if len(workItemIDs) > 0 {
			out.WorkItems = workItemIDs
		}

		for i, ra := range p.RACI {
			roleID, ok := out.Roles[ra.RoleKey]
			if !ok {
				out.Errors = append(out.Errors, fmt.Sprintf("raci[%d]: unknown role_key %q", i, ra.RoleKey))
				continue
			}
			objectiveID, ok := out.Objectives[ra.ObjectiveKey]
			if !ok {
				out.Errors = append(out.Errors, fmt.Sprintf("raci[%d]: unknown objective_key %q", i, ra.ObjectiveKey))
				continue
			}
			row, err := m.CreateRoleAssignment(ctx, models.RoleAssignment{RoleID: roleID, ObjectiveID: objectiveID, Label: models.RACILabel(ra.Label)})
			if err != nil {
				out.Errors = append(out.Errors, fmt.Sprintf("raci[%d]: %v", i, err))
				continue
			}
			out.RoleAssignments = append(out.RoleAssignments, row.ID)
		}

		return nil, out, nil
	}
}
