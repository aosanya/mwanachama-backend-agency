package mwanachamaagency

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) buildDocument(ctx context.Context, tx *gorm.DB, draftID string) (models.AgencyDocument, error) {
	var doc models.AgencyDocument

	scoped := func(role string) *gorm.DB {
		return scopeQuery(tx.WithContext(ctx).Table(m.table(role)), draftID)
	}
	unscoped := func(role string) *gorm.DB {
		return tx.WithContext(ctx).Table(m.table(role))
	}
	fail := func(err error) (models.AgencyDocument, error) {
		return models.AgencyDocument{}, fmt.Errorf("buildDocument: %w", err)
	}

	a, err := m.findAgency(ctx, tx)
	if err != nil {
		return fail(err)
	}
	doc.Agency = models.DocumentAgency{
		Name: a.Name, Description: a.Description, Enabled: a.Enabled,
		DefaultFailurePipelineBudget: a.DefaultFailurePipelineBudget,
		InactivityTimeoutSeconds:     a.InactivityTimeoutSeconds,
		EventFlows:                   a.EventFlows,
	}

	roles, err := list[models.Role](m, scoped(roleRole).Order("created_at"), roleRole)
	if err != nil {
		return fail(err)
	}
	for _, r := range roles {
		doc.Roles = append(doc.Roles, models.DocumentRole{
			ID: r.ID, Code: r.Code, Name: r.Name, Description: r.Description, Powers: r.Powers, Staff: r.Staff,
		})
	}

	goals, err := list[models.Goal](m, scoped(roleGoal).Order("ordinality"), roleGoal)
	if err != nil {
		return fail(err)
	}
	for _, g := range goals {
		doc.Goals = append(doc.Goals, models.DocumentGoal{
			ID: g.ID, Code: g.Code, Title: g.Title, Description: g.Description, Ordinality: g.Ordinality,
		})
	}

	objectives, err := list[models.Objective](m, scoped(roleObjective).Order("ordinality"), roleObjective)
	if err != nil {
		return fail(err)
	}
	for _, o := range objectives {
		doc.Objectives = append(doc.Objectives, models.DocumentObjective{
			ID: o.ID, Code: o.Code, GoalID: o.GoalID, Title: o.Title,
			Description: o.Description, Ordinality: o.Ordinality,
		})
	}

	workflows, err := list[models.Workflow](m, scoped(roleWorkflow).Order("ordinality"), roleWorkflow)
	if err != nil {
		return fail(err)
	}
	for _, w := range workflows {
		doc.Workflows = append(doc.Workflows, models.DocumentWorkflow{
			ID: w.ID, Code: w.Code, Name: w.Name, Description: w.Description,
			Ordinality: w.Ordinality, EventFlows: w.EventFlows,
		})
	}

	workItems, err := list[models.WorkItem](m, scoped(roleWorkItem).Order("ordinality"), roleWorkItem)
	if err != nil {
		return fail(err)
	}
	for _, wi := range workItems {
		doc.WorkItems = append(doc.WorkItems, models.DocumentWorkItem{
			ID: wi.ID, Code: wi.Code, WorkflowID: wi.WorkflowID, ObjectiveID: wi.ObjectiveID,
			Title: wi.Title, Description: wi.Description, Ordinality: wi.Ordinality, Prompt: wi.Prompt,
			Kind: wi.Kind, RoleID: wi.RoleID, RunWorkflowID: wi.RunWorkflowID,
			FormName: wi.FormName, Questions: wi.Questions,
			RecipientRoleID: wi.RecipientRoleID, SentStepID: wi.SentStepID,
			Routes: wi.Routes, OtherwiseStepID: wi.OtherwiseStepID,
		})
	}

	records, err := list[models.Record](m, scoped(roleRecord).Order("created_at"), roleRecord)
	if err != nil {
		return fail(err)
	}
	for _, r := range records {
		doc.Records = append(doc.Records, models.DocumentRecord{
			ID: r.ID, Code: r.Code, WorkflowID: r.WorkflowID, Name: r.Name,
			RoleID: r.RoleID, FormName: r.FormName, Questions: r.Questions,
		})
	}

	assignments, err := list[models.RoleAssignment](m, scoped(roleRoleAssignment).Order("created_at"), roleRoleAssignment)
	if err != nil {
		return fail(err)
	}
	for _, ra := range assignments {
		doc.RoleAssignments = append(doc.RoleAssignments, models.DocumentRoleAssignment{
			RoleID: ra.RoleID, ObjectiveID: ra.ObjectiveID, Label: string(ra.Label),
		})
	}

	values, err := list[models.Value](m, scoped(roleValue).Order("created_at"), roleValue)
	if err != nil {
		return fail(err)
	}
	for _, v := range values {
		doc.Values = append(doc.Values, models.DocumentStatement{ID: v.ID, Code: v.Code, Text: v.Text})
	}

	rules, err := list[models.Rule](m, scoped(roleRule).Order("created_at"), roleRule)
	if err != nil {
		return fail(err)
	}
	for _, r := range rules {
		doc.Rules = append(doc.Rules, models.DocumentStatement{ID: r.ID, Code: r.Code, Text: r.Text})
	}

	valueAttachments, err := list[models.ValueAttachment](m, unscoped(roleValueAttachment).Order("created_at"), roleValueAttachment)
	if err != nil {
		return fail(err)
	}
	for _, a := range valueAttachments {
		doc.ValueAttachments = append(doc.ValueAttachments, models.DocumentAttachment{
			ValueCode: a.ValueCode, TargetType: string(a.TargetType), TargetCode: a.TargetCode,
		})
	}

	ruleAttachments, err := list[models.RuleAttachment](m, unscoped(roleRuleAttachment).Order("created_at"), roleRuleAttachment)
	if err != nil {
		return fail(err)
	}
	for _, a := range ruleAttachments {
		doc.RuleAttachments = append(doc.RuleAttachments, models.DocumentAttachment{
			RuleCode: a.RuleCode, TargetType: string(a.TargetType), TargetCode: a.TargetCode,
		})
	}

	objectiveModules, err := list[models.ObjectiveModule](m, scoped(roleObjectiveModule).Order("created_at"), roleObjectiveModule)
	if err != nil {
		return fail(err)
	}
	linkedModules := make(map[string]bool, len(objectiveModules))
	for _, om := range objectiveModules {
		linkedModules[om.ModuleID] = true
		doc.ObjectiveModules = append(doc.ObjectiveModules, models.DocumentObjectiveModule{
			ObjectiveID: om.ObjectiveID, ModuleID: om.ModuleID, Rationale: om.Rationale,
		})
	}

	modules, err := list[models.Module](m, scoped(roleModule).Order("code"), roleModule)
	if err != nil {
		return fail(err)
	}
	for _, mod := range deriveEnabled(modules, linkedModules) {
		doc.Modules = append(doc.Modules, models.DocumentModule{
			ID: mod.ID, Code: mod.Code, Enabled: mod.Enabled, Spec: json.RawMessage(mod.Spec),
			Settings: json.RawMessage(mod.Settings),
		})
	}

	steps, err := list[models.EventFlowStep](m, scoped(roleEventFlowStep).Order("workflow_code, ordinality, step"), roleEventFlowStep)
	if err != nil {
		return fail(err)
	}
	doc.EventFlowSteps = append(doc.EventFlowSteps, steps...)

	doc.Scenarios, err = m.scenarioDocuments(ctx, tx, draftID)
	if err != nil {
		return fail(err)
	}

	return doc, nil
}

