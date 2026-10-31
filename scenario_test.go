package mwanachamaagency_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type scriptedRunner struct {
	mu      sync.Mutex
	outcome func(mwanachamaagency.RunRequest) (mwanachamaagency.RunOutcome, error)
	seen    []mwanachamaagency.RunRequest
}

func (r *scriptedRunner) Run(_ context.Context, req mwanachamaagency.RunRequest) (mwanachamaagency.RunOutcome, error) {
	r.mu.Lock()
	r.seen = append(r.seen, req)
	r.mu.Unlock()
	return r.outcome(req)
}

func passing(req mwanachamaagency.RunRequest) (mwanachamaagency.RunOutcome, error) {
	steps := make([]models.RunStep, len(req.Scenario.Blocks))
	for i, b := range req.Scenario.Blocks {
		steps[i] = models.RunStep{StepID: b.StepID, Step: b.Title, Actor: b.Actor, Did: b.Action, Tokens: 100}
	}
	return mwanachamaagency.RunOutcome{Result: models.TestRunPassed, Steps: steps, Tokens: 100 * len(steps)}, nil
}

type scenarioWorld struct {
	stepWorld
	runner    *scriptedRunner
	db        *gorm.DB
	buyNow    models.Workflow
	groupOrd  models.Workflow
	agentRole models.Role
}

func newScenarioWorld(t *testing.T, opts ...mwanachamaagency.Option) scenarioWorld {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := mwanachamaagency.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	runner := &scriptedRunner{outcome: passing}
	pool, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	pool.SetMaxOpenConns(1)
	mgr, err := mwanachamaagency.NewAgencyManager(db, s, append([]mwanachamaagency.Option{mwanachamaagency.WithScenarioRunner(runner)}, opts...)...)
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	ctx := context.Background()
	_, _, o, _, firstStep := seedLiveAgency(t, mgr, ctx)
	owner, err := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "owner"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		DraftID: o.DraftID, WorkItemID: firstStep.ID, RoleID: owner.ID, Label: models.RACILabelAccountable,
	}); err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}
	sw := stepWorld{mgr: mgr, draftID: o.DraftID, objective: o, roles: map[string]models.Role{}}
	for _, name := range []string{coordinator, farmer, supplier} {
		r, err := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: name})
		if err != nil {
			t.Fatalf("CreateRole %s: %v", name, err)
		}
		sw.roles[name] = r
	}
	agent, err := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "Farmers' agent", Powers: []string{"ai.use"}})
	if err != nil {
		t.Fatalf("CreateRole agent: %v", err)
	}
	w := scenarioWorld{stepWorld: sw, runner: runner, db: db, agentRole: agent}
	w.buyNow = sw.workflow(t, "Buy now")
	sw.build(t, w.buyNow, buyNowSteps)
	w.groupOrd = sw.workflow(t, "Group order")
	sw.buildRouted(t, w.groupOrd)
	return w
}

func (w stepWorld) buildRouted(t *testing.T, wf models.Workflow) map[string]models.WorkItem {
	t.Helper()
	ctx := context.Background()
	steps := map[string]models.WorkItem{}
	for i, s := range groupOrderSteps {
		in := w.item(wf, i+1, s)
		switch s.title {
		case "Send the invitation":
			in.RecipientRoleID = w.roles[farmer].ID
		case "Answer the invitation":
			in.FormName, in.Questions = "", nil
			in.SentStepID = steps["Send the invitation"].ID
		}
		made, err := w.mgr.CreateWorkItem(ctx, in)
		if err != nil {
			t.Fatalf("CreateWorkItem %s: %v", s.title, err)
		}
		steps[s.title] = made
	}
	sort := steps["Sort the answers"]
	sort.SentStepID = steps["Send the invitation"].ID
	sort.Routes, sort.OtherwiseStepID = routeToOpenOtherwiseThankYou(steps["Send the invitation"], steps["Open"], steps["Thank you"])
	sorted, err := w.mgr.UpdateWorkItem(ctx, sort)
	if err != nil {
		t.Fatalf("UpdateWorkItem route: %v", err)
	}
	steps["Sort the answers"] = sorted
	return steps
}

