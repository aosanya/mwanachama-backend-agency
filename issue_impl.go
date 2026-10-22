package mwanachamaagency

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

type issueScan struct {
	ctx     context.Context
	m       *agencyManager
	tx      *gorm.DB
	draftID string
	now     string
	found   []models.Issue
}

func (m *agencyManager) scanIssues(ctx context.Context, tx *gorm.DB, draftID string) ([]models.Issue, error) {
	s := issueScan{ctx: ctx, m: m, tx: tx, draftID: draftID, now: models.NowRFC3339()}
	for _, check := range []func() error{
		s.checkStructure,
		s.checkRACI,
		s.checkWorkItems,
		s.checkEventFlowSteps,
		s.checkAttachments,
		s.checkModuleSettings,
	} {
		if err := check(); err != nil {
			return nil, fmt.Errorf("scanIssues: %w", err)
		}
	}
	return s.found, nil
}

func (s *issueScan) scoped(role string) *gorm.DB {
	return scopeQuery(s.tx.WithContext(s.ctx).Table(s.m.table(role)), s.draftID)
}

func (s *issueScan) add(sev models.IssueSeverity, entityType, entityID, format string, args ...any) {
	s.found = append(s.found, models.Issue{
		ID:         newID(),
		DraftID:    s.draftID,
		Severity:   sev,
		Message:    fmt.Sprintf(format, args...),
		EntityType: entityType,
		EntityID:   entityID,
		CreatedAt:  s.now,
	})
}

func (s *issueScan) codeSet(role string) (map[string]bool, error) {
	var codes []string
	if err := s.scoped(role).Pluck("code", &codes).Error; err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(codes))
	for _, c := range codes {
		set[c] = true
	}
	return set, nil
}

func (s *issueScan) checkStructure() error {
	goals, err := list[models.Goal](s.m, s.scoped(roleGoal), roleGoal)
	if err != nil {
		return err
	}
	objectives, err := list[models.Objective](s.m, s.scoped(roleObjective), roleObjective)
	if err != nil {
		return err
	}

	if len(goals) == 0 {
		s.add(models.IssueSeverityError, "agency", "", "The Agency has no Goals")
	}
	goalsWithObjective := make(map[string]bool, len(objectives))
	for _, o := range objectives {
		goalsWithObjective[o.GoalID] = true
	}
	for _, g := range goals {
		if !goalsWithObjective[g.ID] {
			s.add(models.IssueSeverityError, "goal", g.ID, "Goal %s %q has no Objectives", g.Code, g.Title)
		}
	}
	return nil
}

func (s *issueScan) checkRACI() error {
	objectives, err := list[models.Objective](s.m, s.scoped(roleObjective), roleObjective)
	if err != nil {
		return err
	}
	roles, err := list[models.Role](s.m, s.scoped(roleRole), roleRole)
	if err != nil {
		return err
	}
	assignments, err := list[models.RoleAssignment](s.m, s.scoped(roleRoleAssignment), roleRoleAssignment)
	if err != nil {
		return err
	}

	objectiveIDs := make(map[string]bool, len(objectives))
	for _, o := range objectives {
		objectiveIDs[o.ID] = true
	}
	roleIDs := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleIDs[r.ID] = true
	}

	objectivesWithAccountable := make(map[string]bool, len(assignments))
	for _, a := range assignments {
		if a.Label == models.RACILabelAccountable {
			objectivesWithAccountable[a.ObjectiveID] = true
		}
		if !roleIDs[a.RoleID] {
			s.add(models.IssueSeverityError, "role_assignment", a.ID,
				"Role assignment %s refers to Role %s, which does not exist", a.ID, a.RoleID)
		}
		if !objectiveIDs[a.ObjectiveID] {
			s.add(models.IssueSeverityError, "role_assignment", a.ID,
				"Role assignment %s refers to Objective %s, which does not exist", a.ID, a.ObjectiveID)
		}
	}
	for _, o := range objectives {
		if !objectivesWithAccountable[o.ID] {
			s.add(models.IssueSeverityError, "objective", o.ID,
				"Objective %s %q has no Accountable role assigned", o.Code, o.Title)
		}
	}
	return nil
}

