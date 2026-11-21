package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

const reefFile = `{
  "agency": {"name": "Reef farm"},
  "goals": [{"id": "g1", "code": "G-1", "title": "Tend the reef"}],
  "objectives": [{"id": "o1", "code": "O-1", "goal_id": "g1", "title": "Plant kelp"}],
  "home": [
    {
      "key": "registers",
      "title": "Registers",
      "icon": "folder",
      "titles": { "sw": "Rejista" },
      "entries": [
        {
          "key": "farms",
          "title": "Farms",
          "entries": [
            { "key": "farm-list", "title": "All farms", "target": { "type": "record", "ref": "RC-1" } },
            { "key": "farm-visit", "title": "Record a visit", "target": { "type": "workflow", "ref": "W-2" } }
          ]
        },
        { "key": "files", "title": "Files", "target": { "type": "files" } }
      ]
    },
    { "key": "work", "title": "Work", "permission": "task.read", "target": { "type": "backlog", "ref": "backlog" } }
  ]
}`

func hcTarget(kind models.HomeTargetType, ref string) *models.HomeTarget {
	return &models.HomeTarget{Type: kind, Ref: ref}
}

func hcEntry(key string, target *models.HomeTarget) models.DocumentHomeNode {
	return models.DocumentHomeNode{Key: key, Title: strings.ToUpper(key[:1]) + key[1:], Target: target}
}

func hcSection(key string, nodes ...models.DocumentHomeNode) models.DocumentHomeNode {
	return models.DocumentHomeNode{Key: key, Title: strings.ToUpper(key[:1]) + key[1:], Entries: nodes}
}

func hcKelpHome() []models.DocumentHomeNode {
	return []models.DocumentHomeNode{
		{
			Key: "zebra-fish", Title: "Zebra fish", Titles: map[string]string{"sw": "Samaki", "hu": "Hal"}, Icon: "fish",
			Entries: []models.DocumentHomeNode{
				{Key: "tides", Title: "Tides", Icon: "waves", Entries: []models.DocumentHomeNode{
					{Key: "tide-table", Title: "Tide table", Permission: "mount.list", Target: hcTarget(models.HomeTargetRecord, "RC-3")},
					{Key: "anchor-drill", Title: "Anchor drill", Target: hcTarget(models.HomeTargetWorkflow, "W-9")},
					{Key: "calm", Title: "Calm waters", Entries: []models.DocumentHomeNode{
						{Key: "deep-menu", Title: "Deep menu", Target: hcTarget(models.HomeTargetMenu, "residents/Businesses")},
					}},
				}},
				{Key: "nets", Title: "Nets", Target: hcTarget(models.HomeTargetFiles, "")},
				{Key: "clinic-queue", Title: "Clinic queue", Target: hcTarget(models.HomeTargetBacklog, "clinic")},
			},
		},
		{Key: "apple-boat", Title: "Apple boat", Target: hcTarget(models.HomeTargetBacklog, "boats")},
		{Key: "mango-reef", Title: "Mango reef", Titles: map[string]string{"sw": "Mwamba"}, Entries: []models.DocumentHomeNode{
			{Key: "kelp", Title: "Kelp", Target: hcTarget(models.HomeTargetRecord, "RC-1")},
			{Key: "abalone", Title: "Abalone", Target: hcTarget(models.HomeTargetRecord, "RC-2")},
		}},
	}
}

func hcDocument(home []models.DocumentHomeNode) models.AgencyDocument {
	return models.AgencyDocument{
		Agency:     models.DocumentAgency{Name: "Kelp co-op"},
		Goals:      []models.DocumentGoal{{ID: "g1", Code: "G-1", Title: "Tend the reef"}},
		Objectives: []models.DocumentObjective{{ID: "o1", Code: "O-1", GoalID: "g1", Title: "Plant kelp"}},
		Home:       home,
	}
}

func hcFileDocument(t *testing.T, raw string) models.AgencyDocument {
	t.Helper()
	var doc models.AgencyDocument
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("the agency file does not read: %v", err)
	}
	return doc
}

func hcImport(t *testing.T, mgr mwanachamaagency.AgencyManager, home []models.DocumentHomeNode) models.Draft {
	t.Helper()
	out, err := mgr.ImportDocument(context.Background(), "kelp", hcDocument(home))
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	return out.Draft
}

func hcRows(t *testing.T, mgr mwanachamaagency.AgencyManager, draftID string) []models.HomeEntry {
	t.Helper()
	rows, err := mgr.ListHomeEntries(context.Background(), draftID)
	if err != nil {
		t.Fatalf("ListHomeEntries: %v", err)
	}
	return rows
}

type hcShape struct {
	Key        string
	Title      string
	Titles     map[string]string
	Icon       string
	Permission string
	TargetType models.HomeTargetType
	Target     string
	Children   []hcShape
}

func hcTree(rows []models.HomeEntry) []hcShape {
	byParent := map[string][]models.HomeEntry{}
	for _, row := range rows {
		byParent[row.ParentID] = append(byParent[row.ParentID], row)
	}
	var build func(parent string) []hcShape
	build = func(parent string) []hcShape {
		siblings := append([]models.HomeEntry(nil), byParent[parent]...)
		sort.SliceStable(siblings, func(i, j int) bool { return siblings[i].Ordinality < siblings[j].Ordinality })
		var out []hcShape
		for _, row := range siblings {
			out = append(out, hcShape{
				Key: row.Key, Title: row.Title, Titles: row.Titles, Icon: row.Icon, Permission: row.Permission,
				TargetType: row.TargetType, Target: row.Target, Children: build(row.ID),
			})
		}
		return out
	}
	return build("")
}

