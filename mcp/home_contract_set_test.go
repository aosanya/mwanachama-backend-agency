package mcp_test

import (
	"encoding/json"
	"testing"
)

func homeContractNestedEntries() []any {
	var entries []any
	raw := `[
  {"key":"deliver","title":"Deliver work","roles":[],"entries":[
    {"key":"projects","title":"Projects","target":{"type":"record","ref":"RC-4"}},
    {"key":"project-page","title":"Project","page_of":"RC-4","entries":[
      {"key":"days","title":"Days worked","target":{"type":"record","ref":"RC-6","where":[{"question":"Project","operator":"is","value":{"row":"Project title"}}]}},
      {"key":"report","title":"Progress report","target":{"type":"workflow","ref":"W-7","step":"WI-48","scope":"row.order"}}
    ]}
  ]},
  {"key":"mine","title":"My work","target":{"type":"start"}},
  {"key":"running","title":"Running","target":{"type":"ongoing"}}
]`
	_ = json.Unmarshal([]byte(raw), &entries)
	return entries
}

func TestHomeContractSetTool_IsListed(t *testing.T) {
	s := newHomeContractSession(t)
	tools, err := s.cs.ListTools(ctxT(t), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	for _, tl := range tools.Tools {
		if tl.Name == "agency_set_home" {
			return
		}
	}
	t.Fatalf("tools/list has no agency_set_home")
}

func TestHomeContractSetTool_ReplacesTheWholeHomeWithTheNestedOne(t *testing.T) {
	s := newHomeContractSession(t)
	s.create(map[string]any{"key": "old-one", "title": "Old one", "target_type": "files"})

	out, err := s.call("agency_set_home", map[string]any{"draft_id": s.draftID, "entries": homeContractNestedEntries()})
	if err != nil || out.IsError {
		t.Fatalf("agency_set_home: %v %s", err, textOf(out))
	}
	items := s.list()
	if len(items) != 7 {
		t.Fatalf("items = %d %+v, want the 7 nodes sent and nothing of the old Home", len(items), items)
	}
	byKey := map[string]map[string]any{}
	for _, item := range items {
		key, _ := item["key"].(string)
		byKey[key] = item
	}
	if _, kept := byKey["old-one"]; kept {
		t.Fatalf("the old row survived the replacement")
	}
	if byKey["days"]["parent_id"] != byKey["project-page"]["id"] || byKey["project-page"]["page_of"] != "RC-4" {
		t.Fatalf("project-page = %+v, days = %+v, want the nesting and page_of kept", byKey["project-page"], byKey["days"])
	}
	if byKey["mine"]["target_type"] != "start" || byKey["running"]["target_type"] != "ongoing" {
		t.Fatalf("mine = %+v, running = %+v, want the start and ongoing types", byKey["mine"], byKey["running"])
	}
}

func TestHomeContractSetTool_AnEmptyListClearsTheHome(t *testing.T) {
	s := newHomeContractSession(t)
	s.create(map[string]any{"key": "old-one", "title": "Old one", "target_type": "files"})
	out, err := s.call("agency_set_home", map[string]any{"draft_id": s.draftID, "entries": []any{}})
	if err != nil || out.IsError {
		t.Fatalf("agency_set_home: %v %s", err, textOf(out))
	}
	if items := s.list(); len(items) != 0 {
		t.Fatalf("items = %+v, want none", items)
	}
}

func TestHomeContractSetTool_ARefusalKeepsTheOldHome(t *testing.T) {
	s := newHomeContractSession(t)
	s.create(map[string]any{"key": "old-one", "title": "Old one", "target_type": "files"})
	bad := map[string][]any{
		"a repeated key": {
			map[string]any{"key": "a", "title": "A", "target": map[string]any{"type": "files"}},
			map[string]any{"key": "a", "title": "Again", "target": map[string]any{"type": "files"}},
		},
		"a role the draft does not hold": {
			map[string]any{"key": "a", "title": "A", "roles": []any{"R-99"}, "target": map[string]any{"type": "files"}},
		},
		"a scope outside a row page": {
			map[string]any{"key": "a", "title": "A", "target": map[string]any{"type": "workflow", "ref": "W-1", "scope": "row.order"}},
		},
		"a good node then a node with no ref": {
			map[string]any{"key": "good", "title": "Good", "target": map[string]any{"type": "files"}},
			map[string]any{"key": "bad", "title": "Bad", "target": map[string]any{"type": "record"}},
		},
	}
	for name, entries := range bad {
		t.Run(name, func(t *testing.T) {
			out, err := s.call("agency_set_home", map[string]any{"draft_id": s.draftID, "entries": entries})
			if err == nil && !out.IsError {
				t.Fatalf("agency_set_home kept it")
			}
			items := s.list()
			if len(items) != 1 || items[0]["key"] != "old-one" {
				t.Fatalf("items = %+v, want the old Home untouched", items)
			}
		})
	}
}

func TestHomeContractSetTool_AnArchivedDraftIsRefused(t *testing.T) {
	s := newHomeContractSession(t)
	s.create(map[string]any{"key": "old-one", "title": "Old one", "target_type": "files"})
	if _, err := s.mgr.ArchiveDraft(ctxT(t), s.draftID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}
	out, err := s.call("agency_set_home", map[string]any{"draft_id": s.draftID, "entries": []any{}})
	if err == nil && !out.IsError {
		t.Fatalf("agency_set_home cleared an archived draft")
	}
	if items := s.list(); len(items) != 1 {
		t.Fatalf("items = %+v, want the old Home", items)
	}
}
