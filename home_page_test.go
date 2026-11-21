package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func projectHome() []models.DocumentHomeNode {
	return []models.DocumentHomeNode{
		{
			Key: "deliver", Title: "Deliver work",
			Entries: []models.DocumentHomeNode{
				{Key: "projects", Title: "Projects", Roles: []string{"R-3"}, Target: &models.HomeTarget{Type: models.HomeTargetRecord, Ref: "RC-4"}},
				{
					Key: "project-page", Title: "Project", PageOf: "RC-4",
					Entries: []models.DocumentHomeNode{
						{Key: "days", Title: "Days worked", Target: &models.HomeTarget{
							Type: models.HomeTargetRecord, Ref: "RC-6",
							Where: []models.HomeCondition{{Question: "Project", Operator: "is", Value: models.HomeValue{Row: "Project title"}}},
						}},
						{Key: "report", Title: "Progress report", Target: &models.HomeTarget{Type: models.HomeTargetWorkflow, Ref: "W-7", Step: "WI-48", Scope: models.HomeScopeRowOrder}},
					},
				},
			},
		},
		{Key: "mine", Title: "My work", Target: &models.HomeTarget{Type: models.HomeTargetStart}},
		{Key: "running", Title: "Running", Target: &models.HomeTarget{Type: models.HomeTargetOngoing}},
	}
}

func projectDocument(home []models.DocumentHomeNode) models.AgencyDocument {
	return models.AgencyDocument{
		Agency: models.DocumentAgency{Name: "Practice"},
		Roles:  []models.DocumentRole{{ID: "r3", Code: "R-3", Name: "Director"}},
		Home:   home,
	}
}

func TestImportDocument_KeepsRowPagesRolesConditionsStepsAndScopes(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	out, err := mgr.ImportDocument(ctx, "practice", projectDocument(projectHome()))
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	rows := homeRowsOf(t, mgr, ctx, out.Draft.ID)
	if got := homeKeyed(rows, "project-page"); got.PageOf != "RC-4" || !got.IsSection() || got.ParentID != homeKeyed(rows, "deliver").ID {
		t.Fatalf("page = %+v, want a section that is the page of RC-4 inside deliver", got)
	}
	if got := homeKeyed(rows, "projects"); len(got.Roles) != 1 || got.Roles[0] != "R-3" {
		t.Fatalf("projects = %+v, want its role kept", got)
	}
	days := homeKeyed(rows, "days")
	if len(days.Conditions) != 1 || days.Conditions[0].Value.Row != "Project title" || days.Conditions[0].Question != "Project" || days.Conditions[0].Operator != "is" {
		t.Fatalf("days = %+v, want its row placeholder condition kept", days)
	}
	if got := homeKeyed(rows, "report"); got.Step != "WI-48" || got.Scope != models.HomeScopeRowOrder || got.Target != "W-7" {
		t.Fatalf("report = %+v, want its step and scope kept", got)
	}
	if homeKeyed(rows, "mine").TargetType != models.HomeTargetStart || homeKeyed(rows, "running").TargetType != models.HomeTargetOngoing {
		t.Fatalf("rows = %+v, want Start and Ongoing kept", rows)
	}
	again := models.NestHomeEntries(rows)
	encoded, _ := json.Marshal(again)
	want, _ := json.Marshal(projectHome())
	if string(encoded) != string(want) {
		t.Fatalf("nested = %s, want %s", encoded, want)
	}
}

