package mcp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestConsequenceTools_SchemasNameTheHazard(t *testing.T) {
	server := newTestServer(newTestManager(t))
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cs := mcpConnect(t, srv.URL)

	listed, err := cs.ListTools(ctxT(t), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	schemas := map[string]map[string]any{}
	for _, tool := range listed.Tools {
		if s, ok := tool.InputSchema.(map[string]any); ok {
			schemas[tool.Name] = s
		}
	}
	for _, name := range []string{"agency_create_consequence", "agency_list_consequences", "agency_update_consequence"} {
		props, _ := schemas[name]["properties"].(map[string]any)
		if _, ok := props["hazard_code"]; !ok {
			t.Errorf("%s has no hazard_code property: %v", name, props)
		}
	}
	required, _ := schemas["agency_create_consequence"]["required"].([]any)
	found := false
	for _, r := range required {
		if r == "hazard_code" {
			found = true
		}
	}
	if !found {
		t.Errorf("agency_create_consequence required = %v, want hazard_code in it", required)
	}
}
