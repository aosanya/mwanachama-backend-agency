package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func bootstrapDraftID(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context) string {
	t.Helper()
	drafts, err := mgr.ListDrafts(ctx, mwanachamaagency.DraftFilter{})
	if err != nil {
		t.Fatalf("ListDrafts: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("expected exactly 1 bootstrap draft right after Migrate, got %d: %+v", len(drafts), drafts)
	}
	return drafts[0].ID
}

func TestCreateAndGetModule(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)

	mod, err := mgr.CreateModule(ctx, models.Module{DraftID: draftID, Code: "custom-module", Name: "Custom", Description: "a custom capability", Enabled: true})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	if mod.ID == "" {
		t.Fatal("expected minted ID")
	}
	if !mod.Enabled || !mod.Explicit || mod.NeededBySteps {
		t.Fatalf("expected a module created switched on to read enabled and explicit, got %+v", mod)
	}

	got, err := mgr.GetModule(ctx, mod.ID)
	if err != nil {
		t.Fatalf("GetModule: %v", err)
	}
	if got.Code != "custom-module" || got.Name != "Custom" {
		t.Fatalf("unexpected module: %+v", got)
	}
}

func TestGetModule_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.GetModule(context.Background(), "unknown"); !errors.Is(err, mwanachamaagency.ErrModuleNotFound) {
		t.Fatalf("expected ErrModuleNotFound, got %v", err)
	}
}

func TestCreateModule_EmptyDraftID(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.CreateModule(context.Background(), models.Module{Name: "no draft"}); !errors.Is(err, mwanachamaagency.ErrDraftRequired) {
		t.Fatalf("expected ErrDraftRequired, got %v", err)
	}
}

func TestCreateModule_EmptyCode(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)
	if _, err := mgr.CreateModule(ctx, models.Module{DraftID: draftID, Name: "no code"}); !errors.Is(err, mwanachamaagency.ErrInvalidModule) {
		t.Fatalf("expected ErrInvalidModule, got %v", err)
	}
}

func TestCreateModule_DuplicateCode(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := bootstrapDraftID(t, mgr, ctx)

	if _, err := mgr.CreateModule(ctx, models.Module{DraftID: draftID, Code: "git", Name: "Git again"}); !errors.Is(err, mwanachamaagency.ErrDuplicateModule) {
		t.Fatalf("expected ErrDuplicateModule for an already-backfilled code, got %v", err)
	}

	if _, err := mgr.CreateModule(ctx, models.Module{DraftID: draftID, Code: "brand-new", Name: "Brand New"}); err != nil {
		t.Fatalf("CreateModule(brand-new): %v", err)
	}
	if _, err := mgr.CreateModule(ctx, models.Module{DraftID: draftID, Code: "brand-new", Name: "Brand New Again"}); !errors.Is(err, mwanachamaagency.ErrDuplicateModule) {
		t.Fatalf("expected ErrDuplicateModule for a repeat of a freshly created code, got %v", err)
	}
}

func TestBackfillModules_SeedsCatalogDefaults(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := bootstrapDraftID(t, mgr, ctx)

	mods, err := mgr.ListModules(ctx, draftID, mwanachamaagency.ModuleFilter{})
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}
	if len(mods) != len(models.ModuleCatalog) {
		t.Fatalf("expected %d backfilled modules, got %d: %+v", len(models.ModuleCatalog), len(mods), mods)
	}
	byCode := make(map[string]models.Module, len(mods))
	for _, m := range mods {
		byCode[m.Code] = m
	}
	for _, entry := range models.ModuleCatalog {
		mod, ok := byCode[entry.Code]
		if !ok {
			t.Fatalf("expected a backfilled module for code %q", entry.Code)
		}
		if mod.Enabled != entry.Enabled {
			t.Fatalf("expected backfilled module %q Enabled=%v, got %+v", entry.Code, entry.Enabled, mod)
		}
		if mod.Locked != entry.Locked {
			t.Fatalf("expected backfilled module %q Locked=%v, got %+v", entry.Code, entry.Locked, mod)
		}
	}

	neverOff := map[string]bool{"auth": true, "actor": true}
	onByDefault := map[string]bool{"auth": true, "actor": true, "comm": true}
	for code := range onByDefault {
		mod, ok := byCode[code]
		if !ok {
			t.Fatalf("expected a backfilled %s module", code)
		}
		if !mod.Enabled || !mod.Explicit {
			t.Fatalf("expected %s to backfill switched on, got %+v", code, mod)
		}
	}
	for code := range neverOff {
		mod, ok := byCode[code]
		if !ok {
			t.Fatalf("expected a backfilled %s module", code)
		}
		if !mod.Enabled || !mod.Locked {
			t.Fatalf("expected %s to backfill Enabled: true, Locked: true, got %+v", code, mod)
		}
	}

	if byCode["comm"].Locked {
		t.Fatalf("comm is on by default but must stay switchable, got %+v", byCode["comm"])
	}
	for _, entry := range models.ModuleCatalog {
		if onByDefault[entry.Code] {
			continue
		}
		mod := byCode[entry.Code]
		if mod.Enabled || mod.Locked {
			t.Fatalf("expected module %q to backfill Enabled: false, Locked: false, got %+v", entry.Code, mod)
		}
	}
}