func TestImportDocument_ABrokenRowPageOrRoleRefusesTheWholeFile(t *testing.T) {
	row := func(prompt string) *models.HomeTarget {
		return &models.HomeTarget{Type: models.HomeTargetRecord, Ref: "RC-6", Where: []models.HomeCondition{{Question: "Project", Operator: "is", Value: models.HomeValue{Row: prompt}}}}
	}
	broken := map[string][]models.DocumentHomeNode{
		"a role the file does not hold":    {{Key: "a", Title: "A", Roles: []string{"R-99"}}},
		"a placeholder outside a row page": {{Key: "a", Title: "A", Target: row("Project title")}},
		"a scope outside a row page":       {{Key: "a", Title: "A", Target: &models.HomeTarget{Type: models.HomeTargetWorkflow, Ref: "W-1", Scope: models.HomeScopeRowOrder}}},
		"two pages for one record":         {{Key: "a", Title: "A", PageOf: "RC-4"}, {Key: "b", Title: "B", PageOf: "RC-4"}},
		"a page that is an entry":          {{Key: "a", Title: "A", PageOf: "RC-4", Target: &models.HomeTarget{Type: models.HomeTargetFiles}}},
		"a step on a record":               {{Key: "a", Title: "A", Target: &models.HomeTarget{Type: models.HomeTargetRecord, Ref: "RC-1", Step: "WI-1"}}},
		"a condition on a workflow":        {{Key: "a", Title: "A", Target: &models.HomeTarget{Type: models.HomeTargetWorkflow, Ref: "W-1", Where: []models.HomeCondition{{Question: "Q", Operator: "is", Value: models.HomeValue{Literal: "x"}}}}}},
		"an unknown operator":              {{Key: "a", Title: "A", Target: &models.HomeTarget{Type: models.HomeTargetRecord, Ref: "RC-1", Where: []models.HomeCondition{{Question: "Q", Operator: "like", Value: models.HomeValue{Literal: "x"}}}}}},
		"a start entry naming a ref":       {{Key: "a", Title: "A", Target: &models.HomeTarget{Type: models.HomeTargetStart, Ref: "x"}}},
		"a condition with no value":        {{Key: "a", Title: "A", Target: &models.HomeTarget{Type: models.HomeTargetRecord, Ref: "RC-1", Where: []models.HomeCondition{{Question: "Q", Operator: "is"}}}}},
	}
	for name, home := range broken {
		t.Run(name, func(t *testing.T) {
			mgr := newTestManager(t)
			ctx := context.Background()
			before, err := mgr.ListDrafts(ctx, mwanachamaagency.DraftFilter{})
			if err != nil {
				t.Fatalf("ListDrafts: %v", err)
			}
			if _, err := mgr.ImportDocument(ctx, "x", projectDocument(home)); !errors.Is(err, mwanachamaagency.ErrInvalidHomeEntry) {
				t.Fatalf("ImportDocument = %v, want ErrInvalidHomeEntry", err)
			}
			after, err := mgr.ListDrafts(ctx, mwanachamaagency.DraftFilter{})
			if err != nil || len(after) != len(before) {
				t.Fatalf("drafts = %+v, %v, want the refused import to have left no draft", after, err)
			}
		})
	}
}

func TestHomeCondition_ValueIsATextOrARowPlaceholderInJSON(t *testing.T) {
	var cond models.HomeCondition
	if err := json.Unmarshal([]byte(`{"question":"Project","operator":"is","value":{"row":"Project title"}}`), &cond); err != nil || cond.Value.Row != "Project title" || cond.Value.Literal != "" {
		t.Fatalf("placeholder = %+v, %v", cond, err)
	}
	if err := json.Unmarshal([]byte(`{"question":"Status","operator":"is","value":"Open"}`), &cond); err != nil || cond.Value.Literal != "Open" || cond.Value.Row != "" {
		t.Fatalf("literal = %+v, %v", cond, err)
	}
	if err := json.Unmarshal([]byte(`{"question":"Status","operator":"is","value":{"row":""}}`), &cond); err == nil {
		t.Fatal("a placeholder naming no question was accepted")
	}
	encoded, _ := json.Marshal(models.HomeCondition{Question: "P", Operator: "is", Value: models.HomeValue{Row: "R"}})
	if string(encoded) != `{"question":"P","operator":"is","value":{"row":"R"}}` {
		t.Fatalf("encoded = %s", encoded)
	}
}

func TestCreateHomeEntry_RolesPagesAndTheTappedRow(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, _, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := g.DraftID
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "Director"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	if _, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draftID, Key: "a", Title: "A", Roles: []string{"R-404"}}); !errors.Is(err, mwanachamaagency.ErrInvalidHomeEntry) {
		t.Fatalf("an unknown role = %v, want ErrInvalidHomeEntry", err)
	}
	if _, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draftID, Key: "a", Title: "A", Roles: []string{role.Code}}); err != nil {
		t.Fatalf("a role of the draft: %v", err)
	}
	page, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draftID, Key: "page", Title: "Project", PageOf: "RC-4"})
	if err != nil {
		t.Fatalf("the page: %v", err)
	}
	if _, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draftID, Key: "page-2", Title: "Project again", PageOf: "RC-4"}); !errors.Is(err, mwanachamaagency.ErrInvalidHomeEntry) {
		t.Fatalf("a second page for one record = %v, want ErrInvalidHomeEntry", err)
	}
	outside := models.HomeEntry{DraftID: draftID, Key: "days", Title: "Days", TargetType: models.HomeTargetRecord, Target: "RC-6",
		Conditions: []models.HomeCondition{{Question: "Project", Operator: "is", Value: models.HomeValue{Row: "Project title"}}}}
	if _, err := mgr.CreateHomeEntry(ctx, outside); !errors.Is(err, mwanachamaagency.ErrInvalidHomeEntry) {
		t.Fatalf("the tapped row outside a page = %v, want ErrInvalidHomeEntry", err)
	}
	outside.ParentID = page.ID
	if _, err := mgr.CreateHomeEntry(ctx, outside); err != nil {
		t.Fatalf("the tapped row inside a page: %v", err)
	}
}