func (w scenarioWorld) scenario(t *testing.T, name string, workflow models.Workflow, actors ...models.ScenarioActor) models.Scenario {
	t.Helper()
	s, err := w.mgr.CreateScenario(context.Background(), models.Scenario{
		DraftID: w.draftID, Name: name, Summary: "tries " + name, WorkflowID: workflow.ID, Actors: actors,
	})
	if err != nil {
		t.Fatalf("CreateScenario %s: %v", name, err)
	}
	return s
}

func (w scenarioWorld) actor(name, role string) models.ScenarioActor {
	return models.ScenarioActor{Name: name, Kind: models.ActorKindRole, RoleID: w.roles[role].ID}
}

func (w scenarioWorld) get(t *testing.T, id string) models.Scenario {
	t.Helper()
	s, err := w.mgr.GetScenario(context.Background(), id)
	if err != nil {
		t.Fatalf("GetScenario: %v", err)
	}
	return s
}

func (w scenarioWorld) runToEnd(ctx context.Context, id string) (models.TestRun, error) {
	pressed, err := w.mgr.RunScenario(ctx, id)
	if err != nil {
		return models.TestRun{}, err
	}
	return w.mgr.AwaitRun(ctx, pressed.ID)
}

func (w scenarioWorld) awaitAll(t *testing.T, runs []models.TestRun) {
	t.Helper()
	for _, run := range runs {
		if _, err := w.mgr.AwaitRun(context.Background(), run.ID); err != nil {
			t.Fatalf("AwaitRun: %v", err)
		}
	}
}

func (w scenarioWorld) pass(t *testing.T, id string) {
	t.Helper()
	run, err := w.runToEnd(context.Background(), id)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	if run.Result != models.TestRunPassed {
		t.Fatalf("run = %+v, want passed", run)
	}
}

func (w scenarioWorld) results(t *testing.T) map[string]models.ScenarioResult {
	t.Helper()
	all, err := w.mgr.ListScenarios(context.Background(), w.draftID, mwanachamaagency.ScenarioFilter{})
	if err != nil {
		t.Fatalf("ListScenarios: %v", err)
	}
	out := map[string]models.ScenarioResult{}
	for _, s := range all {
		out[s.Name] = s.Result
	}
	return out
}

func TestCreateScenario_ReadsItsBlocksFromTheWorkflowAndStartsNotRun(t *testing.T) {
	w := newScenarioWorld(t)
	s := w.scenario(t, "Shop says yes", w.buyNow, w.actor("Farmer 1", farmer), w.actor("Shop", supplier))

	if s.Result != models.ScenarioNotRun || s.LastRunID != "" {
		t.Errorf("result = %q run = %q, want a scenario not run yet", s.Result, s.LastRunID)
	}
	if s.Workflow != "Buy now" {
		t.Errorf("workflow = %q, want Buy now", s.Workflow)
	}
	var titles, actors []string
	for _, b := range s.Blocks {
		titles = append(titles, b.Title)
		actors = append(actors, b.Actor)
	}
	if want := []string{"Ask", "Answered", "Paid"}; !reflect.DeepEqual(titles, want) {
		t.Errorf("blocks = %v, want the steps of Buy now %v", titles, want)
	}
	if want := []string{"Farmer 1", "Shop", "Shop"}; !reflect.DeepEqual(actors, want) {
		t.Errorf("block actors = %v, want %v", actors, want)
	}
	if got := s.Blocks[0]; got.Kind != models.StepKindFillForm || got.Action != "Fills the Buy now form" || len(got.Fields) != 3 || !got.Fields[0].Free {
		t.Errorf("first block = %+v, want the step's kind, action and three free fields", got)
	}
}

