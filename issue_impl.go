package mwanachamaagency

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// scanIssues re-derives every Issue for one scope from that scope's
// current rows. draftID selects the scope the same way every other
// scoped read does: "" scans the live Agency, a Draft's id scans that
// Draft. Called inside ListIssues' and PublishAgency's transactions,
// after any stale rows have already been deleted.
//
// Severity is what [AgencyManager.PublishAgency] gates on: an error
// refuses the publish, a warning or info never does (AGD-004).
func scanIssues(ctx context.Context, tx *gorm.DB, tables TableNames, draftID string) ([]models.Issue, error) {
	s := issueScan{ctx: ctx, tx: tx, tables: tables, draftID: draftID, now: models.NowRFC3339()}
	for _, check := range []func() error{
		s.checkStructure,
		s.checkRACI,
		s.checkWorkItems,
		s.checkEventFlowSteps,
		s.checkAttachments,
	} {
		if err := check(); err != nil {
			return nil, fmt.Errorf("scanIssues: %w", err)
		}
	}
	return s.found, nil
}

// issueScan carries one scan's scope and accumulated findings so each
// check reads as its own rule rather than one long function threading
// the same four values through.
type issueScan struct {
	ctx     context.Context
	tx      *gorm.DB
	tables  TableNames
	draftID string
	now     string
	found   []models.Issue
}

func (s *issueScan) scoped(table string) *gorm.DB {
	return scopeQuery(s.tx.WithContext(s.ctx).Table(table), s.draftID)
}

func (s *issueScan) add(sev models.IssueSeverity, entityType, entityID, format string, args ...any) {
	s.found = append(s.found, models.Issue{
		DraftID:    s.draftID,
		Severity:   sev,
		Message:    fmt.Sprintf(format, args...),
		EntityType: entityType,
		EntityID:   entityID,
		CreatedAt:  s.now,
	})
}

// codeSet reads one scoped table's Code column into a lookup set, for the
// dangling-reference checks that resolve by Code rather than by row id
// (AG22's attachments, EventFlowStep's workflow_code business key).
func (s *issueScan) codeSet(table string) (map[string]bool, error) {
	var codes []string
	if err := s.scoped(table).Pluck("code", &codes).Error; err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(codes))
	for _, c := range codes {
		set[c] = true
	}
	return set, nil
}

// checkStructure flags an Agency with nothing in it and a Goal carrying no
// Objectives — both errors: a publication of either is empty of the thing
// it exists to record.
func (s *issueScan) checkStructure() error {
	var goalRows []gormstore.GoalRow
	if err := s.scoped(s.tables.Goals).Find(&goalRows).Error; err != nil {
		return err
	}
	var objectiveRows []gormstore.ObjectiveRow
	if err := s.scoped(s.tables.Objectives).Find(&objectiveRows).Error; err != nil {
		return err
	}

	if len(goalRows) == 0 {
		s.add(models.IssueSeverityError, "agency", "", "The Agency has no Goals")
	}

	goalsWithObjective := make(map[string]bool, len(objectiveRows))
	for _, or := range objectiveRows {
		goalsWithObjective[gormstore.ObjectiveFromRow(or).GoalID] = true
	}
	for _, gr := range goalRows {
		g := gormstore.GoalFromRow(gr)
		if !goalsWithObjective[g.ID] {
			s.add(models.IssueSeverityError, "goal", g.ID, "Goal %s %q has no Objectives", g.Code, g.Title)
		}
	}
	return nil
}

