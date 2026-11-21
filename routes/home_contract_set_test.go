package routes_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/aosanya/mwanachama-backend-agency/routes"
)

func homeContractSetRoute(t *testing.T) (string, string) {
	t.Helper()
	var file struct {
		Operations map[string]struct {
			Action string `json:"action"`
			Tool   string `json:"tool"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(mwanachamaagency.Operations(), &file); err != nil {
		t.Fatalf("read the operations: %v", err)
	}
	action := ""
	for _, op := range file.Operations {
		if op.Tool == "agency_set_home" {
			action = op.Action
		}
	}
	if action == "" {
		t.Fatalf("no operation is served as the tool agency_set_home")
	}
	for _, rt := range routes.Shape() {
		if rt.Action == action {
			return rt.Method, rt.Path
		}
	}
	t.Fatalf("the route table has no route for the action %s", action)
	return "", ""
}

func (c homeContractClient) setHome(draftID, entriesJSON string, status int) []byte {
	c.t.Helper()
	method, path := homeContractSetRoute(c.t)
	return c.must(method, path, `{"draft_id":"`+draftID+`","entries":`+entriesJSON+`}`, status)
}

const homeContractNested = `[
  {"key":"deliver","title":"Deliver work","entries":[
    {"key":"projects","title":"Projects","target":{"type":"record","ref":"RC-4"}},
    {"key":"project-page","title":"Project","page_of":"RC-4","entries":[
      {"key":"days","title":"Days worked","target":{"type":"record","ref":"RC-6","where":[{"question":"Project","operator":"is","value":{"row":"Project title"}}]}},
      {"key":"report","title":"Progress report","target":{"type":"workflow","ref":"W-7","step":"WI-48","scope":"row.order"}}
    ]}
  ]},
  {"key":"mine","title":"My work","target":{"type":"start"}},
  {"key":"running","title":"Running","target":{"type":"ongoing"}}
]`

func homeContractKeys(rows []models.HomeEntry) map[string]models.HomeEntry {
	out := map[string]models.HomeEntry{}
	for _, row := range rows {
		out[row.Key] = row
	}
	return out
}

func TestHomeContractSet_TheToolHasARouteAndTakesTheWholeNestedHome(t *testing.T) {
	c, draftID := newHomeContractDraft(t)
	c.setHome(draftID, homeContractNested, http.StatusOK)

	rows := c.list(draftID)
	if len(rows) != 7 {
		t.Fatalf("rows = %d, want the 7 nodes sent", len(rows))
	}
	byKey := homeContractKeys(rows)
	if byKey["deliver"].ParentID != "" || byKey["projects"].ParentID != byKey["deliver"].ID || byKey["days"].ParentID != byKey["project-page"].ID || byKey["project-page"].ParentID != byKey["deliver"].ID {
		t.Fatalf("parents do not follow the nesting sent: %+v", rows)
	}
	if !(byKey["deliver"].Ordinality < byKey["mine"].Ordinality && byKey["mine"].Ordinality < byKey["running"].Ordinality) {
		t.Fatalf("top-level ordinality = %d %d %d, want the order sent", byKey["deliver"].Ordinality, byKey["mine"].Ordinality, byKey["running"].Ordinality)
	}
	if !(byKey["projects"].Ordinality < byKey["project-page"].Ordinality) || !(byKey["days"].Ordinality < byKey["report"].Ordinality) {
		t.Fatalf("inner ordinality does not follow the order sent")
	}
}

func TestHomeContractSet_ReplacesTheWholeHomeAndKeepsNothingOfTheOld(t *testing.T) {
	c, draftID := newHomeContractDraft(t)
	c.add(draftID, "", "old-one", "Old one", "files", "")
	old := c.add(draftID, "", "old-two", "Old two", "", "")
	c.add(draftID, old.ID, "old-inner", "Old inner", "files", "")

	c.setHome(draftID, `[{"key":"fresh","title":"Fresh","target":{"type":"files"}},{"key":"old-one","title":"Reused key","target":{"type":"files"}}]`, http.StatusOK)

	rows := c.list(draftID)
	if len(rows) != 2 {
		t.Fatalf("rows = %+v, want only the two sent", rows)
	}
	byKey := homeContractKeys(rows)
	if byKey["fresh"].ID == "" || byKey["old-one"].Title != "Reused key" {
		t.Fatalf("rows = %+v, want the two sent, a key of the old Home reusable", rows)
	}
	if _, kept := byKey["old-inner"]; kept {
		t.Fatalf("a row of the old Home survived")
	}
}

func TestHomeContractSet_AnEmptyListClearsTheHome(t *testing.T) {
	c, draftID := newHomeContractDraft(t)
	c.add(draftID, "", "old-one", "Old one", "files", "")
	c.setHome(draftID, `[]`, http.StatusOK)
	if rows := c.list(draftID); len(rows) != 0 {
		t.Fatalf("rows = %+v, want none", rows)
	}
}

func TestHomeContractSet_ARefusalRefusesTheWholeHomeAndKeepsTheOldOne(t *testing.T) {
	c, draftID := newHomeContractDraft(t)
	c.add(draftID, "", "old-one", "Old one", "files", "")
	cases := map[string]string{
		"a repeated key":                    `[{"key":"a","title":"A","target":{"type":"files"}},{"key":"a","title":"Again","target":{"type":"files"}}]`,
		"entries and target":                `[{"key":"a","title":"A","target":{"type":"files"},"entries":[{"key":"b","title":"B","target":{"type":"files"}}]}]`,
		"an unknown target type":            `[{"key":"a","title":"A","target":{"type":"page","ref":"x"}}]`,
		"a missing ref":                     `[{"key":"a","title":"A","target":{"type":"record"}}]`,
		"a permission not in the catalogue": `[{"key":"a","title":"A","permission":"no.such.permission","target":{"type":"files"}}]`,
		"a role the draft does not hold":    `[{"key":"a","title":"A","roles":["R-99"],"target":{"type":"files"}}]`,
		"a step on a record":                `[{"key":"a","title":"A","target":{"type":"record","ref":"RC-1","step":"WI-1"}}]`,
		"a where on a workflow":             `[{"key":"a","title":"A","target":{"type":"workflow","ref":"W-1","where":[{"question":"Q","operator":"is","value":"v"}]}}]`,
		"a scope outside a row page":        `[{"key":"a","title":"A","target":{"type":"workflow","ref":"W-1","scope":"row.order"}}]`,
		"page_of on an entry":               `[{"key":"a","title":"A","page_of":"RC-1","target":{"type":"files"}}]`,
		"two pages for one record":          `[{"key":"a","title":"A","page_of":"RC-1"},{"key":"b","title":"B","page_of":"RC-1"}]`,
		"a good node then a bad one":        `[{"key":"good","title":"Good","target":{"type":"files"}},{"key":"bad","title":"Bad","target":{"type":"record"}}]`,
		"a sixth level":                     `[{"key":"a","title":"A","entries":[{"key":"b","title":"B","entries":[{"key":"c","title":"C","entries":[{"key":"d","title":"D","entries":[{"key":"e","title":"E","entries":[{"key":"f","title":"F"}]}]}]}]}]}]`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			method, path := homeContractSetRoute(t)
			code, out := c.call(method, path, `{"draft_id":"`+draftID+`","entries":`+body+`}`)
			if code < 400 || code >= 500 {
				t.Fatalf("set home = %d %s, want a client refusal", code, out)
			}
			rows := c.list(draftID)
			if len(rows) != 1 || rows[0].Key != "old-one" {
				t.Fatalf("rows after the refusal = %+v, want the old Home untouched", rows)
			}
		})
	}
}

func TestHomeContractSet_ARoleOfTheDraftIsAccepted(t *testing.T) {
	c, draftID := newHomeContractDraft(t)
	role, err := c.mgr.CreateRole(t.Context(), models.Role{DraftID: draftID, Name: "Diver"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	c.setHome(draftID, `[{"key":"a","title":"A","roles":["`+role.Code+`"],"target":{"type":"files"}}]`, http.StatusOK)
	rows := c.list(draftID)
	if len(rows) != 1 || len(rows[0].Roles) != 1 || rows[0].Roles[0] != role.Code {
		t.Fatalf("rows = %+v, want the role %s kept", rows, role.Code)
	}
}

func TestHomeContractSet_AnArchivedDraftTakesNoNewHome(t *testing.T) {
	c, draftID := newHomeContractDraft(t)
	c.add(draftID, "", "old-one", "Old one", "files", "")
	c.must(http.MethodPost, "/drafts/"+draftID+"/archive", "", http.StatusOK)
	c.setHome(draftID, `[{"key":"late","title":"Late","target":{"type":"files"}}]`, http.StatusConflict)
	if rows := c.list(draftID); len(rows) != 1 || rows[0].Key != "old-one" {
		t.Fatalf("rows = %+v, want the old Home", rows)
	}
}

func TestHomeContractSet_AnUnknownDraftIsRefused(t *testing.T) {
	c, _ := newHomeContractDraft(t)
	method, path := homeContractSetRoute(t)
	if code, out := c.call(method, path, `{"draft_id":"no-such-draft","entries":[]}`); code != http.StatusNotFound {
		t.Fatalf("set home on an unknown draft = %d %s, want 404", code, out)
	}
}

func TestHomeContractSet_TouchesOnlyTheDraftNamed(t *testing.T) {
	c, draftID := newHomeContractDraft(t)
	c.add(draftID, "", "kelp", "Kelp", "files", "")
	var other models.Draft
	if err := json.Unmarshal(c.must(http.MethodPost, "/drafts", `{"description":"second"}`, http.StatusCreated), &other); err != nil {
		t.Fatalf("decode: %v", err)
	}
	c.add(other.ID, "", "reef", "Reef", "files", "")
	c.setHome(other.ID, `[]`, http.StatusOK)
	if rows := c.list(draftID); len(rows) != 1 {
		t.Fatalf("the first draft's rows = %+v, want kelp kept", rows)
	}
	if rows := c.list(other.ID); len(rows) != 0 {
		t.Fatalf("the second draft's rows = %+v, want none", rows)
	}
}

func TestHomeContractSet_TheHomeSetIsCarriedThroughForkAndPublishAsWritten(t *testing.T) {
	c, draftID := newHomeContractDraft(t)
	c.setHome(draftID, homeContractNested, http.StatusOK)

	var fork models.Draft
	if err := json.Unmarshal(c.must(http.MethodPost, "/drafts", `{"description":"fork","forked_from_id":"`+draftID+`","forked_from_type":"draft"}`, http.StatusCreated), &fork); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rows := c.list(fork.ID); len(rows) != 7 {
		t.Fatalf("forked rows = %d, want 7", len(rows))
	}
	var pub struct {
		Tag      string `json:"tag"`
		Document struct {
			Home json.RawMessage `json:"home"`
		} `json:"document"`
	}
	if err := json.Unmarshal(c.must(http.MethodPost, "/publications", `{"draft_id":"`+fork.ID+`","tag":"set-one"}`, http.StatusCreated), &pub); err != nil {
		t.Fatalf("decode: %v", err)
	}
	var got, want any
	if err := json.Unmarshal(pub.Document.Home, &got); err != nil {
		t.Fatalf("the document carries no home: %v", err)
	}
	if err := json.Unmarshal([]byte(homeContractNested), &want); err != nil {
		t.Fatalf("decode: %v", err)
	}
	gotText, _ := json.Marshal(got)
	wantText, _ := json.Marshal(want)
	if string(gotText) != string(wantText) {
		t.Fatalf("published home = %s\nwant %s", gotText, wantText)
	}

	var again models.Draft
	if err := json.Unmarshal(c.must(http.MethodPost, "/drafts", `{"description":"again","forked_from_id":"set-one","forked_from_type":"publication"}`, http.StatusCreated), &again); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rows := c.list(again.ID); len(rows) != 7 {
		t.Fatalf("rows forked from the publication = %d, want 7", len(rows))
	}
}

func TestHomeContractSet_CreateTakesRolesAndPageOfAndTheNewTargetFields(t *testing.T) {
	c, draftID := newHomeContractDraft(t)
	role, err := c.mgr.CreateRole(t.Context(), models.Role{DraftID: draftID, Name: "Diver"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	post := func(body string, status int) {
		t.Helper()
		c.must(http.MethodPost, "/home-entries", body, status)
	}
	post(`{"draft_id":"`+draftID+`","key":"p","title":"P","page_of":"RC-4"}`, http.StatusCreated)
	post(`{"draft_id":"`+draftID+`","key":"g","title":"G","roles":["`+role.Code+`"],"target_type":"start"}`, http.StatusCreated)
	post(`{"draft_id":"`+draftID+`","key":"o","title":"O","target_type":"ongoing"}`, http.StatusCreated)
	post(`{"draft_id":"`+draftID+`","key":"bad-role","title":"B","roles":["R-99"],"target_type":"files"}`, http.StatusBadRequest)
	post(`{"draft_id":"`+draftID+`","key":"two-pages","title":"B","page_of":"RC-4"}`, http.StatusBadRequest)

	byKey := homeContractKeys(c.list(draftID))
	if len(byKey) != 3 || byKey["p"].PageOf != "RC-4" || byKey["g"].TargetType != "start" || byKey["o"].TargetType != "ongoing" || len(byKey["g"].Roles) != 1 {
		t.Fatalf("rows = %+v, want the three good ones with their fields", byKey)
	}
	if !strings.HasPrefix(role.Code, "R-") {
		t.Fatalf("role code = %q", role.Code)
	}
}