func TestListModules_Filter(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := o.DraftID
	if _, err := mgr.CreateModule(ctx, models.Module{DraftID: draftID, Code: "widgets", Name: "Widgets", Enabled: true}); err != nil {
		t.Fatalf("CreateModule: %v", err)
	}

	byCode, err := mgr.ListModules(ctx, draftID, mwanachamaagency.ModuleFilter{Code: "widg"})
	if err != nil {
		t.Fatalf("ListModules(code filter): %v", err)
	}
	if len(byCode) != 1 || byCode[0].Code != "widgets" {
		t.Fatalf("expected exactly the widgets module, got %+v", byCode)
	}

	trueVal := true
	enabled, err := mgr.ListModules(ctx, draftID, mwanachamaagency.ModuleFilter{Enabled: &trueVal})
	if err != nil {
		t.Fatalf("ListModules(enabled filter): %v", err)
	}
	want := map[string]bool{"widgets": true}
	for _, entry := range models.ModuleCatalog {
		if entry.Locked || entry.Enabled {
			want[entry.Code] = true
		}
	}
	got := map[string]bool{}
	for _, mod := range enabled {
		if !want[mod.Code] {
			t.Fatalf("did not expect %q to be enabled: %+v", mod.Code, enabled)
		}
		got[mod.Code] = true
	}
	if len(got) != len(want) || len(enabled) != len(want) {
		t.Fatalf("expected widgets and the default-on catalog modules to be enabled, got %d: %+v", len(enabled), enabled)
	}
}

func addStep(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID string, kind models.StepKind) models.WorkItem {
	t.Helper()
	objectives, err := mgr.ListObjectives(ctx, draftID, mwanachamaagency.ObjectiveFilter{})
	if err != nil || len(objectives) == 0 {
		t.Fatalf("ListObjectives: %+v %v", objectives, err)
	}
	workflows, err := mgr.ListWorkflows(ctx, draftID, mwanachamaagency.WorkflowFilter{})
	if err != nil || len(workflows) == 0 {
		t.Fatalf("ListWorkflows: %+v %v", workflows, err)
	}
	roleID := ""
	if kind != models.StepKindSendForm {
		role, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "Doer " + string(kind)})
		if err != nil {
			t.Fatalf("CreateRole: %v", err)
		}
		roleID = role.ID
	}
	item := models.WorkItem{
		DraftID: draftID, WorkflowID: workflows[0].ID, ObjectiveID: objectives[0].ID,
		Title: "step " + string(kind), Ordinality: 50, Kind: kind, RoleID: roleID, FormName: "A form",
		Questions: []models.StepQuestion{{Prompt: "Write the message", AnswerType: models.AnswerFreeText, Required: true}},
	}
	created, err := mgr.CreateWorkItem(ctx, item)
	if err != nil {
		t.Fatalf("CreateWorkItem(%s): %v", kind, err)
	}
	return created
}

func TestModuleEnabled_NeededByAFillFormStep(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := o.DraftID

	before := moduleByCode(t, mgr, ctx, draftID, "forms")
	if before.Enabled || before.Explicit || before.NeededBySteps {
		t.Fatalf("expected forms to start off with no steps, got %+v", before)
	}

	addStep(t, mgr, ctx, draftID, models.StepKindFillForm)

	after := moduleByCode(t, mgr, ctx, draftID, "forms")
	if !after.Enabled || after.Explicit || !after.NeededBySteps {
		t.Fatalf("expected forms enabled by a fill_form step with the stored flag still off, got %+v", after)
	}
	if got, err := mgr.GetModule(ctx, before.ID); err != nil || got != after {
		t.Fatalf("expected GetModule to agree with the listing, got %+v (%v) want %+v", got, err, after)
	}
	for _, code := range []string{"catalog", "git"} {
		if got := moduleByCode(t, mgr, ctx, draftID, code); !got.Enabled || got.Explicit || got.NeededBySteps {
			t.Fatalf("expected %s enabled only because forms requires it, got %+v", code, got)
		}
	}
}

