package mwanachamaagency

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

// SWELabPersonas is the full persona catalog from
// NeoLabHQ/context-engineering-kit's agents/ directory — the seed source
// for SWELab's Roles. Every persona here is software-development-specific.
var SWELabPersonas = []string{
	"bug-hunter", "business-analyst", "change-expectation-agent",
	"change-failure-agent", "change-impact-agent", "change-story-agent",
	"code-explorer", "code-quality-reviewer", "code-reviewer",
	"contracts-reviewer", "developer", "fpf-agent",
	"historical-context-reviewer", "judge", "meta-judge", "researcher",
	"security-auditor", "software-architect", "tech-lead", "tech-writer",
	"test-coverage-reviewer",
}

// SWELabSeed is everything [SeedSWELab] creates, so a caller doesn't
// need to re-query it.
type SWELabSeed struct {
	Agency          models.Agency
	GroupType       models.GroupType
	Roles           map[string]models.Role   // keyed by persona name
	Modules         map[string]models.Module // keyed by module code
	Goals           []models.Goal
	Objectives      []models.Objective // index-aligned with Goals — this seed creates one Objective per Goal, not a model requirement
	Workflows       []models.Workflow
	WorkItems       []models.WorkItem
	RoleAssignments []models.RoleAssignment
}

// SeedSWELab creates a test Agency ("SWELab"): all 21
// [SWELabPersonas] as Roles, one "Division" Level and one "Department"
// GroupType (Role itself stays unscoped — the GroupType exists for real,
// but nothing points at it), plus a small illustrative Goal/Objective/Workflow/
// WorkItem set with RoleAssignments exercising every RACI label. Three
// separate Goal/Objective pairs, one per illustrative WorkItem — RACI
// moved to Objective-scoped 2026-09-10 (AGD-001), so three distinct RACI
// sets need three distinct Objectives, not three WorkItems sharing one.
// Also collects every models.ModuleCatalog entry into seed.Modules
// (already live from gormstore.BackfillModules, not newly created here).
// Everything is created live; publishing is left to the caller.
func SeedSWELab(ctx context.Context, m AgencyManager) (SWELabSeed, error) {
	var seed SWELabSeed

	agency, err := m.CreateAgency(ctx, models.Agency{
		Name: "SWELab",
	})
	if err != nil {
		return SWELabSeed{}, fmt.Errorf("SeedSWELab: %w", err)
	}
	seed.Agency = agency

	level, err := m.CreateLevel(ctx, models.Level{Name: "Division", Depth: 0})
	if err != nil {
		return SWELabSeed{}, fmt.Errorf("SeedSWELab: %w", err)
	}
	groupType, err := m.CreateGroupType(ctx, models.GroupType{Name: "Department", LevelID: level.ID})
	if err != nil {
		return SWELabSeed{}, fmt.Errorf("SeedSWELab: %w", err)
	}
	seed.GroupType = groupType

	seed.Roles = make(map[string]models.Role, len(SWELabPersonas))
	for _, name := range SWELabPersonas {
		role, err := m.CreateRole(ctx, models.Role{Name: name})
		if err != nil {
			return SWELabSeed{}, fmt.Errorf("SeedSWELab: create role %q: %w", name, err)
		}
		seed.Roles[name] = role
	}

	// Every code in models.ModuleCatalog already has a live row by the
	// time this runs — gormstore.BackfillModules seeds them unconditionally
	// at the end of every Migrate, before any Agency exists. Read those
	// back rather than calling CreateModule again, which would collide with
	// ErrDuplicateModule; CreateModule is still the fallback for a code
	// Migrate somehow didn't backfill, using that same catalog entry's
	// Name/Description/Enabled/Locked rather than a blanket Enabled: true.
	liveModules, err := m.ListModules(ctx, "", ModuleFilter{})
	if err != nil {
		return SWELabSeed{}, fmt.Errorf("SeedSWELab: %w", err)
	}
	byCode := make(map[string]models.Module, len(liveModules))
	for _, mod := range liveModules {
		byCode[mod.Code] = mod
	}
	seed.Modules = make(map[string]models.Module, len(models.ModuleCatalog))
	for _, entry := range models.ModuleCatalog {
		if mod, ok := byCode[entry.Code]; ok {
			seed.Modules[entry.Code] = mod
			continue
		}
		mod, err := m.CreateModule(ctx, models.Module{
			Code: entry.Code, Name: entry.Name, Description: entry.Description,
			Enabled: entry.Enabled, Locked: entry.Locked,
		})
		if err != nil {
			return SWELabSeed{}, fmt.Errorf("SeedSWELab: create module %q: %w", entry.Code, err)
		}
		seed.Modules[entry.Code] = mod
	}

	workflow, err := m.CreateWorkflow(ctx, models.Workflow{Name: "Code Review Process"})
	if err != nil {
		return SWELabSeed{}, fmt.Errorf("SeedSWELab: %w", err)
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
			return SWELabSeed{}, fmt.Errorf("SeedSWELab: create goal %q: %w", item.goalTitle, err)
		}
		seed.Goals = append(seed.Goals, goal)

		objective, err := m.CreateObjective(ctx, models.Objective{GoalID: goal.ID, Title: item.goalTitle})
		if err != nil {
			return SWELabSeed{}, fmt.Errorf("SeedSWELab: create objective for %q: %w", item.goalTitle, err)
		}
		seed.Objectives = append(seed.Objectives, objective)

		workItem, err := m.CreateWorkItem(ctx, models.WorkItem{
			WorkflowID: workflow.ID, ObjectiveID: objective.ID, Title: item.workItem,
		})
		if err != nil {
			return SWELabSeed{}, fmt.Errorf("SeedSWELab: create work item %q: %w", item.workItem, err)
		}
		seed.WorkItems = append(seed.WorkItems, workItem)

		for _, a := range item.assignments {
			role, ok := seed.Roles[a.persona]
			if !ok {
				return SWELabSeed{}, fmt.Errorf("SeedSWELab: %q is not a seeded persona", a.persona)
			}
			ra, err := m.CreateRoleAssignment(ctx, models.RoleAssignment{
				RoleID: role.ID, ObjectiveID: objective.ID, Label: a.label,
			})
			if err != nil {
				return SWELabSeed{}, fmt.Errorf("SeedSWELab: assign %q to %q: %w", a.persona, item.workItem, err)
			}
			seed.RoleAssignments = append(seed.RoleAssignments, ra)
		}
	}

	return seed, nil
}