func (s *issueScan) checkWorkItems() error {
	workflows, err := list[models.Workflow](s.m, s.scoped(roleWorkflow), roleWorkflow)
	if err != nil {
		return err
	}
	objectives, err := list[models.Objective](s.m, s.scoped(roleObjective), roleObjective)
	if err != nil {
		return err
	}
	workItems, err := list[models.WorkItem](s.m, s.scoped(roleWorkItem), roleWorkItem)
	if err != nil {
		return err
	}

	objectiveIDs := make(map[string]bool, len(objectives))
	for _, o := range objectives {
		objectiveIDs[o.ID] = true
	}
	workflowsWithItem := make(map[string]bool, len(workItems))
	for _, w := range workItems {
		workflowsWithItem[w.WorkflowID] = true
		if !objectiveIDs[w.ObjectiveID] {
			s.add(models.IssueSeverityError, "work_item", w.ID,
				"Work item %s %q refers to Objective %s, which does not exist", w.Code, w.Title, w.ObjectiveID)
		}
	}
	for _, w := range workflows {
		if !workflowsWithItem[w.ID] {
			s.add(models.IssueSeverityError, "workflow", w.ID, "Workflow %s %q has no Work Items", w.Code, w.Name)
		}
	}
	return nil
}

func (s *issueScan) checkEventFlowSteps() error {
	workflowCodes, err := s.codeSet(roleWorkflow)
	if err != nil {
		return err
	}
	steps, err := list[models.EventFlowStep](s.m, s.scoped(roleEventFlowStep), roleEventFlowStep)
	if err != nil {
		return err
	}
	for _, step := range steps {
		if step.WorkflowCode == "" || workflowCodes[step.WorkflowCode] {
			continue
		}
		s.add(models.IssueSeverityError, "event_flow_step", step.ID,
			"Event flow step %s %q refers to Workflow code %q, which does not exist", step.Code, step.Name, step.WorkflowCode)
	}
	return nil
}

func (s *issueScan) checkAttachments() error {
	codesByType := make(map[models.TargetType]map[string]bool, 5)
	for _, t := range []models.TargetType{
		models.TargetTypeRole, models.TargetTypeGoal, models.TargetTypeObjective,
		models.TargetTypeWorkflow, models.TargetTypeWorkItem,
	} {
		role, _, ok := s.m.targetRole(t)
		if !ok {
			continue
		}
		codes, err := s.codeSet(role)
		if err != nil {
			return err
		}
		codesByType[t] = codes
	}

	valueAttachments, err := list[models.ValueAttachment](s.m,
		s.tx.WithContext(s.ctx).Table(s.m.table(roleValueAttachment)), roleValueAttachment)
	if err != nil {
		return err
	}
	for _, a := range valueAttachments {
		if codesByType[a.TargetType][a.TargetCode] {
			continue
		}
		s.add(models.IssueSeverityError, "value_attachment", a.ID,
			"Value %s is attached to %s %s, which does not exist", a.ValueCode, a.TargetType, a.TargetCode)
	}

	ruleAttachments, err := list[models.RuleAttachment](s.m,
		s.tx.WithContext(s.ctx).Table(s.m.table(roleRuleAttachment)), roleRuleAttachment)
	if err != nil {
		return err
	}
	for _, a := range ruleAttachments {
		if codesByType[a.TargetType][a.TargetCode] {
			continue
		}
		s.add(models.IssueSeverityError, "rule_attachment", a.ID,
			"Rule %s is attached to %s %s, which does not exist", a.RuleCode, a.TargetType, a.TargetCode)
	}
	return nil
}

func (m *agencyManager) ListIssues(ctx context.Context, draftID string, filter IssueFilter) ([]models.Issue, error) {
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_, err := m.rescanIssues(ctx, tx, draftID)
		return err
	})
	if err != nil {
		return nil, err
	}

	q := scopeQuery(m.q(ctx, roleIssue), draftID).Order("created_at")
	q = likeFilter(q, "message", filter.Message)
	q = eqFilter(q, "severity", filter.Severity)
	q = eqFilter(q, "entity_type", filter.EntityType)

	out, err := list[models.Issue](m, q, roleIssue)
	if err != nil {
		return nil, fmt.Errorf("ListIssues: %w", err)
	}
	return out, nil
}

func (m *agencyManager) rescanIssues(ctx context.Context, tx *gorm.DB, draftID string) ([]models.Issue, error) {
	if err := m.deleteWhere(ctx, tx, roleIssue, "draft_id = ?", draftID); err != nil {
		return nil, fmt.Errorf("rescanIssues: %w", err)
	}
	found, err := m.scanIssues(ctx, tx, draftID)
	if err != nil {
		return nil, err
	}
	for _, issue := range found {
		if err := m.copyRow(ctx, tx, roleIssue, issue); err != nil {
			return nil, fmt.Errorf("rescanIssues: %w", err)
		}
	}
	return found, nil
}