func hcShapeOfDocument(nodes []models.DocumentHomeNode) []hcShape {
	var out []hcShape
	for _, n := range nodes {
		s := hcShape{Key: n.Key, Title: n.Title, Titles: n.Titles, Icon: n.Icon, Permission: n.Permission, Children: hcShapeOfDocument(n.Entries)}
		if n.Target != nil {
			s.TargetType, s.Target = n.Target.Type, n.Target.Ref
		}
		out = append(out, s)
	}
	return out
}

func hcEqual(t *testing.T, what string, got, want []hcShape) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		g, _ := json.MarshalIndent(got, "", " ")
		w, _ := json.MarshalIndent(want, "", " ")
		t.Fatalf("%s differ.\ngot:\n%s\nwant:\n%s", what, g, w)
	}
}

func hcPublish(t *testing.T, mgr mwanachamaagency.AgencyManager, draftID, tag string) models.AgencyPublication {
	t.Helper()
	p, err := mgr.PublishAgency(context.Background(), draftID, tag, "")
	if err != nil {
		t.Fatalf("PublishAgency %s: %v", tag, err)
	}
	return p
}

func hcSeedDraft(t *testing.T, mgr mwanachamaagency.AgencyManager) models.Draft {
	t.Helper()
	ctx := context.Background()
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "Clinic"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draft, err := mgr.CreateDraft(ctx, "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	g, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draft.ID, Title: "Tend the reef"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if _, err := mgr.CreateObjective(ctx, models.Objective{DraftID: draft.ID, GoalID: g.ID, Title: "Plant kelp"}); err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	return draft
}

func hcWide(prefix string, count int) []models.DocumentHomeNode {
	out := make([]models.DocumentHomeNode, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, hcEntry(fmt.Sprintf("%s-%d", prefix, i), hcTarget(models.HomeTargetFiles, "")))
	}
	return out
}

func hcDeep(levels int) []models.DocumentHomeNode {
	node := hcEntry("leaf", hcTarget(models.HomeTargetFiles, ""))
	for level := levels - 1; level >= 1; level-- {
		node = hcSection(fmt.Sprintf("level-%d", level), node)
	}
	return []models.DocumentHomeNode{node}
}

func TestHomeContract_ImportKeepsAThreeLevelMixedHomeInDeclaredOrder(t *testing.T) {
	mgr := newTestManager(t)
	draft := hcImport(t, mgr, hcKelpHome())

	rows := hcRows(t, mgr, draft.ID)
	if len(rows) != 12 {
		t.Fatalf("rows = %d, want the 12 declared nodes", len(rows))
	}
	hcEqual(t, "the imported home", hcTree(rows), hcShapeOfDocument(hcKelpHome()))
}

func TestHomeContract_ImportKeepsDeclaredOrderNotAlphabeticalOrder(t *testing.T) {
	mgr := newTestManager(t)
	draft := hcImport(t, mgr, hcKelpHome())

	var top []string
	for _, c := range hcTree(hcRows(t, mgr, draft.ID)) {
		top = append(top, c.Key)
	}
	if !reflect.DeepEqual(top, []string{"zebra-fish", "apple-boat", "mango-reef"}) {
		t.Fatalf("top-level order = %v, want the declared order, nothing sorted", top)
	}
	rows := hcRows(t, mgr, draft.ID)
	byKey := map[string]models.HomeEntry{}
	for _, row := range rows {
		byKey[row.Key] = row
	}
	if !(byKey["zebra-fish"].Ordinality < byKey["apple-boat"].Ordinality && byKey["apple-boat"].Ordinality < byKey["mango-reef"].Ordinality) {
		t.Fatalf("ordinality of the top rows = %d %d %d, want it to rise in declared order",
			byKey["zebra-fish"].Ordinality, byKey["apple-boat"].Ordinality, byKey["mango-reef"].Ordinality)
	}
	if !(byKey["kelp"].Ordinality < byKey["abalone"].Ordinality) {
		t.Fatalf("kelp = %d, abalone = %d, want declared order inside a section", byKey["kelp"].Ordinality, byKey["abalone"].Ordinality)
	}
}

func TestHomeContract_EveryNodeIsOneRowWithItsParentAndItsDraft(t *testing.T) {
	mgr := newTestManager(t)
	draft := hcImport(t, mgr, hcKelpHome())

	rows := hcRows(t, mgr, draft.ID)
	byKey := map[string]models.HomeEntry{}
	ids := map[string]bool{}
	for _, row := range rows {
		if row.ID == "" || ids[row.ID] {
			t.Fatalf("row %q has id %q, want a distinct id for every row", row.Key, row.ID)
		}
		ids[row.ID] = true
		if row.DraftID != draft.ID {
			t.Errorf("row %q draft = %q, want %q", row.Key, row.DraftID, draft.ID)
		}
		byKey[row.Key] = row
	}
	for key, parent := range map[string]string{
		"zebra-fish": "", "apple-boat": "", "mango-reef": "",
		"tides": "zebra-fish", "nets": "zebra-fish", "clinic-queue": "zebra-fish",
		"tide-table": "tides", "anchor-drill": "tides", "calm": "tides", "deep-menu": "calm",
		"kelp": "mango-reef", "abalone": "mango-reef",
	} {
		want := ""
		if parent != "" {
			want = byKey[parent].ID
		}
		if byKey[key].ParentID != want {
			t.Errorf("row %q parent_id = %q, want the id of %q", key, byKey[key].ParentID, parent)
		}
	}
}

