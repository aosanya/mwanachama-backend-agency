package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

const catalogSpec = `{"module":"catalog","domain":"agency","objects":[` +
	`{"role":"entry","name":"agency","table":"agencies"},` +
	`{"role":"suggestion","name":"suggestion","table":"suggestions"}]}`

func catalogModule(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID string) models.Module {
	t.Helper()
	mods, err := mgr.ListModules(ctx, draftID, mwanachamaagency.ModuleFilter{Code: "catalog"})
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}
	if len(mods) != 1 {
		t.Fatalf("expected one catalog module in %s, got %d", draftID, len(mods))
	}
	return mods[0]
}

func TestModuleSpec_IsSeededEmptyAndCarriesNoInstance(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ag, _, _ := seedPublishableAgency(t, mgr, ctx)
	_ = ag

	drafts, err := mgr.ListDrafts(ctx, mwanachamaagency.DraftFilter{})
	if err != nil || len(drafts) == 0 {
		t.Fatalf("ListDrafts: %v, %+v", err, drafts)
	}
	mod := catalogModule(t, mgr, ctx, drafts[0].ID)
	if mod.Spec != "" {
		t.Errorf("a backfilled Module should carry no spec, got %q", mod.Spec)
	}
}

func TestModuleSpec_SurvivesImportAndPublication(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedPublishableAgency(t, mgr, ctx)

	doc := models.AgencyDocument{
		Agency: models.DocumentAgency{Name: "Speccer", Description: "carries a module spec"},
		Goals:  []models.DocumentGoal{{ID: "g1", Code: "G-1", Title: "Sell templates"}},
		Objectives: []models.DocumentObjective{
			{ID: "o1", Code: "O-1", GoalID: "g1", Title: "Publish only what imports"},
		},
		Roles: []models.DocumentRole{{ID: "r1", Code: "R-1", Name: "owner"}},
		RoleAssignments: []models.DocumentRoleAssignment{
			{RoleID: "r1", ObjectiveID: "o1", Label: string(models.RACILabelAccountable)},
		},
		Workflows: []models.DocumentWorkflow{{ID: "w1", Code: "W-1", Name: "Catalog pipeline"}},
		WorkItems: []models.DocumentWorkItem{
			{ID: "wi1", Code: "WI-1", WorkflowID: "w1", ObjectiveID: "o1", Title: "Mine an archetype"},
		},
		Modules: []models.DocumentModule{
			{ID: "m1", Code: "catalog", Enabled: true, Spec: json.RawMessage(catalogSpec)},
		},
		ObjectiveModules: []models.DocumentObjectiveModule{
			{ObjectiveID: "o1", ModuleID: "m1", Rationale: "the shop is this module"},
		},
	}

	out, err := mgr.ImportDocument(ctx, "imported with a spec", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}

	imported := catalogModule(t, mgr, ctx, out.Draft.ID)
	if imported.Spec == "" {
		t.Fatal("the imported Module lost its spec")
	}
	if strings.Contains(imported.Spec, `"instance"`) {
		t.Errorf("a stored spec must carry no instance, got %s", imported.Spec)
	}
	if !imported.Enabled {
		t.Error("a module linked to an Objective should read as enabled")
	}

	if _, err := mgr.PublishAgency(ctx, out.Draft.ID, "spec-1", "production"); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}

	forked, err := mgr.CreateDraft(ctx, "revisit spec-1", "spec-1", "publication")
	if err != nil {
		t.Fatalf("CreateDraft(publication): %v", err)
	}

	after := catalogModule(t, mgr, ctx, forked.ID)
	if after.Spec == "" {
		t.Fatal("the spec did not survive publish and fork")
	}

	var want, got struct {
		Module  string `json:"module"`
		Objects []struct {
			Role  string `json:"role"`
			Name  string `json:"name"`
			Table string `json:"table"`
		} `json:"objects"`
	}
	if err := json.Unmarshal([]byte(catalogSpec), &want); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	if err := json.Unmarshal([]byte(after.Spec), &got); err != nil {
		t.Fatalf("the forked spec does not parse: %v", err)
	}
	if got.Module != want.Module || len(got.Objects) != len(want.Objects) {
		t.Fatalf("spec = %+v, want %+v", got, want)
	}
	if got.Objects[0].Name != "agency" || got.Objects[0].Table != "agencies" {
		t.Errorf("the domain's own naming was lost: %+v", got.Objects[0])
	}
}
