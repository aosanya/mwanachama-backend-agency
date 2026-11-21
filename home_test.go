package mwanachamaagency_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func homeRowsOf(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID string) []models.HomeEntry {
	t.Helper()
	rows, err := mgr.ListHomeEntries(ctx, draftID)
	if err != nil {
		t.Fatalf("ListHomeEntries: %v", err)
	}
	return rows
}

func homeKeyed(rows []models.HomeEntry, key string) models.HomeEntry {
	for _, row := range rows {
		if row.Key == key {
			return row
		}
	}
	return models.HomeEntry{}
}

func threeLevelHome() []models.DocumentHomeNode {
	return []models.DocumentHomeNode{
		{
			Key: "registers", Title: "Registers", Titles: map[string]string{"sw": "Rejista"}, Icon: "folder",
			Entries: []models.DocumentHomeNode{
				{
					Key: "farms", Title: "Farms",
					Entries: []models.DocumentHomeNode{
						{Key: "farm-list", Title: "All farms", Target: &models.HomeTarget{Type: models.HomeTargetRecord, Ref: "RC-1"}},
						{Key: "farm-visit", Title: "Record a visit", Permission: "mount.list", Target: &models.HomeTarget{Type: models.HomeTargetWorkflow, Ref: "W-2"}},
					},
				},
				{Key: "shared-files", Title: "Files", Target: &models.HomeTarget{Type: models.HomeTargetFiles}},
			},
		},
		{Key: "work", Title: "Work", Target: &models.HomeTarget{Type: models.HomeTargetBacklog, Ref: "backlog"}},
	}
}

func TestCreateHomeEntry_ASectionAndItsEntriesAreListedInDeclaredOrder(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, _, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := g.DraftID

	section, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draftID, Key: "registers", Title: "Registers", Titles: map[string]string{"sw": "Rejista"}})
	if err != nil {
		t.Fatalf("CreateHomeEntry (section): %v", err)
	}
	if section.ID == "" || section.ParentID != "" || section.Ordinality != 0 || !section.IsSection() {
		t.Fatalf("section = %+v, want a minted id, on Home itself, first", section)
	}
	second, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draftID, Key: "work", Title: "Work", TargetType: models.HomeTargetBacklog, Target: "backlog"})
	if err != nil {
		t.Fatalf("CreateHomeEntry (top entry): %v", err)
	}
	inner, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draftID, ParentID: section.ID, Key: "farm-list", Title: "All farms", TargetType: models.HomeTargetRecord, Target: "RC-1"})
	if err != nil {
		t.Fatalf("CreateHomeEntry (inner): %v", err)
	}
	if second.Ordinality != 1 || inner.Ordinality != 0 || inner.ParentID != section.ID {
		t.Fatalf("second = %+v inner = %+v, want places counted within each section", second, inner)
	}

	rows := homeRowsOf(t, mgr, ctx, draftID)
	if len(rows) != 3 {
		t.Fatalf("rows = %+v, want the three declared", rows)
	}
	if got := homeKeyed(rows, "registers"); got.Titles["sw"] != "Rejista" {
		t.Fatalf("section = %+v, want its translated title kept", got)
	}
	if got := homeKeyed(rows, "farm-list"); got.TargetType != models.HomeTargetRecord || got.Target != "RC-1" || got.DraftID != draftID {
		t.Fatalf("entry = %+v, want its target and draft kept", got)
	}
}

