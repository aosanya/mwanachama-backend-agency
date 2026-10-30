package mwanachamaagency_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestModuleRequires_SwitchingFormsOnEnablesCatalogAndGit(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := o.DraftID

	created := map[string]models.Module{}
	for _, code := range []string{"forms", "catalog", "git", "taskmanager"} {
		mod, err := seededModule(mgr, ctx, draftID, code)
		if err != nil {
			t.Fatalf("seededModule %s: %v", code, err)
		}
		created[code] = mod
	}

	if _, err := mgr.SetModuleEnabled(ctx, draftID, created["forms"].ID, true); err != nil {
		t.Fatalf("SetModuleEnabled: %v", err)
	}

	for _, code := range []string{"forms", "catalog", "git"} {
		if got := moduleByCode(t, mgr, ctx, draftID, code); !got.Enabled {
			t.Fatalf("expected %s enabled while forms is on, got %+v", code, got)
		}
		if got, err := mgr.GetModule(ctx, created[code].ID); err != nil || !got.Enabled {
			t.Fatalf("expected GetModule to agree %s is enabled, got %+v (%v)", code, got, err)
		}
	}
	if got := moduleByCode(t, mgr, ctx, draftID, "taskmanager"); got.Enabled {
		t.Fatalf("expected taskmanager untouched by forms, got %+v", got)
	}

	if _, err := mgr.SetModuleEnabled(ctx, draftID, created["forms"].ID, false); err != nil {
		t.Fatalf("SetModuleEnabled: %v", err)
	}
	for _, code := range []string{"forms", "catalog", "git"} {
		if got := moduleByCode(t, mgr, ctx, draftID, code); got.Enabled {
			t.Fatalf("expected %s disabled once forms is off, got %+v", code, got)
		}
	}
}

func TestModuleRequires_ACatalogSwitchedOnByHandStaysOnWhenFormsGoesOff(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := o.DraftID
	forms := moduleByCode(t, mgr, ctx, draftID, "forms")
	catalog := moduleByCode(t, mgr, ctx, draftID, "catalog")

	for _, id := range []string{forms.ID, catalog.ID} {
		if _, err := mgr.SetModuleEnabled(ctx, draftID, id, true); err != nil {
			t.Fatalf("SetModuleEnabled: %v", err)
		}
	}
	if _, err := mgr.SetModuleEnabled(ctx, draftID, forms.ID, false); err != nil {
		t.Fatalf("SetModuleEnabled: %v", err)
	}
	if got := moduleByCode(t, mgr, ctx, draftID, "catalog"); !got.Enabled || !got.Explicit {
		t.Fatalf("expected catalog to stay on by its own flag, got %+v", got)
	}
}

func TestModuleRequires_FormsDeclaresCatalogAndGit(t *testing.T) {
	got := models.ModuleRequires("forms")
	if len(got) != 2 || got[0] != "catalog" || got[1] != "git" {
		t.Fatalf("expected forms to require catalog and git, got %v", got)
	}
}