func TestCreateScenario_TheBuiltInActorsAreAlwaysThereAndReadOnly(t *testing.T) {
	w := newScenarioWorld(t)
	s := w.scenario(t, "Shop says yes", w.buyNow, w.actor("Farmer 1", farmer))

	built := map[string]models.ScenarioActor{}
	for _, a := range s.Actors {
		if a.BuiltIn {
			built[a.Name] = a
		}
	}
	if sys, ok := built[models.SystemActorName]; !ok || sys.Kind != models.ActorKindSystem {
		t.Errorf("built-in actors = %+v, want the System", built)
	}
	if ag, ok := built["Farmers' agent"]; !ok || ag.Kind != models.ActorKindAgent || ag.RoleID != w.agentRole.ID || ag.RoleName != "Farmers' agent" {
		t.Errorf("built-in actors = %+v, want the Farmers' agent holding its role", built)
	}
	if len(s.Actors) != 3 {
		t.Errorf("actors = %d, want Farmer 1, the Farmers' agent and the System", len(s.Actors))
	}
	for _, id := range []string{"system", w.agentRole.ID} {
		err := w.mgr.RemoveScenarioActor(context.Background(), id)
		if !errors.Is(err, mwanachamaagency.ErrScenarioActorNotFound) {
			t.Errorf("removing built-in %q: %v, want ErrScenarioActorNotFound", id, err)
		}
	}
}

func TestCreateScenario_StepsNobodyTakesAreThe_System(t *testing.T) {
	w := newScenarioWorld(t)
	s := w.scenario(t, "Group", w.groupOrd, w.actor("Coordinator", coordinator), w.actor("Farmer 1", farmer), w.actor("Farmer 2", farmer), w.actor("Shop", supplier))

	got := map[string]string{}
	for _, b := range s.Blocks {
		got[b.Title] = b.Actor
	}
	want := map[string]string{
		"Start": "Coordinator", "Send the invitation": models.SystemActorName, "Answer the invitation": "Farmer 1",
		"Sort the answers": models.SystemActorName, "Open": "Farmer 1", "Send": "Coordinator", "Answered": "Shop",
		"Thank you": models.SystemActorName,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("block actors = %v, want %v", got, want)
	}
}

func TestCreateScenario_ShowsTheRulesAttachedToItsWorkflow(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	rule, err := w.mgr.CreateRule(ctx, models.Rule{DraftID: w.draftID, Text: "Ask the shop once"})
	if err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	if _, err := w.mgr.CreateRuleAttachment(ctx, rule.ID, models.TargetTypeWorkflow, w.buyNow.ID); err != nil {
		t.Fatalf("CreateRuleAttachment: %v", err)
	}
	s := w.scenario(t, "Shop says yes", w.buyNow)
	if !reflect.DeepEqual(s.Rules, []string{rule.Code}) {
		t.Errorf("rules = %v, want [%s]", s.Rules, rule.Code)
	}
	if other := w.scenario(t, "Group", w.groupOrd); len(other.Rules) != 0 {
		t.Errorf("a scenario on another workflow shows rules %v", other.Rules)
	}
}

