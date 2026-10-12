package mcp_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestAToolGivenNullForEveryRequiredArgumentBesidesItsInstanceRefuses(t *testing.T) {
	server := newTestServer(newTestManager(t))
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cs := mcpConnect(t, srv.URL)

	created, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_create_agency",
		Arguments: map[string]any{"instance_id": "any", "name": "Null probe"},
	})
	if err != nil || created.IsError {
		t.Fatalf("precondition: agency_create_agency = %+v, %v", created, err)
	}

	listed, err := cs.ListTools(ctxT(t), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	probed := 0
	for _, tool := range listed.Tools {
		var shape struct {
			Required []string `json:"required"`
		}
		raw, _ := json.Marshal(tool.InputSchema)
		if err := json.Unmarshal(raw, &shape); err != nil {
			t.Fatalf("%s schema: %v", tool.Name, err)
		}
		args := map[string]any{"instance_id": "any"}
		for _, key := range shape.Required {
			if key != "instance_id" {
				args[key] = nil
			}
		}
		if len(args) == 1 {
			continue
		}
		probed++
		res, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{Name: tool.Name, Arguments: args})
		if err != nil {
			continue
		}
		if !res.IsError {
			sc, _ := json.Marshal(res.StructuredContent)
			t.Errorf("%s with every required argument null answered %s; it should refuse", tool.Name, sc)
		}
	}
	if probed < 40 {
		t.Fatalf("precondition: only %d tools declare a required argument besides instance_id", probed)
	}
}
