package mwanachamaagency_test

import (
	"context"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func formsOnlyDocument(linkForms bool) models.AgencyDocument {
	doc := models.AgencyDocument{
		Agency:     models.DocumentAgency{Name: "Ward office"},
		Goals:      []models.DocumentGoal{{ID: "g1", Code: "G-1", Title: "Hear residents"}},
		Objectives: []models.DocumentObjective{{ID: "o1", Code: "O-1", GoalID: "g1", Title: "Ask residents"}},
		Modules: []models.DocumentModule{
			{ID: "m-forms", Code: "forms", Enabled: !linkForms},
			{ID: "m-comm", Code: "comm"},
		},
	}
	if linkForms {
		doc.ObjectiveModules = []models.DocumentObjectiveModule{{ObjectiveID: "o1", ModuleID: "m-forms", Rationale: "residents answer forms"}}
	}
	return doc
}

func modulesByCode(t *testing.T, mgr mwanachamaagency.AgencyManager, draftID string) map[string]models.Module {
	t.Helper()
	mods, err := mgr.ListModules(context.Background(), draftID, mwanachamaagency.ModuleFilter{})
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}
	out := make(map[string]models.Module, len(mods))
	for _, mod := range mods {
		out[mod.Code] = mod
	}
	return out
}

func TestImportDocument_FormsWithoutCatalogOrGitGetsBothEnabled(t *testing.T) {
	mgr := newTestManager(t)
	out, err := mgr.ImportDocument(context.Background(), "ward", formsOnlyDocument(true))
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	mods := modulesByCode(t, mgr, out.Draft.ID)
	for _, code := range []string{"forms", "catalog", "git"} {
		mod, ok := mods[code]
		if !ok {
			t.Fatalf("expected a %s module after import, got %v", code, mods)
		}
		if !mod.Enabled {
			t.Fatalf("expected %s enabled after import, got %+v", code, mod)
		}
	}
	if mods["comm"].Enabled {
		t.Fatalf("expected comm left off, got %+v", mods["comm"])
	}
	if len(mods) != 4 {
		t.Fatalf("expected forms, comm, catalog and git only, got %v", mods)
	}
}

func TestImportDocument_FormsSwitchedOnWithoutALinkStillGetsCatalogAndGit(t *testing.T) {
	mgr := newTestManager(t)
	out, err := mgr.ImportDocument(context.Background(), "ward", formsOnlyDocument(false))
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	mods := modulesByCode(t, mgr, out.Draft.ID)
	for _, code := range []string{"catalog", "git"} {
		if _, ok := mods[code]; !ok {
			t.Fatalf("expected a %s module after import, got %v", code, mods)
		}
	}
}

func TestImportDocument_CatalogAndGitAlreadyCarriedAreNotDuplicated(t *testing.T) {
	doc := formsOnlyDocument(true)
	doc.Modules = append(doc.Modules,
		models.DocumentModule{ID: "m-catalog", Code: "catalog"},
		models.DocumentModule{ID: "m-git", Code: "git"},
	)
	mgr := newTestManager(t)
	out, err := mgr.ImportDocument(context.Background(), "ward", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	mods, err := mgr.ListModules(context.Background(), out.Draft.ID, mwanachamaagency.ModuleFilter{})
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}
	if len(mods) != 4 {
		t.Fatalf("expected the four carried modules and no more, got %+v", mods)
	}
	for _, mod := range mods {
		if mod.Code != "comm" && !mod.Enabled {
			t.Fatalf("expected %s enabled while forms is on, got %+v", mod.Code, mod)
		}
	}
}

func TestImportDocument_NoFormsCreatesNoCatalogOrGit(t *testing.T) {
	doc := formsOnlyDocument(false)
	doc.Modules = []models.DocumentModule{{ID: "m-comm", Code: "comm"}}
	mgr := newTestManager(t)
	out, err := mgr.ImportDocument(context.Background(), "ward", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	mods := modulesByCode(t, mgr, out.Draft.ID)
	if len(mods) != 1 {
		t.Fatalf("expected comm alone, got %v", mods)
	}
}