func TestCreateHomeEntry_Refusals(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, _, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := g.DraftID
	entry, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draftID, Key: "work", Title: "Work", TargetType: models.HomeTargetBacklog, Target: "backlog"})
	if err != nil {
		t.Fatalf("CreateHomeEntry: %v", err)
	}
	section, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draftID, Key: "registers", Title: "Registers"})
	if err != nil {
		t.Fatalf("CreateHomeEntry: %v", err)
	}

	cases := []struct {
		name string
		in   models.HomeEntry
		want error
	}{
		{"no key", models.HomeEntry{DraftID: draftID, Title: "A"}, mwanachamaagency.ErrInvalidHomeEntry},
		{"a key with a space", models.HomeEntry{DraftID: draftID, Key: "a b", Title: "A"}, mwanachamaagency.ErrInvalidHomeEntry},
		{"no title", models.HomeEntry{DraftID: draftID, Key: "a", Title: "  "}, mwanachamaagency.ErrInvalidHomeEntry},
		{"a repeated key", models.HomeEntry{DraftID: draftID, Key: "work", Title: "Again"}, mwanachamaagency.ErrInvalidHomeEntry},
		{"an unknown target type", models.HomeEntry{DraftID: draftID, Key: "a", Title: "A", TargetType: "page", Target: "x"}, mwanachamaagency.ErrInvalidHomeEntry},
		{"a record with no code", models.HomeEntry{DraftID: draftID, Key: "a", Title: "A", TargetType: models.HomeTargetRecord}, mwanachamaagency.ErrInvalidHomeEntry},
		{"files naming a target", models.HomeEntry{DraftID: draftID, Key: "a", Title: "A", TargetType: models.HomeTargetFiles, Target: "x"}, mwanachamaagency.ErrInvalidHomeEntry},
		{"a section naming a target", models.HomeEntry{DraftID: draftID, Key: "a", Title: "A", Target: "x"}, mwanachamaagency.ErrInvalidHomeEntry},
		{"a menu item with no section", models.HomeEntry{DraftID: draftID, Key: "a", Title: "A", TargetType: models.HomeTargetMenu, Target: "Businesses"}, mwanachamaagency.ErrInvalidHomeEntry},
		{"a permission outside the catalogue", models.HomeEntry{DraftID: draftID, Key: "a", Title: "A", Permission: "no.such.permission"}, mwanachamaagency.ErrInvalidHomeEntry},
		{"a parent that is an entry", models.HomeEntry{DraftID: draftID, ParentID: entry.ID, Key: "a", Title: "A"}, mwanachamaagency.ErrInvalidHomeEntry},
		{"a parent that does not exist", models.HomeEntry{DraftID: draftID, ParentID: "nope", Key: "a", Title: "A"}, mwanachamaagency.ErrHomeEntryNotFound},
		{"no draft", models.HomeEntry{Key: "a", Title: "A"}, mwanachamaagency.ErrDraftRequired},
		{"a section in an unknown draft", models.HomeEntry{DraftID: "nope", ParentID: section.ID, Key: "a", Title: "A"}, mwanachamaagency.ErrDraftNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := mgr.CreateHomeEntry(ctx, c.in); !errors.Is(err, c.want) {
				t.Fatalf("CreateHomeEntry = %v, want %v", err, c.want)
			}
		})
	}
	if rows := homeRowsOf(t, mgr, ctx, draftID); len(rows) != 2 {
		t.Fatalf("rows = %+v, want the refusals to have kept nothing", rows)
	}
}

func TestCreateHomeEntry_AnArchivedDraftRefusesIt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, _, _, _ := seedLiveAgency(t, mgr, ctx)
	if _, err := mgr.ArchiveDraft(ctx, g.DraftID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}
	if _, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: g.DraftID, Key: "a", Title: "A"}); !errors.Is(err, mwanachamaagency.ErrDraftNotOpen) {
		t.Fatalf("CreateHomeEntry = %v, want ErrDraftNotOpen", err)
	}
}

func TestCreateHomeEntry_TheDepthLimitStopsASixthLevel(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, _, _, _ := seedLiveAgency(t, mgr, ctx)
	parent := ""
	for level := 1; level <= models.HomeMaxDepth; level++ {
		row, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: g.DraftID, ParentID: parent, Key: fmt.Sprintf("level-%d", level), Title: "Level"})
		if err != nil {
			t.Fatalf("level %d: %v", level, err)
		}
		parent = row.ID
	}
	if _, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: g.DraftID, ParentID: parent, Key: "too-deep", Title: "Too deep"}); !errors.Is(err, mwanachamaagency.ErrInvalidHomeEntry) {
		t.Fatalf("a row at level %d = %v, want ErrInvalidHomeEntry", models.HomeMaxDepth+1, err)
	}
}

