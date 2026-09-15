package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

func TestDeleteWorkItem_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if err := mgr.DeleteWorkItem(context.Background(), "unknown"); !errors.Is(err, mwanachamaagency.ErrWorkItemNotFound) {
		t.Fatalf("expected ErrWorkItemNotFound, got %v", err)
	}
}

func TestDeleteWorkItem(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, _, _, wi := seedLiveAgency(t, mgr, ctx)

	if err := mgr.DeleteWorkItem(ctx, wi.ID); err != nil {
		t.Fatalf("DeleteWorkItem: %v", err)
	}
	if _, err := mgr.GetWorkItem(ctx, wi.ID); !errors.Is(err, mwanachamaagency.ErrWorkItemNotFound) {
		t.Fatalf("expected ErrWorkItemNotFound, got %v", err)
	}
}
