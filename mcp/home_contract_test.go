package mcp_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type homeContractSession struct {
	t       *testing.T
	mgr     mwanachamaagency.AgencyManager
	cs      *gosdkmcp.ClientSession
	draftID string
}

func newHomeContractSession(t *testing.T) homeContractSession {
	t.Helper()
	mgr := newTestManager(t)
	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	cs := mcpConnect(t, srv.URL)
	if _, err := mgr.CreateAgency(ctxT(t), models.Agency{Name: "Kelp co-op"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draft, err := mgr.CreateDraft(ctxT(t), "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	return homeContractSession{t: t, mgr: mgr, cs: cs, draftID: draft.ID}
}

func (s homeContractSession) call(name string, args map[string]any) (*gosdkmcp.CallToolResult, error) {
	s.t.Helper()
	args["instance_id"] = "any"
	return s.cs.CallTool(ctxT(s.t), &gosdkmcp.CallToolParams{Name: name, Arguments: args})
}

func (s homeContractSession) create(args map[string]any) map[string]any {
	s.t.Helper()
	args["draft_id"] = s.draftID
	out, err := s.call("agency_create_home_entry", args)
	if err != nil || out.IsError {
		s.t.Fatalf("agency_create_home_entry %v: %v %s", args, err, textOf(out))
	}
	row, _ := out.StructuredContent.(map[string]any)
	return row
}

func (s homeContractSession) list() []map[string]any {
	s.t.Helper()
	out, err := s.call("agency_list_home_entries", map[string]any{"draft_id": s.draftID})
	if err != nil || out.IsError {
		s.t.Fatalf("agency_list_home_entries: %v %s", err, textOf(out))
	}
	raw, _ := json.Marshal(out.StructuredContent)
	var holder struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(raw, &holder); err != nil {
		s.t.Fatalf("decode: %v", err)
	}
	return holder.Items
}

func TestHomeContractTools_BothToolsAreListed(t *testing.T) {
	s := newHomeContractSession(t)
	tools, err := s.cs.ListTools(ctxT(t), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	names := map[string]bool{}
	for _, tl := range tools.Tools {
		names[tl.Name] = true
	}
	for _, want := range []string{"agency_create_home_entry", "agency_list_home_entries"} {
		if !names[want] {
			t.Errorf("tools/list missing %q", want)
		}
	}
}

func TestHomeContractTools_TheToolsAreDeclaredUnderTheirPermissionActions(t *testing.T) {
	var file struct {
		Operations map[string]struct {
			Action string `json:"action"`
			Tool   string `json:"tool"`
			Method string `json:"method"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(mwanachamaagency.Operations(), &file); err != nil {
		t.Fatalf("read the operations: %v", err)
	}
	byTool := map[string]string{}
	for _, op := range file.Operations {
		byTool[op.Tool] = op.Action
	}
	if byTool["agency_create_home_entry"] != "agency.home_entry.create" {
		t.Errorf("agency_create_home_entry is under the action %q, want agency.home_entry.create", byTool["agency_create_home_entry"])
	}
	if byTool["agency_list_home_entries"] != "agency.home_entry.list" {
		t.Errorf("agency_list_home_entries is under the action %q, want agency.home_entry.list", byTool["agency_list_home_entries"])
	}
}

func TestHomeContractTools_CreateNestedThenListReadsEveryFieldBack(t *testing.T) {
	s := newHomeContractSession(t)
	section := s.create(map[string]any{"key": "registers", "title": "Registers", "titles": map[string]any{"sw": "Rejista"}, "icon": "folder"})
	sectionID, _ := section["id"].(string)
	if sectionID == "" || section["key"] != "registers" {
		t.Fatalf("created section = %+v, want a minted id", section)
	}
	s.create(map[string]any{"parent_id": sectionID, "key": "farm-list", "title": "All farms", "permission": "task.read", "target_type": "record", "target": "RC-1"})

	items := s.list()
	if len(items) != 2 {
		t.Fatalf("items = %+v, want both rows", items)
	}
	byKey := map[string]map[string]any{}
	for _, item := range items {
		key, _ := item["key"].(string)
		byKey[key] = item
	}
	if byKey["farm-list"]["parent_id"] != sectionID || byKey["farm-list"]["target_type"] != "record" || byKey["farm-list"]["target"] != "RC-1" || byKey["farm-list"]["permission"] != "task.read" {
		t.Fatalf("farm-list = %+v, want its parent, target and permission", byKey["farm-list"])
	}
	titles, _ := byKey["registers"]["titles"].(map[string]any)
	if titles["sw"] != "Rejista" || byKey["registers"]["icon"] != "folder" {
		t.Fatalf("registers = %+v, want its translated title and icon", byKey["registers"])
	}
	if _, has := byKey["registers"]["ordinality"]; !has {
		t.Fatalf("registers = %+v, want an ordinality", byKey["registers"])
	}
}

func TestHomeContractTools_CreateKeepsTheOrderRowsWereAddedIn(t *testing.T) {
	s := newHomeContractSession(t)
	section := s.create(map[string]any{"key": "zeta", "title": "Zeta"})
	sectionID, _ := section["id"].(string)
	for _, key := range []string{"pearl", "coral", "anemone"} {
		s.create(map[string]any{"parent_id": sectionID, "key": key, "title": key, "target_type": "files"})
	}
	places := map[string]float64{}
	for _, item := range s.list() {
		key, _ := item["key"].(string)
		places[key], _ = item["ordinality"].(float64)
	}
	if !(places["pearl"] < places["coral"] && places["coral"] < places["anemone"]) {
		t.Fatalf("ordinality = %v, want it to rise in the order the rows were added", places)
	}
}

func TestHomeContractTools_CreateRefusals(t *testing.T) {
	s := newHomeContractSession(t)
	entry := s.create(map[string]any{"key": "files", "title": "Files", "target_type": "files"})
	entryID, _ := entry["id"].(string)
	cases := []struct {
		name string
		args map[string]any
	}{
		{"no title", map[string]any{"key": "a"}},
		{"a repeated key", map[string]any{"key": "files", "title": "Again", "target_type": "files"}},
		{"an unknown target type", map[string]any{"key": "a", "title": "A", "target_type": "page", "target": "x"}},
		{"a workflow with no ref", map[string]any{"key": "a", "title": "A", "target_type": "workflow"}},
		{"a permission outside the catalogue", map[string]any{"key": "a", "title": "A", "permission": "no.such.permission"}},
		{"a parent that is an entry", map[string]any{"key": "a", "title": "A", "parent_id": entryID}},
		{"a parent that does not exist", map[string]any{"key": "a", "title": "A", "parent_id": "nope"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.args["draft_id"] = s.draftID
			out, err := s.call("agency_create_home_entry", tc.args)
			if err == nil && !out.IsError {
				t.Fatalf("agency_create_home_entry kept it: %s", textOf(out))
			}
		})
	}
	if items := s.list(); len(items) != 1 {
		t.Fatalf("items = %+v, want only the first entry", items)
	}
}

func TestHomeContractTools_CreateRefusesTheSixthLevelAndTheFortyFirstRow(t *testing.T) {
	s := newHomeContractSession(t)
	parent := ""
	for level := 1; level <= 5; level++ {
		args := map[string]any{"key": fmt.Sprintf("level-%d", level), "title": "Level"}
		if parent != "" {
			args["parent_id"] = parent
		}
		parent, _ = s.create(args)["id"].(string)
	}
	out, err := s.call("agency_create_home_entry", map[string]any{"draft_id": s.draftID, "parent_id": parent, "key": "level-6", "title": "Level"})
	if err == nil && !out.IsError {
		t.Fatalf("a sixth level was kept")
	}
	crowd, _ := s.create(map[string]any{"key": "crowd", "title": "Crowd"})["id"].(string)
	for i := 0; i < 40; i++ {
		s.create(map[string]any{"parent_id": crowd, "key": fmt.Sprintf("crowd-%d", i), "title": "Crowd", "target_type": "files"})
	}
	out, err = s.call("agency_create_home_entry", map[string]any{"draft_id": s.draftID, "parent_id": crowd, "key": "crowd-41", "title": "Crowd", "target_type": "files"})
	if err == nil && !out.IsError {
		t.Fatalf("a forty-first row in one section was kept")
	}
}

func TestHomeContractTools_AnArchivedDraftIsRefused(t *testing.T) {
	s := newHomeContractSession(t)
	if _, err := s.mgr.ArchiveDraft(ctxT(t), s.draftID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}
	out, err := s.call("agency_create_home_entry", map[string]any{"draft_id": s.draftID, "key": "late", "title": "Late", "target_type": "files"})
	if err == nil && !out.IsError {
		t.Fatalf("a row was kept in an archived draft")
	}
	if items := s.list(); len(items) != 0 {
		t.Fatalf("items = %+v, want none", items)
	}
}

func TestHomeContractTools_ListReadsOnlyTheDraftAsked(t *testing.T) {
	s := newHomeContractSession(t)
	s.create(map[string]any{"key": "kelp", "title": "Kelp", "target_type": "files"})
	other, err := s.mgr.CreateDraft(ctxT(t), "second", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if _, err := s.mgr.CreateHomeEntry(ctxT(t), models.HomeEntry{DraftID: other.ID, Key: "reef", Title: "Reef", TargetType: models.HomeTargetFiles}); err != nil {
		t.Fatalf("CreateHomeEntry: %v", err)
	}
	items := s.list()
	if len(items) != 1 || items[0]["key"] != "kelp" {
		t.Fatalf("items = %+v, want only the first draft's kelp", items)
	}
}