func TestCreateHomeEntry_ASectionHoldsAtMostFortyRows(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, _, _, _ := seedLiveAgency(t, mgr, ctx)
	for i := 0; i < models.HomeMaxPerSection; i++ {
		if _, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: g.DraftID, Key: fmt.Sprintf("row-%d", i), Title: "Row"}); err != nil {
			t.Fatalf("row %d: %v", i, err)
		}
	}
	if _, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: g.DraftID, Key: "one-more", Title: "One more"}); !errors.Is(err, mwanachamaagency.ErrInvalidHomeEntry) {
		t.Fatalf("the 41st row = %v, want ErrInvalidHomeEntry", err)
	}
}

func TestImportDocument_KeepsANestedHomeThreeLevelsDeepAsDeclared(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	out, err := mgr.ImportDocument(ctx, "farms", models.AgencyDocument{Agency: models.DocumentAgency{Name: "Farms"}, Home: threeLevelHome()})
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	rows := homeRowsOf(t, mgr, ctx, out.Draft.ID)
	if len(rows) != 6 {
		t.Fatalf("rows = %+v, want the six declared", rows)
	}
	registers, farms, visit := homeKeyed(rows, "registers"), homeKeyed(rows, "farms"), homeKeyed(rows, "farm-visit")
	if registers.ParentID != "" || farms.ParentID != registers.ID || visit.ParentID != farms.ID {
		t.Fatalf("parents = %q %q %q, want each row inside the one above it", registers.ParentID, farms.ParentID, visit.ParentID)
	}
	if homeKeyed(rows, "shared-files").Ordinality != 1 || homeKeyed(rows, "work").Ordinality != 1 || farms.Ordinality != 0 {
		t.Fatalf("places = %+v, want the declared order kept", rows)
	}
	if visit.Permission != "mount.list" || visit.TargetType != models.HomeTargetWorkflow || visit.Target != "W-2" {
		t.Fatalf("visit = %+v, want its permission and target", visit)
	}
	nested := models.NestHomeEntries(rows)
	if len(nested) != 2 || nested[0].Key != "registers" || len(nested[0].Entries) != 2 || nested[0].Entries[0].Entries[1].Key != "farm-visit" {
		t.Fatalf("nested = %+v, want the declaration back as written", nested)
	}
}

func TestImportDocument_AFileWithNoHomeKeepsNoHomeRows(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	out, err := mgr.ImportDocument(ctx, "plain", models.AgencyDocument{Agency: models.DocumentAgency{Name: "Plain"}})
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	if rows := homeRowsOf(t, mgr, ctx, out.Draft.ID); len(rows) != 0 {
		t.Fatalf("rows = %+v, want none", rows)
	}
}

func TestImportDocument_ABrokenHomeRefusesTheWholeImport(t *testing.T) {
	broken := map[string][]models.DocumentHomeNode{
		"a repeated key":            {{Key: "a", Title: "A"}, {Key: "a", Title: "B"}},
		"both a target and entries": {{Key: "a", Title: "A", Target: &models.HomeTarget{Type: models.HomeTargetFiles}, Entries: []models.DocumentHomeNode{{Key: "b", Title: "B"}}}},
		"a sixth level":             {{Key: "1", Title: "1", Entries: []models.DocumentHomeNode{{Key: "2", Title: "2", Entries: []models.DocumentHomeNode{{Key: "3", Title: "3", Entries: []models.DocumentHomeNode{{Key: "4", Title: "4", Entries: []models.DocumentHomeNode{{Key: "5", Title: "5", Entries: []models.DocumentHomeNode{{Key: "6", Title: "6"}}}}}}}}}}}},
		"an unknown permission":     {{Key: "a", Title: "A", Permission: "no.such.permission"}},
	}
	for name, home := range broken {
		t.Run(name, func(t *testing.T) {
			mgr := newTestManager(t)
			ctx := context.Background()
			before, err := mgr.ListDrafts(ctx, mwanachamaagency.DraftFilter{})
			if err != nil {
				t.Fatalf("ListDrafts: %v", err)
			}
			if _, err := mgr.ImportDocument(ctx, "x", models.AgencyDocument{Agency: models.DocumentAgency{Name: "X"}, Home: home}); !errors.Is(err, mwanachamaagency.ErrInvalidHomeEntry) {
				t.Fatalf("ImportDocument = %v, want ErrInvalidHomeEntry", err)
			}
			after, err := mgr.ListDrafts(ctx, mwanachamaagency.DraftFilter{})
			if err != nil || len(after) != len(before) {
				t.Fatalf("drafts = %+v, %v, want the refused import to have left no draft beside %+v", after, err, before)
			}
		})
	}
}

