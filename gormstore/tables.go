package gormstore

import (
	"strings"

	"gorm.io/gorm"
)

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
	Departments        string
	Levels             string
	Roles              string
	RoleAssignments    string
	Modules            string
	ObjectiveModules   string
	Values             string
	ValueAttachments   string
	Rules              string
	RuleAttachments    string
	Issues             string
	// CodeSequences backs [NextCode]'s per-entity-type counters, minting
	// the stable "<prefix>-<n>" Code every named business-entity type
	// above carries.
	CodeSequences string
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
		Departments:        instance + "_departments",
		Levels:             instance + "_levels",
		Roles:              instance + "_roles",
		RoleAssignments:    instance + "_role_assignments",
		Modules:            instance + "_modules",
		ObjectiveModules:   instance + "_objective_modules",
		Values:             instance + "_values",
		ValueAttachments:   instance + "_value_attachments",
		Rules:              instance + "_rules",
		RuleAttachments:    instance + "_rule_attachments",
		Issues:             instance + "_issues",
		CodeSequences:      instance + "_code_sequences",
	}
}

// Migrate creates or updates the tables named by t, then backfills a Code
// onto any pre-existing row of a named business-entity type that predates
// it (see [BackfillCodes]).
func Migrate(db *gorm.DB, t TableNames) error {
	if err := db.Table(t.CodeSequences).AutoMigrate(&CodeSequenceRow{}); err != nil {
		return err
	}
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
	if err := db.Table(t.Modules).AutoMigrate(&ModuleRow{}); err != nil {
		return err
	}
	if err := db.Table(t.ObjectiveModules).AutoMigrate(&ObjectiveModuleRow{}); err != nil {
		return err
	}
	if err := db.Table(t.Values).AutoMigrate(&ValueRow{}); err != nil {
		return err
	}
	if err := db.Table(t.ValueAttachments).AutoMigrate(&ValueAttachmentRow{}); err != nil {
		return err
	}
	if err := db.Table(t.Rules).AutoMigrate(&RuleRow{}); err != nil {
		return err
	}
	if err := db.Table(t.RuleAttachments).AutoMigrate(&RuleAttachmentRow{}); err != nil {
		return err
	}
	if err := db.Table(t.Issues).AutoMigrate(&IssueRow{}); err != nil {
		return err
	}

	for _, b := range []struct {
		table, entityType, prefix string
	}{
		{t.Goals, "goal", "G"},
		{t.Objectives, "objective", "O"},
		{t.Roles, "role", "R"},
		{t.Departments, "department", "D"},
		{t.Levels, "level", "L"},
		{t.Workflows, "workflow", "W"},
		{t.WorkItems, "workitem", "WI"},
		{t.Values, "value", "V"},
		{t.Rules, "rule", "RU"},
	} {
		if err := BackfillCodes(db, b.table, t.CodeSequences, b.entityType, b.prefix); err != nil {
			return err
		}
	}

	// Module.Code is caller-supplied, not minted via BackfillCodes/NextCode
	// (AGD-002) — BackfillModules is its own idempotent seed-if-missing
	// step instead, covering every deployment (fresh or pre-existing) so a
	// gateway gating routes on Module.Enabled never finds an empty catalog.
	if err := BackfillModules(db, t.Modules); err != nil {
		return err
	}
	return DropRetiredTables(db, t)
}

// retiredTableSuffixes names table suffixes this package used to create
// and no longer does. Dropping is driven from a list rather than done
// inline so a future retirement is one entry, not another block here.
var retiredTableSuffixes = []string{
	// WorkPlan/ContextSource, retired 2026-09-16 (AGD-005 decision #6) —
	// event-topic dispatch config nothing consumed: no external caller of
	// MatchWorkPlans, and mwanachama-wakala-studio had already hidden the
	// module from its own shortcuts.
	"_work_plans",
	"_context_sources",
}

// DropRetiredTables removes the tables of retired types for one instance,
// derived from the instance prefix TableNames itself is built from. Runs
// at the end of every [Migrate], so a pre-existing deployment sheds them
// on its next boot; a no-op everywhere else.
func DropRetiredTables(db *gorm.DB, t TableNames) error {
	instance := strings.TrimSuffix(t.Agencies, "_agencies")
	for _, suffix := range retiredTableSuffixes {
		if err := db.Migrator().DropTable(instance + suffix); err != nil {
			return err
		}
	}
	return nil
}
