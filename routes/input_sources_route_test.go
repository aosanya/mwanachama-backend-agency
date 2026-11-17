package routes

import "testing"

func TestTheInputSourceListIsDeclaredAndServed(t *testing.T) {
	for _, rt := range Shape() {
		if rt.Action == "agency.input_source.list" {
			if rt.Method != "GET" || rt.Path != "/input-sources" {
				t.Fatalf("agency.input_source.list is %s %s; want GET /input-sources", rt.Method, rt.Path)
			}
			return
		}
	}
	t.Fatal("Shape() carries no agency.input_source.list, so GET /input-sources and agency_list_input_sources are served by no mount")
}
