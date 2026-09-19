package gormstore

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aosanya/mwanachama-backend-agency/models"
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

// draftScopedTables names every table whose DraftID column is non-nullable
// (AGD-006 — every row belongs to a real Draft). Driven from a list so
// [EnsureWorkingDraft]'s pre-migration backfill and the AutoMigrate calls
// below stay in lockstep with what actually needs one.
func draftScopedTables(t TableNames) []string {
	return []string{
		t.Goals, t.Objectives, t.Workflows, t.WorkItems, t.EventFlowSteps,
		t.Roles, t.RoleAssignments, t.Modules, t.ObjectiveModules,
		t.Values, t.Rules, t.Issues,
	}
}

// EnsureWorkingDraft guarantees at least one Draft row exists for this
// Instance and returns its id — the home for BackfillModules' catalog rows
// and the anchor [backfillLegacyDraftScope] reassigns any pre-AGD-006
// NULL-scoped row to. Returns the oldest open Draft if any Draft already
// exists (idempotent across repeated Migrate calls), or creates one
// ("Initial draft") if the Drafts table is empty — the bootstrap path a
// brand-new Instance needs, since [AgencyManager.CreateDraft] can no
// longer fork from a "live" row-scope that doesn't exist (AGD-006).
func EnsureWorkingDraft(db *gorm.DB, t TableNames) (string, error) {
	var existing DraftRow
	err := db.Table(t.Drafts).Where("status = ?", string(models.DraftStatusOpen)).
		Order("created_at").First(&existing).Error
	if err == nil {
		return existing.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", fmt.Errorf("EnsureWorkingDraft: %w", err)
	}
	now := models.NowRFC3339()
	row := DraftRow{
		Description: "Initial draft",
		Status:      string(models.DraftStatusOpen),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := db.Table(t.Drafts).Create(&row).Error; err != nil {
		return "", fmt.Errorf("EnsureWorkingDraft: %w", err)
	}
	return row.ID, nil
}

// backfillLegacyDraftScope reassigns every currently NULL/""-scoped row
// across every table [draftScopedTables] names to draftID, ahead of that
// table's own AutoMigrate call adding the NOT NULL constraint (AGD-006) —
// a live ALTER COLUMN ... SET NOT NULL fails outright while NULLs remain.
// A no-op wherever the table doesn't exist yet (a brand-new Instance has
// nothing to backfill; AutoMigrate creates the column NOT NULL from
// scratch) or already carries no such rows (every deployment migrated
// since AGD-006 shipped).
func backfillLegacyDraftScope(db *gorm.DB, t TableNames, draftID string) error {
	for _, table := range draftScopedTables(t) {
		if !db.Migrator().HasTable(table) {
			continue
		}
		if err := db.Table(table).Where("draft_id IS NULL OR draft_id = ?", "").
			Updates(map[string]any{"draft_id": draftID}).Error; err != nil {
			return fmt.Errorf("backfillLegacyDraftScope(%s): %w", table, err)
		}
	}
	return nil
}

// Migrate creates or updates the tables named by t, backfills any
// pre-AGD-006 NULL-scoped row into a real Draft, then backfills a Code
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

	workingDraftID, err := EnsureWorkingDraft(db, t)
	if err != nil {
		return err
	}
	if err := backfillLegacyDraftScope(db, t, workingDraftID); err != nil {
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
	if err := BackfillModules(db, t.Modules, workingDraftID); err != nil {
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

// DropRetiredTables removes the tables and columns of retired types for
// one instance, derived from the instance prefix TableNames itself is
// built from. Runs at the end of every [Migrate], so a pre-existing
// deployment sheds them on its next boot; a no-op everywhere else.
func DropRetiredTables(db *gorm.DB, t TableNames) error {
	instance := strings.TrimSuffix(t.Agencies, "_agencies")
	for _, suffix := range retiredTableSuffixes {
		if err := db.Migrator().DropTable(instance + suffix); err != nil {
			return err
		}
	}
	// AgencyPublication.Status became Channel (AGD-005 #3). AutoMigrate
	// adds the new column but never drops the old one, which would
	// otherwise sit there holding a stale "active" nothing reads. Every
	// pre-AGD-005 publication lands on ChannelArchived, which is what
	// AGD-005 says they become: history, with no document to serve.
	publications := db.Table(t.AgencyPublications)
	if publications.Migrator().HasColumn(&AgencyPublicationRow{}, "status") {
		if err := publications.Migrator().DropColumn(&AgencyPublicationRow{}, "status"); err != nil {
			return err
		}
	}
	return nil
}
