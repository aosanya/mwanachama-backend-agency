package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

func TestRecordRow_AStepIsSentOncePerRun(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	first := paymentRow(o, "order-a", "Wanjiru", "100")
	first.SendKey = "step-1:run-1"
	if _, err := o.world.mgr.CreateRecordRow(ctx, first); err != nil {
		t.Fatalf("the first send: %v", err)
	}
	second := paymentRow(o, "order-b", "Otieno", "200")
	second.SendKey = "step-1:run-1"
	if _, err := o.world.mgr.CreateRecordRow(ctx, second); !errors.Is(err, mwanachamaagency.ErrStepAlreadySent) {
		t.Fatalf("a second send of the same step in the same run = %v, want ErrStepAlreadySent", err)
	}
	otherRun := paymentRow(o, "order-c", "Achieng", "300")
	otherRun.SendKey = "step-1:run-2"
	if _, err := o.world.mgr.CreateRecordRow(ctx, otherRun); err != nil {
		t.Errorf("the same step in another run was refused: %v", err)
	}
	for n, amount := range map[string]string{"order-d": "400", "order-e": "500"} {
		keyless := paymentRow(o, n, "Wanjiru", amount)
		if _, err := o.world.mgr.CreateRecordRow(ctx, keyless); err != nil {
			t.Errorf("a row with no send key was refused: %v", err)
		}
	}
}
