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

func TestUpdateAgency(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	created, err := mgr.CreateAgency(ctx, models.Agency{Name: "NeoLabHQ", Mission: "Ship reliable software."})
	if err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}

	updated, err := mgr.UpdateAgency(ctx, models.Agency{Name: "NeoLabHQ Renamed", Mission: "New mission.", Vision: "New vision."})
	if err != nil {
		t.Fatalf("UpdateAgency: %v", err)
	}
	if updated.Name != "NeoLabHQ Renamed" || updated.Mission != "New mission." || updated.Vision != "New vision." {
		t.Fatalf("unexpected updated agency: %+v", updated)
	}
	if updated.ID != created.ID || updated.CreatedAt != created.CreatedAt {
		t.Fatalf("UpdateAgency must not change ID/CreatedAt: got %+v, want id=%s createdAt=%s", updated, created.ID, created.CreatedAt)
	}

	got, err := mgr.GetAgency(ctx)
	if err != nil {
		t.Fatalf("GetAgency: %v", err)
	}
	if got.Name != "NeoLabHQ Renamed" || got.Mission != "New mission." || got.Vision != "New vision." {
		t.Fatalf("update did not persist: %+v", got)
	}
}

func TestUpdateAgency_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.UpdateAgency(context.Background(), models.Agency{Name: "Nobody"}); !errors.Is(err, mwanachamaagency.ErrAgencyNotFound) {
		t.Fatalf("expected ErrAgencyNotFound, got %v", err)
	}
}
