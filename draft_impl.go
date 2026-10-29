package mwanachamaagency

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func scopeQuery(q *gorm.DB, draftID string) *gorm.DB {
	return q.Where("draft_id = ?", draftID)
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func likeFilter(q *gorm.DB, column, value string) *gorm.DB {
	if value == "" {
		return q
	}
	return q.Where("LOWER("+column+") LIKE ? ESCAPE '\\'", "%"+likeEscaper.Replace(strings.ToLower(value))+"%")
}

func eqFilter(q *gorm.DB, column, value string) *gorm.DB {
	if value == "" {
		return q
	}
	return q.Where(column+" = ?", value)
}

func (m *agencyManager) findDraft(ctx context.Context, tx *gorm.DB, id string) (models.Draft, error) {
	var d models.Draft
	if err := m.find(ctx, tx, roleDraft, id, &d, ErrDraftNotFound); err != nil {
		return models.Draft{}, err
	}
	return d, nil
}

func (m *agencyManager) validateOpenDraftID(ctx context.Context, draftID string) error {
	if draftID == "" {
		return ErrDraftRequired
	}
	d, err := m.findDraft(ctx, m.db, draftID)
	if err != nil {
		return err
	}
	if d.Status != models.DraftStatusOpen {
		return ErrDraftNotOpen
	}
	return nil
}

func (m *agencyManager) CreateDraft(ctx context.Context, description, forkedFromID, forkedFromType string) (models.Draft, error) {
	var out models.Draft
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := m.findAgency(ctx, tx); err != nil {
			return err
		}

		var sourceScope string
		var source models.AgencyPublication
		switch forkedFromType {
		case "":
		case "draft":
			src, err := m.findDraft(ctx, tx, forkedFromID)
			if err != nil {
				return err
			}
			if src.Status != models.DraftStatusOpen {
				return ErrDraftNotOpen
			}
			sourceScope = forkedFromID
		case "publication":
			p, err := m.findPublicationBy(ctx, tx, "tag = ?", forkedFromID)
			if err != nil {
				return err
			}
			source = p
		default:
			return ErrInvalidForkSource
		}

		now := models.NowRFC3339()
		draft := models.Draft{
			ID:             newID(),
			Description:    description,
			ForkedFromID:   forkedFromID,
			ForkedFromType: forkedFromType,
			Status:         models.DraftStatusOpen,
			CreatedAt:      now,
			LastUpdated:    now,
		}
		row, err := m.encode(roleDraft, draft)
		if err != nil {
			return err
		}
		if err := tx.Table(m.table(roleDraft)).Create(row).Error; err != nil {
			return fmt.Errorf("CreateDraft: %w", err)
		}
		out = draft

		if forkedFromType == "publication" {
			return applyDocumentToDraft(ctx, m, tx, draft.ID, source.Document, now)
		}
		if forkedFromType == "" {
			return seedModuleCatalog(tx, m.st.Spec(), draft.ID)
		}
		return m.forkScope(ctx, tx, sourceScope, draft.ID, now)
	})
	if err != nil {
		return models.Draft{}, err
	}
	return out, nil
}