// checkRACI flags an Objective nobody is Accountable for, and a
// RoleAssignment pointing at a Role or Objective that no longer exists in
// this scope.
func (s *issueScan) checkRACI() error {
	var objectiveRows []gormstore.ObjectiveRow
	if err := s.scoped(s.tables.Objectives).Find(&objectiveRows).Error; err != nil {
		return err
	}
	var roleRows []gormstore.RoleRow
	if err := s.scoped(s.tables.Roles).Find(&roleRows).Error; err != nil {
		return err
	}
	var assignmentRows []gormstore.RoleAssignmentRow
	if err := s.scoped(s.tables.RoleAssignments).Find(&assignmentRows).Error; err != nil {
		return err
	}

	objectiveIDs := make(map[string]bool, len(objectiveRows))
	for _, or := range objectiveRows {
		objectiveIDs[or.ID] = true
	}
	roleIDs := make(map[string]bool, len(roleRows))
	for _, rr := range roleRows {
		roleIDs[rr.ID] = true
	}

	objectivesWithAccountable := make(map[string]bool, len(assignmentRows))
	for _, ar := range assignmentRows {
		a := gormstore.RoleAssignmentFromRow(ar)
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

	for _, or := range objectiveRows {
		o := gormstore.ObjectiveFromRow(or)
		if !objectivesWithAccountable[o.ID] {
			s.add(models.IssueSeverityError, "objective", o.ID,
				"Objective %s %q has no Accountable role assigned", o.Code, o.Title)
		}
	}
	return nil
}

// checkWorkItems flags a WorkItem orphaned from its required Objective,
// and a Workflow with nothing in it.
func (s *issueScan) checkWorkItems() error {
	var workflowRows []gormstore.WorkflowRow
	if err := s.scoped(s.tables.Workflows).Find(&workflowRows).Error; err != nil {
		return err
	}
	var objectiveRows []gormstore.ObjectiveRow
	if err := s.scoped(s.tables.Objectives).Find(&objectiveRows).Error; err != nil {
		return err
	}
	var workItemRows []gormstore.WorkItemRow
	if err := s.scoped(s.tables.WorkItems).Find(&workItemRows).Error; err != nil {
		return err
	}

	objectiveIDs := make(map[string]bool, len(objectiveRows))
	for _, or := range objectiveRows {
		objectiveIDs[or.ID] = true
	}

	workflowsWithItem := make(map[string]bool, len(workItemRows))
	for _, wr := range workItemRows {
		w := gormstore.WorkItemFromRow(wr)
		workflowsWithItem[w.WorkflowID] = true
		if !objectiveIDs[w.ObjectiveID] {
			s.add(models.IssueSeverityError, "work_item", w.ID,
				"Work item %s %q refers to Objective %s, which does not exist", w.Code, w.Title, w.ObjectiveID)
		}
	}

	for _, wr := range workflowRows {
		w := gormstore.WorkflowFromRow(wr)
		if !workflowsWithItem[w.ID] {
			s.add(models.IssueSeverityError, "workflow", w.ID, "Workflow %s %q has no Work Items", w.Code, w.Name)
		}
	}
	return nil
}

// checkEventFlowSteps flags a step whose WorkflowCode matches no Workflow.
// The join is by business key, never by row id, so nothing at write time
// enforces it.
func (s *issueScan) checkEventFlowSteps() error {
	workflowCodes, err := s.codeSet(s.tables.Workflows)
	if err != nil {
		return err
	}
	var stepRows []gormstore.EventFlowStepRow
	if err := s.scoped(s.tables.EventFlowSteps).Find(&stepRows).Error; err != nil {
		return err
	}
	for _, sr := range stepRows {
		step := gormstore.EventFlowStepFromRow(sr)
		if step.WorkflowCode == "" || workflowCodes[step.WorkflowCode] {
			continue
		}
		s.add(models.IssueSeverityError, "event_flow_step", step.ID,
			"Event flow step %s %q refers to Workflow code %q, which does not exist", step.Code, step.Name, step.WorkflowCode)
	}
	return nil
}

// checkAttachments flags a Value/Rule attachment whose TargetCode no
// longer resolves. Attachments are never Draft-scoped (AGD-003) but their
// targets are, and a Code survives promotion unchanged (AG16) — so the
// target is resolved against whichever scope is being scanned, which also
// surfaces "this Draft deletes something a Rule is attached to" before a
// promotion makes it true of live.
func (s *issueScan) checkAttachments() error {
	codesByType := make(map[models.TargetType]map[string]bool, 5)
	for _, t := range []models.TargetType{
		models.TargetTypeRole, models.TargetTypeGoal, models.TargetTypeObjective,
		models.TargetTypeWorkflow, models.TargetTypeWorkItem,
	} {
		table, ok := targetTableFor(s.tables, t)
		if !ok {
			continue
		}
		codes, err := s.codeSet(table)
		if err != nil {
			return err
		}
		codesByType[t] = codes
	}

	var valueAttachments []gormstore.ValueAttachmentRow
	if err := s.tx.WithContext(s.ctx).Table(s.tables.ValueAttachments).Find(&valueAttachments).Error; err != nil {
		return err
	}
	for _, ar := range valueAttachments {
		a := gormstore.ValueAttachmentFromRow(ar)
		if codesByType[a.TargetType][a.TargetCode] {
			continue
		}
		s.add(models.IssueSeverityError, "value_attachment", a.ID,
			"Value %s is attached to %s %s, which does not exist", a.ValueCode, a.TargetType, a.TargetCode)
	}

	var ruleAttachments []gormstore.RuleAttachmentRow
	if err := s.tx.WithContext(s.ctx).Table(s.tables.RuleAttachments).Find(&ruleAttachments).Error; err != nil {
		return err
	}
	for _, ar := range ruleAttachments {
		a := gormstore.RuleAttachmentFromRow(ar)
		if codesByType[a.TargetType][a.TargetCode] {
			continue
		}
		s.add(models.IssueSeverityError, "rule_attachment", a.ID,
			"Rule %s is attached to %s %s, which does not exist", a.RuleCode, a.TargetType, a.TargetCode)
	}
	return nil
}

// targetTableFor is [agencyManager.targetTable]'s table half, taken as a
// plain TableNames so the scan can call it without a manager.
func targetTableFor(tables TableNames, t models.TargetType) (string, bool) {
	switch t {
	case models.TargetTypeRole:
		return tables.Roles, true
	case models.TargetTypeGoal:
		return tables.Goals, true
	case models.TargetTypeObjective:
		return tables.Objectives, true
	case models.TargetTypeWorkflow:
		return tables.Workflows, true
	case models.TargetTypeWorkItem:
		return tables.WorkItems, true
	default:
		return "", false
	}
}

// ListIssues re-scans draftID's scope before returning: it deletes that
// scope's existing Issue rows, re-runs [scanIssues] against its current
// data, re-inserts the result, then reads it back through filter. draftID
// is "" for the live Agency, matching every other scoped List.
func (m *agencyManager) ListIssues(ctx context.Context, draftID string, filter IssueFilter) ([]models.Issue, error) {
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_, err := rescanIssues(ctx, tx, m.tables, draftID)
		return err
	})
	if err != nil {
		return nil, err
	}

	q := scopeQuery(m.db.WithContext(ctx).Table(m.tables.Issues), draftID).Order("created_at")
	q = likeFilter(q, "message", filter.Message)
	q = eqFilter(q, "severity", filter.Severity)
	q = eqFilter(q, "entity_type", filter.EntityType)
	var rows []gormstore.IssueRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListIssues: %w", err)
	}
	out := make([]models.Issue, len(rows))
	for i, r := range rows {
		out[i] = gormstore.IssueFromRow(r)
	}
	return out, nil
}

// rescanIssues replaces one scope's stored Issue rows with a fresh scan,
// returning what it found. Shared by ListIssues and PublishAgency's
// validation gate so both read the same rules from the same place.
func rescanIssues(ctx context.Context, tx *gorm.DB, tables TableNames, draftID string) ([]models.Issue, error) {
	del := scopeQuery(tx.WithContext(ctx).Table(tables.Issues), draftID)
	if err := del.Delete(&gormstore.IssueRow{}).Error; err != nil {
		return nil, fmt.Errorf("rescanIssues: %w", err)
	}
	found, err := scanIssues(ctx, tx, tables, draftID)
	if err != nil {
		return nil, err
	}
	for _, issue := range found {
		row := gormstore.IssueToRow(issue)
		if err := tx.WithContext(ctx).Table(tables.Issues).Create(&row).Error; err != nil {
			return nil, fmt.Errorf("rescanIssues: %w", err)
		}
	}
	return found, nil
}
