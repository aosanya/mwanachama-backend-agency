package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCreateAndGetAgency(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "NeoLabHQ", Mission: "Ship reliable software."}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}

	got, err := mgr.GetAgency(ctx)
	if err != nil {
		t.Fatalf("GetAgency: %v", err)
	}
	if got.Name != "NeoLabHQ" || got.ID == "" || got.CreatedAt == "" {
		t.Fatalf("unexpected agency: %+v", got)
	}
}

func TestGetAgency_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.GetAgency(context.Background()); !errors.Is(err, mwanachamaagency.ErrAgencyNotFound) {
		t.Fatalf("expected ErrAgencyNotFound, got %v", err)
	}
}
