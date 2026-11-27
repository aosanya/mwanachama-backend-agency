package mwanachamaagency_test

import (
	"context"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func paymentRowInFork(t *testing.T, o routedOrder, forkID, orderID, amount string) models.RecordRow {
	t.Helper()
	ctx := context.Background()
	records, err := o.world.mgr.ListRecords(ctx, forkID, mwanachamaagency.RecordFilter{})
	if err != nil {
		t.Fatalf("ListRecords: %v", err)
	}
	roles, err := o.world.mgr.ListRoles(ctx, forkID, mwanachamaagency.RoleFilter{})
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	row := models.RecordRow{
		DraftID: forkID, AddedBy: "member-1", OrderID: orderID,
		Answers: map[string]string{"Select the farmer": "Wanjiru", "Enter the amount paid": amount},
	}
	for _, record := range records {
		if record.Identity() == o.record.Identity() {
			row.RecordID = record.ID
		}
	}
	for _, role := range roles {
		if role.Name == coordinator {
			row.RoleID = role.ID
		}
	}
	if row.RecordID == "" || row.RoleID == "" {
		t.Fatalf("the fork has no copy of %s or %s", o.record.Name, coordinator)
	}
	return row
}

func forkOf(t *testing.T, o routedOrder) string {
	t.Helper()
	fork, err := o.world.mgr.CreateDraft(context.Background(), "next version", o.world.draftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	return fork.ID
}

func TestRecordRow_ASubjectRowOfAnEarlierDraftIsTheSubjectOfARowInALaterOne(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	subject, err := o.world.mgr.CreateRecordRow(ctx, paymentRow(o, "", "Otieno", "10"))
	if err != nil {
		t.Fatalf("subject row: %v", err)
	}
	child, err := o.world.mgr.CreateRecordRow(ctx, subjectedRow(o, "run-a", subject.ID, "100"))
	if err != nil {
		t.Fatalf("child row: %v", err)
	}
	fork := forkOf(t, o)

	fix := paymentRowInFork(t, o, fork, "", "150")
	fix.Replaces = child.ID
	made, err := o.world.mgr.CreateRecordRow(ctx, fix)
	if err != nil || made.SubjectRowID != subject.ID {
		t.Fatalf("an edit in the later draft of a child of an earlier draft = %+v, %v; want subject %q", made, err, subject.ID)
	}
}

func TestRecordRow_ARunStartedInAnEarlierDraftKeepsItsSubjectInALaterOne(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	subject, err := o.world.mgr.CreateRecordRow(ctx, paymentRow(o, "", "Otieno", "10"))
	if err != nil {
		t.Fatalf("subject row: %v", err)
	}
	if _, err := o.world.mgr.CreateRecordRow(ctx, subjectedRow(o, "run-a", subject.ID, "100")); err != nil {
		t.Fatalf("first row of the run: %v", err)
	}
	fork := forkOf(t, o)

	later, err := o.world.mgr.CreateRecordRow(ctx, paymentRowInFork(t, o, fork, "run-a", "250"))
	if err != nil || later.SubjectRowID != subject.ID {
		t.Fatalf("a later row of the run in the later draft = %+v, %v; want subject %q", later, err, subject.ID)
	}
}
