package mwanachamaagency_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestModuleRequires_EnablingFormsEnablesCatalogAndGit(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := o.DraftID

	created := map[string]models.Module{}
	for _, code := range []string{"forms", "catalog", "git", "comm"} {
		mod, err := seededModule(mgr, ctx, draftID, code)
		if err != nil {
			t.Fatalf("CreateModule %s: %v", code, err)
		}
		created[code] = mod
	}

	link, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{DraftID: draftID, ObjectiveID: o.ID, ModuleID: created["forms"].ID, Rationale: "residents answer forms"})
	if err != nil {
		t.Fatalf("CreateObjectiveModule: %v", err)
	}

	for _, code := range []string{"forms", "catalog", "git"} {
		if got := moduleByCode(t, mgr, ctx, draftID, code); !got.Enabled {
			t.Fatalf("expected %s enabled while forms is on, got %+v", code, got)
		}
		if got, err := mgr.GetModule(ctx, created[code].ID); err != nil || !got.Enabled {
			t.Fatalf("expected GetModule to agree %s is enabled, got %+v (%v)", code, got, err)
		}
	}
	if got := moduleByCode(t, mgr, ctx, draftID, "comm"); got.Enabled {
		t.Fatalf("expected comm untouched by forms, got %+v", got)
	}

	if err := mgr.DeleteObjectiveModule(ctx, link.ID); err != nil {
		t.Fatalf("DeleteObjectiveModule: %v", err)
	}
	for _, code := range []string{"forms", "catalog", "git"} {
		if got := moduleByCode(t, mgr, ctx, draftID, code); got.Enabled {
			t.Fatalf("expected %s disabled once forms is off, got %+v", code, got)
		}
	}
}

func TestModuleRequires_FormsDeclaresCatalogAndGit(t *testing.T) {
	got := models.ModuleRequires("forms")
	if len(got) != 2 || got[0] != "catalog" || got[1] != "git" {
		t.Fatalf("expected forms to require catalog and git, got %v", got)
	}
}