func TestCreateDraft_ForkFromDraftCopiesTheHomeInsideTheNewDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	imported, err := mgr.ImportDocument(ctx, "farms", models.AgencyDocument{Agency: models.DocumentAgency{Name: "Farms"}, Home: threeLevelHome()})
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}

	fork, err := mgr.CreateDraft(ctx, "fork", imported.Draft.ID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	source, copied := homeRowsOf(t, mgr, ctx, imported.Draft.ID), homeRowsOf(t, mgr, ctx, fork.ID)
	if len(copied) != 6 {
		t.Fatalf("fork rows = %+v, want the six copied", copied)
	}
	for _, row := range copied {
		if row.DraftID != fork.ID || homeKeyed(source, row.Key).ID == row.ID {
			t.Fatalf("fork row = %+v, want a new row in the fork", row)
		}
	}
	if got := homeKeyed(copied, "farm-visit"); got.ParentID != homeKeyed(copied, "farms").ID || got.ParentID == homeKeyed(source, "farms").ID {
		t.Fatalf("fork visit = %+v, want it inside the fork's own farms section", got)
	}
	if got := models.NestHomeEntries(copied); len(got) != 2 || got[0].Entries[0].Entries[0].Key != "farm-list" {
		t.Fatalf("fork nested = %+v, want the same declaration", got)
	}
}

func declareThreeLevelHome(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID string) {
	t.Helper()
	var add func(parentID string, nodes []models.DocumentHomeNode)
	add = func(parentID string, nodes []models.DocumentHomeNode) {
		for _, node := range nodes {
			row := models.HomeEntryOfNode(node)
			row.DraftID, row.ParentID = draftID, parentID
			created, err := mgr.CreateHomeEntry(ctx, row)
			if err != nil {
				t.Fatalf("CreateHomeEntry %s: %v", node.Key, err)
			}
			add(created.ID, node.Entries)
		}
	}
	add("", threeLevelHome())
}

func TestPublishAgency_CarriesTheHomeAndAForkFromTheTagGetsIt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	declareThreeLevelHome(t, mgr, ctx, o.DraftID)

	pub, err := mgr.PublishAgency(ctx, o.DraftID, "release-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if len(pub.Document.Home) != 2 || pub.Document.Home[0].Entries[0].Entries[0].Target.Ref != "RC-1" {
		t.Fatalf("document home = %+v, want the declaration nested as written", pub.Document.Home)
	}

	fork, err := mgr.CreateDraft(ctx, "back to release-1", "release-1", "publication")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if got := models.NestHomeEntries(homeRowsOf(t, mgr, ctx, fork.ID)); len(got) != 2 || got[0].Entries[0].Entries[1].Permission != "mount.list" {
		t.Fatalf("fork home = %+v, want the published declaration", got)
	}
}

func TestPublishAgency_ChangingTheHomeIsAChange(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)

	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if _, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: o.DraftID, Key: "files", Title: "Files", TargetType: models.HomeTargetFiles}); err != nil {
		t.Fatalf("CreateHomeEntry: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); err != nil {
		t.Fatalf("PublishAgency after a new home entry: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("PublishAgency unchanged = %v, want ErrNoChangesDetected", err)
	}
}
