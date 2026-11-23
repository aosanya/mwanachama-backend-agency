package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func rowsOf(t *testing.T, o routedOrder, filter mwanachamaagency.RecordRowFilter) []models.RecordRow {
	t.Helper()
	filter.RecordCode = o.record.Code
	rows, err := o.world.mgr.ListRecordRows(context.Background(), o.world.draftID, filter)
	if err != nil {
		t.Fatalf("ListRecordRows: %v", err)
	}
	return rows
}

func TestRetireRecordRow_MarksTheRowRetiredWithWhoWhyAndWhenAndKeepsIt(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	made, err := o.world.mgr.CreateRecordRow(ctx, paymentRow(o, "", "Wanjiru", "100"))
	if err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	retired, err := o.world.mgr.RetireRecordRow(ctx, made.ID, "member-2", "Entered twice")
	if err != nil {
		t.Fatalf("RetireRecordRow: %v", err)
	}
	if !retired.Deleted || retired.DeletedBy != "member-2" || retired.DeletedReason != "Entered twice" || retired.DeletedOn == "" || retired.ID != made.ID {
		t.Errorf("retired row = %+v", retired)
	}
	if got := rowsOf(t, o, mwanachamaagency.RecordRowFilter{}); len(got) != 0 {
		t.Errorf("a retired row is hidden from the default listing, got %+v", got)
	}
	kept := rowsOf(t, o, mwanachamaagency.RecordRowFilter{IncludeRetired: true})
	if len(kept) != 1 || !kept[0].Deleted || kept[0].DeletedBy != "member-2" || kept[0].DeletedReason != "Entered twice" || kept[0].Answers["Enter the amount paid"] != "100" {
		t.Errorf("the retired row is kept with its answers and its reason: %+v", kept)
	}
}

func TestRetireRecordRow_RefusesARowThatIsMissingRetiredAlreadyReplacedOrWithoutAReason(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	mgr := o.world.mgr
	made, err := mgr.CreateRecordRow(ctx, paymentRow(o, "", "Wanjiru", "100"))
	if err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	if _, err := mgr.RetireRecordRow(ctx, "no-such-row", "member-2", "x"); !errors.Is(err, mwanachamaagency.ErrInvalidRecordRow) {
		t.Errorf("a row that does not exist: want ErrInvalidRecordRow, got %v", err)
	}
	if _, err := mgr.RetireRecordRow(ctx, made.ID, "member-2", "  "); !errors.Is(err, mwanachamaagency.ErrInvalidRecordRow) {
		t.Errorf("a blank reason: want ErrInvalidRecordRow, got %v", err)
	}
	if _, err := mgr.RetireRecordRow(ctx, made.ID, "member-2", "Entered twice"); err != nil {
		t.Fatalf("first retire: %v", err)
	}
	if _, err := mgr.RetireRecordRow(ctx, made.ID, "member-3", "Again"); !errors.Is(err, mwanachamaagency.ErrInvalidRecordRow) {
		t.Errorf("an already retired row: want ErrInvalidRecordRow, got %v", err)
	}
	kept := rowsOf(t, o, mwanachamaagency.RecordRowFilter{IncludeRetired: true})
	if len(kept) != 1 || kept[0].DeletedBy != "member-2" || kept[0].DeletedReason != "Entered twice" {
		t.Errorf("the second retire changed the row: %+v", kept)
	}

	old, err := mgr.CreateRecordRow(ctx, paymentRow(o, "", "Otieno", "300"))
	if err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	fix := paymentRow(o, "", "Otieno", "350")
	fix.Replaces = old.ID
	if _, err := mgr.CreateRecordRow(ctx, fix); err != nil {
		t.Fatalf("replacing row: %v", err)
	}
	if _, err := mgr.RetireRecordRow(ctx, old.ID, "member-2", "Old"); !errors.Is(err, mwanachamaagency.ErrInvalidRecordRow) {
		t.Errorf("a replaced row is history, not current: want ErrInvalidRecordRow, got %v", err)
	}
}

func TestRetireRecordRow_ARetiredRowCannotBeReplacedAndItsTwinMayBeAddedAgain(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	mgr := o.world.mgr
	made, err := mgr.CreateRecordRow(ctx, paymentRow(o, "", "Wanjiru", "100"))
	if err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	if _, err := mgr.RetireRecordRow(ctx, made.ID, "member-2", "Entered twice"); err != nil {
		t.Fatalf("RetireRecordRow: %v", err)
	}
	fix := paymentRow(o, "", "Wanjiru", "150")
	fix.Replaces = made.ID
	if _, err := mgr.CreateRecordRow(ctx, fix); !errors.Is(err, mwanachamaagency.ErrInvalidRecordRow) {
		t.Errorf("replacing a retired row: want ErrInvalidRecordRow, got %v", err)
	}
	if _, err := mgr.CreateRecordRow(ctx, paymentRow(o, "", "Wanjiru", "100")); err != nil {
		t.Errorf("the same answers added again after a retire are not a duplicate: %v", err)
	}
}

func TestListRecordRows_RowsNeverRetiredListAsBeforeWithOrWithoutTheFlag(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	for _, in := range []models.RecordRow{paymentRow(o, "order-a", "Wanjiru", "100"), paymentRow(o, "", "Otieno", "300")} {
		if _, err := o.world.mgr.CreateRecordRow(ctx, in); err != nil {
			t.Fatalf("CreateRecordRow: %v", err)
		}
	}
	plain := rowsOf(t, o, mwanachamaagency.RecordRowFilter{})
	flagged := rowsOf(t, o, mwanachamaagency.RecordRowFilter{IncludeRetired: true})
	if len(plain) != 2 || len(flagged) != 2 {
		t.Fatalf("plain %d, flagged %d, want 2 and 2", len(plain), len(flagged))
	}
	for i := range plain {
		if plain[i].ID != flagged[i].ID || plain[i].Deleted {
			t.Errorf("row %d differs: %+v and %+v", i, plain[i], flagged[i])
		}
	}
}