func TestHomeContract_ImportReadsTheDesignPageExampleFileExactly(t *testing.T) {
	mgr := newTestManager(t)
	doc := hcFileDocument(t, reefFile)
	out, err := mgr.ImportDocument(context.Background(), "reef", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}

	got := hcTree(hcRows(t, mgr, out.Draft.ID))
	want := []hcShape{
		{
			Key: "registers", Title: "Registers", Icon: "folder", Titles: map[string]string{"sw": "Rejista"},
			Children: []hcShape{
				{Key: "farms", Title: "Farms", Children: []hcShape{
					{Key: "farm-list", Title: "All farms", TargetType: models.HomeTargetRecord, Target: "RC-1"},
					{Key: "farm-visit", Title: "Record a visit", TargetType: models.HomeTargetWorkflow, Target: "W-2"},
				}},
				{Key: "files", Title: "Files", TargetType: models.HomeTargetFiles},
			},
		},
		{Key: "work", Title: "Work", Permission: "task.read", TargetType: models.HomeTargetBacklog, Target: "backlog"},
	}
	hcEqual(t, "the example home", got, want)
}

func TestHomeContract_AnIconTheAppDoesNotShipIsKept(t *testing.T) {
	mgr := newTestManager(t)
	draft := hcImport(t, mgr, []models.DocumentHomeNode{
		{Key: "odd", Title: "Odd", Icon: "no-such-icon-anywhere", Target: hcTarget(models.HomeTargetFiles, "")},
	})
	rows := hcRows(t, mgr, draft.ID)
	if len(rows) != 1 || rows[0].Icon != "no-such-icon-anywhere" {
		t.Fatalf("rows = %+v, want the unknown icon name kept as written", rows)
	}
}

func TestHomeContract_ASectionWithNoEntriesIsDeclarable(t *testing.T) {
	mgr := newTestManager(t)
	draft := hcImport(t, mgr, []models.DocumentHomeNode{hcSection("empty-bay")})
	rows := hcRows(t, mgr, draft.ID)
	if len(rows) != 1 || rows[0].Key != "empty-bay" || rows[0].TargetType != "" || rows[0].Target != "" {
		t.Fatalf("rows = %+v, want the one section with no target", rows)
	}
}

func TestHomeContract_RefsAreNotCheckedAgainstTheDraftAtImport(t *testing.T) {
	mgr := newTestManager(t)
	draft := hcImport(t, mgr, []models.DocumentHomeNode{
		hcEntry("ghost-record", hcTarget(models.HomeTargetRecord, "RC-404")),
		hcEntry("ghost-flow", hcTarget(models.HomeTargetWorkflow, "W-404")),
		hcEntry("ghost-backlog", hcTarget(models.HomeTargetBacklog, "no-such-backlog")),
	})
	if rows := hcRows(t, mgr, draft.ID); len(rows) != 3 {
		t.Fatalf("rows = %+v, want a dangling ref kept, the app reads it later", rows)
	}
	hcPublish(t, mgr, draft.ID, "ghosts")
}

func TestHomeContract_Limits(t *testing.T) {
	tenSections := func(perSection int) []models.DocumentHomeNode {
		var out []models.DocumentHomeNode
		for i := 0; i < 10; i++ {
			out = append(out, hcSection(fmt.Sprintf("bay-%d", i), hcWide(fmt.Sprintf("bay-%d-item", i), perSection)...))
		}
		return out
	}
	cases := []struct {
		name   string
		home   []models.DocumentHomeNode
		accept bool
	}{
		{"depth 5 is the deepest", hcDeep(5), true},
		{"depth 6 is refused", hcDeep(6), false},
		{"depth 7 is refused", hcDeep(7), false},
		{"40 nodes in a section", []models.DocumentHomeNode{hcSection("full", hcWide("full-item", 40)...)}, true},
		{"41 nodes in a section", []models.DocumentHomeNode{hcSection("over", hcWide("over-item", 41)...)}, false},
		{"41 nodes in a section three levels down", []models.DocumentHomeNode{hcSection("a", hcSection("b", hcWide("deep-item", 41)...))}, false},
		{"300 nodes in all", tenSections(29), true},
		{"301 nodes in all", append(tenSections(29)[:9:9], hcSection("bay-9", hcWide("bay-9-item", 30)...)), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := newTestManager(t)
			_, err := mgr.ImportDocument(context.Background(), "limits", hcDocument(tc.home))
			if tc.accept && err != nil {
				t.Fatalf("ImportDocument refused a home inside the limits: %v", err)
			}
			if !tc.accept && err == nil {
				t.Fatalf("ImportDocument kept a home that breaks a limit")
			}
		})
	}
}

func TestHomeContract_TheCountOfAllTheNodesIsRefusedAcrossNestedSections(t *testing.T) {
	var home []models.DocumentHomeNode
	for i := 0; i < 8; i++ {
		inner := hcSection(fmt.Sprintf("inner-%d", i), hcWide(fmt.Sprintf("inner-%d-item", i), 20)...)
		home = append(home, hcSection(fmt.Sprintf("outer-%d", i), inner, hcEntry(fmt.Sprintf("outer-%d-file", i), hcTarget(models.HomeTargetFiles, ""))))
		home = append(home, hcWide(fmt.Sprintf("loose-%d", i), 5)...)
	}
	mgr := newTestManager(t)
	if _, err := mgr.ImportDocument(context.Background(), "total", hcDocument(home)); err == nil {
		t.Fatalf("ImportDocument kept a declaration of %d nodes, want it refused over 300", 8*(1+1+20+1+5))
	}
}