func TestModuleEnabled_EveryFormsStepKindNeedsForms(t *testing.T) {
	for _, kind := range []models.StepKind{models.StepKindFillForm, models.StepKindSendForm, models.StepKindYesNo} {
		t.Run(string(kind), func(t *testing.T) {
			mgr := newTestManager(t)
			ctx := context.Background()
			_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
			addStep(t, mgr, ctx, o.DraftID, kind)
			if got := moduleByCode(t, mgr, ctx, o.DraftID, "forms"); !got.Enabled || !got.NeededBySteps {
				t.Fatalf("expected a %s step to need forms, got %+v", kind, got)
			}
		})
	}
}

func TestModuleEnabled_NotifyStepNeedsCommEvenWhenSwitchedOff(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := o.DraftID
	comm := moduleByCode(t, mgr, ctx, draftID, "comm")

	off, err := mgr.SetModuleEnabled(ctx, draftID, comm.ID, false)
	if err != nil {
		t.Fatalf("SetModuleEnabled: %v", err)
	}
	if off.Enabled || off.Explicit || off.NeededBySteps {
		t.Fatalf("expected comm off with no notify step, got %+v", off)
	}

	addStep(t, mgr, ctx, draftID, models.StepKindNotify)

	needed := moduleByCode(t, mgr, ctx, draftID, "comm")
	if !needed.Enabled || needed.Explicit || !needed.NeededBySteps {
		t.Fatalf("expected a notify step to keep comm running while its switch is off, got %+v", needed)
	}
}

func TestModuleEnabled_FollowsTheStepsThatNeedIt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := o.DraftID

	step := addStep(t, mgr, ctx, draftID, models.StepKindFillForm)
	if !moduleByCode(t, mgr, ctx, draftID, "forms").Enabled {
		t.Fatal("expected forms enabled while a fill_form step exists")
	}
	if err := mgr.DeleteWorkItem(ctx, step.ID); err != nil {
		t.Fatalf("DeleteWorkItem: %v", err)
	}
	if got := moduleByCode(t, mgr, ctx, draftID, "forms"); got.Enabled || got.NeededBySteps {
		t.Fatalf("expected forms off again once its step is gone, got %+v", got)
	}
}

func TestModuleEnabled_FromTheStoredFlag(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := o.DraftID
	taskmanager := moduleByCode(t, mgr, ctx, draftID, "taskmanager")
	if taskmanager.Enabled || taskmanager.Explicit {
		t.Fatalf("expected taskmanager to start off, got %+v", taskmanager)
	}

	on, err := mgr.SetModuleEnabled(ctx, draftID, taskmanager.ID, true)
	if err != nil {
		t.Fatalf("SetModuleEnabled: %v", err)
	}
	if !on.Enabled || !on.Explicit || on.NeededBySteps {
		t.Fatalf("expected the stored flag alone to switch it on, got %+v", on)
	}
	if got := moduleByCode(t, mgr, ctx, draftID, "taskmanager"); got != on {
		t.Fatalf("expected the listing to agree, got %+v want %+v", got, on)
	}

	off, err := mgr.SetModuleEnabled(ctx, draftID, taskmanager.ID, false)
	if err != nil {
		t.Fatalf("SetModuleEnabled: %v", err)
	}
	if off.Enabled || off.Explicit {
		t.Fatalf("expected the flag to switch it off again, got %+v", off)
	}
}

func TestModuleEnabled_RequiresPullsInWhatItNeeds(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := o.DraftID
	forms := moduleByCode(t, mgr, ctx, draftID, "forms")

	if _, err := mgr.SetModuleEnabled(ctx, draftID, forms.ID, true); err != nil {
		t.Fatalf("SetModuleEnabled: %v", err)
	}
	for _, code := range models.ModuleRequires("forms") {
		got := moduleByCode(t, mgr, ctx, draftID, code)
		if !got.Enabled || got.Explicit || got.NeededBySteps {
			t.Fatalf("expected %s enabled only because forms requires it, got %+v", code, got)
		}
	}
}

func TestModuleEnabled_LockedStaysEnabled(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := bootstrapDraftID(t, mgr, ctx)

	auth := moduleByCode(t, mgr, ctx, draftID, "auth")
	if !auth.Locked || !auth.Enabled {
		t.Fatalf("expected auth to be locked and enabled with no steps at all, got %+v", auth)
	}
}

