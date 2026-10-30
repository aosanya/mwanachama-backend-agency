package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func votingModes(_ context.Context, module string, settings json.RawMessage) []string {
	if module != "voting" {
		return nil
	}
	var values struct {
		Mode string `json:"mode"`
	}
	_ = json.Unmarshal(settings, &values)
	switch values.Mode {
	case "", "one", "up_to", "quadratic":
		return nil
	}
	return []string{"mode must be one, up_to or quadratic"}
}

func TestVotingIsAModuleTheDefinitionCanSwitchOn(t *testing.T) {
	found := false
	for _, entry := range models.ModuleCatalog {
		if entry.Code == "voting" {
			found = true
			if entry.Enabled || entry.Locked {
				t.Fatalf("voting must start off and be switchable, got %+v", entry)
			}
		}
	}
	if !found {
		t.Fatal("voting is missing from the module catalog")
	}
	power, ok := models.LookupPower("poll.run")
	if !ok || power.Locked || power.Name != "Run a poll" {
		t.Fatalf("poll.run = %+v, %v; want an unlocked Run a poll", power, ok)
	}
}

func TestVotingDefinition_SettingsAndRunAPollPublishAndForkTogether(t *testing.T) {
	mgr := newSettingsCheckedManager(t, votingModes)
	ctx := context.Background()
	_, o, role := seedPublishableAgency(t, mgr, ctx)

	voting, err := seededModule(mgr, ctx, o.DraftID, "voting")
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	if _, err := mgr.SetModuleEnabled(ctx, o.DraftID, voting.ID, true); err != nil {
		t.Fatalf("SetModuleEnabled: %v", err)
	}
	role.Powers = []string{"poll.run"}
	if _, err := mgr.UpdateRole(ctx, role); err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}

	if _, err := mgr.SetModuleSettings(ctx, voting.ID, json.RawMessage(`{"mode": "ranked"}`)); err != nil {
		t.Fatalf("SetModuleSettings: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) {
		t.Fatalf("expected voting's own check to block an unknown mode, got %v", err)
	}

	defaults := `{"mode":"quadratic","budget":100,"change_allowed":true,"per_group":"once"}`
	if _, err := mgr.SetModuleSettings(ctx, voting.ID, json.RawMessage(defaults)); err != nil {
		t.Fatalf("SetModuleSettings: %v", err)
	}
	pub, err := mgr.PublishAgency(ctx, o.DraftID, "voting-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	var published *models.DocumentModule
	for i := range pub.Document.Modules {
		if pub.Document.Modules[i].Code == "voting" {
			published = &pub.Document.Modules[i]
		}
	}
	if published == nil || string(published.Settings) != defaults {
		t.Fatalf("published voting module = %+v, want settings %s", published, defaults)
	}
	if len(pub.Document.Roles) != 1 || !reflect.DeepEqual(pub.Document.Roles[0].Powers, []string{"poll.run"}) {
		t.Fatalf("document roles = %+v, want Run a poll", pub.Document.Roles)
	}

	forked, err := mgr.CreateDraft(ctx, "revisit voting-1", "voting-1", "publication")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	mods, err := mgr.ListModules(ctx, forked.ID, mwanachamaagency.ModuleFilter{Code: "voting"})
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}
	if len(mods) != 1 || mods[0].Settings != defaults {
		t.Fatalf("forked voting modules = %+v, want settings %s", mods, defaults)
	}
	roles, err := mgr.ListRoles(ctx, forked.ID, mwanachamaagency.RoleFilter{})
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	if len(roles) != 1 || !reflect.DeepEqual(roles[0].Powers, []string{"poll.run"}) {
		t.Fatalf("forked roles = %+v, want Run a poll", roles)
	}
}
