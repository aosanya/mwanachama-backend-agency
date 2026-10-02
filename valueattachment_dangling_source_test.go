package mwanachamaagency_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAG64_OpenBug_DeletingAnAttachedValueOrphansItsAttachment(t *testing.T) {
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
		t.Errorf("AG64 open bug: DeleteValue removed the Value row for code %s but left %d value_attachment row(s) pointing at it — nothing in the delete path, the Issue scan (checkAttachments only validates TargetCode, never ValueCode/RuleCode) or PublishAgency notices the dangling link; want DeleteValue to cascade-delete its own attachments, the same way DeleteRole already cleans up RoleAssignment and DeleteGoal/DeleteObjective already clean up WorkItem/RoleAssignment (AG27)", v.Code, remaining)
	}
}
