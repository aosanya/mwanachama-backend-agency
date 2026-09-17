package mwanachamaagency

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// buildDocument serializes one scope into a [models.AgencyDocument] —
// draftID "" for the working set, a Draft's id for that Draft. This is
// what PublishAgency freezes onto a tag, and what production then reads
// instead of querying rows at all (AGD-005).
//
// Attachments are the one section not read through scopeQuery: they are
// never Draft-scoped (AGD-003), so every scope's document carries the same
// attachment rows, resolved against that scope's own Codes by whoever
// reads it.
func buildDocument(ctx context.Context, tx *gorm.DB, tables TableNames, draftID string) (models.AgencyDocument, error) {
	var doc models.AgencyDocument
	scoped := func(table string) *gorm.DB {
		return scopeQuery(tx.WithContext(ctx).Table(table), draftID)
	}
	fail := func(err error) (models.AgencyDocument, error) {
		return models.AgencyDocument{}, fmt.Errorf("buildDocument: %w", err)
	}

	var agencyRow gormstore.AgencyRow
	if err := tx.WithContext(ctx).Table(tables.Agencies).Order("created_at").First(&agencyRow).Error; err != nil {
		return fail(err)
	}
	a := gormstore.AgencyFromRow(agencyRow)
	doc.Agency = models.DocumentAgency{
		Name: a.Name, Description: a.Description, Enabled: a.Enabled,
		DefaultFailurePipelineBudget: a.DefaultFailurePipelineBudget,
		InactivityTimeoutSeconds:     a.InactivityTimeoutSeconds,
		EventFlows:                   a.EventFlows,
	}

	var levelRows []gormstore.LevelRow
	if err := tx.WithContext(ctx).Table(tables.Levels).Order("depth").Find(&levelRows).Error; err != nil {
		return fail(err)
	}
	for _, r := range levelRows {
		l := gormstore.LevelFromRow(r)
		doc.Levels = append(doc.Levels, models.DocumentLevel{ID: l.ID, Code: l.Code, Name: l.Name, Depth: l.Depth})
	}

	var departmentRows []gormstore.DepartmentRow
	if err := tx.WithContext(ctx).Table(tables.Departments).Order("created_at").Find(&departmentRows).Error; err != nil {
		return fail(err)
	}
	for _, r := range departmentRows {
		d := gormstore.DepartmentFromRow(r)
		doc.Departments = append(doc.Departments, models.DocumentDepartment{
			ID: d.ID, Code: d.Code, ParentID: d.ParentID, LevelID: d.LevelID, Name: d.Name,
		})
	}

	var roleRows []gormstore.RoleRow
	if err := scoped(tables.Roles).Order("created_at").Find(&roleRows).Error; err != nil {
		return fail(err)
	}
	for _, r := range roleRows {
		role := gormstore.RoleFromRow(r)
		doc.Roles = append(doc.Roles, models.DocumentRole{
			ID: role.ID, Code: role.Code, Name: role.Name, Description: role.Description,
		})
	}

	var goalRows []gormstore.GoalRow
	if err := scoped(tables.Goals).Order("ordinality").Find(&goalRows).Error; err != nil {
		return fail(err)
	}
	for _, r := range goalRows {
		g := gormstore.GoalFromRow(r)
		doc.Goals = append(doc.Goals, models.DocumentGoal{
			ID: g.ID, Code: g.Code, Title: g.Title, Description: g.Description, Ordinality: g.Ordinality,
		})
	}

	var objectiveRows []gormstore.ObjectiveRow
	if err := scoped(tables.Objectives).Order("ordinality").Find(&objectiveRows).Error; err != nil {
		return fail(err)
	}
	for _, r := range objectiveRows {
		o := gormstore.ObjectiveFromRow(r)
		doc.Objectives = append(doc.Objectives, models.DocumentObjective{
			ID: o.ID, Code: o.Code, GoalID: o.GoalID, Title: o.Title,
			Description: o.Description, Ordinality: o.Ordinality,
		})
	}

	var workflowRows []gormstore.WorkflowRow
	if err := scoped(tables.Workflows).Order("ordinality").Find(&workflowRows).Error; err != nil {
		return fail(err)
	}
	for _, r := range workflowRows {
		w := gormstore.WorkflowFromRow(r)
		doc.Workflows = append(doc.Workflows, models.DocumentWorkflow{
			ID: w.ID, Code: w.Code, Name: w.Name, Description: w.Description,
			Ordinality: w.Ordinality, EventFlows: w.EventFlows,
		})
	}

	var workItemRows []gormstore.WorkItemRow
	if err := scoped(tables.WorkItems).Order("ordinality").Find(&workItemRows).Error; err != nil {
		return fail(err)
	}
	for _, r := range workItemRows {
		wi := gormstore.WorkItemFromRow(r)
		doc.WorkItems = append(doc.WorkItems, models.DocumentWorkItem{
			ID: wi.ID, Code: wi.Code, WorkflowID: wi.WorkflowID, ObjectiveID: wi.ObjectiveID,
			Title: wi.Title, Description: wi.Description, Ordinality: wi.Ordinality, Prompt: wi.Prompt,
		})
	}

	var assignmentRows []gormstore.RoleAssignmentRow
	if err := scoped(tables.RoleAssignments).Order("created_at").Find(&assignmentRows).Error; err != nil {
		return fail(err)
	}
	for _, r := range assignmentRows {
		ra := gormstore.RoleAssignmentFromRow(r)
		doc.RoleAssignments = append(doc.RoleAssignments, models.DocumentRoleAssignment{
			RoleID: ra.RoleID, ObjectiveID: ra.ObjectiveID, Label: string(ra.Label),
		})
	}

	var valueRows []gormstore.ValueRow
	if err := scoped(tables.Values).Order("created_at").Find(&valueRows).Error; err != nil {
		return fail(err)
	}
	for _, r := range valueRows {
		v := gormstore.ValueFromRow(r)
		doc.Values = append(doc.Values, models.DocumentStatement{ID: v.ID, Code: v.Code, Text: v.Text})
	}

	var ruleRows []gormstore.RuleRow
	if err := scoped(tables.Rules).Order("created_at").Find(&ruleRows).Error; err != nil {
		return fail(err)
	}
	for _, r := range ruleRows {
		rule := gormstore.RuleFromRow(r)
		doc.Rules = append(doc.Rules, models.DocumentStatement{ID: rule.ID, Code: rule.Code, Text: rule.Text})
	}

	var valueAttachmentRows []gormstore.ValueAttachmentRow
	if err := tx.WithContext(ctx).Table(tables.ValueAttachments).Order("created_at").Find(&valueAttachmentRows).Error; err != nil {
		return fail(err)
	}
	for _, r := range valueAttachmentRows {
		a := gormstore.ValueAttachmentFromRow(r)
		doc.ValueAttachments = append(doc.ValueAttachments, models.DocumentAttachment{
			ValueCode: a.ValueCode, TargetType: string(a.TargetType), TargetCode: a.TargetCode,
		})
	}

	var ruleAttachmentRows []gormstore.RuleAttachmentRow
	if err := tx.WithContext(ctx).Table(tables.RuleAttachments).Order("created_at").Find(&ruleAttachmentRows).Error; err != nil {
		return fail(err)
	}
	for _, r := range ruleAttachmentRows {
		a := gormstore.RuleAttachmentFromRow(r)
		doc.RuleAttachments = append(doc.RuleAttachments, models.DocumentAttachment{
			RuleCode: a.RuleCode, TargetType: string(a.TargetType), TargetCode: a.TargetCode,
		})
	}

	// Links are read before Modules: a Module's Enabled is derived from
	// them (see models.Module), not taken from its own row.
	var objectiveModuleRows []gormstore.ObjectiveModuleRow
	if err := scoped(tables.ObjectiveModules).Order("created_at").Find(&objectiveModuleRows).Error; err != nil {
		return fail(err)
	}
	linkedModules := make(map[string]bool, len(objectiveModuleRows))
	for _, r := range objectiveModuleRows {
		om := gormstore.ObjectiveModuleFromRow(r)
		linkedModules[om.ModuleID] = true
		doc.ObjectiveModules = append(doc.ObjectiveModules, models.DocumentObjectiveModule{
			ObjectiveID: om.ObjectiveID, ModuleID: om.ModuleID,
		})
	}

	var moduleRows []gormstore.ModuleRow
	if err := scoped(tables.Modules).Order("code").Find(&moduleRows).Error; err != nil {
		return fail(err)
	}
	for _, r := range moduleRows {
		m := withDerivedEnabled(gormstore.ModuleFromRow(r), linkedModules[r.ID])
		doc.Modules = append(doc.Modules, models.DocumentModule{ID: m.ID, Code: m.Code, Enabled: m.Enabled})
	}

	var stepRows []gormstore.EventFlowStepRow
	if err := scoped(tables.EventFlowSteps).Order("workflow_code, ordinality, step").Find(&stepRows).Error; err != nil {
		return fail(err)
	}
	for _, r := range stepRows {
		doc.EventFlowSteps = append(doc.EventFlowSteps, gormstore.EventFlowStepFromRow(r))
	}

	return doc, nil
}

