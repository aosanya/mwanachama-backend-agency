package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func paymentRow(o routedOrder, orderID, farmer, amount string) models.RecordRow {
	return models.RecordRow{
		DraftID: o.world.draftID, RecordID: o.record.ID, RoleID: o.world.roles[coordinator].ID, AddedBy: "member-1",
		OrderID: orderID, Answers: map[string]string{"Select the farmer": farmer, "Enter the amount paid": amount},
	}
}

func TestRecordRow_KeepsTheOrderItWasAddedFromAndListsByIt(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	for _, in := range []models.RecordRow{
		paymentRow(o, "order-a", "Wanjiru", "100"),
		paymentRow(o, "order-b", "Wanjiru", "200"),
		paymentRow(o, "", "Otieno", "300"),
	} {
		if _, err := o.world.mgr.CreateRecordRow(ctx, in); err != nil {
			t.Fatalf("CreateRecordRow: %v", err)
		}
	}
	forA, err := o.world.mgr.ListRecordRows(ctx, o.world.draftID, mwanachamaagency.RecordRowFilter{RecordCode: o.record.Code, OrderID: "order-a"})
	if err != nil || len(forA) != 1 || forA[0].OrderID != "order-a" || forA[0].Answers["Enter the amount paid"] != "100" {
		t.Fatalf("rows of order-a = %+v, %v", forA, err)
	}
	all, err := o.world.mgr.ListRecordRows(ctx, o.world.draftID, mwanachamaagency.RecordRowFilter{RecordCode: o.record.Code})
	if err != nil || len(all) != 3 {
		t.Fatalf("every row = %+v, %v", all, err)
	}
}

func TestRecordRow_AReplacingRowKeepsTheOldOneAndNamesTheCorrectionDate(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	old, err := o.world.mgr.CreateRecordRow(ctx, paymentRow(o, "order-a", "Wanjiru", "100"))
	if err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	fix := paymentRow(o, "", "Wanjiru", "150")
	fix.Replaces = old.ID
	made, err := o.world.mgr.CreateRecordRow(ctx, fix)
	if err != nil {
		t.Fatalf("replacing row: %v", err)
	}
	if made.Replaces != old.ID || made.OrderID != "order-a" || made.CorrectedOn == "" || made.CorrectedOn != made.CreatedAt {
		t.Errorf("replacing row = %+v", made)
	}
	if old.CorrectedOn != "" {
		t.Errorf("the first row corrects nothing, got corrected_on %q", old.CorrectedOn)
	}
	rows, err := o.world.mgr.ListRecordRows(ctx, o.world.draftID, mwanachamaagency.RecordRowFilter{RecordCode: o.record.Code})
	if err != nil || len(rows) != 2 {
		t.Fatalf("the old row stays readable: %+v, %v", rows, err)
	}
}

func TestRecordRow_AReplacementIsRefusedWhenItDoesNotFit(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	old, err := o.world.mgr.CreateRecordRow(ctx, paymentRow(o, "order-a", "Wanjiru", "100"))
	if err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	cases := map[string]func(*models.RecordRow){
		"a row that does not exist": func(r *models.RecordRow) { r.Replaces = "no-such-row" },
		"another order":             func(r *models.RecordRow) { r.Replaces = old.ID; r.OrderID = "order-b" },
	}
	for name, mutate := range cases {
		in := paymentRow(o, "", "Wanjiru", "175")
		mutate(&in)
		if _, err := o.world.mgr.CreateRecordRow(ctx, in); !errors.Is(err, mwanachamaagency.ErrInvalidRecordRow) {
			t.Errorf("replacing %s: want ErrInvalidRecordRow, got %v", name, err)
		}
	}
	first := paymentRow(o, "", "Wanjiru", "175")
	first.Replaces = old.ID
	if _, err := o.world.mgr.CreateRecordRow(ctx, first); err != nil {
		t.Fatalf("first correction: %v", err)
	}
	second := paymentRow(o, "", "Wanjiru", "190")
	second.Replaces = old.ID
	if _, err := o.world.mgr.CreateRecordRow(ctx, second); !errors.Is(err, mwanachamaagency.ErrInvalidRecordRow) {
		t.Errorf("correcting a row twice: want ErrInvalidRecordRow, got %v", err)
	}
}

func TestRecordRow_TheSameFarmerAndAmountTwiceWithinAMinuteIsRefused(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	defer mwanachamaagency.SetRowClock(func() time.Time { return now })()

	if _, err := o.world.mgr.CreateRecordRow(ctx, paymentRow(o, "order-a", "Wanjiru", "100")); err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	now = now.Add(59 * time.Second)
	if _, err := o.world.mgr.CreateRecordRow(ctx, paymentRow(o, "order-a", "Wanjiru", "100")); !errors.Is(err, mwanachamaagency.ErrDuplicateRecordRow) {
		t.Fatalf("the same payment again: want ErrDuplicateRecordRow, got %v", err)
	}
	if _, err := o.world.mgr.CreateRecordRow(ctx, paymentRow(o, "order-b", "Wanjiru", "100")); err != nil {
		t.Errorf("the same farmer and amount on another order: %v", err)
	}
	if _, err := o.world.mgr.CreateRecordRow(ctx, paymentRow(o, "order-a", "Wanjiru", "101")); err != nil {
		t.Errorf("another amount: %v", err)
	}
	now = now.Add(2 * time.Second)
	if _, err := o.world.mgr.CreateRecordRow(ctx, paymentRow(o, "order-a", "Wanjiru", "100")); err != nil {
		t.Errorf("the same payment after a minute: %v", err)
	}
}

func TestRecordRow_ARowThatWasCorrectedAwayCanBeSavedAgain(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	defer mwanachamaagency.SetRowClock(func() time.Time { return now })()

	old, err := o.world.mgr.CreateRecordRow(ctx, paymentRow(o, "order-a", "Wanjiru", "100"))
	if err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	now = now.Add(10 * time.Second)
	fix := paymentRow(o, "", "Wanjiru", "150")
	fix.Replaces = old.ID
	if _, err := o.world.mgr.CreateRecordRow(ctx, fix); err != nil {
		t.Fatalf("correction: %v", err)
	}
	now = now.Add(10 * time.Second)
	if _, err := o.world.mgr.CreateRecordRow(ctx, paymentRow(o, "order-a", "Wanjiru", "100")); err != nil {
		t.Errorf("saving the corrected-away payment as a new one: %v", err)
	}
}
