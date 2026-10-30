package routes

import "testing"

func TestTheDefaultWorkflowListIsDeclaredAndServed(t *testing.T) {
	for _, rt := range Shape() {
		if rt.Action == "agency.workflow.list_defaults" {
			if rt.Method != "GET" || rt.Path != "/workflows/defaults" {
				t.Fatalf("agency.workflow.list_defaults is %s %s; want GET /workflows/defaults", rt.Method, rt.Path)
			}
			return
		}
	}
	t.Fatal("Shape() carries no agency.workflow.list_defaults, so GET /workflows/defaults and agency_list_default_workflows are served by no mount")
}
