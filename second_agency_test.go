package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAG73_CreateAgencyRefusesASecondAgencyAndKeepsTheFirst(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	first, err := mgr.CreateAgency(ctx, models.Agency{Name: "First"})
	if err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "Another"}); !errors.Is(err, mwanachamaagency.ErrAgencyExists) {
			t.Fatalf("CreateAgency %d: got %v, want ErrAgencyExists", i, err)
		}
	}
	got, err := mgr.GetAgency(ctx)
	if err != nil || got.ID != first.ID || got.Name != "First" {
		t.Fatalf("GetAgency = %+v, %v, want the first agency untouched", got, err)
	}
}
