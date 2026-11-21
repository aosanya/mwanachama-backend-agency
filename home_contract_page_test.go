package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

const kelpFilePrefix = `{
  "agency": {"name": "Kelp co-op"},
  "goals": [{"id": "g1", "code": "G-1", "title": "Tend the reef"}],
  "objectives": [{"id": "o1", "code": "O-1", "goal_id": "g1", "title": "Plant kelp"}],
  "roles": [
    {"id": "r3", "code": "R-3", "name": "Diver", "description": "Dives"},
    {"id": "r4", "code": "R-4", "name": "Skipper", "description": "Steers"}
  ],
  "home": `

const kelpFileSuffix = `
}`

const rowPageHome = `[
  {
    "key": "deliver", "title": "Deliver work", "icon": "folder", "titles": { "sw": "Kazi" },
    "entries": [
      { "key": "projects", "title": "Projects", "roles": ["R-3", "R-4"],
        "target": { "type": "record", "ref": "RC-4" } },
      {
        "key": "project-page", "title": "Project", "page_of": "RC-4",
        "entries": [
          { "key": "days", "title": "Days worked",
            "target": { "type": "record", "ref": "RC-6",
                        "where": [ { "question": "Project", "operator": "is", "value": { "row": "Project title" } } ] } },
          { "key": "report", "title": "Progress report",
            "target": { "type": "workflow", "ref": "W-7", "step": "WI-48", "scope": "row.order" } }
        ]
      }
    ]
  },
  { "key": "mine", "title": "My work", "target": { "type": "start" } },
  { "key": "running", "title": "Running", "target": { "type": "ongoing" } },
  { "key": "shared", "title": "Files", "target": { "type": "files" } }
]`

func hcFileWithHome(t *testing.T, homeJSON string) models.AgencyDocument {
	t.Helper()
	return hcFileDocument(t, kelpFilePrefix+homeJSON+kelpFileSuffix)
}

func hcJSONHomeOfPublication(t *testing.T, pub models.AgencyPublication) any {
	t.Helper()
	raw, err := json.Marshal(pub.Document)
	if err != nil {
		t.Fatalf("marshal the document: %v", err)
	}
	var whole map[string]any
	if err := json.Unmarshal(raw, &whole); err != nil {
		t.Fatalf("unmarshal the document: %v", err)
	}
	return whole["home"]
}

func hcJSONOf(t *testing.T, text string) any {
	t.Helper()
	var out any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("read %s: %v", text, err)
	}
	return out
}

func hcSameJSON(t *testing.T, what string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		g, _ := json.MarshalIndent(got, "", " ")
		w, _ := json.MarshalIndent(want, "", " ")
		t.Fatalf("%s differ.\ngot:\n%s\nwant:\n%s", what, g, w)
	}
}

func TestHomeContractPages_TheRowPageExampleIsPublishedExactlyAsWritten(t *testing.T) {
	mgr := newTestManager(t)
	out, err := mgr.ImportDocument(context.Background(), "kelp", hcFileWithHome(t, rowPageHome))
	if err != nil {
		t.Fatalf("ImportDocument of the design page example: %v", err)
	}
	pub := hcPublish(t, mgr, out.Draft.ID, "page-one")
	hcSameJSON(t, "the published home", hcJSONHomeOfPublication(t, pub), hcJSONOf(t, rowPageHome))
}

func TestHomeContractPages_TheHomeSurvivesForkOfADraftAndForkOfAPublicationAsWritten(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	out, err := mgr.ImportDocument(ctx, "kelp", hcFileWithHome(t, rowPageHome))
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	want := hcJSONOf(t, rowPageHome)

	fromDraft, err := mgr.CreateDraft(ctx, "fork", out.Draft.ID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft from a draft: %v", err)
	}
	one := hcPublish(t, mgr, fromDraft.ID, "from-draft")
	hcSameJSON(t, "the home published from a draft fork", hcJSONHomeOfPublication(t, one), want)

	fromPublication, err := mgr.CreateDraft(ctx, "fork", one.Tag, "publication")
	if err != nil {
		t.Fatalf("CreateDraft from a publication: %v", err)
	}
	two := hcPublish(t, mgr, fromPublication.ID, "from-publication")
	hcSameJSON(t, "the home published from a publication fork", hcJSONHomeOfPublication(t, two), want)
}

func TestHomeContractPages_ThePublishedDocumentImportsBackAsWritten(t *testing.T) {
	first := newTestManager(t)
	out, err := first.ImportDocument(context.Background(), "kelp", hcFileWithHome(t, rowPageHome))
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	pub := hcPublish(t, first, out.Draft.ID, "one")
	raw, err := json.Marshal(pub.Document)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	second := newTestManager(t)
	again, err := second.ImportDocument(context.Background(), "again", hcFileDocument(t, string(raw)))
	if err != nil {
		t.Fatalf("ImportDocument of the exported document: %v", err)
	}
	two := hcPublish(t, second, again.Draft.ID, "two")
	hcSameJSON(t, "the home after the round trip", hcJSONHomeOfPublication(t, two), hcJSONOf(t, rowPageHome))
}