// applyDocumentToDraft materializes a published document into fresh rows
// scoped to draftID — the inverse of [buildDocument], and what lets a
// Draft fork from a tag rather than from the working set (AGD-005 #5).
//
// Every row is minted with a new id and the document's ids are remapped
// onto them, the same way mwanachama-wakala-api's import endpoint does.
// Code is carried across verbatim, not re-minted: a Draft copy shares its
// original's Code (AG16), which is also what makes EventFlowStep's
// WorkflowCode join and the attachment tables keep resolving.
//
// Attachments are deliberately not applied — they are never Draft-scoped
// (AGD-003), so the live attachment rows already cover this Draft's
// copies by Code.
func applyDocumentToDraft(ctx context.Context, tx *gorm.DB, tables TableNames, draftID string, doc models.AgencyDocument, now string) error {
	fail := func(err error) error { return fmt.Errorf("applyDocumentToDraft: %w", err) }

	goalIDs := make(map[string]string, len(doc.Goals))
	for _, g := range doc.Goals {
		row := gormstore.GoalToRow(models.Goal{
			Code: g.Code, DraftID: draftID, Title: g.Title,
			Description: g.Description, Ordinality: g.Ordinality, CreatedAt: now,
		})
		if err := tx.WithContext(ctx).Table(tables.Goals).Create(&row).Error; err != nil {
			return fail(err)
		}
		goalIDs[g.ID] = row.ID
	}

	objectiveIDs := make(map[string]string, len(doc.Objectives))
	for _, o := range doc.Objectives {
		row := gormstore.ObjectiveToRow(models.Objective{
			Code: o.Code, DraftID: draftID, GoalID: goalIDs[o.GoalID], Title: o.Title,
			Description: o.Description, Ordinality: o.Ordinality, CreatedAt: now,
		})
		if err := tx.WithContext(ctx).Table(tables.Objectives).Create(&row).Error; err != nil {
			return fail(err)
		}
		objectiveIDs[o.ID] = row.ID
	}

	workflowIDs := make(map[string]string, len(doc.Workflows))
	for _, w := range doc.Workflows {
		row := gormstore.WorkflowToRow(models.Workflow{
			Code: w.Code, DraftID: draftID, Name: w.Name, Description: w.Description,
			Ordinality: w.Ordinality, EventFlows: w.EventFlows, CreatedAt: now,
		})
		if err := tx.WithContext(ctx).Table(tables.Workflows).Create(&row).Error; err != nil {
			return fail(err)
		}
		workflowIDs[w.ID] = row.ID
	}

	for _, wi := range doc.WorkItems {
		row := gormstore.WorkItemToRow(models.WorkItem{
			Code: wi.Code, DraftID: draftID, WorkflowID: workflowIDs[wi.WorkflowID],
			ObjectiveID: objectiveIDs[wi.ObjectiveID], Title: wi.Title,
			Description: wi.Description, Ordinality: wi.Ordinality, Prompt: wi.Prompt, CreatedAt: now,
		})
		if err := tx.WithContext(ctx).Table(tables.WorkItems).Create(&row).Error; err != nil {
			return fail(err)
		}
	}

	roleIDs := make(map[string]string, len(doc.Roles))
	for _, r := range doc.Roles {
		row := gormstore.RoleToRow(models.Role{
			Code: r.Code, DraftID: draftID, Name: r.Name, Description: r.Description, CreatedAt: now,
		})
		if err := tx.WithContext(ctx).Table(tables.Roles).Create(&row).Error; err != nil {
			return fail(err)
		}
		roleIDs[r.ID] = row.ID
	}

	for _, ra := range doc.RoleAssignments {
		row := gormstore.RoleAssignmentToRow(models.RoleAssignment{
			DraftID: draftID, RoleID: roleIDs[ra.RoleID], ObjectiveID: objectiveIDs[ra.ObjectiveID],
			Label: models.RACILabel(ra.Label), CreatedAt: now,
		})
		if err := tx.WithContext(ctx).Table(tables.RoleAssignments).Create(&row).Error; err != nil {
			return fail(err)
		}
	}

	for _, v := range doc.Values {
		row := gormstore.ValueToRow(models.Value{Code: v.Code, DraftID: draftID, Text: v.Text, CreatedAt: now})
		if err := tx.WithContext(ctx).Table(tables.Values).Create(&row).Error; err != nil {
			return fail(err)
		}
	}
	for _, r := range doc.Rules {
		row := gormstore.RuleToRow(models.Rule{Code: r.Code, DraftID: draftID, Text: r.Text, CreatedAt: now})
		if err := tx.WithContext(ctx).Table(tables.Rules).Create(&row).Error; err != nil {
			return fail(err)
		}
	}

	// Module is a fixed catalog, and the document carries only each one's
	// id/code/Enabled — Name/Description/Locked come back from
	// [models.ModuleCatalog], so a restored locked code is still locked and
	// still mounted (Enabled is derived from Locked plus this scope's own
	// ObjectiveModule rows, both of which the loop below re-creates).
	catalog := make(map[string]models.ModuleCatalogEntry, len(models.ModuleCatalog))
	for _, entry := range models.ModuleCatalog {
		catalog[entry.Code] = entry
	}
	moduleIDs := make(map[string]string, len(doc.Modules))
	for _, mod := range doc.Modules {
		entry := catalog[mod.Code]
		row := gormstore.ModuleToRow(models.Module{
			Code: mod.Code, DraftID: draftID, Name: entry.Name, Description: entry.Description,
			Enabled: mod.Enabled, Locked: entry.Locked, CreatedAt: now,
		})
		if err := tx.WithContext(ctx).Table(tables.Modules).Create(&row).Error; err != nil {
			return fail(err)
		}
		moduleIDs[mod.ID] = row.ID
	}

	for _, om := range doc.ObjectiveModules {
		row := gormstore.ObjectiveModuleToRow(models.ObjectiveModule{
			DraftID: draftID, ObjectiveID: objectiveIDs[om.ObjectiveID],
			ModuleID: moduleIDs[om.ModuleID], CreatedAt: now,
		})
		if err := tx.WithContext(ctx).Table(tables.ObjectiveModules).Create(&row).Error; err != nil {
			return fail(err)
		}
	}

	for _, step := range doc.EventFlowSteps {
		step.ID, step.DraftID, step.CreatedAt = "", draftID, now
		row := gormstore.EventFlowStepToRow(step)
		if err := tx.WithContext(ctx).Table(tables.EventFlowSteps).Create(&row).Error; err != nil {
			return fail(err)
		}
	}

	return nil
}
