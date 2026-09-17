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

	var moduleRows []gormstore.ModuleRow
	if err := scoped(tables.Modules).Order("code").Find(&moduleRows).Error; err != nil {
		return fail(err)
	}
	for _, r := range moduleRows {
		m := gormstore.ModuleFromRow(r)
		doc.Modules = append(doc.Modules, models.DocumentModule{ID: m.ID, Code: m.Code, Enabled: m.Enabled})
	}

	var objectiveModuleRows []gormstore.ObjectiveModuleRow
	if err := scoped(tables.ObjectiveModules).Order("created_at").Find(&objectiveModuleRows).Error; err != nil {
		return fail(err)
	}
	for _, r := range objectiveModuleRows {
		om := gormstore.ObjectiveModuleFromRow(r)
		doc.ObjectiveModules = append(doc.ObjectiveModules, models.DocumentObjectiveModule{
			ObjectiveID: om.ObjectiveID, ModuleID: om.ModuleID,
		})
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
