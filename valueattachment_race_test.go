package mwanachamaagency_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAG63_OpenBug_TwoValueAttachmentsLinkTheSameValueAndTarget(t *testing.T) {
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

	insertErr := db.WithContext(ctx).Table(tableOf(t, s, "value_attachment")).Create(map[string]any{
		"id":          "concurrent-attachment",
		"value_code":  v.Code,
		"target_type": string(models.TargetTypeRole),
		"target_code": role.Code,
		"created_at":  models.NowRFC3339(),
	}).Error

	list, err := mgr.ListValueAttachments(ctx, v.ID)
	if err != nil {
		t.Fatalf("ListValueAttachments: %v", err)
	}

	if insertErr == nil && len(list) > 1 {
		t.Errorf("AG63 open bug: a second value_attachment row linking value %s to role %s, written as a concurrent CreateValueAttachment writes it after both counted zero, was accepted by the store (insert error %v) and the value now holds %d attachments to the same target; want the store to refuse the second row, the same way CreateValueAttachment's own pre-check means to", v.Code, role.Code, insertErr, len(list))
	}
}