func TestCreateScenario_Refusals(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	other, err := w.mgr.CreateDraft(ctx, "other", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	foreignWorkflow, err := w.mgr.CreateWorkflow(ctx, models.Workflow{DraftID: other.ID, Name: "Elsewhere"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	archived, err := w.mgr.CreateDraft(ctx, "archived", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if _, err := w.mgr.ArchiveDraft(ctx, archived.ID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}

	base := models.Scenario{DraftID: w.draftID, Name: "n", WorkflowID: w.buyNow.ID}
	with := func(f func(*models.Scenario)) models.Scenario { s := base; f(&s); return s }
	cases := []struct {
		name string
		in   models.Scenario
		want error
	}{
		{"no draft", with(func(s *models.Scenario) { s.DraftID = "" }), mwanachamaagency.ErrDraftRequired},
		{"archived draft", with(func(s *models.Scenario) { s.DraftID = archived.ID }), mwanachamaagency.ErrDraftNotOpen},
		{"no name", with(func(s *models.Scenario) { s.Name = " " }), mwanachamaagency.ErrInvalidScenario},
		{"no workflow", with(func(s *models.Scenario) { s.WorkflowID = "" }), mwanachamaagency.ErrInvalidScenario},
		{"a workflow of another draft", with(func(s *models.Scenario) { s.WorkflowID = foreignWorkflow.ID }), mwanachamaagency.ErrWorkflowNotFound},
		{"a role that does not exist", with(func(s *models.Scenario) {
			s.Actors = []models.ScenarioActor{{Name: "Ghost", RoleID: "no-such-role"}}
		}), mwanachamaagency.ErrRoleNotFound},
		{"an actor with no name", with(func(s *models.Scenario) {
			s.Actors = []models.ScenarioActor{{Name: "", RoleID: w.roles[farmer].ID}}
		}), mwanachamaagency.ErrInvalidScenario},
		{"an actor with no role", with(func(s *models.Scenario) {
			s.Actors = []models.ScenarioActor{{Name: "Nobody"}}
		}), mwanachamaagency.ErrInvalidScenario},
		{"two actors with one name", with(func(s *models.Scenario) {
			s.Actors = []models.ScenarioActor{w.actor("Farmer 1", farmer), w.actor("farmer 1", farmer)}
		}), mwanachamaagency.ErrDuplicateScenarioActor},
		{"an actor named like the System", with(func(s *models.Scenario) {
			s.Actors = []models.ScenarioActor{w.actor("System", farmer)}
		}), mwanachamaagency.ErrDuplicateScenarioActor},
		{"an actor named like an agent", with(func(s *models.Scenario) {
			s.Actors = []models.ScenarioActor{w.actor("Farmers' agent", farmer)}
		}), mwanachamaagency.ErrDuplicateScenarioActor},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := w.mgr.CreateScenario(ctx, c.in); !errors.Is(err, c.want) {
				t.Errorf("err = %v, want %v", err, c.want)
			}
		})
	}
	if all := w.results(t); len(all) != 0 {
		t.Errorf("a refused scenario left rows behind: %v", all)
	}
}

func TestAddScenarioActor_NamesAreUniqueAndRolesBelongToTheDraft(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow, w.actor("Farmer 1", farmer))

	added, err := w.mgr.AddScenarioActor(ctx, models.ScenarioActor{ScenarioID: s.ID, Name: "Farmer 2", RoleID: w.roles[farmer].ID})
	if err != nil {
		t.Fatalf("AddScenarioActor: %v", err)
	}
	if added.ID == "" || added.Kind != models.ActorKindRole || added.RoleName != farmer || added.Ordinality != 2 {
		t.Errorf("added = %+v, want a role actor numbered 2 reading its role's name", added)
	}
	if _, err := w.mgr.AddScenarioActor(ctx, models.ScenarioActor{ScenarioID: s.ID, Name: " farmer 2 ", RoleID: w.roles[farmer].ID}); !errors.Is(err, mwanachamaagency.ErrDuplicateScenarioActor) {
		t.Errorf("a repeated name: %v, want ErrDuplicateScenarioActor", err)
	}
	if _, err := w.mgr.AddScenarioActor(ctx, models.ScenarioActor{ScenarioID: s.ID, Name: "X", RoleID: "no-such-role"}); !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Errorf("an unknown role: %v, want ErrRoleNotFound", err)
	}
	if _, err := w.mgr.AddScenarioActor(ctx, models.ScenarioActor{ScenarioID: "no-such-scenario", Name: "X", RoleID: w.roles[farmer].ID}); !errors.Is(err, mwanachamaagency.ErrScenarioNotFound) {
		t.Errorf("an unknown scenario: %v, want ErrScenarioNotFound", err)
	}
}

