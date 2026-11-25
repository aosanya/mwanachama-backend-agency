package mwanachamaagency_test

import (
	"context"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

func TestReturnRecordRow_KeepsTheSenderIdBesideTheSenderName(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := mwanachamaagency.WithReturnedByID(context.Background(), "account-42")
	made, err := o.world.mgr.CreateRecordRow(ctx, sentRow(o, "order-a", "step-1:order-a"))
	if err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	returned, err := o.world.mgr.ReturnRecordRow(ctx, made.ID, "Njeri Kamau", "The amount is not in the budget")
	if err != nil {
		t.Fatalf("ReturnRecordRow: %v", err)
	}
	if returned.ReturnedBy != "Njeri Kamau" || returned.ReturnedByID != "account-42" {
		t.Errorf("returned row names %q and %q, want Njeri Kamau and account-42", returned.ReturnedBy, returned.ReturnedByID)
	}
	kept := rowsOf(t, o, mwanachamaagency.RecordRowFilter{OrderID: "order-a", IncludeReturned: true})
	if len(kept) != 1 || kept[0].ReturnedBy != "Njeri Kamau" || kept[0].ReturnedByID != "account-42" {
		t.Errorf("the listing of returned rows reads %+v", kept)
	}
}

func TestReturnRecordRow_AReturnWithoutASenderIdReadsAsBefore(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	made, err := o.world.mgr.CreateRecordRow(ctx, sentRow(o, "order-a", "step-1:order-a"))
	if err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	returned, err := o.world.mgr.ReturnRecordRow(ctx, made.ID, "member-2", "Check the amount")
	if err != nil {
		t.Fatalf("ReturnRecordRow: %v", err)
	}
	if returned.ReturnedBy != "member-2" || returned.ReturnedByID != "" {
		t.Errorf("returned row = %+v", returned)
	}
	kept := rowsOf(t, o, mwanachamaagency.RecordRowFilter{OrderID: "order-a", IncludeReturned: true})
	if len(kept) != 1 || kept[0].ReturnedBy != "member-2" || kept[0].ReturnedByID != "" || kept[0].ReturnedReason != "Check the amount" {
		t.Errorf("the old-style return reads %+v", kept)
	}
}

func TestReturnRecordRow_TheSenderIdSurvivesTheSendThatFollowsTheReturn(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := mwanachamaagency.WithReturnedByID(context.Background(), "account-42")
	mgr := o.world.mgr
	first, err := mgr.CreateRecordRow(ctx, sentRow(o, "order-a", "step-1:order-a"))
	if err != nil {
		t.Fatalf("first send: %v", err)
	}
	if _, err := mgr.ReturnRecordRow(ctx, first.ID, "Njeri Kamau", "Check the amount"); err != nil {
		t.Fatalf("ReturnRecordRow: %v", err)
	}
	again := paymentRow(o, "order-a", "Wanjiru", "150")
	again.SendKey = "step-1:order-a"
	resent, err := mgr.CreateRecordRow(ctx, again)
	if err != nil {
		t.Fatalf("send again: %v", err)
	}
	if resent.ReturnedByID != "" {
		t.Errorf("the new send carries the sender id %q of the return", resent.ReturnedByID)
	}
	all := rowsOf(t, o, mwanachamaagency.RecordRowFilter{OrderID: "order-a", IncludeReturned: true})
	byID := map[string]string{}
	for _, r := range all {
		byID[r.ID] = r.ReturnedByID
	}
	if len(all) != 2 || byID[first.ID] != "account-42" || byID[resent.ID] != "" {
		t.Errorf("after the resend the rows read %+v", all)
	}
}
