package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCreateValue_RequiresText(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.CreateValue(context.Background(), models.Value{}); !errors.Is(err, mwanachamaagency.ErrInvalidValue) {
		t.Fatalf("expected ErrInvalidValue, got %v", err)
	}
}

// TestValue_AgencyWideNeverForkable asserts a Value with no attachment is
// permanently immutable: CreateDraft never copies it, and Update/Delete
// against the live row are refused.
func TestValue_AgencyWideNeverForkable(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, _, _, _, _ := seedLiveAgency(t, mgr, ctx)

	v, err := mgr.CreateValue(ctx, models.Value{Text: "we prioritize correctness over speed"})
	if err != nil {
		t.Fatalf("CreateValue: %v", err)
	}

	v.Text = "changed"
	if _, err := mgr.UpdateValue(ctx, v); !errors.Is(err, mwanachamaagency.ErrCannotEditLive) {
		t.Fatalf("expected ErrCannotEditLive, got %v", err)
	}

	d, err := mgr.CreateDraft(ctx, "try to edit the value", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftValues, err := mgr.ListValues(ctx, d.ID, mwanachamaagency.ValueFilter{})
	if err != nil {
		t.Fatalf("ListValues(draft): %v", err)
	}
	if len(draftValues) != 0 {
		t.Fatalf("expected an Agency-wide Value to never fork into a Draft, got %+v", draftValues)
	}
}

func TestCreateValueAttachment_InvalidTargetType(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	v, err := mgr.CreateValue(ctx, models.Value{Text: "x"})
	if err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	_, err = mgr.CreateValueAttachment(ctx, v.ID, models.TargetType("bogus"), "any")
	if !errors.Is(err, mwanachamaagency.ErrInvalidTargetType) {
		t.Fatalf("expected ErrInvalidTargetType, got %v", err)
	}
}

func TestCreateValueAttachment_TargetNotFound(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	v, err := mgr.CreateValue(ctx, models.Value{Text: "x"})
	if err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	_, err = mgr.CreateValueAttachment(ctx, v.ID, models.TargetTypeRole, "unknown-role")
	if !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestCreateValueAttachment_Duplicate(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	v, err := mgr.CreateValue(ctx, models.Value{Text: "x"})
	if err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	role, err := mgr.CreateRole(ctx, models.Role{Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := mgr.CreateValueAttachment(ctx, v.ID, models.TargetTypeRole, role.ID); err != nil {
		t.Fatalf("CreateValueAttachment: %v", err)
	}
	if _, err := mgr.CreateValueAttachment(ctx, v.ID, models.TargetTypeRole, role.ID); !errors.Is(err, mwanachamaagency.ErrDuplicateAttachment) {
		t.Fatalf("expected ErrDuplicateAttachment, got %v", err)
	}
}

// TestValueAttachment_Lifecycle covers attach, list-by-value,
// filter-by-target, and detach — all live-editable, no Draft involved.
func TestValueAttachment_Lifecycle(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	v, err := mgr.CreateValue(ctx, models.Value{Text: "never touch production data"})
	if err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	role, err := mgr.CreateRole(ctx, models.Role{Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	att, err := mgr.CreateValueAttachment(ctx, v.ID, models.TargetTypeRole, role.ID)
	if err != nil {
		t.Fatalf("CreateValueAttachment: %v", err)
	}
	if att.ValueCode != v.Code || att.TargetCode != role.Code || att.TargetType != models.TargetTypeRole {
		t.Fatalf("unexpected attachment: %+v", att)
	}

	atts, err := mgr.ListValueAttachments(ctx, v.ID)
	if err != nil {
		t.Fatalf("ListValueAttachments: %v", err)
	}
	if len(atts) != 1 || atts[0].ID != att.ID {
		t.Fatalf("expected exactly the one attachment, got %+v", atts)
	}

	found, err := mgr.ListValues(ctx, "", mwanachamaagency.ValueFilter{RoleID: role.ID})
	if err != nil {
		t.Fatalf("ListValues(RoleID filter): %v", err)
	}
	if len(found) != 1 || found[0].ID != v.ID {
		t.Fatalf("expected filter to find the attached Value, got %+v", found)
	}

	notFound, err := mgr.ListValues(ctx, "", mwanachamaagency.ValueFilter{GoalID: "unknown-goal-id"})
	if err != nil {
		t.Fatalf("ListValues(unknown GoalID filter): %v", err)
	}
	if len(notFound) != 0 {
		t.Fatalf("expected an unresolvable target filter to yield no results, got %+v", notFound)
	}

	if err := mgr.DeleteValueAttachment(ctx, att.ID); err != nil {
		t.Fatalf("DeleteValueAttachment: %v", err)
	}
	if err := mgr.DeleteValueAttachment(ctx, att.ID); !errors.Is(err, mwanachamaagency.ErrValueAttachmentNotFound) {
		t.Fatalf("expected ErrValueAttachmentNotFound on repeat delete, got %v", err)
	}
	afterDelete, err := mgr.ListValues(ctx, "", mwanachamaagency.ValueFilter{RoleID: role.ID})
	if err != nil {
		t.Fatalf("ListValues after detach: %v", err)
	}
	if len(afterDelete) != 0 {
		t.Fatalf("expected no Values attached to the role after detach, got %+v", afterDelete)
	}
}

// TestValue_AttachedForksAndSurvivesPromote is the key correctness case for
// AGD-003's Code-based attachment design: a Value attached to a WorkItem
// forks into a Draft (its Text is edited there), and after PromoteDraft
// replaces the live row with a brand-new id, the attachment — created
// before the fork ever happened — still resolves against the new live row,
// because it was always keyed by Code, not id.
func TestValue_AttachedForksAndSurvivesPromote(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, _, _, _, wi := seedLiveAgency(t, mgr, ctx)

	v, err := mgr.CreateValue(ctx, models.Value{Text: "ship it fast"})
	if err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	if _, err := mgr.CreateValueAttachment(ctx, v.ID, models.TargetTypeWorkItem, wi.ID); err != nil {
		t.Fatalf("CreateValueAttachment: %v", err)
	}

	d, err := mgr.CreateDraft(ctx, "reword the value", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftValues, err := mgr.ListValues(ctx, d.ID, mwanachamaagency.ValueFilter{})
	if err != nil {
		t.Fatalf("ListValues(draft): %v", err)
	}
	if len(draftValues) != 1 {
		t.Fatalf("expected the attached Value to fork into the draft, got %+v", draftValues)
	}
	draftV := draftValues[0]
	if draftV.ID == v.ID {
		t.Fatal("expected the draft-scoped copy to have a new id")
	}
	draftV.Text = "ship it fast, but correctly"
	if _, err := mgr.UpdateValue(ctx, draftV); err != nil {
		t.Fatalf("UpdateValue(draft copy): %v", err)
	}

	if _, err := mgr.PromoteDraft(ctx, d.ID); err != nil {
		t.Fatalf("PromoteDraft: %v", err)
	}

	live, err := mgr.ListValues(ctx, "", mwanachamaagency.ValueFilter{})
	if err != nil {
		t.Fatalf("ListValues(live): %v", err)
	}
	if len(live) != 1 {
		t.Fatalf("expected exactly 1 live Value after promotion, got %d", len(live))
	}
	if live[0].ID == v.ID {
		t.Fatal("expected the old live Value id to have been replaced, not reused")
	}
	if live[0].Code != v.Code {
		t.Fatalf("expected Code to survive promotion, got %q want %q", live[0].Code, v.Code)
	}
	if live[0].Text != "ship it fast, but correctly" {
		t.Fatalf("expected the promoted Text, got %q", live[0].Text)
	}

	atts, err := mgr.ListValueAttachments(ctx, live[0].ID)
	if err != nil {
		t.Fatalf("ListValueAttachments(new live id): %v", err)
	}
	if len(atts) != 1 {
		t.Fatalf("expected the attachment created before the fork to still resolve, got %+v", atts)
	}
	if atts[0].TargetType != models.TargetTypeWorkItem {
		t.Fatalf("unexpected attachment after promote: %+v", atts[0])
	}
}
