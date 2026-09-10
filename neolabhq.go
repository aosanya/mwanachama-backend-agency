package mwanachamaagency

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

// NeoLabHQPersonas is the full persona catalog from
// NeoLabHQ/context-engineering-kit's agents/ directory — the seed source
// for NeoLabHQ's Roles. Every persona here is software-development-specific.
var NeoLabHQPersonas = []string{
	"bug-hunter", "business-analyst", "change-expectation-agent",
	"change-failure-agent", "change-impact-agent", "change-story-agent",
	"code-explorer", "code-quality-reviewer", "code-reviewer",
	"contracts-reviewer", "developer", "fpf-agent",
	"historical-context-reviewer", "judge", "meta-judge", "researcher",
	"security-auditor", "software-architect", "tech-lead", "tech-writer",
	"test-coverage-reviewer",
}

// NeoLabHQSeed is everything [SeedNeoLabHQ] creates, so a caller doesn't
// need to re-query it.
type NeoLabHQSeed struct {
	Agency          models.Agency
	Department      models.Department
	Roles           map[string]models.Role // keyed by persona name
	Goals           []models.Goal
	Objectives      []models.Objective // index-aligned with Goals — this seed creates one Objective per Goal, not a model requirement
	Workflows       []models.Workflow
	WorkItems       []models.WorkItem
	RoleAssignments []models.RoleAssignment
}

// SeedNeoLabHQ creates a test Agency ("NeoLabHQ"): all 21
// [NeoLabHQPersonas] as Roles under one "Software" Department (Role
// itself stays unscoped — the Department exists for real, but nothing
// points at it), plus a small illustrative Goal/Objective/Workflow/
// WorkItem set with RoleAssignments exercising every RACI label. Three
// separate Goal/Objective pairs, one per illustrative WorkItem — RACI
// moved to Objective-scoped 2026-09-10 (AGD-001), so three distinct RACI
// sets need three distinct Objectives, not three WorkItems sharing one.
// Everything is created live; publishing is left to the caller.
func SeedNeoLabHQ(ctx context.Context, m AgencyManager) (NeoLabHQSeed, error) {
	var seed NeoLabHQSeed

	agency, err := m.CreateAgency(ctx, models.Agency{
		Name: "NeoLabHQ",
	})
	if err != nil {
		return NeoLabHQSeed{}, fmt.Errorf("SeedNeoLabHQ: %w", err)
	}
	seed.Agency = agency

	level, err := m.CreateLevel(ctx, models.Level{Name: "Department", Depth: 0})
	if err != nil {
		return NeoLabHQSeed{}, fmt.Errorf("SeedNeoLabHQ: %w", err)
	}
	department, err := m.CreateDepartment(ctx, models.Department{Name: "Software", LevelID: level.ID})
	if err != nil {
		return NeoLabHQSeed{}, fmt.Errorf("SeedNeoLabHQ: %w", err)
	}
	seed.Department = department

	seed.Roles = make(map[string]models.Role, len(NeoLabHQPersonas))
	for _, name := range NeoLabHQPersonas {
		role, err := m.CreateRole(ctx, models.Role{Name: name})
		if err != nil {
			return NeoLabHQSeed{}, fmt.Errorf("SeedNeoLabHQ: create role %q: %w", name, err)
		}
		seed.Roles[name] = role
	}

	workflow, err := m.CreateWorkflow(ctx, models.Workflow{Name: "Code Review Process"})
	if err != nil {
		return NeoLabHQSeed{}, fmt.Errorf("SeedNeoLabHQ: %w", err)
	}
	seed.Workflows = []models.Workflow{workflow}

	type assignment struct {
		persona string
		label   models.RACILabel
	}
	items := []struct {
		goalTitle   string
		workItem    string
		assignments []assignment
	}{
		{
			goalTitle: "Ship reliable software",
			workItem:  "Review pull request",
			assignments: []assignment{
				{"code-reviewer", models.RACILabelResponsible},
				{"tech-lead", models.RACILabelAccountable},
				{"security-auditor", models.RACILabelConsulted},
				{"developer", models.RACILabelInformed},
			},
		},
		{
			goalTitle: "Resolve reported defects",
			workItem:  "Triage a bug report",
			assignments: []assignment{
				{"bug-hunter", models.RACILabelResponsible},
				{"tech-lead", models.RACILabelAccountable},
			},
		},
		{
			goalTitle: "Keep the architecture coherent",
			workItem:  "Approve architecture change",
			assignments: []assignment{
				{"software-architect", models.RACILabelResponsible},
				{"tech-lead", models.RACILabelAccountable},
				{"contracts-reviewer", models.RACILabelConsulted},
			},
		},
	}

	for _, item := range items {
		goal, err := m.CreateGoal(ctx, models.Goal{Title: item.goalTitle})
		if err != nil {
			return NeoLabHQSeed{}, fmt.Errorf("SeedNeoLabHQ: create goal %q: %w", item.goalTitle, err)
		}
		seed.Goals = append(seed.Goals, goal)

		objective, err := m.CreateObjective(ctx, models.Objective{GoalID: goal.ID, Title: item.goalTitle})
		if err != nil {
			return NeoLabHQSeed{}, fmt.Errorf("SeedNeoLabHQ: create objective for %q: %w", item.goalTitle, err)
		}
		seed.Objectives = append(seed.Objectives, objective)

		workItem, err := m.CreateWorkItem(ctx, models.WorkItem{
			WorkflowID: workflow.ID, ObjectiveID: objective.ID, Title: item.workItem,
		})
		if err != nil {
			return NeoLabHQSeed{}, fmt.Errorf("SeedNeoLabHQ: create work item %q: %w", item.workItem, err)
		}
		seed.WorkItems = append(seed.WorkItems, workItem)

		for _, a := range item.assignments {
			role, ok := seed.Roles[a.persona]
			if !ok {
				return NeoLabHQSeed{}, fmt.Errorf("SeedNeoLabHQ: %q is not a seeded persona", a.persona)
			}
			ra, err := m.CreateRoleAssignment(ctx, models.RoleAssignment{
				RoleID: role.ID, ObjectiveID: objective.ID, Label: a.label,
			})
			if err != nil {
				return NeoLabHQSeed{}, fmt.Errorf("SeedNeoLabHQ: assign %q to %q: %w", a.persona, item.workItem, err)
			}
			seed.RoleAssignments = append(seed.RoleAssignments, ra)
		}
	}

	return seed, nil
}
