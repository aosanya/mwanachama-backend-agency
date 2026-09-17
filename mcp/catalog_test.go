package mcp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestListTypes_NoNameListsEveryType(t *testing.T) {
	server := newTestServer(newTestManager(t))
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cs := mcpConnect(t, srv.URL)
	result, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{Name: "agency_list_types", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("CallTool agency_list_types: %v", err)
	}
	if result.IsError {
		t.Fatalf("agency_list_types: IsError = true (content: %+v)", result.Content)
	}
	sc, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structuredContent is %T, want map[string]any", result.StructuredContent)
	}
	items, ok := sc["items"].([]any)
	if !ok {
		t.Fatalf("structuredContent.items is %T, want []any", sc["items"])
	}
	if len(items) < 14 {
		t.Errorf("got %d types, want the full catalog (at least 14)", len(items))
	}
}

func TestListTypes_NameFiltersToOneType(t *testing.T) {
	server := newTestServer(newTestManager(t))
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cs := mcpConnect(t, srv.URL)
	result, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_list_types",
		Arguments: map[string]any{"name": "objective"},
	})
	if err != nil {
		t.Fatalf("CallTool agency_list_types: %v", err)
	}
	if result.IsError {
		t.Fatalf("agency_list_types: IsError = true (content: %+v)", result.Content)
	}
	sc := result.StructuredContent.(map[string]any)
	items := sc["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("got %d types matching %q, want exactly 1", len(items), "objective")
	}
	entry := items[0].(map[string]any)
	if entry["name"] != "Objective" {
		t.Errorf("matched type = %q, want %q", entry["name"], "Objective")
	}
	if entry["what_it_is"] == "" || entry["how_to_add"] == "" {
		t.Errorf("Objective entry missing what_it_is/how_to_add: %+v", entry)
	}
}

func TestListTypes_UnknownNameReturnsEmpty(t *testing.T) {
	server := newTestServer(newTestManager(t))
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cs := mcpConnect(t, srv.URL)
	result, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_list_types",
		Arguments: map[string]any{"name": "nonexistent-type"},
	})
	if err != nil {
		t.Fatalf("CallTool agency_list_types: %v", err)
	}
	if result.IsError {
		t.Fatalf("agency_list_types: IsError = true (content: %+v)", result.Content)
	}
	sc := result.StructuredContent.(map[string]any)
	if items, ok := sc["items"].([]any); ok && len(items) != 0 {
		t.Errorf("got %d types for an unknown name, want 0", len(items))
	}
}