func TestHomeContract_RefusalsRefuseTheWholeHomeAndKeepNothing(t *testing.T) {
	good := []models.DocumentHomeNode{
		hcSection("fine-first", hcEntry("fine-inner", hcTarget(models.HomeTargetRecord, "RC-1"))),
		hcEntry("fine-second", hcTarget(models.HomeTargetFiles, "")),
	}
	withBad := func(bad ...models.DocumentHomeNode) []models.DocumentHomeNode {
		return append(append([]models.DocumentHomeNode(nil), good...), bad...)
	}
	cases := []struct {
		name string
		home []models.DocumentHomeNode
	}{
		{"depth over 5", withBad(hcDeep(6)...)},
		{"a section over 40 nodes", withBad(hcSection("crowded", hcWide("crowd", 41)...))},
		{"a declaration over 300", withBad(append(hcWide("first-lot", 40), hcSection("lot-b", hcWide("b", 40)...), hcSection("lot-c", hcWide("c", 40)...), hcSection("lot-d", hcWide("d", 40)...), hcSection("lot-e", hcWide("e", 40)...), hcSection("lot-f", hcWide("f", 40)...), hcSection("lot-g", hcWide("g", 40)...), hcSection("lot-h", hcWide("h", 40)...))...)},
		{"a key repeated at the top", withBad(hcEntry("fine-second", hcTarget(models.HomeTargetFiles, "")))},
		{"a key repeated at another level", withBad(hcSection("holder", hcEntry("fine-inner", hcTarget(models.HomeTargetFiles, ""))))},
		{"a key repeated by a section and an entry", withBad(hcSection("fine-second", hcEntry("another", hcTarget(models.HomeTargetFiles, ""))))},
		{"a node with both entries and target", withBad(models.DocumentHomeNode{
			Key: "both", Title: "Both", Target: hcTarget(models.HomeTargetFiles, ""),
			Entries: []models.DocumentHomeNode{hcEntry("child", hcTarget(models.HomeTargetFiles, ""))},
		})},
		{"an unknown target type", withBad(hcEntry("odd", hcTarget("page", "x")))},
		{"a permission outside the catalogue", withBad(models.DocumentHomeNode{Key: "gated", Title: "Gated", Permission: "no.such.permission", Target: hcTarget(models.HomeTargetFiles, "")})},
		{"a permission outside the catalogue on a section", withBad(models.DocumentHomeNode{Key: "gated", Title: "Gated", Permission: "no.such.permission", Entries: []models.DocumentHomeNode{hcEntry("kid", hcTarget(models.HomeTargetFiles, ""))}})},
		{"a record with no ref", withBad(hcEntry("bare-record", hcTarget(models.HomeTargetRecord, "")))},
		{"a workflow with no ref", withBad(hcEntry("bare-flow", hcTarget(models.HomeTargetWorkflow, "")))},
		{"a backlog with no ref", withBad(hcEntry("bare-backlog", hcTarget(models.HomeTargetBacklog, "")))},
		{"a menu item with no ref", withBad(hcEntry("bare-menu", hcTarget(models.HomeTargetMenu, "")))},
		{"a bad node three levels down", withBad(hcSection("a", hcSection("b", hcEntry("bad-record", hcTarget(models.HomeTargetRecord, "")))))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mgr, db, s := newTestManagerDB(t)
			before, err := mgr.ListDrafts(context.Background(), mwanachamaagency.DraftFilter{})
			if err != nil {
				t.Fatalf("ListDrafts: %v", err)
			}
			_, err = mgr.ImportDocument(context.Background(), "refusals", hcDocument(tc.home))
			if err == nil {
				t.Fatalf("ImportDocument kept a home it should refuse")
			}
			var kept int64
			if err := db.Table(tableOf(t, s, "home_entry")).Count(&kept).Error; err != nil {
				t.Fatalf("count home rows: %v", err)
			}
			if kept != 0 {
				t.Fatalf("%d home rows kept after the refusal, want none (the good nodes included)", kept)
			}
			drafts, err := mgr.ListDrafts(context.Background(), mwanachamaagency.DraftFilter{})
			if err != nil {
				t.Fatalf("ListDrafts: %v", err)
			}
			if len(drafts) != len(before) {
				t.Fatalf("%d drafts after a refused import, want the %d there were", len(drafts), len(before))
			}
		})
	}
}

func TestHomeContract_AFilesTargetNeedsNoRef(t *testing.T) {
	mgr := newTestManager(t)
	draft := hcImport(t, mgr, []models.DocumentHomeNode{hcEntry("shared-files", hcTarget(models.HomeTargetFiles, ""))})
	rows := hcRows(t, mgr, draft.ID)
	if len(rows) != 1 || rows[0].TargetType != models.HomeTargetFiles {
		t.Fatalf("rows = %+v, want the files entry", rows)
	}
}

func TestHomeContract_EveryTargetTypeOfTheTableIsAccepted(t *testing.T) {
	mgr := newTestManager(t)
	draft := hcImport(t, mgr, []models.DocumentHomeNode{
		hcEntry("t-record", hcTarget(models.HomeTargetRecord, "RC-1")),
		hcEntry("t-workflow", hcTarget(models.HomeTargetWorkflow, "W-2")),
		hcEntry("t-backlog", hcTarget(models.HomeTargetBacklog, "reef-backlog")),
		hcEntry("t-menu", hcTarget(models.HomeTargetMenu, "residents/Businesses")),
		hcEntry("t-files", hcTarget(models.HomeTargetFiles, "")),
	})
	got := map[string]string{}
	for _, row := range hcRows(t, mgr, draft.ID) {
		got[row.Key] = string(row.TargetType) + "|" + row.Target
	}
	want := map[string]string{
		"t-record": "record|RC-1", "t-workflow": "workflow|W-2", "t-backlog": "backlog|reef-backlog",
		"t-menu": "menu|residents/Businesses", "t-files": "files|",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("targets = %v, want %v", got, want)
	}
}