func (m *agencyManager) forkScope(ctx context.Context, tx *gorm.DB, sourceScope, newDraftID, now string) error {
	goals, err := list[models.Goal](m, scopeQuery(tx.Table(m.table(roleGoal)), sourceScope), roleGoal)
	if err != nil {
		return fmt.Errorf("CreateDraft: %w", err)
	}
	goalIDs := make(map[string]string, len(goals))
	for _, g := range goals {
		old := g.ID
		g.ID, g.DraftID, g.CreatedAt, g.LastUpdated = newID(), newDraftID, now, now
		if err := m.copyRow(ctx, tx, roleGoal, g); err != nil {
			return err
		}
		goalIDs[old] = g.ID
	}

	objectives, err := list[models.Objective](m, scopeQuery(tx.Table(m.table(roleObjective)), sourceScope), roleObjective)
	if err != nil {
		return fmt.Errorf("CreateDraft: %w", err)
	}
	objectiveIDs := make(map[string]string, len(objectives))
	for _, o := range objectives {
		old := o.ID
		o.ID, o.DraftID, o.CreatedAt, o.LastUpdated = newID(), newDraftID, now, now
		o.GoalID = remap(goalIDs, o.GoalID)
		if err := m.copyRow(ctx, tx, roleObjective, o); err != nil {
			return err
		}
		objectiveIDs[old] = o.ID
	}

	workflows, err := list[models.Workflow](m, scopeQuery(tx.Table(m.table(roleWorkflow)), sourceScope), roleWorkflow)
	if err != nil {
		return fmt.Errorf("CreateDraft: %w", err)
	}
	workflowIDs := make(map[string]string, len(workflows))
	for _, w := range workflows {
		old := w.ID
		w.ID, w.DraftID, w.CreatedAt, w.LastUpdated = newID(), newDraftID, now, now
		if err := m.copyRow(ctx, tx, roleWorkflow, w); err != nil {
			return err
		}
		workflowIDs[old] = w.ID
	}

	roles, err := list[models.Role](m, scopeQuery(tx.Table(m.table(roleRole)), sourceScope), roleRole)
	if err != nil {
		return fmt.Errorf("CreateDraft: %w", err)
	}
	roleIDs := make(map[string]string, len(roles))
	for _, r := range roles {
		old := r.ID
		r.ID, r.DraftID, r.CreatedAt, r.LastUpdated = newID(), newDraftID, now, now
		if err := m.copyRow(ctx, tx, roleRole, r); err != nil {
			return err
		}
		roleIDs[old] = r.ID
	}

	workItems, err := list[models.WorkItem](m, scopeQuery(tx.Table(m.table(roleWorkItem)), sourceScope), roleWorkItem)
	if err != nil {
		return fmt.Errorf("CreateDraft: %w", err)
	}
	stepIDs := make(map[string]string, len(workItems))
	for i := range workItems {
		old := workItems[i].ID
		workItems[i].ID = newID()
		stepIDs[old] = workItems[i].ID
	}
	for _, wi := range workItems {
		wi.DraftID, wi.CreatedAt, wi.LastUpdated = newDraftID, now, now
		wi.WorkflowID = remap(workflowIDs, wi.WorkflowID)
		wi.ObjectiveID = remap(objectiveIDs, wi.ObjectiveID)
		wi.RoleID = remap(roleIDs, wi.RoleID)
		wi.RunWorkflowID = remap(workflowIDs, wi.RunWorkflowID)
		wi.RecipientRoleID = remap(roleIDs, wi.RecipientRoleID)
		wi.SentStepID = remap(stepIDs, wi.SentStepID)
		wi.OtherwiseStepID = remap(stepIDs, wi.OtherwiseStepID)
		wi.Routes = remapRoutes(stepIDs, wi.Routes)
		if err := m.copyRow(ctx, tx, roleWorkItem, wi); err != nil {
			return err
		}
	}

	records, err := list[models.Record](m, scopeQuery(tx.Table(m.table(roleRecord)), sourceScope), roleRecord)
	if err != nil {
		return fmt.Errorf("CreateDraft: %w", err)
	}
	for _, r := range records {
		r.ID, r.DraftID, r.CreatedAt, r.LastUpdated = newID(), newDraftID, now, now
		r.WorkflowID = remap(workflowIDs, r.WorkflowID)
		r.RoleID = remap(roleIDs, r.RoleID)
		if err := m.copyRow(ctx, tx, roleRecord, r); err != nil {
			return err
		}
	}

	steps, err := list[models.EventFlowStep](m, scopeQuery(tx.Table(m.table(roleEventFlowStep)), sourceScope), roleEventFlowStep)
	if err != nil {
		return fmt.Errorf("CreateDraft: %w", err)
	}
	for _, s := range steps {
		s.ID, s.DraftID, s.CreatedAt, s.LastUpdated = newID(), newDraftID, now, now
		if err := m.copyRow(ctx, tx, roleEventFlowStep, s); err != nil {
			return err
		}
	}

	modules, err := list[models.Module](m, scopeQuery(tx.Table(m.table(roleModule)), sourceScope), roleModule)
	if err != nil {
		return fmt.Errorf("CreateDraft: %w", err)
	}
	moduleIDs := make(map[string]string, len(modules))
	for _, mod := range modules {
		old := mod.ID
		mod.ID, mod.DraftID, mod.CreatedAt, mod.LastUpdated = newID(), newDraftID, now, now
		if err := m.copyRow(ctx, tx, roleModule, mod); err != nil {
			return err
		}
		moduleIDs[old] = mod.ID
	}

	assignments, err := list[models.RoleAssignment](m, scopeQuery(tx.Table(m.table(roleRoleAssignment)), sourceScope), roleRoleAssignment)
	if err != nil {
		return fmt.Errorf("CreateDraft: %w", err)
	}
	for _, a := range assignments {
		a.ID, a.DraftID, a.CreatedAt, a.LastUpdated = newID(), newDraftID, now, now
		a.RoleID = remap(roleIDs, a.RoleID)
		a.ObjectiveID = remap(objectiveIDs, a.ObjectiveID)
		if err := m.copyRow(ctx, tx, roleRoleAssignment, a); err != nil {
			return err
		}
	}

	objectiveModules, err := list[models.ObjectiveModule](m, scopeQuery(tx.Table(m.table(roleObjectiveModule)), sourceScope), roleObjectiveModule)
	if err != nil {
		return fmt.Errorf("CreateDraft: %w", err)
	}
	for _, om := range objectiveModules {
		om.ID, om.DraftID, om.CreatedAt, om.LastUpdated = newID(), newDraftID, now, now
		om.ObjectiveID = remap(objectiveIDs, om.ObjectiveID)
		om.ModuleID = remap(moduleIDs, om.ModuleID)
		if err := m.copyRow(ctx, tx, roleObjectiveModule, om); err != nil {
			return err
		}
	}

	attached := func(role, attachmentRole, codeColumn string) *gorm.DB {
		return scopeQuery(tx.Table(m.table(role)), sourceScope).
			Where("code IN (SELECT " + codeColumn + " FROM " + m.table(attachmentRole) + ")")
	}

	values, err := list[models.Value](m, attached(roleValue, roleValueAttachment, "value_code"), roleValue)
	if err != nil {
		return fmt.Errorf("CreateDraft: %w", err)
	}
	for _, v := range values {
		v.ID, v.DraftID, v.CreatedAt, v.LastUpdated = newID(), newDraftID, now, now
		if err := m.copyRow(ctx, tx, roleValue, v); err != nil {
			return err
		}
	}

	rules, err := list[models.Rule](m, attached(roleRule, roleRuleAttachment, "rule_code"), roleRule)
	if err != nil {
		return fmt.Errorf("CreateDraft: %w", err)
	}
	for _, r := range rules {
		r.ID, r.DraftID, r.CreatedAt, r.LastUpdated = newID(), newDraftID, now, now
		if err := m.copyRow(ctx, tx, roleRule, r); err != nil {
			return err
		}
	}
	return nil
}

