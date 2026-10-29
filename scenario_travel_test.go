package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type travelWorld struct {
	scenarioWorld
	base models.Scenario
}

func newTravelWorld(t *testing.T) travelWorld {
	t.Helper()
	w := travelWorld{scenarioWorld: newScenarioWorld(t)}
	s := w.scenario(t, "Shop says yes", w.buyNow, w.actor("Farmer 1", farmer), w.actor("Shop", supplier))
	if _, err := w.mgr.SetScenarioInput(context.Background(), models.ScenarioInput{
		ScenarioID: s.ID, StepID: s.Blocks[0].StepID, Field: "Enter the product", Value: "urea",
	}); err != nil {
		t.Fatalf("SetScenarioInput: %v", err)
	}
	w.pass(t, s.ID)
	w.base = w.get(t, s.ID)
	return w
}

func (w travelWorld) onlyScenario(t *testing.T, draftID string) models.Scenario {
	t.Helper()
	all, err := w.mgr.ListScenarios(context.Background(), draftID, mwanachamaagency.ScenarioFilter{})
	if err != nil || len(all) != 1 {
		t.Fatalf("ListScenarios(%s) = %v %v, want one scenario", draftID, all, err)
	}
	full, err := w.mgr.GetScenario(context.Background(), all[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	return full
}

func (w travelWorld) checkCopy(t *testing.T, copied models.Scenario, draftID string) {
	t.Helper()
	ctx := context.Background()
	if copied.ID == w.base.ID || copied.DraftID != draftID {
		t.Errorf("copy = %s in %s, want a new row in %s", copied.ID, copied.DraftID, draftID)
	}
	if copied.Name != "Shop says yes" || copied.Summary != w.base.Summary {
		t.Errorf("copy = %q %q, want the same words", copied.Name, copied.Summary)
	}
	if copied.Result != models.ScenarioNotRun || copied.LastRunID != "" {
		t.Errorf("copy result = %q run = %q, want not run yet: nothing has run on this draft", copied.Result, copied.LastRunID)
	}
	if copied.Workflow != "Buy now" || copied.WorkflowID == w.buyNow.ID {
		t.Errorf("copy mirrors workflow %q (%s), want Buy now in the new draft", copied.Workflow, copied.WorkflowID)
	}
	if len(copied.Blocks) != 3 {
		t.Fatalf("copy has %d blocks, want the 3 steps of the copied workflow", len(copied.Blocks))
	}
	for _, b := range copied.Blocks {
		if b.StepID == "" {
			t.Errorf("a copied block has no step")
		}
		for _, original := range w.base.Blocks {
			if b.StepID == original.StepID {
				t.Errorf("block %q still points at the old draft's step", b.Title)
			}
		}
	}
	if got := copied.Blocks[0].Fields[1]; got.Free || got.Value != "urea" {
		t.Errorf("fixed input = %+v, want it carried to the copy", got)
	}
	roles, _ := w.mgr.ListRoles(ctx, draftID, mwanachamaagency.RoleFilter{})
	byID := map[string]string{}
	for _, r := range roles {
		byID[r.ID] = r.Name
	}
	var names []string
	for _, a := range copied.Actors {
		if a.BuiltIn {
			continue
		}
		names = append(names, a.Name+"/"+byID[a.RoleID])
	}
	if got := strings.Join(names, ","); got != "Farmer 1/Farmer,Shop/Supplier" {
		t.Errorf("copied actors = %q, want each holding the new draft's role", got)
	}
	runs, err := w.mgr.ListTestRuns(ctx, copied.ID)
	if err != nil || len(runs) != 0 {
		t.Errorf("runs of the copy = %v %v, want none: runs never travel", runs, err)
	}
}

func TestForkingADraftCopiesItsScenariosAndLeavesTheRunsBehind(t *testing.T) {
	w := newTravelWorld(t)
	ctx := context.Background()
	fork, err := w.mgr.CreateDraft(ctx, "fork", w.draftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	w.checkCopy(t, w.onlyScenario(t, fork.ID), fork.ID)

	if got := w.get(t, w.base.ID); got.Result != models.ScenarioPassed || got.DraftID != w.draftID {
		t.Errorf("the original = %q in %s, want it untouched", got.Result, got.DraftID)
	}
	if _, err := w.mgr.PublishAgency(ctx, fork.ID, "", ""); !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) {
		t.Errorf("publishing the fork: %v, want it blocked until its scenarios pass", err)
	}
}

func TestAPublicationCarriesScenariosButNeverRunsOrResults(t *testing.T) {
	w := newTravelWorld(t)
	ctx := context.Background()
	p, err := w.mgr.PublishAgency(ctx, w.draftID, "v-scn", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if len(p.Document.Scenarios) != 1 {
		t.Fatalf("document scenarios = %+v, want one", p.Document.Scenarios)
	}
	ds := p.Document.Scenarios[0]
	if ds.Name != "Shop says yes" || ds.WorkflowID != w.buyNow.ID || len(ds.Actors) != 2 || len(ds.Inputs) != 1 {
		t.Errorf("document scenario = %+v", ds)
	}
	raw, _ := json.Marshal(p.Document)
	for _, leaked := range []string{`"result"`, `"last_run_id"`, `"tokens"`, `"steps":[{"step_id"`, `"draft_version"`} {
		if strings.Contains(string(raw), leaked) {
			t.Errorf("the document carries %s, which belongs to a run", leaked)
		}
	}

	read, err := w.mgr.GetPublication(ctx, p.Version)
	if err != nil || len(read.Document.Scenarios) != 1 {
		t.Fatalf("GetPublication: %v %+v", err, read.Document.Scenarios)
	}

	fork, err := w.mgr.CreateDraft(ctx, "from publication", "v-scn", "publication")
	if err != nil {
		t.Fatalf("CreateDraft from publication: %v", err)
	}
	w.checkCopy(t, w.onlyScenario(t, fork.ID), fork.ID)
}

func TestImportingADocumentWithScenariosWritesThem(t *testing.T) {
	w := newTravelWorld(t)
	ctx := context.Background()
	p, err := w.mgr.PublishAgency(ctx, w.draftID, "v-scn", "")
	if err != nil {
		t.Fatal(err)
	}
	imported, err := w.mgr.ImportDocument(ctx, "imported", p.Document)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	w.checkCopy(t, w.onlyScenario(t, imported.Draft.ID), imported.Draft.ID)
}

func TestImportingAScenarioOfAWorkflowTheDocumentLacksIsRefused(t *testing.T) {
	w := newTravelWorld(t)
	ctx := context.Background()
	p, err := w.mgr.PublishAgency(ctx, w.draftID, "v-scn", "")
	if err != nil {
		t.Fatal(err)
	}
	doc := p.Document
	doc.Scenarios = []models.DocumentScenario{{ID: "s", Name: "orphan", WorkflowID: "no-such-workflow"}}
	if _, err := w.mgr.ImportDocument(ctx, "bad", doc); !errors.Is(err, mwanachamaagency.ErrInvalidScenario) {
		t.Errorf("import: %v, want ErrInvalidScenario", err)
	}
	drafts, _ := w.mgr.ListDrafts(ctx, mwanachamaagency.DraftFilter{})
	for _, d := range drafts {
		if d.Description == "bad" {
			t.Errorf("a refused import left the draft %s behind", d.ID)
		}
	}
}

func TestADocumentWithoutScenariosStillImportsAndForks(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	p, err := mgr.PublishAgency(ctx, o.DraftID, "plain", "")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(p.Document)
	if strings.Contains(string(raw), "scenarios") {
		t.Errorf("a document with no scenarios names the key: %s", raw)
	}
	if _, err := mgr.ImportDocument(ctx, "plain import", p.Document); err != nil {
		t.Errorf("import: %v", err)
	}
}