func TestHomeContract_FieldBounds(t *testing.T) {
	cases := []struct {
		name   string
		node   models.DocumentHomeNode
		accept bool
	}{
		{"a key of 60", models.DocumentHomeNode{Key: strings.Repeat("k", 60), Title: "T", Target: hcTarget(models.HomeTargetFiles, "")}, true},
		{"a key of 61", models.DocumentHomeNode{Key: strings.Repeat("k", 61), Title: "T", Target: hcTarget(models.HomeTargetFiles, "")}, false},
		{"a key with hyphen, underscore and digits", models.DocumentHomeNode{Key: "Reef_9-b", Title: "T", Target: hcTarget(models.HomeTargetFiles, "")}, true},
		{"a key with a space", models.DocumentHomeNode{Key: "reef bay", Title: "T", Target: hcTarget(models.HomeTargetFiles, "")}, false},
		{"a key with a slash", models.DocumentHomeNode{Key: "reef/bay", Title: "T", Target: hcTarget(models.HomeTargetFiles, "")}, false},
		{"no key", models.DocumentHomeNode{Title: "T", Target: hcTarget(models.HomeTargetFiles, "")}, false},
		{"a title of 80", models.DocumentHomeNode{Key: "k", Title: strings.Repeat("t", 80), Target: hcTarget(models.HomeTargetFiles, "")}, true},
		{"a title of 81", models.DocumentHomeNode{Key: "k", Title: strings.Repeat("t", 81), Target: hcTarget(models.HomeTargetFiles, "")}, false},
		{"no title", models.DocumentHomeNode{Key: "k", Target: hcTarget(models.HomeTargetFiles, "")}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := newTestManager(t)
			_, err := mgr.ImportDocument(context.Background(), "fields", hcDocument([]models.DocumentHomeNode{tc.node}))
			if tc.accept && err != nil {
				t.Fatalf("ImportDocument refused: %v", err)
			}
			if !tc.accept && err == nil {
				t.Fatalf("ImportDocument kept it")
			}
		})
	}
}

func TestHomeContract_AFileWithoutHomeImportsAsBefore(t *testing.T) {
	mgr := newTestManager(t)
	out, err := mgr.ImportDocument(context.Background(), "plain", hcDocument(nil))
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	if rows := hcRows(t, mgr, out.Draft.ID); len(rows) != 0 {
		t.Fatalf("rows = %+v, want none", rows)
	}
	pub := hcPublish(t, mgr, out.Draft.ID, "plain")
	if len(pub.Document.Home) != 0 {
		t.Fatalf("document home = %+v, want none", pub.Document.Home)
	}
	raw, err := json.Marshal(pub.Document)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var whole map[string]json.RawMessage
	if err := json.Unmarshal(raw, &whole); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if home, present := whole["home"]; present {
		t.Fatalf("the document of an agency with no home carries home = %s, want the key left out as before", home)
	}
}

