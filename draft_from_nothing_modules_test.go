package mwanachamaagency_test

import (
	"context"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestADraftStartedFromNothingHoldsTheWholeModuleCatalog(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "A"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	d, err := mgr.CreateDraft(ctx, "fresh", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	mods, err := mgr.ListModules(ctx, d.ID, mwanachamaagency.ModuleFilter{})
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}
	byCode := map[string]models.Module{}
	for _, m := range mods {
		byCode[m.Code] = m
	}
	if len(mods) != len(models.ModuleCatalog) {
		t.Fatalf("expected %d modules, got %d: %+v", len(models.ModuleCatalog), len(mods), mods)
	}
	for _, entry := range models.ModuleCatalog {
		m, ok := byCode[entry.Code]
		if !ok {
			t.Fatalf("module %q missing from the fresh draft", entry.Code)
		}
		if m.Locked != entry.Locked || m.DraftID != d.ID {
			t.Fatalf("module %q seeded wrongly: %+v", entry.Code, m)
		}
	}
}
