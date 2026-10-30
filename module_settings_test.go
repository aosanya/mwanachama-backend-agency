package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func newSettingsCheckedManager(t *testing.T, check mwanachamaagency.SettingsChecker) mwanachamaagency.AgencyManager {
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
	mgr, err := mwanachamaagency.NewAgencyManager(db, s, mwanachamaagency.WithSettingsChecker(check))
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	return mgr
}

func requireWindow(_ context.Context, module string, settings json.RawMessage) []string {
	if module != "forms" {
		return nil
	}
	var values struct {
		WindowDays int `json:"window_days"`
	}
	_ = json.Unmarshal(settings, &values)
	if values.WindowDays <= 0 {
		return []string{"window_days must be positive"}
	}
	return nil
}

func TestModuleSettingsFailingTheModulesCheckBlockPublishThenPublishWithTheDraft(t *testing.T) {
	mgr := newSettingsCheckedManager(t, requireWindow)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)

	forms, err := seededModule(mgr, ctx, o.DraftID, "forms")
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	if _, err := mgr.SetModuleEnabled(ctx, o.DraftID, forms.ID, true); err != nil {
		t.Fatalf("SetModuleEnabled: %v", err)
	}

	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) {
		t.Fatalf("expected the module's own check to block publish, got %v", err)
	}
	issues, err := mgr.ListIssues(ctx, o.DraftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	found := false
	for _, issue := range issues {
		if issue.EntityType == "module" && issue.EntityID == forms.ID && issue.Severity == models.IssueSeverityError &&
			strings.Contains(issue.Message, "window_days must be positive") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a blocking module settings issue, got %+v", issues)
	}

	updated, err := mgr.SetModuleSettings(ctx, forms.ID, json.RawMessage(`{"window_days": 14}`))
	if err != nil {
		t.Fatalf("SetModuleSettings: %v", err)
	}
	if updated.Settings != `{"window_days": 14}` {
		t.Fatalf("settings not carried on the row: %q", updated.Settings)
	}

	pub, err := mgr.PublishAgency(ctx, o.DraftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	doc := pub.Document
	published := false
	for _, mod := range doc.Modules {
		if mod.Code == "forms" && strings.Contains(string(mod.Settings), `"window_days"`) {
			published = true
		}
	}
	if !published {
		t.Fatalf("expected the forms settings in the published document, got %+v", doc.Modules)
	}

	if _, err := mgr.SetModuleSettings(ctx, forms.ID, json.RawMessage(`{"window_days": 30}`)); err != nil {
		t.Fatalf("SetModuleSettings: %v", err)
	}
	second, err := mgr.PublishAgency(ctx, o.DraftID, "", "")
	if err != nil {
		t.Fatalf("a settings-only change should be a new version: %v", err)
	}
	if second.Version != pub.Version+1 {
		t.Fatalf("expected version %d, got %d", pub.Version+1, second.Version)
	}

	fork, err := mgr.CreateDraft(ctx, "from the tag", second.Tag, "publication")
	if err != nil {
		t.Fatalf("CreateDraft from publication: %v", err)
	}
	mods, err := mgr.ListModules(ctx, fork.ID, mwanachamaagency.ModuleFilter{Code: "forms"})
	if err != nil || len(mods) != 1 || !strings.Contains(mods[0].Settings, "30") {
		t.Fatalf("expected the fork to carry the published settings, got %+v %v", mods, err)
	}
}

func TestSetModuleSettingsRefusesAnythingButAnObject(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)
	mod, err := seededModule(mgr, ctx, draftID, "forms")
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	for _, raw := range []string{`[1]`, `"x"`, `42`, `{`} {
		if _, err := mgr.SetModuleSettings(ctx, mod.ID, json.RawMessage(raw)); !errors.Is(err, mwanachamaagency.ErrInvalidModuleSettings) {
			t.Fatalf("%s: expected ErrInvalidModuleSettings, got %v", raw, err)
		}
	}
	cleared, err := mgr.SetModuleSettings(ctx, mod.ID, nil)
	if err != nil || cleared.Settings != "" {
		t.Fatalf("expected clearing to succeed, got %+v %v", cleared, err)
	}
	if _, err := mgr.SetModuleSettings(ctx, "missing", json.RawMessage(`{}`)); !errors.Is(err, mwanachamaagency.ErrModuleNotFound) {
		t.Fatalf("expected ErrModuleNotFound, got %v", err)
	}
}

func TestNoSettingsCheckerMeansNoSettingsCheck(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	forms, err := seededModule(mgr, ctx, o.DraftID, "forms")
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	if _, err := mgr.SetModuleEnabled(ctx, o.DraftID, forms.ID, true); err != nil {
		t.Fatalf("SetModuleEnabled: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); err != nil {
		t.Fatalf("expected publish with no checker to pass, got %v", err)
	}
}

func TestASwitchedOffModulesBadSettingsWarnAndSwitchingItOnBlocks(t *testing.T) {
	mgr := newSettingsCheckedManager(t, requireWindow)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)

	forms, err := seededModule(mgr, ctx, o.DraftID, "forms")
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	if _, err := mgr.SetModuleSettings(ctx, forms.ID, json.RawMessage(`{"window_days": 0}`)); err != nil {
		t.Fatalf("SetModuleSettings: %v", err)
	}

	issues, err := mgr.ListIssues(ctx, o.DraftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	warned := false
	for _, issue := range issues {
		if issue.EntityType != "module" || issue.EntityID != forms.ID {
			continue
		}
		if issue.Severity != models.IssueSeverityWarning {
			t.Fatalf("a switched-off module's bad settings must warn, not block: %+v", issue)
		}
		if strings.Contains(issue.Message, "window_days must be positive") {
			warned = true
		}
	}
	if !warned {
		t.Fatalf("expected a warning for the switched-off module's settings, got %+v", issues)
	}
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); err != nil {
		t.Fatalf("a warning must not block publish, got %v", err)
	}

	if _, err := mgr.SetModuleEnabled(ctx, o.DraftID, forms.ID, true); err != nil {
		t.Fatalf("SetModuleEnabled: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) {
		t.Fatalf("switching the module on must check its settings again and block, got %v", err)
	}
}