func TestHomeContract_TheWakalaTemplateStillImportsWithAnEmptyHome(t *testing.T) {
	doc := readDocument(t, wakalaDocument)
	if len(doc.Home) != 0 {
		t.Skip("the template declares a home")
	}
	mgr := newTestManager(t)
	out, err := mgr.ImportDocument(context.Background(), "wakala", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	if rows := hcRows(t, mgr, out.Draft.ID); len(rows) != 0 {
		t.Fatalf("rows = %d, want none for a file with no home", len(rows))
	}
}

func TestHomeContract_AFreshDraftWithNoHomeRowsPublishesAsBefore(t *testing.T) {
	mgr := newTestManager(t)
	draft := hcSeedDraft(t, mgr)
	pub := hcPublish(t, mgr, draft.ID, "bare")
	if len(pub.Document.Home) != 0 {
		t.Fatalf("home = %+v, want none", pub.Document.Home)
	}
	if rows := hcRows(t, mgr, draft.ID); len(rows) != 0 {
		t.Fatalf("rows = %+v, want none", rows)
	}
}

func TestHomeContract_ThePublicationDocumentHoldsHomeNestedExactlyAsWritten(t *testing.T) {
	mgr := newTestManager(t)
	want := hcKelpHome()
	draft := hcImport(t, mgr, want)
	pub := hcPublish(t, mgr, draft.ID, "v-kelp")

	if !reflect.DeepEqual(pub.Document.Home, want) {
		g, _ := json.MarshalIndent(pub.Document.Home, "", " ")
		w, _ := json.MarshalIndent(want, "", " ")
		t.Fatalf("published home differs from the declaration.\ngot:\n%s\nwant:\n%s", g, w)
	}

	read, err := mgr.GetPublication(context.Background(), pub.Version)
	if err != nil {
		t.Fatalf("GetPublication: %v", err)
	}
	if !reflect.DeepEqual(read.Document.Home, want) {
		t.Fatalf("a re-read publication lost the nesting or the order")
	}
	live, err := mgr.GetPublicationForChannel(context.Background(), models.DefaultChannel)
	if err != nil {
		t.Fatalf("GetPublicationForChannel: %v", err)
	}
	if !reflect.DeepEqual(live.Document.Home, want) {
		t.Fatalf("the production publication lost the nesting or the order")
	}
}

func TestHomeContract_ThePublicationJSONCarriesHomeAsNestedEntriesAndTargets(t *testing.T) {
	mgr := newTestManager(t)
	out, err := mgr.ImportDocument(context.Background(), "reef", hcFileDocument(t, reefFile))
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	pub := hcPublish(t, mgr, out.Draft.ID, "v-reef")
	raw, err := json.Marshal(pub.Document)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var wrote struct {
		Home []map[string]any `json:"home"`
	}
	if err := json.Unmarshal(raw, &wrote); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(wrote.Home) != 2 || wrote.Home[0]["key"] != "registers" || wrote.Home[1]["key"] != "work" {
		t.Fatalf("home = %+v, want registers then work at the top", wrote.Home)
	}
	registers := wrote.Home[0]
	entries, _ := registers["entries"].([]any)
	if len(entries) != 2 {
		t.Fatalf("registers.entries = %+v, want farms and files", registers["entries"])
	}
	farms, _ := entries[0].(map[string]any)
	inner, _ := farms["entries"].([]any)
	if farms["key"] != "farms" || len(inner) != 2 {
		t.Fatalf("farms = %+v, want two entries nested in it", farms)
	}
	visit, _ := inner[1].(map[string]any)
	target, _ := visit["target"].(map[string]any)
	if visit["key"] != "farm-visit" || target["type"] != "workflow" || target["ref"] != "W-2" {
		t.Fatalf("farm-visit = %+v, want a workflow target W-2", visit)
	}
	if _, has := visit["entries"]; has {
		t.Fatalf("an entry carries entries: %+v", visit)
	}
	if _, has := farms["target"]; has {
		t.Fatalf("a section carries a target: %+v", farms)
	}
	if registers["icon"] != "folder" || registers["titles"].(map[string]any)["sw"] != "Rejista" {
		t.Fatalf("registers = %+v, want its icon and translated title as written", registers)
	}
	work := wrote.Home[1]
	if work["permission"] != "task.read" {
		t.Fatalf("work = %+v, want its permission", work)
	}
}

func TestHomeContract_TheDocumentOfAPublicationImportsBackIntoTheSameHome(t *testing.T) {
	first := newTestManager(t)
	draft := hcImport(t, first, hcKelpHome())
	pub := hcPublish(t, first, draft.ID, "v1")
	raw, err := json.Marshal(pub.Document)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	second := newTestManager(t)
	out, err := second.ImportDocument(context.Background(), "again", hcFileDocument(t, string(raw)))
	if err != nil {
		t.Fatalf("ImportDocument of the exported document: %v", err)
	}
	hcEqual(t, "the home after the round trip", hcTree(hcRows(t, second, out.Draft.ID)), hcShapeOfDocument(hcKelpHome()))
}

func TestHomeContract_ForkFromADraftKeepsTheNestingAndTheOrder(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	source := hcImport(t, mgr, hcKelpHome())

	fork, err := mgr.CreateDraft(ctx, "fork", source.ID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft (fork of a draft): %v", err)
	}
	forkRows := hcRows(t, mgr, fork.ID)
	hcEqual(t, "the forked home", hcTree(forkRows), hcShapeOfDocument(hcKelpHome()))

	sourceIDs := map[string]bool{}
	for _, row := range hcRows(t, mgr, source.ID) {
		sourceIDs[row.ID] = true
	}
	forkIDs := map[string]bool{}
	for _, row := range forkRows {
		if row.DraftID != fork.ID {
			t.Errorf("forked row %q draft = %q, want the fork %q", row.Key, row.DraftID, fork.ID)
		}
		if sourceIDs[row.ID] {
			t.Errorf("forked row %q reuses the source row id %q", row.Key, row.ID)
		}
		if row.ParentID != "" && sourceIDs[row.ParentID] {
			t.Errorf("forked row %q hangs from the source's row %q", row.Key, row.ParentID)
		}
		forkIDs[row.ID] = true
	}
	for _, row := range forkRows {
		if row.ParentID != "" && !forkIDs[row.ParentID] {
			t.Errorf("forked row %q has a parent %q outside the fork", row.Key, row.ParentID)
		}
	}
	if len(hcRows(t, mgr, source.ID)) != 12 {
		t.Fatalf("the source lost rows when it was forked")
	}
}

func TestHomeContract_AForkAndItsSourceDoNotShareRows(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	source := hcImport(t, mgr, hcKelpHome())
	fork, err := mgr.CreateDraft(ctx, "fork", source.ID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	var sectionID string
	for _, row := range hcRows(t, mgr, fork.ID) {
		if row.Key == "mango-reef" {
			sectionID = row.ID
		}
	}
	if _, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: fork.ID, ParentID: sectionID, Key: "urchin", Title: "Urchin", TargetType: models.HomeTargetFiles}); err != nil {
		t.Fatalf("CreateHomeEntry in the fork: %v", err)
	}
	if n := len(hcRows(t, mgr, source.ID)); n != 12 {
		t.Fatalf("the source holds %d rows after the fork gained one, want 12", n)
	}
	if n := len(hcRows(t, mgr, fork.ID)); n != 13 {
		t.Fatalf("the fork holds %d rows, want 13", n)
	}
}

