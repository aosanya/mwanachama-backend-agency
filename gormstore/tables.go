package gormstore

import "gorm.io/gorm"

// TableNames names the physical tables one mounted copy of this package
// reads and writes.
type TableNames struct {
	Agencies           string
	Drafts             string
	Goals              string
	Objectives         string
	Workflows          string
	WorkItems          string
	EventFlowSteps     string
	AgencySnapshots    string
	AgencyPublications string
	WorkPlans          string
	ContextSources     string
	Departments        string
	Levels             string
	Roles              string
	RoleAssignments    string
	Values             string
	Rules              string
}

// DefaultTableNames builds the conventional table set for one mounted
// instance of this package.
func DefaultTableNames(instance string) TableNames {
	return TableNames{
		Agencies:           instance + "_agencies",
		Drafts:             instance + "_drafts",
		Goals:              instance + "_goals",
		Objectives:         instance + "_objectives",
		Workflows:          instance + "_workflows",
		WorkItems:          instance + "_work_items",
		EventFlowSteps:     instance + "_event_flow_steps",
		AgencySnapshots:    instance + "_agency_snapshots",
		AgencyPublications: instance + "_agency_publications",
		WorkPlans:          instance + "_work_plans",
		ContextSources:     instance + "_context_sources",
		Departments:        instance + "_departments",
		Levels:             instance + "_levels",
		Roles:              instance + "_roles",
		RoleAssignments:    instance + "_role_assignments",
		Values:             instance + "_values",
		Rules:              instance + "_rules",
	}
}

// Migrate creates or updates the tables named by t.
func Migrate(db *gorm.DB, t TableNames) error {
	if err := db.Table(t.Agencies).AutoMigrate(&AgencyRow{}); err != nil {
		return err
	}
	if err := db.Table(t.Drafts).AutoMigrate(&DraftRow{}); err != nil {
		return err
	}
	if err := db.Table(t.Goals).AutoMigrate(&GoalRow{}); err != nil {
		return err
	}
	if err := db.Table(t.Objectives).AutoMigrate(&ObjectiveRow{}); err != nil {
		return err
	}
	if err := db.Table(t.Workflows).AutoMigrate(&WorkflowRow{}); err != nil {
		return err
	}
	if err := db.Table(t.WorkItems).AutoMigrate(&WorkItemRow{}); err != nil {
		return err
	}
	if err := db.Table(t.EventFlowSteps).AutoMigrate(&EventFlowStepRow{}); err != nil {
		return err
	}
	if err := db.Table(t.AgencySnapshots).AutoMigrate(&AgencySnapshotRow{}); err != nil {
		return err
	}
	if err := db.Table(t.AgencyPublications).AutoMigrate(&AgencyPublicationRow{}); err != nil {
		return err
	}
	if err := db.Table(t.WorkPlans).AutoMigrate(&WorkPlanRow{}); err != nil {
		return err
	}
	if err := db.Table(t.ContextSources).AutoMigrate(&ContextSourceRow{}); err != nil {
		return err
	}
	if err := db.Table(t.Levels).AutoMigrate(&LevelRow{}); err != nil {
		return err
	}
	if err := db.Table(t.Departments).AutoMigrate(&DepartmentRow{}); err != nil {
		return err
	}
	if err := db.Table(t.Roles).AutoMigrate(&RoleRow{}); err != nil {
		return err
	}
	if err := db.Table(t.RoleAssignments).AutoMigrate(&RoleAssignmentRow{}); err != nil {
		return err
	}
	if err := db.Table(t.Values).AutoMigrate(&ValueRow{}); err != nil {
		return err
	}
	if err := db.Table(t.Rules).AutoMigrate(&RuleRow{}); err != nil {
		return err
	}
	return nil
}
