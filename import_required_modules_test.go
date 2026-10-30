package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func formsOnlyDocument(formsOn bool) models.AgencyDocument {
	return models.AgencyDocument{
		Agency:     models.DocumentAgency{Name: "Ward office"},
		Goals:      []models.DocumentGoal{{ID: "g1", Code: "G-1", Title: "Hear residents"}},
		Objectives: []models.DocumentObjective{{ID: "o1", Code: "O-1", GoalID: "g1", Title: "Ask residents"}},
		Modules: []models.DocumentModule{
			{ID: "m-forms", Code: "forms", Enabled: formsOn},
			{ID: "m-comm", Code: "comm"},
		},
	}
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

func TestImportDocument_FormsSwitchedOffAndUnneededGetsNoCatalogOrGit(t *testing.T) {
	mgr := newTestManager(t)
	out, err := mgr.ImportDocument(context.Background(), "ward", formsOnlyDocument(false))
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	mods := modulesByCode(t, mgr, out.Draft.ID)
	if len(mods) != 2 || mods["forms"].Enabled || mods["forms"].Explicit {
		t.Fatalf("expected forms and comm only, forms off, got %v", mods)
	}
}

func TestImportDocument_AFillFormStepNeedsFormsCatalogAndGit(t *testing.T) {
	doc := formsOnlyDocument(false)
	doc.Workflows = []models.DocumentWorkflow{{ID: "w1", Code: "W-1", Name: "Ask"}}
	doc.Roles = []models.DocumentRole{{ID: "r1", Code: "R-1", Name: "Resident"}}
	doc.WorkItems = []models.DocumentWorkItem{{
		ID: "wi1", Code: "WI-1", WorkflowID: "w1", ObjectiveID: "o1", Title: "Answer", Ordinality: 1,
		Kind: models.StepKindFillForm, RoleID: "r1", FormName: "Survey",
		Questions: []models.StepQuestion{{Prompt: "Write the message", AnswerType: models.AnswerFreeText, Required: true}},
	}}
	mgr := newTestManager(t)
	out, err := mgr.ImportDocument(context.Background(), "ward", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	mods := modulesByCode(t, mgr, out.Draft.ID)
	for _, code := range []string{"forms", "catalog", "git"} {
		mod, ok := mods[code]
		if !ok || !mod.Enabled {
			t.Fatalf("expected %s present and enabled, got %v", code, mods)
		}
	}
	if mods["forms"].Explicit || !mods["forms"].NeededBySteps {
		t.Fatalf("expected forms needed by the step with its own flag off, got %+v", mods["forms"])
	}
}

func TestImportDocument_AnIncomingObjectiveModulesKeyIsIgnored(t *testing.T) {
	raw := `{"agency":{"name":"Ward office"},"goals":[{"id":"g1","code":"G-1","title":"Hear residents"}],
		"objectives":[{"id":"o1","code":"O-1","goal_id":"g1","title":"Ask residents"}],
		"modules":[{"id":"m-forms","code":"forms","enabled":false}],
		"objective_modules":[{"objective_id":"o1","module_id":"m-forms","rationale":"residents answer forms"}]}`
	var doc models.AgencyDocument
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("an old template must still decode: %v", err)
	}
	mgr := newTestManager(t)
	out, err := mgr.ImportDocument(context.Background(), "ward", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	if forms := modulesByCode(t, mgr, out.Draft.ID)["forms"]; forms.Enabled || forms.Explicit {
		t.Fatalf("a link in an old template must not switch forms on, got %+v", forms)
	}
}

func TestImportDocument_AModulesEnabledSetsTheStoredFlag(t *testing.T) {
	mgr := newTestManager(t)
	out, err := mgr.ImportDocument(context.Background(), "ward", formsOnlyDocument(true))
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	mods := modulesByCode(t, mgr, out.Draft.ID)
	if !mods["forms"].Explicit || mods["comm"].Explicit {
		t.Fatalf("expected forms stored on and comm stored off exactly as the document says, got %+v %+v", mods["forms"], mods["comm"])
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