func TestHomeContract_PublishThenForkFromThePublicationKeepsTheNestingAndTheOrder(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	source := hcImport(t, mgr, hcKelpHome())
	pub := hcPublish(t, mgr, source.ID, "v-reef")

	fork, err := mgr.CreateDraft(ctx, "from a publication", pub.Tag, "publication")
	if err != nil {
		t.Fatalf("CreateDraft (fork of a publication): %v", err)
	}
	hcEqual(t, "the home of a draft forked from the publication", hcTree(hcRows(t, mgr, fork.ID)), hcShapeOfDocument(hcKelpHome()))
	for _, row := range hcRows(t, mgr, fork.ID) {
		if row.DraftID != fork.ID {
			t.Errorf("row %q draft = %q, want %q", row.Key, row.DraftID, fork.ID)
		}
	}
}

func TestHomeContract_AHomeCarriedThroughForkPublishForkIsPublishedAgainAsWritten(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	source := hcImport(t, mgr, hcKelpHome())
	one := hcPublish(t, mgr, source.ID, "one")

	fromDraft, err := mgr.CreateDraft(ctx, "a", source.ID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	var sectionID string
	for _, row := range hcRows(t, mgr, fromDraft.ID) {
		if row.Key == "mango-reef" {
			sectionID = row.ID
		}
	}
	if _, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: fromDraft.ID, ParentID: sectionID, Key: "urchin", Title: "Urchin", TargetType: models.HomeTargetRecord, Target: "RC-5"}); err != nil {
		t.Fatalf("CreateHomeEntry: %v", err)
	}
	two := hcPublish(t, mgr, fromDraft.ID, "two")

	want := hcKelpHome()
	want[2].Entries = append(want[2].Entries, models.DocumentHomeNode{Key: "urchin", Title: "Urchin", Target: hcTarget(models.HomeTargetRecord, "RC-5")})
	if !reflect.DeepEqual(two.Document.Home, want) {
		g, _ := json.MarshalIndent(two.Document.Home, "", " ")
		t.Fatalf("second publication home = %s, want the first plus urchin last in mango-reef", g)
	}
	if !reflect.DeepEqual(one.Document.Home, hcKelpHome()) {
		t.Fatalf("the first publication changed after the second was cut")
	}

	fromPublication, err := mgr.CreateDraft(ctx, "b", two.Tag, "publication")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	hcEqual(t, "the home forked from the second publication", hcTree(hcRows(t, mgr, fromPublication.ID)), hcShapeOfDocument(want))
}

func TestHomeContract_ADraftWhoseOnlyChangeIsOneHomeRowPublishesAgain(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draft := hcImport(t, mgr, hcKelpHome())
	one := hcPublish(t, mgr, draft.ID, "one")
	if _, err := mgr.PublishAgency(ctx, draft.ID, "same", ""); err == nil {
		t.Fatalf("PublishAgency of an unchanged draft was kept, want ErrNoChangesDetected")
	}
	if _, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draft.ID, Key: "urchin", Title: "Urchin", TargetType: models.HomeTargetFiles}); err != nil {
		t.Fatalf("CreateHomeEntry: %v", err)
	}
	two, err := mgr.PublishAgency(ctx, draft.ID, "two", "")
	if err != nil {
		t.Fatalf("PublishAgency of a draft whose only change is one home row: %v", err)
	}
	if len(two.Document.Home) != len(one.Document.Home)+1 || two.Document.Home[len(two.Document.Home)-1].Key != "urchin" {
		t.Fatalf("second home = %+v, want urchin added last", two.Document.Home)
	}
}

func TestHomeContract_CreateHomeEntryPlacesRowsInCallOrderWithinTheirSection(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draft := hcSeedDraft(t, mgr)
	section, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draft.ID, Key: "zeta", Title: "Zeta"})
	if err != nil {
		t.Fatalf("CreateHomeEntry: %v", err)
	}
	for _, key := range []string{"pearl", "coral", "anemone"} {
		if _, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draft.ID, ParentID: section.ID, Key: key, Title: key, TargetType: models.HomeTargetFiles}); err != nil {
			t.Fatalf("CreateHomeEntry %s: %v", key, err)
		}
	}
	if _, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draft.ID, Key: "alpha", Title: "Alpha", TargetType: models.HomeTargetFiles}); err != nil {
		t.Fatalf("CreateHomeEntry alpha: %v", err)
	}

	rows := hcRows(t, mgr, draft.ID)
	byKey := map[string]models.HomeEntry{}
	for _, row := range rows {
		byKey[row.Key] = row
	}
	if !(byKey["pearl"].Ordinality < byKey["coral"].Ordinality && byKey["coral"].Ordinality < byKey["anemone"].Ordinality) {
		t.Fatalf("ordinality pearl %d, coral %d, anemone %d, want it to rise in the order the rows were added",
			byKey["pearl"].Ordinality, byKey["coral"].Ordinality, byKey["anemone"].Ordinality)
	}
	if !(byKey["zeta"].Ordinality < byKey["alpha"].Ordinality) {
		t.Fatalf("ordinality zeta %d, alpha %d, want a later top-level row after an earlier one", byKey["zeta"].Ordinality, byKey["alpha"].Ordinality)
	}
	hcEqual(t, "the created home", hcTree(rows), []hcShape{
		{Key: "zeta", Title: "Zeta", Children: []hcShape{
			{Key: "pearl", Title: "pearl", TargetType: models.HomeTargetFiles},
			{Key: "coral", Title: "coral", TargetType: models.HomeTargetFiles},
			{Key: "anemone", Title: "anemone", TargetType: models.HomeTargetFiles},
		}},
		{Key: "alpha", Title: "Alpha", TargetType: models.HomeTargetFiles},
	})

	pub := hcPublish(t, mgr, draft.ID, "created")
	if len(pub.Document.Home) != 2 || pub.Document.Home[0].Key != "zeta" || pub.Document.Home[1].Key != "alpha" {
		t.Fatalf("published home = %+v, want zeta then alpha", pub.Document.Home)
	}
	var inner []string
	for _, n := range pub.Document.Home[0].Entries {
		inner = append(inner, n.Key)
	}
	if !reflect.DeepEqual(inner, []string{"pearl", "coral", "anemone"}) {
		t.Fatalf("zeta entries = %v, want the order they were added", inner)
	}
}