func (m *agencyManager) copyRow(ctx context.Context, tx *gorm.DB, role string, v any) error {
	row, err := m.encode(role, v)
	if err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Table(m.table(role)).Create(row).Error; err != nil {
		return fmt.Errorf("CreateDraft: %w", err)
	}
	return nil
}

func remapRoutes(stepIDs map[string]string, routes []models.StepRoute) []models.StepRoute {
	if len(routes) == 0 {
		return nil
	}
	out := make([]models.StepRoute, len(routes))
	for i, r := range routes {
		out[i] = models.StepRoute{Match: r.Match, Rules: append([]models.RouteRule(nil), r.Rules...), GoToStep: remap(stepIDs, r.GoToStep)}
	}
	return out
}

func remap(ids map[string]string, id string) string {
	if mapped, ok := ids[id]; ok {
		return mapped
	}
	return id
}

func (m *agencyManager) GetDraft(ctx context.Context, id string) (models.Draft, error) {
	return m.findDraft(ctx, m.db, id)
}

func (m *agencyManager) ListDrafts(ctx context.Context, filter DraftFilter) ([]models.Draft, error) {
	q := m.q(ctx, roleDraft).Order("created_at")
	q = likeFilter(q, "description", filter.Description)
	q = eqFilter(q, "status", filter.Status)

	out, err := list[models.Draft](m, q, roleDraft)
	if err != nil {
		return nil, fmt.Errorf("ListDrafts: %w", err)
	}
	return out, nil
}

func (m *agencyManager) UpdateDraftDescription(ctx context.Context, id, description string) (models.Draft, error) {
	var out models.Draft
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		draft, err := m.findDraft(ctx, tx, id)
		if err != nil {
			return err
		}
		if draft.Status != models.DraftStatusOpen {
			return ErrDraftNotOpen
		}
		draft.Description = description
		draft.LastUpdated = models.NowRFC3339()

		if err := m.writeUpdate(ctx, tx, roleDraft, id, draft, ErrDraftNotFound); err != nil {
			return err
		}
		out = draft
		return nil
	})
	if err != nil {
		return models.Draft{}, fmt.Errorf("UpdateDraftDescription: %w", err)
	}
	return out, nil
}

func (m *agencyManager) ArchiveDraft(ctx context.Context, id string) (models.Draft, error) {
	var out models.Draft
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		draft, err := m.findDraft(ctx, tx, id)
		if err != nil {
			return err
		}
		if !draft.Status.CanTransitionTo(models.DraftStatusArchived) {
			return ErrDraftNotOpen
		}
		draft.Status = models.DraftStatusArchived
		draft.LastUpdated = models.NowRFC3339()

		if err := m.writeUpdate(ctx, tx, roleDraft, id, draft, ErrDraftNotFound); err != nil {
			return err
		}
		out = draft
		return nil
	})
	if err != nil {
		return models.Draft{}, fmt.Errorf("ArchiveDraft: %w", err)
	}
	return out, nil
}
