package mwanachamaagency

import (
	"context"
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
			ID: r.ID, Code: r.Code, Name: r.Name, Description: r.Description,
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
	for _, mod := range modules {
		mod = withDerivedEnabled(mod, linkedModules[mod.ID])
		doc.Modules = append(doc.Modules, models.DocumentModule{ID: mod.ID, Code: mod.Code, Enabled: mod.Enabled})
	}

	steps, err := list[models.EventFlowStep](m, scoped(roleEventFlowStep).Order("workflow_code, ordinality, step"), roleEventFlowStep)
	if err != nil {
		return fail(err)
	}
	doc.EventFlowSteps = append(doc.EventFlowSteps, steps...)

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

	for _, wi := range doc.WorkItems {
		row := models.WorkItem{
			ID: newID(), Code: wi.Code, DraftID: draftID, WorkflowID: workflowIDs[wi.WorkflowID],
			ObjectiveID: objectiveIDs[wi.ObjectiveID], Title: wi.Title,
			Description: wi.Description, Ordinality: wi.Ordinality, Prompt: wi.Prompt,
			CreatedAt: now, LastUpdated: now,
		}
		if err := m.copyRow(ctx, tx, roleWorkItem, row); err != nil {
			return fail(err)
		}
	}

	roleIDs := make(map[string]string, len(doc.Roles))
	for _, r := range doc.Roles {
		row := models.Role{
			ID: newID(), Code: r.Code, DraftID: draftID, Name: r.Name,
			Description: r.Description, CreatedAt: now, LastUpdated: now,
		}
		if err := m.copyRow(ctx, tx, roleRole, row); err != nil {
			return fail(err)
		}
		roleIDs[r.ID] = row.ID
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
			CreatedAt: now, LastUpdated: now,
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
	return nil
}