func TestRenameScenarioActor_KeepsNamesUniqueAndFollowsIntoTheBlocks(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow, w.actor("Farmer 1", farmer), w.actor("Shop", supplier))
	var farmerActor models.ScenarioActor
	for _, a := range s.Actors {
		if a.Name == "Farmer 1" {
			farmerActor = a
		}
	}

	if _, err := w.mgr.RenameScenarioActor(ctx, farmerActor.ID, "Shop"); !errors.Is(err, mwanachamaagency.ErrDuplicateScenarioActor) {
		t.Errorf("renaming onto another actor: %v, want ErrDuplicateScenarioActor", err)
	}
	if _, err := w.mgr.RenameScenarioActor(ctx, farmerActor.ID, "  "); !errors.Is(err, mwanachamaagency.ErrInvalidScenario) {
		t.Errorf("renaming to nothing: %v, want ErrInvalidScenario", err)
	}
	if renamed, err := w.mgr.RenameScenarioActor(ctx, farmerActor.ID, "Farmer 1"); err != nil || renamed.Name != "Farmer 1" {
		t.Errorf("renaming to its own name: %v %+v, want it allowed", err, renamed)
	}
	renamed, err := w.mgr.RenameScenarioActor(ctx, farmerActor.ID, "Mama Akinyi")
	if err != nil {
		t.Fatalf("RenameScenarioActor: %v", err)
	}
	if renamed.Name != "Mama Akinyi" {
		t.Errorf("renamed = %+v", renamed)
	}
	if got := w.get(t, s.ID).Blocks[0].Actor; got != "Mama Akinyi" {
		t.Errorf("first block actor = %q, want the new name", got)
	}
	if _, err := w.mgr.RenameScenarioActor(ctx, "no-such-actor", "X"); !errors.Is(err, mwanachamaagency.ErrScenarioActorNotFound) {
		t.Errorf("an unknown actor: %v, want ErrScenarioActorNotFound", err)
	}
}

func TestScenarioEdits_SendTheScenarioBackToNotRun(t *testing.T) {
	ctx := context.Background()
	edits := map[string]func(w scenarioWorld, s models.Scenario) error{
		"update": func(w scenarioWorld, s models.Scenario) error {
			s.Name = "Renamed"
			_, err := w.mgr.UpdateScenario(ctx, s)
			return err
		},
		"add an actor": func(w scenarioWorld, s models.Scenario) error {
			_, err := w.mgr.AddScenarioActor(ctx, models.ScenarioActor{ScenarioID: s.ID, Name: "Shop", RoleID: w.roles[supplier].ID})
			return err
		},
		"rename an actor": func(w scenarioWorld, s models.Scenario) error {
			_, err := w.mgr.RenameScenarioActor(ctx, s.Actors[0].ID, "Someone")
			return err
		},
		"remove an actor": func(w scenarioWorld, s models.Scenario) error {
			return w.mgr.RemoveScenarioActor(ctx, s.Actors[0].ID)
		},
		"set an input": func(w scenarioWorld, s models.Scenario) error {
			_, err := w.mgr.SetScenarioInput(ctx, models.ScenarioInput{ScenarioID: s.ID, StepID: s.Blocks[0].StepID, Field: "Enter the product", Value: "urea"})
			return err
		},
	}
	for name, edit := range edits {
		t.Run(name, func(t *testing.T) {
			w := newScenarioWorld(t)
			s := w.scenario(t, "Shop says yes", w.buyNow, w.actor("Farmer 1", farmer))
			w.pass(t, s.ID)
			if err := edit(w, w.get(t, s.ID)); err != nil {
				t.Fatalf("edit: %v", err)
			}
			if got := w.get(t, s.ID).Result; got != models.ScenarioNotRun {
				t.Errorf("result after %s = %q, want not_run", name, got)
			}
		})
	}
}

func TestUpdateScenario_ChangesTheWordsAndNotTheWorkflow(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow)
	w.pass(t, s.ID)

	cur := w.get(t, s.ID)
	cur.Name, cur.Summary = "Shop agrees", "the shop agrees at once"
	out, err := w.mgr.UpdateScenario(ctx, cur)
	if err != nil {
		t.Fatalf("UpdateScenario: %v", err)
	}
	if out.Name != "Shop agrees" || out.Summary != "the shop agrees at once" || out.Result != models.ScenarioNotRun {
		t.Errorf("updated = %+v", out)
	}
	if out.LastRunID == "" {
		t.Errorf("last_run_id was cleared; the old runs stay in the Runs list")
	}

	moved := w.get(t, s.ID)
	moved.WorkflowID = w.groupOrd.ID
	if _, err := w.mgr.UpdateScenario(ctx, moved); !errors.Is(err, mwanachamaagency.ErrInvalidScenario) {
		t.Errorf("moving to another workflow: %v, want ErrInvalidScenario", err)
	}
	blank := w.get(t, s.ID)
	blank.Name = ""
	if _, err := w.mgr.UpdateScenario(ctx, blank); !errors.Is(err, mwanachamaagency.ErrInvalidScenario) {
		t.Errorf("a blank name: %v, want ErrInvalidScenario", err)
	}
	forged := w.get(t, s.ID)
	forged.Result = models.ScenarioPassed
	if got, err := w.mgr.UpdateScenario(ctx, forged); err != nil || got.Result != models.ScenarioNotRun {
		t.Errorf("a caller-set result: %v %q, want it ignored and the scenario not run", err, got.Result)
	}
	if _, err := w.mgr.UpdateScenario(ctx, models.Scenario{ID: "no-such-scenario", Name: "x", WorkflowID: w.buyNow.ID}); !errors.Is(err, mwanachamaagency.ErrScenarioNotFound) {
		t.Errorf("an unknown scenario: %v, want ErrScenarioNotFound", err)
	}
}

