package mwanachamaagency_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func subjectedRow(o routedOrder, orderID, subjectRowID, amount string) models.RecordRow {
	row := paymentRow(o, orderID, "Wanjiru", amount)
	row.SubjectRowID = subjectRowID
	return row
}

func TestRecordRow_KeepsTheSubjectRowItsRunWasStartedOnAndListsByIt(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	subject, err := o.world.mgr.CreateRecordRow(ctx, paymentRow(o, "", "Otieno", "10"))
	if err != nil {
		t.Fatalf("subject row: %v", err)
	}
	made, err := o.world.mgr.CreateRecordRow(ctx, subjectedRow(o, "run-a", subject.ID, "100"))
	if err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	if made.SubjectRowID != subject.ID {
		t.Errorf("created row subject_row_id = %q, want %q", made.SubjectRowID, subject.ID)
	}
	if _, err := o.world.mgr.CreateRecordRow(ctx, paymentRow(o, "run-b", "Otieno", "200")); err != nil {
		t.Fatalf("a row with no subject: %v", err)
	}
	got, err := o.world.mgr.ListRecordRows(ctx, o.world.draftID, mwanachamaagency.RecordRowFilter{SubjectRowID: subject.ID})
	if err != nil || len(got) != 1 || got[0].ID != made.ID || got[0].SubjectRowID != subject.ID {
		t.Fatalf("rows of the subject = %+v, %v", got, err)
	}
}

func TestRecordRow_ARowWithNoSubjectBehavesAsToday(t *testing.T) {
	o := buildRoutedOrder(t)
	made, err := o.world.mgr.CreateRecordRow(context.Background(), paymentRow(o, "run-a", "Wanjiru", "100"))
	if err != nil || made.SubjectRowID != "" {
		t.Fatalf("a row with no subject = %+v, %v", made, err)
	}
}

func TestRecordRow_ASubjectRowTheDraftDoesNotKeepIsRefusedNamingIt(t *testing.T) {
	o := buildRoutedOrder(t)
	_, err := o.world.mgr.CreateRecordRow(context.Background(), subjectedRow(o, "run-a", "no-such-row", "100"))
	if !errors.Is(err, mwanachamaagency.ErrInvalidRecordRow) || !strings.Contains(err.Error(), "no-such-row") {
		t.Fatalf("want ErrInvalidRecordRow naming no-such-row, got %v", err)
	}
}

func TestRecordRow_EveryRowOfARunCarriesTheSubjectTheRunWasStartedOn(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	subject, err := o.world.mgr.CreateRecordRow(ctx, paymentRow(o, "", "Otieno", "10"))
	if err != nil {
		t.Fatalf("subject row: %v", err)
	}
	if _, err := o.world.mgr.CreateRecordRow(ctx, subjectedRow(o, "run-a", subject.ID, "100")); err != nil {
		t.Fatalf("first row of the run: %v", err)
	}
	later, err := o.world.mgr.CreateRecordRow(ctx, paymentRow(o, "run-a", "Wanjiru", "250"))
	if err != nil {
		t.Fatalf("a later row of the run: %v", err)
	}
	if later.SubjectRowID != subject.ID {
		t.Errorf("a later row of the run carries %q, want %q", later.SubjectRowID, subject.ID)
	}
}

func TestRecordRow_ARunIsNotMovedToAnotherSubjectRow(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	first, err := o.world.mgr.CreateRecordRow(ctx, paymentRow(o, "", "Otieno", "10"))
	if err != nil {
		t.Fatalf("first subject row: %v", err)
	}
	second, err := o.world.mgr.CreateRecordRow(ctx, paymentRow(o, "", "Otieno", "20"))
	if err != nil {
		t.Fatalf("second subject row: %v", err)
	}
	if _, err := o.world.mgr.CreateRecordRow(ctx, subjectedRow(o, "run-a", first.ID, "100")); err != nil {
		t.Fatalf("first row of the run: %v", err)
	}
	_, err = o.world.mgr.CreateRecordRow(ctx, subjectedRow(o, "run-a", second.ID, "300"))
	if !errors.Is(err, mwanachamaagency.ErrInvalidRecordRow) {
		t.Fatalf("a row naming another subject for the same run: want ErrInvalidRecordRow, got %v", err)
	}
}

func TestRecordRow_ARunOpenBeforeSubjectsExistedKeepsWorking(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	if _, err := o.world.mgr.CreateRecordRow(ctx, paymentRow(o, "run-a", "Wanjiru", "100")); err != nil {
		t.Fatalf("an older row of the run: %v", err)
	}
	later, err := o.world.mgr.CreateRecordRow(ctx, paymentRow(o, "run-a", "Wanjiru", "250"))
	if err != nil || later.SubjectRowID != "" {
		t.Fatalf("a later row of an older run = %+v, %v", later, err)
	}
}

func TestRecordRow_AReplacingRowKeepsTheSubjectRowOfTheRowItReplaces(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	subject, err := o.world.mgr.CreateRecordRow(ctx, paymentRow(o, "", "Otieno", "10"))
	if err != nil {
		t.Fatalf("subject row: %v", err)
	}
	old, err := o.world.mgr.CreateRecordRow(ctx, subjectedRow(o, "run-a", subject.ID, "100"))
	if err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	fix := paymentRow(o, "", "Wanjiru", "150")
	fix.Replaces = old.ID
	made, err := o.world.mgr.CreateRecordRow(ctx, fix)
	if err != nil || made.SubjectRowID != subject.ID {
		t.Fatalf("replacing row = %+v, %v; want subject %q", made, err, subject.ID)
	}
}