func applyDocumentToDraft(ctx context.Context, m *agencyManager, tx *gorm.DB, draftID string, doc models.AgencyDocument, now string) error {
	fail := func(err error) error { return fmt.Errorf("applyDocumentToDraft: %w", err) }

	goalIDs := make(map[string]string, len(doc.Goals))
	for _, g := range doc.Goals {
		row := models.Goal{
			ID: newID(), Code: g.Code, DraftID: draftID, Title: g.Title,
			Description: g.Description, Ordinality: g.Ordinality, CreatedAt: now, LastUpdated: now,
		}
		if err := m.copyRow(ctx, tx, roleGoal, row); err != nil {
			return fail(err)
		}
		goalIDs[g.ID] = row.ID
	}

	objectiveIDs := make(map[string]string, len(doc.Objectives))
	for _, o := range doc.Objectives {
		row := models.Objective{
			ID: newID(), Code: o.Code, DraftID: draftID, GoalID: goalIDs[o.GoalID], Title: o.Title,
			Description: o.Description, Ordinality: o.Ordinality, CreatedAt: now, LastUpdated: now,
		}
		if err := m.copyRow(ctx, tx, roleObjective, row); err != nil {
			return fail(err)
		}
		objectiveIDs[o.ID] = row.ID
	}

	workflowIDs := make(map[string]string, len(doc.Workflows))
	for _, w := range doc.Workflows {
		row := models.Workflow{
			ID: newID(), Code: w.Code, DraftID: draftID, Name: w.Name, Description: w.Description,
			Ordinality: w.Ordinality, EventFlows: w.EventFlows, CreatedAt: now, LastUpdated: now,
		}
		if err := m.copyRow(ctx, tx, roleWorkflow, row); err != nil {
			return fail(err)
		}
		workflowIDs[w.ID] = row.ID
	}

	roleIDs := make(map[string]string, len(doc.Roles))
	for _, r := range doc.Roles {
		row := models.Role{
			ID: newID(), Code: r.Code, DraftID: draftID, Name: r.Name,
			Description: r.Description, Powers: r.Powers, Staff: r.Staff, CreatedAt: now, LastUpdated: now,
		}
		if err := m.copyRow(ctx, tx, roleRole, row); err != nil {
			return fail(err)
		}
		roleIDs[r.ID] = row.ID
	}

	stepIDs := make(map[string]string, len(doc.WorkItems))
	for _, wi := range doc.WorkItems {
		stepIDs[wi.ID] = newID()
	}
	for _, wi := range doc.WorkItems {
		row := models.WorkItem{
			ID: stepIDs[wi.ID], Code: wi.Code, DraftID: draftID, WorkflowID: workflowIDs[wi.WorkflowID],
			ObjectiveID: objectiveIDs[wi.ObjectiveID], Title: wi.Title,
			Description: wi.Description, Ordinality: wi.Ordinality, Prompt: wi.Prompt,
			Kind: wi.Kind, RoleID: roleIDs[wi.RoleID], RunWorkflowID: workflowIDs[wi.RunWorkflowID],
			FormName: wi.FormName, Questions: wi.Questions,
			RecipientRoleID: remap(roleIDs, wi.RecipientRoleID), SentStepID: remap(stepIDs, wi.SentStepID),
			Routes: remapRoutes(stepIDs, wi.Routes), OtherwiseStepID: remap(stepIDs, wi.OtherwiseStepID),
			CreatedAt: now, LastUpdated: now,
		}
		if err := m.copyRow(ctx, tx, roleWorkItem, row); err != nil {
			return fail(err)
		}
		stepIDs[wi.ID] = row.ID
	}

	for _, r := range doc.Records {
		row := models.Record{
			ID: newID(), Code: r.Code, DraftID: draftID, WorkflowID: remap(workflowIDs, r.WorkflowID),
			Name: r.Name, RoleID: remap(roleIDs, r.RoleID), FormName: r.FormName, Questions: r.Questions,
			CreatedAt: now, LastUpdated: now,
		}
		if err := m.copyRow(ctx, tx, roleRecord, row); err != nil {
			return fail(err)
		}
	}

	for _, ra := range doc.RoleAssignments {
		row := models.RoleAssignment{
			ID: newID(), DraftID: draftID, RoleID: roleIDs[ra.RoleID],
			ObjectiveID: objectiveIDs[ra.ObjectiveID], Label: models.RACILabel(ra.Label),
			CreatedAt: now, LastUpdated: now,
		}
		if err := m.copyRow(ctx, tx, roleRoleAssignment, row); err != nil {
			return fail(err)
		}
	}

	for _, v := range doc.Values {
		row := models.Value{
			ID: newID(), Code: v.Code, DraftID: draftID, Text: v.Text,
			CreatedAt: now, LastUpdated: now,
		}
		if err := m.copyRow(ctx, tx, roleValue, row); err != nil {
			return fail(err)
		}
	}

	for _, r := range doc.Rules {
		row := models.Rule{
			ID: newID(), Code: r.Code, DraftID: draftID, Text: r.Text,
			CreatedAt: now, LastUpdated: now,
		}
		if err := m.copyRow(ctx, tx, roleRule, row); err != nil {
			return fail(err)
		}
	}

	catalog := make(map[string]models.ModuleCatalogEntry, len(models.ModuleCatalog))
	for _, entry := range models.ModuleCatalog {
		catalog[entry.Code] = entry
	}
	moduleIDs := make(map[string]string, len(doc.Modules))
	for _, mod := range doc.Modules {
		entry := catalog[mod.Code]
		row := models.Module{
			ID: newID(), Code: mod.Code, DraftID: draftID, Name: entry.Name,
			Description: entry.Description, Enabled: mod.Enabled, Locked: entry.Locked,
			Spec: string(mod.Spec), Settings: string(mod.Settings), CreatedAt: now, LastUpdated: now,
		}
		if err := m.copyRow(ctx, tx, roleModule, row); err != nil {
			return fail(err)
		}
		moduleIDs[mod.ID] = row.ID
	}

	for _, om := range doc.ObjectiveModules {
		row := models.ObjectiveModule{
			ID: newID(), DraftID: draftID, ObjectiveID: objectiveIDs[om.ObjectiveID],
			ModuleID: moduleIDs[om.ModuleID], Rationale: om.Rationale,
			CreatedAt: now, LastUpdated: now,
		}
		if err := m.copyRow(ctx, tx, roleObjectiveModule, row); err != nil {
			return fail(err)
		}
	}

	for _, step := range doc.EventFlowSteps {
		step.ID, step.DraftID, step.CreatedAt, step.LastUpdated = newID(), draftID, now, now
		if err := m.copyRow(ctx, tx, roleEventFlowStep, step); err != nil {
			return fail(err)
		}
	}
	remap := scenarioRemap{workflows: workflowIDs, roles: roleIDs, steps: stepIDs}
	return m.applyScenarioDocuments(ctx, tx, draftID, doc.Scenarios, now, remap)
}
