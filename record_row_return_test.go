package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func sentRow(o routedOrder, orderID, key string) models.RecordRow {
	row := paymentRow(o, orderID, "Wanjiru", "100")
	row.SendKey = key
	return row
}

func TestReturnRecordRow_MarksTheRowReturnedWithWhoWhyAndWhenAndKeepsItInHistory(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	made, err := o.world.mgr.CreateRecordRow(ctx, sentRow(o, "order-a", "step-1:order-a"))
	if err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	returned, err := o.world.mgr.ReturnRecordRow(ctx, made.ID, "member-2", "The amount is not in the budget")
	if err != nil {
		t.Fatalf("ReturnRecordRow: %v", err)
	}
	if returned.ID != made.ID || returned.ReturnedBy != "member-2" || returned.ReturnedReason != "The amount is not in the budget" || returned.ReturnedOn == "" || returned.Deleted {
		t.Errorf("returned row = %+v", returned)
	}
	if got := rowsOf(t, o, mwanachamaagency.RecordRowFilter{OrderID: "order-a"}); len(got) != 0 {
		t.Errorf("a returned row no longer counts as done, yet the default listing holds %+v", got)
	}
	kept := rowsOf(t, o, mwanachamaagency.RecordRowFilter{OrderID: "order-a", IncludeReturned: true})
	if len(kept) != 1 || kept[0].ID != made.ID || kept[0].ReturnedBy != "member-2" || kept[0].ReturnedReason != "The amount is not in the budget" || kept[0].ReturnedOn == "" || answerTo(o.record, kept[0], "Enter the amount paid") != "100" {
		t.Errorf("the returned row stays in history with its answers and its reason: %+v", kept)
	}
}

func TestReturnRecordRow_FreesTheSendKeyOfItsStepAndRun(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	mgr := o.world.mgr
	first, err := mgr.CreateRecordRow(ctx, sentRow(o, "order-a", "step-1:order-a"))
	if err != nil {
		t.Fatalf("first send: %v", err)
	}
	if _, err := mgr.CreateRecordRow(ctx, paymentRow(o, "order-a", "Otieno", "200")); err != nil {
		t.Fatalf("a keyless row of the same order: %v", err)
	}
	again := paymentRow(o, "order-a", "Achieng", "300")
	again.SendKey = "step-1:order-a"
	if _, err := mgr.CreateRecordRow(ctx, again); !errors.Is(err, mwanachamaagency.ErrStepAlreadySent) {
		t.Fatalf("before the return, a second send of the step = %v, want ErrStepAlreadySent", err)
	}
	if _, err := mgr.ReturnRecordRow(ctx, first.ID, "member-2", "Check the amount"); err != nil {
		t.Fatalf("ReturnRecordRow: %v", err)
	}
	resent, err := mgr.CreateRecordRow(ctx, again)
	if err != nil {
		t.Fatalf("after the return, the step is sent again for the same run: %v", err)
	}
	third := paymentRow(o, "order-a", "Kamau", "400")
	third.SendKey = "step-1:order-a"
	if _, err := mgr.CreateRecordRow(ctx, third); !errors.Is(err, mwanachamaagency.ErrStepAlreadySent) {
		t.Errorf("the new row holds the key again: want ErrStepAlreadySent, got %v", err)
	}
	current := rowsOf(t, o, mwanachamaagency.RecordRowFilter{OrderID: "order-a"})
	found := false
	for _, r := range current {
		if r.ID == first.ID {
			t.Errorf("the returned row is still in the default listing")
		}
		found = found || r.ID == resent.ID
	}
	if !found {
		t.Errorf("the new send is missing from the default listing: %+v", current)
	}
}

