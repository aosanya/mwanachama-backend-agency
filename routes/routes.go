package routes

import (
	"net/http"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

// Route is one address this package answers, relative to wherever the
// mounting process prefixes it.
type Route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
}

// Pattern returns the http.ServeMux registration pattern for this route
// once mounted under prefix.
func (r Route) Pattern(prefix string) string {
	return r.Method + " " + prefix + r.Path
}

// ResourceNames configures the URL noun each type is addressed under.
// Every field defaults to this package's own noun for the type when left
// "" — a mounting process overrides only the ones its own domain renames.
type ResourceNames struct {
	Agency         string
	Goal           string
	Objective      string
	Workflow       string
	WorkItem       string
	Draft          string
	Publication    string
	EventFlowStep  string
	Department     string
	Level          string
	Role           string
	RoleAssignment string
	Module         string
	WorkPlan       string
	ContextSource  string
	Value          string
	Rule           string
	Issue          string
}

func (n ResourceNames) withDefaults() ResourceNames {
	if n.Agency == "" {
		n.Agency = "agency"
	}
	if n.Goal == "" {
		n.Goal = "goals"
	}
	if n.Objective == "" {
		n.Objective = "objectives"
	}
	if n.Workflow == "" {
		n.Workflow = "workflows"
	}
	if n.WorkItem == "" {
		n.WorkItem = "work-items"
	}
	if n.Draft == "" {
		n.Draft = "drafts"
	}
	if n.Publication == "" {
		n.Publication = "publications"
	}
	if n.EventFlowStep == "" {
		n.EventFlowStep = "event-flow-steps"
	}
	if n.Department == "" {
		n.Department = "departments"
	}
	if n.Level == "" {
		n.Level = "levels"
	}
	if n.Role == "" {
		n.Role = "roles"
	}
	if n.RoleAssignment == "" {
		n.RoleAssignment = "role-assignments"
	}
	if n.Module == "" {
		n.Module = "modules"
	}
	if n.WorkPlan == "" {
		n.WorkPlan = "work-plans"
	}
	if n.ContextSource == "" {
		n.ContextSource = "context-sources"
	}
	if n.Value == "" {
		n.Value = "values"
	}
	if n.Rule == "" {
		n.Rule = "rules"
	}
	if n.Issue == "" {
		n.Issue = "issues"
	}
	return n
}

// Routes is every address this package answers today, sharing one
// ResourceNames so all of it stays consistent (e.g. WorkPlan's noun
// override is honoured in ContextSourceRoutes' nested path too). A
// mounting process that wants to wrap one domain's routes differently
// from another's calls the per-domain *Routes functions separately
// instead.
func Routes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	var out []Route
	out = append(out, AgencyRoutes(m, names)...)
	out = append(out, GoalRoutes(m, names)...)
	out = append(out, ObjectiveRoutes(m, names)...)
	out = append(out, WorkflowRoutes(m, names)...)
	out = append(out, WorkItemRoutes(m, names)...)
	out = append(out, DraftRoutes(m, names)...)
	out = append(out, PublicationRoutes(m, names)...)
	out = append(out, EventFlowStepRoutes(m, names)...)
	out = append(out, DepartmentRoutes(m, names)...)
	out = append(out, LevelRoutes(m, names)...)
	out = append(out, RoleRoutes(m, names)...)
	out = append(out, RoleAssignmentRoutes(m, names)...)
	out = append(out, ModuleRoutes(m, names)...)
	out = append(out, WorkPlanRoutes(m, names)...)
	out = append(out, ContextSourceRoutes(m, names)...)
	out = append(out, ValueRoutes(m, names)...)
	out = append(out, RuleRoutes(m, names)...)
	out = append(out, IssueRoutes(m, names)...)
	return out
}
