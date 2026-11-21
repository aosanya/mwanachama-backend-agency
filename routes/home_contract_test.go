package routes_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/aosanya/mwanachama-backend-agency/routes"
)

type homeContractClient struct {
	t   *testing.T
	mgr mwanachamaagency.AgencyManager
}

func (c homeContractClient) call(method, path, body string) (int, []byte) {
	c.t.Helper()
	rec := httptest.NewRecorder()
	serve(c.mgr, rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec.Code, rec.Body.Bytes()
}

func (c homeContractClient) must(method, path, body string, status int) []byte {
	c.t.Helper()
	code, out := c.call(method, path, body)
	if code != status {
		c.t.Fatalf("%s %s = %d %s, want %d", method, path, code, out, status)
	}
	return out
}

func (c homeContractClient) add(draftID, parentID, key, title, targetType, target string) models.HomeEntry {
	c.t.Helper()
	body := map[string]string{"draft_id": draftID, "parent_id": parentID, "key": key, "title": title, "target_type": targetType, "target": target}
	raw, _ := json.Marshal(body)
	var row models.HomeEntry
	if err := json.Unmarshal(c.must(http.MethodPost, "/home-entries", string(raw), http.StatusCreated), &row); err != nil {
		c.t.Fatalf("decode: %v", err)
	}
	return row
}

func (c homeContractClient) list(draftID string) []models.HomeEntry {
	c.t.Helper()
	var rows []models.HomeEntry
	if err := json.Unmarshal(c.must(http.MethodGet, "/home-entries?draft_id="+draftID, "", http.StatusOK), &rows); err != nil {
		c.t.Fatalf("decode: %v", err)
	}
	return rows
}

func newHomeContractDraft(t *testing.T) (homeContractClient, string) {
	t.Helper()
	mgr, draftID := newTestManagerWithDraft(t)
	goal, err := mgr.CreateGoal(t.Context(), models.Goal{DraftID: draftID, Title: "Tend the reef"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if _, err := mgr.CreateObjective(t.Context(), models.Objective{DraftID: draftID, GoalID: goal.ID, Title: "Plant kelp"}); err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	return homeContractClient{t: t, mgr: mgr}, draftID
}

func TestHomeContractRoutes_TheListAndCreateOperationsAreDeclaredWithTheirActions(t *testing.T) {
	want := map[string]string{
		"POST /home-entries": "agency.home_entry.create",
		"GET /home-entries":  "agency.home_entry.list",
	}
	got := map[string]string{}
	for _, rt := range routes.Shape() {
		if strings.HasPrefix(rt.Action, "agency.home_entry.") {
			got[rt.Method+" "+rt.Path] = rt.Action
		}
	}
	for route, action := range want {
		if got[route] != action {
			t.Errorf("Shape() has %q = %q, want the action %q", route, got[route], action)
		}
	}
	if len(got) != len(want) {
		t.Errorf("Shape() holds the home actions %v, want only create and list", got)
	}
}

func TestHomeContractRoutes_BothOperationsAreBoundOnTheRouteTable(t *testing.T) {
	mgr := newTestManager(t)
	bound := map[string]string{}
	for _, rt := range routes.Routes(mgr, nil) {
		if strings.HasPrefix(rt.Action, "agency.home_entry.") {
			bound[rt.Action] = rt.Method + " " + rt.Path
		}
	}
	if bound["agency.home_entry.create"] != "POST /home-entries" || bound["agency.home_entry.list"] != "GET /home-entries" {
		t.Fatalf("bound home routes = %v, want create on POST and list on GET /home-entries", bound)
	}
}

func TestHomeContractRoutes_CreateAnswers201WithTheRowAndTheListReadsTheDraftInPlace(t *testing.T) {
	c, draftID := newHomeContractDraft(t)

	section := c.add(draftID, "", "registers", "Registers", "", "")
	if section.ID == "" || section.DraftID != draftID || section.Key != "registers" || section.Title != "Registers" || section.TargetType != "" {
		t.Fatalf("created section = %+v, want the row as stored", section)
	}
	farms := c.add(draftID, section.ID, "farms", "Farms", "", "")
	list := c.add(draftID, farms.ID, "farm-list", "All farms", "record", "RC-1")
	if list.ParentID != farms.ID || list.TargetType != models.HomeTargetRecord || list.Target != "RC-1" {
		t.Fatalf("created entry = %+v, want its parent, type and ref", list)
	}

	rows := c.list(draftID)
	if len(rows) != 3 {
		t.Fatalf("listed rows = %+v, want all three", rows)
	}
	byKey := map[string]models.HomeEntry{}
	for _, row := range rows {
		byKey[row.Key] = row
	}
	if byKey["farms"].ParentID != section.ID || byKey["farm-list"].ParentID != farms.ID || byKey["registers"].ParentID != "" {
		t.Fatalf("listed parents = %+v, want the nesting declared", rows)
	}
}

func TestHomeContractRoutes_CreateCarriesTitlesIconAndPermission(t *testing.T) {
	c, draftID := newHomeContractDraft(t)
	body := `{"draft_id":"` + draftID + `","key":"clinic","title":"Clinic","titles":{"sw":"Zahanati","hu":"Rendelo"},"icon":"heart","permission":"task.read","target_type":"backlog","target":"clinic"}`
	var row models.HomeEntry
	if err := json.Unmarshal(c.must(http.MethodPost, "/home-entries", body, http.StatusCreated), &row); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if row.Titles["sw"] != "Zahanati" || row.Titles["hu"] != "Rendelo" || row.Icon != "heart" || row.Permission != "task.read" {
		t.Fatalf("created row = %+v, want titles, icon and permission kept", row)
	}
	listed := c.list(draftID)
	if len(listed) != 1 || listed[0].Titles["sw"] != "Zahanati" || listed[0].Icon != "heart" || listed[0].Permission != "task.read" {
		t.Fatalf("listed = %+v, want the same fields read back", listed)
	}
}

func TestHomeContractRoutes_ListKeepsToTheDraftAsked(t *testing.T) {
	c, draftID := newHomeContractDraft(t)
	c.add(draftID, "", "kelp", "Kelp", "files", "")

	var other models.Draft
	if err := json.Unmarshal(c.must(http.MethodPost, "/drafts", `{"description":"second"}`, http.StatusCreated), &other); err != nil {
		t.Fatalf("decode: %v", err)
	}
	c.add(other.ID, "", "reef", "Reef", "files", "")
	c.add(other.ID, "", "farm", "Farm", "files", "")

	if rows := c.list(draftID); len(rows) != 1 || rows[0].Key != "kelp" {
		t.Fatalf("first draft = %+v, want only kelp", rows)
	}
	if rows := c.list(other.ID); len(rows) != 2 {
		t.Fatalf("second draft = %+v, want reef and farm", rows)
	}
}

func TestHomeContractRoutes_CreateRefusals(t *testing.T) {
	c, draftID := newHomeContractDraft(t)
	entry := c.add(draftID, "", "files", "Files", "files", "")
	c.add(draftID, "", "taken", "Taken", "files", "")
	cases := []struct {
		name   string
		body   string
		status int
	}{
		{"no key", `{"draft_id":"` + draftID + `","title":"A"}`, http.StatusBadRequest},
		{"no title", `{"draft_id":"` + draftID + `","key":"a"}`, http.StatusBadRequest},
		{"a repeated key", `{"draft_id":"` + draftID + `","key":"taken","title":"Again","target_type":"files"}`, http.StatusBadRequest},
		{"an unknown target type", `{"draft_id":"` + draftID + `","key":"a","title":"A","target_type":"page","target":"x"}`, http.StatusBadRequest},
		{"a record with no ref", `{"draft_id":"` + draftID + `","key":"a","title":"A","target_type":"record"}`, http.StatusBadRequest},
		{"a permission outside the catalogue", `{"draft_id":"` + draftID + `","key":"a","title":"A","permission":"no.such.permission"}`, http.StatusBadRequest},
		{"a parent that is an entry", `{"draft_id":"` + draftID + `","parent_id":"` + entry.ID + `","key":"a","title":"A"}`, http.StatusBadRequest},
		{"a parent that does not exist", `{"draft_id":"` + draftID + `","parent_id":"nope","key":"a","title":"A"}`, http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if code, out := c.call(http.MethodPost, "/home-entries", tc.body); code != tc.status {
				t.Fatalf("POST /home-entries = %d %s, want %d", code, out, tc.status)
			}
		})
	}
	if rows := c.list(draftID); len(rows) != 2 {
		t.Fatalf("rows = %+v, want only the two good ones", rows)
	}
}

func TestHomeContractRoutes_CreateRefusesTheSixthLevelAndTheFortyFirstRow(t *testing.T) {
	c, draftID := newHomeContractDraft(t)
	parent := ""
	for level := 1; level <= 5; level++ {
		parent = c.add(draftID, parent, fmt.Sprintf("level-%d", level), "Level", "", "").ID
	}
	body := `{"draft_id":"` + draftID + `","parent_id":"` + parent + `","key":"level-6","title":"Level"}`
	if code, out := c.call(http.MethodPost, "/home-entries", body); code != http.StatusBadRequest {
		t.Fatalf("a sixth level = %d %s, want 400", code, out)
	}

	crowd := c.add(draftID, "", "crowd", "Crowd", "", "")
	for i := 0; i < 40; i++ {
		c.add(draftID, crowd.ID, fmt.Sprintf("crowd-%d", i), "Crowd", "files", "")
	}
	body = `{"draft_id":"` + draftID + `","parent_id":"` + crowd.ID + `","key":"crowd-41","title":"Crowd","target_type":"files"}`
	if code, out := c.call(http.MethodPost, "/home-entries", body); code != http.StatusBadRequest {
		t.Fatalf("a forty-first row = %d %s, want 400", code, out)
	}
}

func TestHomeContractRoutes_CreateInAnArchivedDraftIsRefusedWithConflict(t *testing.T) {
	c, draftID := newHomeContractDraft(t)
	c.must(http.MethodPost, "/drafts/"+draftID+"/archive", "", http.StatusOK)
	body := `{"draft_id":"` + draftID + `","key":"late","title":"Late","target_type":"files"}`
	if code, out := c.call(http.MethodPost, "/home-entries", body); code != http.StatusConflict {
		t.Fatalf("POST /home-entries into an archived draft = %d %s, want 409", code, out)
	}
}

func TestHomeContractRoutes_TheRowsSurviveForkAndPublishOverHTTP(t *testing.T) {
	c, draftID := newHomeContractDraft(t)
	section := c.add(draftID, "", "zeta", "Zeta", "", "")
	c.add(draftID, section.ID, "pearl", "Pearl", "files", "")
	c.add(draftID, section.ID, "coral", "Coral", "record", "RC-1")
	c.add(draftID, "", "alpha", "Alpha", "backlog", "reef")

	var fork models.Draft
	forkBody := `{"description":"fork","forked_from_id":"` + draftID + `","forked_from_type":"draft"}`
	if err := json.Unmarshal(c.must(http.MethodPost, "/drafts", forkBody, http.StatusCreated), &fork); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rows := c.list(fork.ID); len(rows) != 4 {
		t.Fatalf("forked rows = %+v, want the four", rows)
	}

	var pub models.AgencyPublication
	if err := json.Unmarshal(c.must(http.MethodPost, "/publications", `{"draft_id":"`+fork.ID+`","tag":"reef-one"}`, http.StatusCreated), &pub); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(pub.Document.Home) != 2 || pub.Document.Home[0].Key != "zeta" || pub.Document.Home[1].Key != "alpha" {
		t.Fatalf("published home = %+v, want zeta then alpha", pub.Document.Home)
	}
	inner := pub.Document.Home[0].Entries
	if len(inner) != 2 || inner[0].Key != "pearl" || inner[1].Key != "coral" || inner[1].Target == nil || inner[1].Target.Ref != "RC-1" {
		t.Fatalf("zeta entries = %+v, want pearl then coral with its target", inner)
	}

	var again models.Draft
	fromPub := `{"description":"again","forked_from_id":"reef-one","forked_from_type":"publication"}`
	if err := json.Unmarshal(c.must(http.MethodPost, "/drafts", fromPub, http.StatusCreated), &again); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rows := c.list(again.ID); len(rows) != 4 {
		t.Fatalf("rows forked from the publication = %+v, want the four", rows)
	}
}