func TestReturnRecordRow_RefusesARowThatIsMissingReturnedRetiredReplacedOrWithoutAReason(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	mgr := o.world.mgr
	made, err := mgr.CreateRecordRow(ctx, paymentRow(o, "order-a", "Wanjiru", "100"))
	if err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	if _, err := mgr.ReturnRecordRow(ctx, "no-such-row", "member-2", "x"); !errors.Is(err, mwanachamaagency.ErrInvalidRecordRow) {
		t.Errorf("a row that does not exist: want ErrInvalidRecordRow, got %v", err)
	}
	if _, err := mgr.ReturnRecordRow(ctx, made.ID, "member-2", "  "); !errors.Is(err, mwanachamaagency.ErrInvalidRecordRow) {
		t.Errorf("a blank reason: want ErrInvalidRecordRow, got %v", err)
	}
	if _, err := mgr.ReturnRecordRow(ctx, made.ID, "member-2", "Check the amount"); err != nil {
		t.Fatalf("first return: %v", err)
	}
	if _, err := mgr.ReturnRecordRow(ctx, made.ID, "member-3", "Again"); !errors.Is(err, mwanachamaagency.ErrInvalidRecordRow) {
		t.Errorf("an already returned row: want ErrInvalidRecordRow, got %v", err)
	}
	kept := rowsOf(t, o, mwanachamaagency.RecordRowFilter{IncludeReturned: true})
	if len(kept) != 1 || kept[0].ReturnedBy != "member-2" || kept[0].ReturnedReason != "Check the amount" {
		t.Errorf("the second return changed the row: %+v", kept)
	}

	retired, err := mgr.CreateRecordRow(ctx, paymentRow(o, "order-b", "Otieno", "300"))
	if err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	if _, err := mgr.RetireRecordRow(ctx, retired.ID, "member-2", "Entered twice"); err != nil {
		t.Fatalf("RetireRecordRow: %v", err)
	}
	if _, err := mgr.ReturnRecordRow(ctx, retired.ID, "member-2", "Check"); !errors.Is(err, mwanachamaagency.ErrInvalidRecordRow) {
		t.Errorf("a retired row: want ErrInvalidRecordRow, got %v", err)
	}

	old, err := mgr.CreateRecordRow(ctx, paymentRow(o, "order-c", "Achieng", "500"))
	if err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	fix := paymentRow(o, "order-c", "Achieng", "550")
	fix.Replaces = old.ID
	if _, err := mgr.CreateRecordRow(ctx, fix); err != nil {
		t.Fatalf("replacing row: %v", err)
	}
	if _, err := mgr.ReturnRecordRow(ctx, old.ID, "member-2", "Old"); !errors.Is(err, mwanachamaagency.ErrInvalidRecordRow) {
		t.Errorf("a replaced row is history: want ErrInvalidRecordRow, got %v", err)
	}
}

func TestReturnRecordRow_AReturnedRowCannotBeRetiredOrReplacedAndItsTwinMayBeAddedAgain(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	mgr := o.world.mgr
	made, err := mgr.CreateRecordRow(ctx, paymentRow(o, "order-a", "Wanjiru", "100"))
	if err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	if _, err := mgr.ReturnRecordRow(ctx, made.ID, "member-2", "Check the amount"); err != nil {
		t.Fatalf("ReturnRecordRow: %v", err)
	}
	if _, err := mgr.RetireRecordRow(ctx, made.ID, "member-2", "Entered twice"); !errors.Is(err, mwanachamaagency.ErrInvalidRecordRow) {
		t.Errorf("retiring a returned row: want ErrInvalidRecordRow, got %v", err)
	}
	fix := paymentRow(o, "order-a", "Wanjiru", "150")
	fix.Replaces = made.ID
	if _, err := mgr.CreateRecordRow(ctx, fix); !errors.Is(err, mwanachamaagency.ErrInvalidRecordRow) {
		t.Errorf("replacing a returned row: want ErrInvalidRecordRow, got %v", err)
	}
	if _, err := mgr.CreateRecordRow(ctx, paymentRow(o, "order-a", "Wanjiru", "100")); err != nil {
		t.Errorf("the same answers added again after a return are not a duplicate: %v", err)
	}
}

func TestListRecordRows_RowsNeverReturnedListAsBeforeWithOrWithoutTheFlag(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	for _, in := range []models.RecordRow{paymentRow(o, "order-a", "Wanjiru", "100"), paymentRow(o, "", "Otieno", "300")} {
		if _, err := o.world.mgr.CreateRecordRow(ctx, in); err != nil {
			t.Fatalf("CreateRecordRow: %v", err)
		}
	}
	plain := rowsOf(t, o, mwanachamaagency.RecordRowFilter{})
	flagged := rowsOf(t, o, mwanachamaagency.RecordRowFilter{IncludeReturned: true})
	if len(plain) != 2 || len(flagged) != 2 {
		t.Fatalf("plain %d, flagged %d, want 2 and 2", len(plain), len(flagged))
	}
	for i := range plain {
		if plain[i].ID != flagged[i].ID || plain[i].ReturnedOn != "" || plain[i].ReturnedBy != "" || plain[i].ReturnedReason != "" {
			t.Errorf("row %d differs: %+v and %+v", i, plain[i], flagged[i])
		}
	}
}
