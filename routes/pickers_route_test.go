package routes

import "testing"

func TestThePickerListIsDeclaredAndServed(t *testing.T) {
	for _, rt := range Shape() {
		if rt.Action == "agency.picker.list" {
			if rt.Method != "GET" || rt.Path != "/pickers" {
				t.Fatalf("agency.picker.list is %s %s; want GET /pickers", rt.Method, rt.Path)
			}
			return
		}
	}
	t.Fatal("Shape() carries no agency.picker.list, so GET /pickers and agency_list_pickers are served by no mount")
}