func hcRowsAsJSON(t *testing.T, mgr mwanachamaagency.AgencyManager, draftID string) map[string]map[string]any {
	t.Helper()
	out := map[string]map[string]any{}
	for _, row := range hcRows(t, mgr, draftID) {
		raw, err := json.Marshal(row)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var fields map[string]any
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		out[row.Key] = fields
	}
	return out
}

func TestHomeContractPages_RowsCarryRolesAndPageOfAndTheStartAndOngoingTypes(t *testing.T) {
	mgr := newTestManager(t)
	out, err := mgr.ImportDocument(context.Background(), "kelp", hcFileWithHome(t, rowPageHome))
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	rows := hcRowsAsJSON(t, mgr, out.Draft.ID)
	if len(rows) != 8 {
		t.Fatalf("rows = %d, want the 8 declared nodes", len(rows))
	}
	if !reflect.DeepEqual(rows["projects"]["roles"], []any{"R-3", "R-4"}) {
		t.Errorf("projects roles = %v, want R-3 and R-4 in the order written", rows["projects"]["roles"])
	}
	if rows["project-page"]["page_of"] != "RC-4" {
		t.Errorf("project-page page_of = %v, want RC-4", rows["project-page"]["page_of"])
	}
	if rows["mine"]["target_type"] != "start" || rows["running"]["target_type"] != "ongoing" {
		t.Errorf("mine = %v, running = %v, want the start and ongoing types kept", rows["mine"]["target_type"], rows["running"]["target_type"])
	}
	if rows["days"]["parent_id"] != rows["project-page"]["id"] {
		t.Errorf("days parent = %v, want the row page", rows["days"]["parent_id"])
	}
}

func TestHomeContractPages_AcceptedDeclarations(t *testing.T) {
	where := func(n int) string {
		var parts []string
		for i := 0; i < n; i++ {
			parts = append(parts, fmt.Sprintf(`{"question":"Q%d","operator":"is","value":"v"}`, i))
		}
		return strings.Join(parts, ",")
	}
	cases := []struct{ name, home string }{
		{"start with no ref", `[{"key":"a","title":"A","target":{"type":"start"}}]`},
		{"ongoing with no ref", `[{"key":"a","title":"A","target":{"type":"ongoing"}}]`},
		{"roles the file holds", `[{"key":"a","title":"A","roles":["R-3"],"target":{"type":"files"}}]`},
		{"roles on a section", `[{"key":"a","title":"A","roles":["R-4"],"entries":[{"key":"b","title":"B","target":{"type":"files"}}]}]`},
		{"permission and roles together", `[{"key":"a","title":"A","permission":"task.read","roles":["R-3"],"target":{"type":"files"}}]`},
		{"a row page with nothing in it", `[{"key":"p","title":"P","page_of":"RC-4"}]`},
		{"row pages for two records", `[{"key":"p","title":"P","page_of":"RC-4"},{"key":"q","title":"Q","page_of":"RC-5"}]`},
		{"a row page deep in the tree", `[{"key":"a","title":"A","entries":[{"key":"b","title":"B","entries":[{"key":"p","title":"P","page_of":"RC-4"}]}]}]`},
		{"where with 5 conditions", `[{"key":"a","title":"A","target":{"type":"record","ref":"RC-1","where":[` + where(5) + `]}}]`},
		{"a row placeholder in a section inside a row page", `[{"key":"p","title":"P","page_of":"RC-4","entries":[{"key":"s","title":"S","entries":[{"key":"d","title":"D","target":{"type":"record","ref":"RC-6","where":[{"question":"Project","operator":"is","value":{"row":"Project title"}}]}}]}]}]`},
		{"a row scope inside a section inside a row page", `[{"key":"p","title":"P","page_of":"RC-4","entries":[{"key":"s","title":"S","entries":[{"key":"w","title":"W","target":{"type":"workflow","ref":"W-7","scope":"row.order"}}]}]}]`},
		{"a step on a workflow", `[{"key":"a","title":"A","target":{"type":"workflow","ref":"W-7","step":"WI-48"}}]`},
		{"a ref of 200 characters", `[{"key":"a","title":"A","target":{"type":"record","ref":"` + strings.Repeat("r", 200) + `"}}]`},
		{"a step of 200 characters", `[{"key":"a","title":"A","target":{"type":"workflow","ref":"W-7","step":"` + strings.Repeat("s", 200) + `"}}]`},
		{"a row page at level 5", `[{"key":"a","title":"A","entries":[{"key":"b","title":"B","entries":[{"key":"c","title":"C","entries":[{"key":"p","title":"P","page_of":"RC-4","entries":[{"key":"d","title":"D","target":{"type":"files"}}]}]}]}]}]`},
	}
	for _, op := range []string{"is", "is_not", "contains", "more_than", "less_than", "on_or_before", "on_or_after"} {
		cases = append(cases, struct{ name, home string }{
			"the operator " + op,
			`[{"key":"a","title":"A","target":{"type":"record","ref":"RC-1","where":[{"question":"Q","operator":"` + op + `","value":"v"}]}}]`,
		})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := newTestManager(t)
			out, err := mgr.ImportDocument(context.Background(), "kelp", hcFileWithHome(t, tc.home))
			if err != nil {
				t.Fatalf("ImportDocument refused a declaration the page allows: %v", err)
			}
			pub := hcPublish(t, mgr, out.Draft.ID, "ok")
			hcSameJSON(t, "the published home", hcJSONHomeOfPublication(t, pub), hcJSONOf(t, tc.home))
		})
	}
}

