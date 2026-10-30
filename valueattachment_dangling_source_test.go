package mwanachamaagency_test

import (
	"context"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAG64_DeletingAnAttachedValueTakesItsAttachmentsWithIt(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)

	v, err := mgr.CreateValue(ctx, models.Value{DraftID: draftID, Text: "never touch production data"})
	if err != nil {
		t.Fatalf("precondition: CreateValue: %v", err)
	}
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("precondition: CreateRole: %v", err)
	}
	if _, err := mgr.CreateValueAttachment(ctx, v.ID, models.TargetTypeRole, role.ID); err != nil {
		t.Fatalf("precondition: CreateValueAttachment: %v", err)
	}

	if err := mgr.DeleteValue(ctx, v.ID); err != nil {
		t.Fatalf("precondition: DeleteValue (allowed on an attached Value since AGD-006): %v", err)
	}

	var remaining int64
	if err := db.WithContext(ctx).Table(tableOf(t, s, "value_attachment")).
		Where("value_code = ?", v.Code).Count(&remaining).Error; err != nil {
		t.Fatalf("counting value_attachment rows: %v", err)
	}

	if remaining > 0 {
		t.Errorf("AG64: DeleteValue removed the Value row for code %s but left %d value_attachment row(s) pointing at it — nothing in the delete path, the Issue scan (checkAttachments only validates TargetCode, never ValueCode/RuleCode) or PublishAgency notices the dangling link; want DeleteValue to cascade-delete its own attachments, the same way DeleteRole already cleans up RoleAssignment and DeleteGoal/DeleteObjective already clean up WorkItem/RoleAssignment (AG27)", v.Code, remaining)
	}
}

func TestAG64_DeletingAnAttachedRuleTakesItsAttachmentsWithIt(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)

	r, err := mgr.CreateRule(ctx, models.Rule{DraftID: draftID, Text: "review before merge"})
	if err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := mgr.CreateRuleAttachment(ctx, r.ID, models.TargetTypeRole, role.ID); err != nil {
		t.Fatalf("CreateRuleAttachment: %v", err)
	}
	if err := mgr.DeleteRule(ctx, r.ID); err != nil {
		t.Fatalf("DeleteRule: %v", err)
	}

	var remaining int64
	if err := db.WithContext(ctx).Table(tableOf(t, s, "rule_attachment")).
		Where("rule_code = ?", r.Code).Count(&remaining).Error; err != nil {
		t.Fatalf("counting rule_attachment rows: %v", err)
	}
	if remaining > 0 {
		t.Errorf("DeleteRule left %d rule_attachment row(s) pointing at %s", remaining, r.Code)
	}
}

func TestAG64_TheIssueScanNamesAnAttachmentWhoseValueIsGone(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)

	role, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	row := map[string]any{
		"id": "orphan-1", "value_code": "V-404", "target_type": string(models.TargetTypeRole),
		"target_code": role.Code, "created_at": models.NowRFC3339(),
	}
	if err := db.WithContext(ctx).Table(tableOf(t, s, "value_attachment")).Create(row).Error; err != nil {
		t.Fatalf("planting the dangling attachment: %v", err)
	}

	issues, err := mgr.ListIssues(ctx, draftID, mwanachamaagency.IssueFilter{EntityType: "value_attachment"})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(issues) != 1 || issues[0].Severity != models.IssueSeverityError {
		t.Fatalf("want one error naming the attachment whose Value V-404 is gone, got %+v", issues)
	}
}