func TestHomeContract_CreateHomeEntryRefusesWhatTheImportRefuses(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draft := hcSeedDraft(t, mgr)
	entry, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draft.ID, Key: "files", Title: "Files", TargetType: models.HomeTargetFiles})
	if err != nil {
		t.Fatalf("CreateHomeEntry: %v", err)
	}
	cases := []struct {
		name string
		in   models.HomeEntry
	}{
		{"a repeated key", models.HomeEntry{DraftID: draft.ID, Key: "files", Title: "Again", TargetType: models.HomeTargetFiles}},
		{"an unknown target type", models.HomeEntry{DraftID: draft.ID, Key: "odd", Title: "Odd", TargetType: "page", Target: "x"}},
		{"a permission outside the catalogue", models.HomeEntry{DraftID: draft.ID, Key: "gated", Title: "Gated", Permission: "no.such.permission"}},
		{"a record with no ref", models.HomeEntry{DraftID: draft.ID, Key: "bare", Title: "Bare", TargetType: models.HomeTargetRecord}},
		{"a row inside an entry", models.HomeEntry{DraftID: draft.ID, ParentID: entry.ID, Key: "inside", Title: "Inside", TargetType: models.HomeTargetFiles}},
		{"a row inside a parent that is not there", models.HomeEntry{DraftID: draft.ID, ParentID: "no-such-row", Key: "lost", Title: "Lost", TargetType: models.HomeTargetFiles}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := mgr.CreateHomeEntry(ctx, tc.in); err == nil {
				t.Fatalf("CreateHomeEntry kept it")
			}
		})
	}
	if rows := hcRows(t, mgr, draft.ID); len(rows) != 1 {
		t.Fatalf("rows = %+v, want only the first entry kept", rows)
	}
}

func TestHomeContract_CreateHomeEntryRefusesTheSixthLevelAndTheFortyFirstRow(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draft := hcSeedDraft(t, mgr)
	parent := ""
	for level := 1; level <= 5; level++ {
		row, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draft.ID, ParentID: parent, Key: fmt.Sprintf("level-%d", level), Title: "Level"})
		if err != nil {
			t.Fatalf("level %d refused: %v", level, err)
		}
		parent = row.ID
	}
	if _, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draft.ID, ParentID: parent, Key: "level-6", Title: "Level"}); err == nil {
		t.Fatalf("a sixth level was kept")
	}

	crowd, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draft.ID, Key: "crowd", Title: "Crowd"})
	if err != nil {
		t.Fatalf("CreateHomeEntry: %v", err)
	}
	for i := 0; i < 40; i++ {
		if _, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draft.ID, ParentID: crowd.ID, Key: fmt.Sprintf("crowd-%d", i), Title: "Crowd", TargetType: models.HomeTargetFiles}); err != nil {
			t.Fatalf("row %d of 40 refused: %v", i+1, err)
		}
	}
	if _, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draft.ID, ParentID: crowd.ID, Key: "crowd-41", Title: "Crowd", TargetType: models.HomeTargetFiles}); err == nil {
		t.Fatalf("a forty-first row in one section was kept")
	}
}

func TestHomeContract_AnArchivedDraftTakesNoNewHomeRow(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draft := hcImport(t, mgr, hcKelpHome())
	if _, err := mgr.ArchiveDraft(ctx, draft.ID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}
	if _, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draft.ID, Key: "late", Title: "Late", TargetType: models.HomeTargetFiles}); err == nil {
		t.Fatalf("CreateHomeEntry kept a row in an archived draft")
	}
	if n := len(hcRows(t, mgr, draft.ID)); n != 12 {
		t.Fatalf("rows = %d, want the 12 from the import", n)
	}
}

func TestHomeContract_ListHomeEntriesReadsOnlyTheDraftAsked(t *testing.T) {
	mgr := newTestManager(t)
	one := hcImport(t, mgr, []models.DocumentHomeNode{hcEntry("kelp", hcTarget(models.HomeTargetFiles, ""))})
	two := hcImport(t, mgr, []models.DocumentHomeNode{hcEntry("reef", hcTarget(models.HomeTargetFiles, "")), hcEntry("farm", hcTarget(models.HomeTargetFiles, ""))})
	if rows := hcRows(t, mgr, one.ID); len(rows) != 1 || rows[0].Key != "kelp" {
		t.Fatalf("first draft rows = %+v, want only kelp", rows)
	}
	if rows := hcRows(t, mgr, two.ID); len(rows) != 2 {
		t.Fatalf("second draft rows = %+v, want reef and farm", rows)
	}
}

func TestHomeContract_TheSameKeyMayLiveInTwoDrafts(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	source := hcImport(t, mgr, []models.DocumentHomeNode{hcEntry("kelp", hcTarget(models.HomeTargetFiles, ""))})
	if _, err := mgr.CreateDraft(ctx, "fork", source.ID, "draft"); err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	other := hcImport(t, mgr, []models.DocumentHomeNode{hcEntry("kelp", hcTarget(models.HomeTargetFiles, ""))})
	if rows := hcRows(t, mgr, other.ID); len(rows) != 1 {
		t.Fatalf("rows = %+v, want the key kept in a draft of its own", rows)
	}
}