func TestHomeContractPages_RefusedDeclarationsKeepNothingAtAll(t *testing.T) {
	where := func(n int) string {
		var parts []string
		for i := 0; i < n; i++ {
			parts = append(parts, fmt.Sprintf(`{"question":"Q%d","operator":"is","value":"v"}`, i))
		}
		return strings.Join(parts, ",")
	}
	fine := `{"key":"fine","title":"Fine","entries":[{"key":"fine-inner","title":"Inner","target":{"type":"record","ref":"RC-1"}}]}`
	cases := []struct{ name, bad string }{
		{"a role the file does not hold", `{"key":"a","title":"A","roles":["R-9"],"target":{"type":"files"}}`},
		{"one role the file holds and one it does not", `{"key":"a","title":"A","roles":["R-3","R-9"],"target":{"type":"files"}}`},
		{"a role the file does not hold on a section", `{"key":"a","title":"A","roles":["R-9"],"entries":[{"key":"b","title":"B","target":{"type":"files"}}]}`},
		{"a role the file does not hold three levels down", `{"key":"a","title":"A","entries":[{"key":"b","title":"B","entries":[{"key":"c","title":"C","roles":["R-9"],"target":{"type":"files"}}]}]}`},
		{"a step on a record", `{"key":"a","title":"A","target":{"type":"record","ref":"RC-1","step":"WI-1"}}`},
		{"a step on files", `{"key":"a","title":"A","target":{"type":"files","step":"WI-1"}}`},
		{"a step on a backlog", `{"key":"a","title":"A","target":{"type":"backlog","ref":"reef","step":"WI-1"}}`},
		{"a where on a workflow", `{"key":"a","title":"A","target":{"type":"workflow","ref":"W-7","where":[{"question":"Q","operator":"is","value":"v"}]}}`},
		{"a where on a backlog", `{"key":"a","title":"A","target":{"type":"backlog","ref":"reef","where":[{"question":"Q","operator":"is","value":"v"}]}}`},
		{"a where on files", `{"key":"a","title":"A","target":{"type":"files","where":[{"question":"Q","operator":"is","value":"v"}]}}`},
		{"a scope outside a row page", `{"key":"a","title":"A","target":{"type":"workflow","ref":"W-7","scope":"row.order"}}`},
		{"a scope inside a section that is not a row page", `{"key":"a","title":"A","entries":[{"key":"b","title":"B","target":{"type":"workflow","ref":"W-7","scope":"row.order"}}]}`},
		{"a scope on a record", `{"key":"p","title":"P","page_of":"RC-4","entries":[{"key":"b","title":"B","target":{"type":"record","ref":"RC-1","scope":"row.order"}}]}`},
		{"page_of on an entry", `{"key":"a","title":"A","page_of":"RC-4","target":{"type":"record","ref":"RC-4"}}`},
		{"two row pages for one record at the top", `{"key":"p","title":"P","page_of":"RC-4"},{"key":"q","title":"Q","page_of":"RC-4"}`},
		{"two row pages for one record at different levels", `{"key":"p","title":"P","page_of":"RC-4"},{"key":"x","title":"X","entries":[{"key":"q","title":"Q","page_of":"RC-4"}]}`},
		{"a where with 6 conditions", `{"key":"a","title":"A","target":{"type":"record","ref":"RC-1","where":[` + where(6) + `]}}`},
		{"a where with an unknown operator", `{"key":"a","title":"A","target":{"type":"record","ref":"RC-1","where":[{"question":"Q","operator":"resembles","value":"v"}]}}`},
		{"a where with no question", `{"key":"a","title":"A","target":{"type":"record","ref":"RC-1","where":[{"operator":"is","value":"v"}]}}`},
		{"a row placeholder outside a row page", `{"key":"a","title":"A","target":{"type":"record","ref":"RC-1","where":[{"question":"Q","operator":"is","value":{"row":"Project title"}}]}}`},
		{"a ref of 201 characters", `{"key":"a","title":"A","target":{"type":"record","ref":"` + strings.Repeat("r", 201) + `"}}`},
		{"a step of 201 characters", `{"key":"a","title":"A","target":{"type":"workflow","ref":"W-7","step":"` + strings.Repeat("s", 201) + `"}}`},
		{"a row page sixth level", `{"key":"a","title":"A","entries":[{"key":"b","title":"B","entries":[{"key":"c","title":"C","entries":[{"key":"d","title":"D","entries":[{"key":"p","title":"P","page_of":"RC-4","entries":[{"key":"e","title":"E","target":{"type":"files"}}]}]}]}]}]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mgr, db, s := newTestManagerDB(t)
			ctx := context.Background()
			before, err := mgr.ListDrafts(ctx, mwanachamaagency.DraftFilter{})
			if err != nil {
				t.Fatalf("ListDrafts: %v", err)
			}
			doc := hcFileWithHome(t, "["+fine+","+tc.bad+"]")
			if _, err := mgr.ImportDocument(ctx, "refused", doc); err == nil {
				t.Fatalf("ImportDocument kept a declaration the page refuses")
			}
			var kept int64
			if err := db.Table(tableOf(t, s, "home_entry")).Count(&kept).Error; err != nil {
				t.Fatalf("count: %v", err)
			}
			if kept != 0 {
				t.Fatalf("%d home rows kept after the refusal, want none, the fine nodes included", kept)
			}
			after, err := mgr.ListDrafts(ctx, mwanachamaagency.DraftFilter{})
			if err != nil {
				t.Fatalf("ListDrafts: %v", err)
			}
			if len(after) != len(before) {
				t.Fatalf("%d drafts after the refusal, want %d", len(after), len(before))
			}
			if _, err := mgr.GetAgency(ctx); err == nil {
				t.Fatalf("the refused file left an agency behind")
			}
		})
	}
}

func TestHomeContractPages_ARowPlaceholderIsNotCheckedAgainstTheRecordsQuestions(t *testing.T) {
	mgr := newTestManager(t)
	home := `[{"key":"p","title":"P","page_of":"RC-404","entries":[{"key":"d","title":"D","target":{"type":"record","ref":"RC-405","where":[{"question":"No such question","operator":"is","value":{"row":"No such prompt"}}]}}]}]`
	out, err := mgr.ImportDocument(context.Background(), "kelp", hcFileWithHome(t, home))
	if err != nil {
		t.Fatalf("ImportDocument refused a placeholder for a prompt the record does not have: %v", err)
	}
	if rows := hcRows(t, mgr, out.Draft.ID); len(rows) != 2 {
		t.Fatalf("rows = %d, want both kept", len(rows))
	}
}

func TestHomeContractPages_ARoleIsARoleCodeOfTheFileNotAPermissionCode(t *testing.T) {
	mgr := newTestManager(t)
	home := `[{"key":"a","title":"A","roles":["task.read"],"target":{"type":"files"}}]`
	if _, err := mgr.ImportDocument(context.Background(), "kelp", hcFileWithHome(t, home)); err == nil {
		t.Fatalf("ImportDocument took a permission code as a role code")
	}
}

func TestHomeContractPages_AFileWithNoRolesRefusesAnyRoleOnTheHome(t *testing.T) {
	mgr := newTestManager(t)
	doc := hcDocument([]models.DocumentHomeNode{{Key: "a", Title: "A", Roles: []string{"R-1"}, Target: &models.HomeTarget{Type: models.HomeTargetFiles}}})
	if _, err := mgr.ImportDocument(context.Background(), "kelp", doc); err == nil {
		t.Fatalf("ImportDocument took a role the file does not hold")
	}
}

func TestHomeContractPages_RolesAndRowPagesSurviveAForkFromAPublication(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	out, err := mgr.ImportDocument(ctx, "kelp", hcFileWithHome(t, rowPageHome))
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	pub := hcPublish(t, mgr, out.Draft.ID, "one")
	fork, err := mgr.CreateDraft(ctx, "fork", pub.Tag, "publication")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	rows := hcRowsAsJSON(t, mgr, fork.ID)
	if !reflect.DeepEqual(rows["projects"]["roles"], []any{"R-3", "R-4"}) || rows["project-page"]["page_of"] != "RC-4" {
		t.Fatalf("forked rows lost roles or page_of: projects %v, page %v", rows["projects"]["roles"], rows["project-page"]["page_of"])
	}
}