func TestSetScenarioInput_FixesAFieldAndFreeingItDropsTheRow(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow)
	ask, answered := s.Blocks[0].StepID, s.Blocks[1].StepID

	in, err := w.mgr.SetScenarioInput(ctx, models.ScenarioInput{ScenarioID: s.ID, StepID: ask, Field: "Enter the product", Value: "urea"})
	if err != nil || in.ID == "" {
		t.Fatalf("SetScenarioInput: %v %+v", err, in)
	}
	again, err := w.mgr.SetScenarioInput(ctx, models.ScenarioInput{ScenarioID: s.ID, StepID: ask, Field: "Enter the product", Value: "lime"})
	if err != nil || again.ID != in.ID {
		t.Fatalf("fixing the same field twice: %v %+v, want the same row changed", err, again)
	}
	got := w.get(t, s.ID)
	if len(got.Inputs) != 1 || got.Inputs[0].Value != "lime" {
		t.Errorf("inputs = %+v, want one row holding lime", got.Inputs)
	}
	product := got.Blocks[0].Fields[1]
	if product.Prompt != "Enter the product" || product.Free || product.Value != "lime" {
		t.Errorf("block field = %+v, want it fixed to lime", product)
	}
	if !got.Blocks[0].Fields[0].Free || !got.Blocks[1].Fields[0].Free {
		t.Errorf("fields nobody fixed are not Free: %+v", got.Blocks)
	}

	if _, err := w.mgr.SetScenarioInput(ctx, models.ScenarioInput{ScenarioID: s.ID, StepID: ask, Field: "Enter the product", Value: ""}); err != nil {
		t.Fatalf("freeing a field: %v", err)
	}
	got = w.get(t, s.ID)
	if len(got.Inputs) != 0 || !got.Blocks[0].Fields[1].Free {
		t.Errorf("after freeing: inputs = %+v, field = %+v, want no row and Free", got.Inputs, got.Blocks[0].Fields[1])
	}

	other := w.stepsOf(t, w.draftID, w.groupOrd)[0]
	for name, in := range map[string]models.ScenarioInput{
		"a step of another workflow": {ScenarioID: s.ID, StepID: other.ID, Field: "Select the target group", Value: "x"},
		"a step that does not exist": {ScenarioID: s.ID, StepID: "no-such-step", Field: "x", Value: "x"},
		"a question the step lacks":  {ScenarioID: s.ID, StepID: answered, Field: "Enter the product", Value: "x"},
		"no field":                   {ScenarioID: s.ID, StepID: ask, Field: "", Value: "x"},
	} {
		if _, err := w.mgr.SetScenarioInput(ctx, in); !errors.Is(err, mwanachamaagency.ErrInvalidScenario) {
			t.Errorf("%s: %v, want ErrInvalidScenario", name, err)
		}
	}
	if _, err := w.mgr.SetScenarioInput(ctx, models.ScenarioInput{ScenarioID: "no-such-scenario", StepID: ask, Field: "x", Value: "x"}); !errors.Is(err, mwanachamaagency.ErrScenarioNotFound) {
		t.Errorf("an unknown scenario: %v, want ErrScenarioNotFound", err)
	}
}

