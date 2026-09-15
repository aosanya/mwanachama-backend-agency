package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

// targetFilters is the five ID-based fields ValueFilter/RuleFilter share —
// "narrow to whatever is attached to this Role/Goal/Objective/Workflow/
// WorkItem".
type targetFilters struct {
	RoleID, GoalID, ObjectiveID, WorkflowID, WorkItemID string
}

// codeOnly reads just the Code column off whichever business-entity table
// a query targets.
type codeOnly struct {
	Code string
}

// resolveCodeByID looks up id's Code in table — any scope, live or a
// Draft-scoped copy, since Code is shared between them (AG16). Returns
// notFoundErr if no row matches.
func resolveCodeByID(ctx context.Context, tx *gorm.DB, table, id string, notFoundErr error) (string, error) {
	var row codeOnly
	err := tx.WithContext(ctx).Table(table).Select("code").Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", notFoundErr
	}
	if err != nil {
		return "", fmt.Errorf("resolveCodeByID: %w", err)
	}
	return row.Code, nil
}

// targetTable maps a TargetType to the table it's stored in and the
// sentinel error to return when an id names no row there.
func (m *agencyManager) targetTable(t models.TargetType) (table string, notFound error, ok bool) {
	switch t {
	case models.TargetTypeRole:
		return m.tables.Roles, ErrRoleNotFound, true
	case models.TargetTypeGoal:
		return m.tables.Goals, ErrGoalNotFound, true
	case models.TargetTypeObjective:
		return m.tables.Objectives, ErrObjectiveNotFound, true
	case models.TargetTypeWorkflow:
		return m.tables.Workflows, ErrWorkflowNotFound, true
	case models.TargetTypeWorkItem:
		return m.tables.WorkItems, ErrWorkItemNotFound, true
	default:
		return "", nil, false
	}
}

// resolveTargetCode validates targetType and resolves targetID to its
// Code within that type's table.
func (m *agencyManager) resolveTargetCode(ctx context.Context, tx *gorm.DB, targetType models.TargetType, targetID string) (string, error) {
	table, notFound, ok := m.targetTable(targetType)
	if !ok {
		return "", ErrInvalidTargetType
	}
	return resolveCodeByID(ctx, tx, table, targetID, notFound)
}

// applyTargetFilters chains an "attached to this target" subquery onto q
// for each non-empty field in f, resolving each caller-supplied id to its
// Code first (an attachment row stores Code, not id — see
// [models.ValueAttachment]'s doc comment). codeColumn is the attachments
// table's own FK column ("value_code" or "rule_code"). If any given id
// names no row at all, the filter can never match anything — matchable
// comes back false so the caller returns an empty result instead of
// running a query, the same way every other filter here treats "narrows
// to nothing" as an empty list, not an error.
func (m *agencyManager) applyTargetFilters(ctx context.Context, q *gorm.DB, attachmentsTable, codeColumn string, f targetFilters) (out *gorm.DB, matchable bool, err error) {
	targets := []struct {
		id         string
		targetType models.TargetType
	}{
		{f.RoleID, models.TargetTypeRole},
		{f.GoalID, models.TargetTypeGoal},
		{f.ObjectiveID, models.TargetTypeObjective},
		{f.WorkflowID, models.TargetTypeWorkflow},
		{f.WorkItemID, models.TargetTypeWorkItem},
	}
	for _, t := range targets {
		if t.id == "" {
			continue
		}
		table, notFound, _ := m.targetTable(t.targetType)
		code, err := resolveCodeByID(ctx, m.db.WithContext(ctx), table, t.id, notFound)
		if errors.Is(err, notFound) {
			return q, false, nil
		}
		if err != nil {
			return q, false, err
		}
		q = q.Where("code IN (SELECT "+codeColumn+" FROM "+attachmentsTable+
			" WHERE target_type = ? AND target_code = ?)", string(t.targetType), code)
	}
	return q, true, nil
}