func TestSetModuleEnabled_RefusesSwitchingALockedModuleOff(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := bootstrapDraftID(t, mgr, ctx)
	auth := moduleByCode(t, mgr, ctx, draftID, "auth")

	if _, err := mgr.SetModuleEnabled(ctx, draftID, auth.ID, false); !errors.Is(err, mwanachamaagency.ErrModuleLocked) {
		t.Fatalf("expected ErrModuleLocked, got %v", err)
	}
	if got := moduleByCode(t, mgr, ctx, draftID, "auth"); !got.Enabled || !got.Explicit {
		t.Fatalf("expected auth untouched by the refused switch, got %+v", got)
	}
	if _, err := mgr.SetModuleEnabled(ctx, draftID, auth.ID, true); err != nil {
		t.Fatalf("switching a locked module on is harmless, got %v", err)
	}
}

func TestSetModuleEnabled_RefusesAnotherDraftAnUnknownIDAndAnArchivedDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, goal, _, _, _ := seedLiveAgency(t, mgr, ctx)
	seedDraftID := goal.DraftID
	other, err := mgr.CreateDraft(ctx, "other", seedDraftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	git := moduleByCode(t, mgr, ctx, seedDraftID, "git")

	if _, err := mgr.SetModuleEnabled(ctx, other.ID, git.ID, true); !errors.Is(err, mwanachamaagency.ErrModuleNotFound) {
		t.Fatalf("expected ErrModuleNotFound for another draft's id, got %v", err)
	}
	if _, err := mgr.SetModuleEnabled(ctx, seedDraftID, "unknown", true); !errors.Is(err, mwanachamaagency.ErrModuleNotFound) {
		t.Fatalf("expected ErrModuleNotFound, got %v", err)
	}
	if moduleByCode(t, mgr, ctx, seedDraftID, "git").Explicit || moduleByCode(t, mgr, ctx, other.ID, "git").Explicit {
		t.Fatal("a refused call switched a module")
	}

	otherGit := moduleByCode(t, mgr, ctx, other.ID, "git")
	if _, err := mgr.ArchiveDraft(ctx, other.ID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}
	if _, err := mgr.SetModuleEnabled(ctx, other.ID, otherGit.ID, true); !errors.Is(err, mwanachamaagency.ErrDraftNotOpen) {
		t.Fatalf("expected ErrDraftNotOpen, got %v", err)
	}
}

func TestModuleEnabled_ScopedToItsOwnDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, seedGoal, _, _, _ := seedLiveAgency(t, mgr, ctx)
	seedDraftID := seedGoal.DraftID

	d, err := mgr.CreateDraft(ctx, "d", seedDraftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	seedGit := moduleByCode(t, mgr, ctx, seedDraftID, "git")
	draftGit := moduleByCode(t, mgr, ctx, d.ID, "git")
	if draftGit.ID == seedGit.ID {
		t.Fatal("expected the fork to mint a new Module id")
	}

	if _, err := mgr.SetModuleEnabled(ctx, d.ID, draftGit.ID, true); err != nil {
		t.Fatalf("SetModuleEnabled(draft): %v", err)
	}
	if got := moduleByCode(t, mgr, ctx, d.ID, "git"); !got.Enabled {
		t.Fatalf("expected the new draft's git copy to be enabled, got %+v", got)
	}
	if got := moduleByCode(t, mgr, ctx, seedDraftID, "git"); got.Enabled {
		t.Fatalf("expected the seed draft's git to stay off, got %+v", got)
	}
}

func TestFreshAgencyListsAuthActorAndCommEnabledWithNoSteps(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "Fresh"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draft, err := mgr.CreateDraft(ctx, "fresh", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	mods, err := mgr.ListModules(ctx, draft.ID, mwanachamaagency.ModuleFilter{})
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}
	enabled := map[string]models.Module{}
	for _, mod := range mods {
		if mod.NeededBySteps {
			t.Fatalf("no steps exist yet, but %s reads needed_by_steps: %+v", mod.Code, mod)
		}
		if mod.Enabled {
			enabled[mod.Code] = mod
		}
	}
	if len(enabled) != 3 || enabled["auth"].ID == "" || enabled["actor"].ID == "" || enabled["comm"].ID == "" {
		t.Fatalf("expected exactly auth, actor and comm enabled, got %+v", enabled)
	}
	if !enabled["auth"].Locked || !enabled["actor"].Locked || enabled["comm"].Locked {
		t.Fatalf("expected auth and actor locked and comm not, got %+v", enabled)
	}
}

func moduleByCode(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID, code string) models.Module {
	t.Helper()
	mods, err := mgr.ListModules(ctx, draftID, mwanachamaagency.ModuleFilter{Code: code})
	if err != nil {
		t.Fatalf("ListModules(%q): %v", code, err)
	}
	for _, m := range mods {
		if m.Code == code {
			return m
		}
	}
	t.Fatalf("no module %q in scope %q: %+v", code, draftID, mods)
	return models.Module{}
}