func TestListScenarios_FiltersAndScopesToTheDraft(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	a := w.scenario(t, "Shop says yes", w.buyNow)
	w.scenario(t, "Group joins", w.groupOrd)
	w.pass(t, a.ID)

	names := func(f mwanachamaagency.ScenarioFilter) string {
		out, err := w.mgr.ListScenarios(ctx, w.draftID, f)
		if err != nil {
			t.Fatalf("ListScenarios: %v", err)
		}
		var n []string
		for _, s := range out {
			n = append(n, s.Name)
		}
		return strings.Join(n, ",")
	}
	if got := names(mwanachamaagency.ScenarioFilter{}); got != "Shop says yes,Group joins" {
		t.Errorf("all = %q, want both in the order written", got)
	}
	if got := names(mwanachamaagency.ScenarioFilter{Result: "passed"}); got != "Shop says yes" {
		t.Errorf("passed = %q", got)
	}
	if got := names(mwanachamaagency.ScenarioFilter{Name: "GROUP"}); got != "Group joins" {
		t.Errorf("name filter = %q", got)
	}
	other, _ := w.mgr.CreateDraft(ctx, "other", "", "")
	if out, err := w.mgr.ListScenarios(ctx, other.ID, mwanachamaagency.ScenarioFilter{}); err != nil || len(out) != 0 {
		t.Errorf("another draft lists %v %v, want nothing", out, err)
	}
}

func TestDeleteScenario_TakesItsActorsInputsAndRunsWithIt(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow, w.actor("Farmer 1", farmer))
	if _, err := w.mgr.SetScenarioInput(ctx, models.ScenarioInput{ScenarioID: s.ID, StepID: s.Blocks[0].StepID, Field: "Enter the product", Value: "urea"}); err != nil {
		t.Fatalf("SetScenarioInput: %v", err)
	}
	run, err := w.runToEnd(ctx, s.ID)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}

	if err := w.mgr.DeleteScenario(ctx, s.ID); err != nil {
		t.Fatalf("DeleteScenario: %v", err)
	}
	if _, err := w.mgr.GetScenario(ctx, s.ID); !errors.Is(err, mwanachamaagency.ErrScenarioNotFound) {
		t.Errorf("GetScenario after delete: %v", err)
	}
	if _, err := w.mgr.GetTestRun(ctx, run.ID); !errors.Is(err, mwanachamaagency.ErrTestRunNotFound) {
		t.Errorf("GetTestRun after delete: %v, want its run gone", err)
	}
	for _, table := range []string{"scenario_actors", "scenario_inputs", "test_runs"} {
		var n int64
		w.db.Table(w.tableName(t, table)).Count(&n)
		if n != 0 {
			t.Errorf("%s still holds %d rows", table, n)
		}
	}
	if err := w.mgr.DeleteScenario(ctx, s.ID); !errors.Is(err, mwanachamaagency.ErrScenarioNotFound) {
		t.Errorf("deleting twice: %v, want ErrScenarioNotFound", err)
	}
}

func (w scenarioWorld) tableName(t *testing.T, table string) string {
	t.Helper()
	s, err := mwanachamaagency.SpecFor("test")
	if err != nil {
		t.Fatal(err)
	}
	tables := mwanachamaagency.TablesFor(s)
	return map[string]string{
		"scenario_actors": tables.ScenarioActors, "scenario_inputs": tables.ScenarioInputs, "test_runs": tables.TestRuns,
	}[table]
}

func TestDeleteWorkItem_DropsTheInputsFixedOnIt(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow)
	ask := s.Blocks[0].StepID
	if _, err := w.mgr.SetScenarioInput(ctx, models.ScenarioInput{ScenarioID: s.ID, StepID: ask, Field: "Enter the product", Value: "urea"}); err != nil {
		t.Fatal(err)
	}
	if err := w.mgr.DeleteWorkItem(ctx, ask); err != nil {
		t.Fatalf("DeleteWorkItem: %v", err)
	}
	got := w.get(t, s.ID)
	if len(got.Inputs) != 0 || len(got.Blocks) != 2 {
		t.Errorf("after deleting the step: %d inputs, %d blocks, want none and 2", len(got.Inputs), len(got.Blocks))
	}
}
