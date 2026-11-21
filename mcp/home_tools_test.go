package mcp_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestHomeTools_CreateNestedThenListTheDraft(t *testing.T) {
	mgr := newTestManager(t)
	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cs := mcpConnect(t, srv.URL)

	if _, err := mgr.CreateAgency(ctxT(t), models.Agency{Name: "Farmers"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draft, err := mgr.CreateDraft(ctxT(t), "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	created, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_create_home_entry",
		Arguments: map[string]any{"instance_id": "any", "draft_id": draft.ID, "key": "registers", "title": "Registers"},
	})
	if err != nil || created.IsError {
		t.Fatalf("agency_create_home_entry: %v %+v", err, textOf(created))
	}
	section, _ := created.StructuredContent.(map[string]any)
	sectionID, _ := section["id"].(string)
	if sectionID == "" {
		t.Fatalf("section = %+v, want an id", section)
	}
	inner, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name: "agency_create_home_entry",
		Arguments: map[string]any{
			"instance_id": "any", "draft_id": draft.ID, "parent_id": sectionID, "key": "farms", "title": "Farms",
			"target_type": "record", "target": "RC-1",
		},
	})
	if err != nil || inner.IsError {
		t.Fatalf("agency_create_home_entry inside: %v %+v", err, textOf(inner))
	}

	result, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_list_home_entries",
		Arguments: map[string]any{"instance_id": "any", "draft_id": draft.ID},
	})
	if err != nil || result.IsError {
		t.Fatalf("agency_list_home_entries: %v %+v", err, textOf(result))
	}
	structured, _ := result.StructuredContent.(map[string]any)
	items, _ := structured["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items = %+v, want both rows", structured)
	}
}

func TestHomeTools_CreateRefusesAnUnknownParent(t *testing.T) {
	mgr := newTestManager(t)
	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cs := mcpConnect(t, srv.URL)

	if _, err := mgr.CreateAgency(ctxT(t), models.Agency{Name: "Farmers"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draft, err := mgr.CreateDraft(ctxT(t), "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	result, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_create_home_entry",
		Arguments: map[string]any{"instance_id": "any", "draft_id": draft.ID, "parent_id": "nope", "key": "a", "title": "A"},
	})
	if err != nil || !result.IsError || !strings.Contains(textOf(result), "home entry not found") {
		t.Fatalf("agency_create_home_entry = %v %q, want a refusal saying the home entry was not found", err, textOf(result))
	}
}

func TestHomeTools_SetHomeReplacesTheHomeWithANestedOne(t *testing.T) {
	mgr := newTestManager(t)
	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cs := mcpConnect(t, srv.URL)

	if _, err := mgr.CreateAgency(ctxT(t), models.Agency{Name: "Farmers"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draft, err := mgr.CreateDraft(ctxT(t), "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	entries := []any{
		map[string]any{"key": "deliver", "title": "Deliver", "entries": []any{
			map[string]any{"key": "projects", "title": "Projects", "target": map[string]any{"type": "record", "ref": "RC-4"}},
			map[string]any{"key": "page", "title": "Project", "page_of": "RC-4", "entries": []any{
				map[string]any{"key": "days", "title": "Days", "target": map[string]any{"type": "record", "ref": "RC-6", "where": []any{
					map[string]any{"question": "Project", "operator": "is", "value": map[string]any{"row": "Project title"}},
				}}},
			}},
		}},
	}
	result, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_set_home",
		Arguments: map[string]any{"instance_id": "any", "draft_id": draft.ID, "entries": entries},
	})
	if err != nil || result.IsError {
		t.Fatalf("agency_set_home: %v %+v", err, textOf(result))
	}
	rows, err := mgr.ListHomeEntries(ctxT(t), draft.ID)
	if err != nil || len(rows) != 4 {
		t.Fatalf("rows = %+v, %v, want the four nested rows", rows, err)
	}
}
