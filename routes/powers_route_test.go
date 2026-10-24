package routes

import (
	"encoding/json"
	"os"
	"testing"
)

func TestThePowerListIsDeclaredAndServed(t *testing.T) {
	raw, err := os.ReadFile("../agency.operations.json")
	if err != nil {
		t.Fatalf("read agency.operations.json: %v", err)
	}
	var whole any
	if err := json.Unmarshal(raw, &whole); err != nil {
		t.Errorf("AG90: agency.operations.json is not one JSON document (%v); the operations after its closing brace are dropped by a first-value decoder", err)
	}
	for _, rt := range Shape() {
		if rt.Action == "agency.power.list" {
			if rt.Method != "GET" || rt.Path != "/powers" {
				t.Fatalf("agency.power.list is %s %s; want GET /powers", rt.Method, rt.Path)
			}
			return
		}
	}
	t.Fatalf("AG90: Shape() carries no agency.power.list, so GET /powers is served by no mount and agency_list_powers by no MCP server (AG88, member app decision 410)")
}
