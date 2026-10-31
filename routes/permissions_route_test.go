package routes

import (
	"encoding/json"
	"os"
	"testing"
)

func TestThePermissionListIsDeclaredAndServed(t *testing.T) {
	raw, err := os.ReadFile("../agency.operations.json")
	if err != nil {
		t.Fatalf("read agency.operations.json: %v", err)
	}
	var whole any
	if err := json.Unmarshal(raw, &whole); err != nil {
		t.Errorf("AG90: agency.operations.json is not one JSON document (%v); the operations after its closing brace are dropped by a first-value decoder", err)
	}
	for _, rt := range Shape() {
		if rt.Action == "agency.permission.list" {
			if rt.Method != "GET" || rt.Path != "/permissions" {
				t.Fatalf("agency.permission.list is %s %s; want GET /permissions", rt.Method, rt.Path)
			}
			return
		}
	}
	t.Fatalf("AG90: Shape() carries no agency.permission.list, so GET /permissions is served by no mount and agency_list_permissions by no MCP server (AG88, member app decision 410)")
}