func TestSetHome_ReplacesTheWholeHomeAndARefusalKeepsTheOldOne(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, _, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := g.DraftID
	if _, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "Director"}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	sent := func(nodes []models.DocumentHomeNode) []map[string]any {
		raw, _ := json.Marshal(nodes)
		var out []map[string]any
		_ = json.Unmarshal(raw, &out)
		return out
	}
	first := []models.DocumentHomeNode{{Key: "old", Title: "Old"}, {Key: "older", Title: "Older"}}
	if _, err := mgr.SetHome(ctx, draftID, sent(first)); err != nil {
		t.Fatalf("SetHome (first): %v", err)
	}

	rows, err := mgr.SetHome(ctx, draftID, sent(projectHome()[1:]))
	if err != nil || len(rows) != 2 || homeKeyed(rows, "mine").Key == "" {
		t.Fatalf("SetHome (second) = %+v, %v, want only the two rows sent", rows, err)
	}
	if listed := homeRowsOf(t, mgr, ctx, draftID); len(listed) != 2 || homeKeyed(listed, "old").Key != "" {
		t.Fatalf("listed = %+v, want the old rows gone", listed)
	}

	broken := sent([]models.DocumentHomeNode{{Key: "x", Title: "X"}, {Key: "x", Title: "Again"}})
	if _, err := mgr.SetHome(ctx, draftID, broken); !errors.Is(err, mwanachamaagency.ErrInvalidHomeEntry) {
		t.Fatalf("SetHome (broken) = %v, want ErrInvalidHomeEntry", err)
	}
	if listed := homeRowsOf(t, mgr, ctx, draftID); len(listed) != 2 || homeKeyed(listed, "mine").Key == "" {
		t.Fatalf("listed = %+v, want the Home from before the refusal", listed)
	}
	if _, err := mgr.SetHome(ctx, draftID, sent([]models.DocumentHomeNode{{Key: "a", Title: "A", Roles: []string{"R-404"}}})); !errors.Is(err, mwanachamaagency.ErrInvalidHomeEntry) {
		t.Fatalf("SetHome with an unknown role = %v, want ErrInvalidHomeEntry", err)
	}
	if rows, err := mgr.SetHome(ctx, draftID, nil); err != nil || len(rows) != 0 {
		t.Fatalf("SetHome with nothing = %+v, %v, want the Home cleared", rows, err)
	}
}

func TestSetHome_AnArchivedDraftRefusesIt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, _, _, _ := seedLiveAgency(t, mgr, ctx)
	if _, err := mgr.ArchiveDraft(ctx, g.DraftID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}
	if _, err := mgr.SetHome(ctx, g.DraftID, nil); !errors.Is(err, mwanachamaagency.ErrDraftNotOpen) {
		t.Fatalf("SetHome = %v, want ErrDraftNotOpen", err)
	}
}

func TestPublishAgency_SettingTheSameHomeAgainIsNoChange(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	raw, _ := json.Marshal([]models.DocumentHomeNode{{Key: "files", Title: "Files", Target: &models.HomeTarget{Type: models.HomeTargetFiles}}})
	var sent []map[string]any
	_ = json.Unmarshal(raw, &sent)

	if _, err := mgr.SetHome(ctx, o.DraftID, sent); err != nil {
		t.Fatalf("SetHome: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if _, err := mgr.SetHome(ctx, o.DraftID, sent); err != nil {
		t.Fatalf("SetHome (again): %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("PublishAgency after the same Home = %v, want ErrNoChangesDetected", err)
	}
}
