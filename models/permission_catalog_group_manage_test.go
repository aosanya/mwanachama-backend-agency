package models_test

import (
	"slices"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

var groupAdministrationActions = []string{
	"instance.group_tag.give",
	"instance.group_tag.take",
	"instance.group.disable",
	"instance.group.enable",
}

func TestCatalogCarriesGroupAdministrationInGroupManage(t *testing.T) {
	requireCatalogRows(t, []catalogRow{
		{"group.manage", "", groupAdministrationActions},
		{"group.manage", "", []string{"actor.group.create", "actor.group.update", "actor.group.move", "actor.group.set_area_node", "actor.group.set_boundary", "actor.assignment.create"}},
	})
}

func TestGroupManageIsGivableAndOnlyItCarriesGroupTaggingAndDisabling(t *testing.T) {
	for _, p := range models.PermissionCatalog() {
		holds := false
		for _, action := range groupAdministrationActions {
			if slices.Contains(p.Actions, action) {
				holds = true
			}
		}
		if holds && p.Code != "group.manage" {
			t.Errorf("%s carries a group administration action, which stays with group.manage", p.Code)
		}
	}
	p, ok := models.LookupPermission("group.manage")
	if !ok {
		t.Fatal("catalogue lacks group.manage")
	}
	if p.Locked {
		t.Error("group.manage is locked, want it givable to a role")
	}
}

func TestTagManageAndGroupSettingsDoNotCarryGroupAdministration(t *testing.T) {
	for _, code := range []string{"tag.manage", "group.settings", "group.view", "group.read"} {
		p, ok := models.LookupPermission(code)
		if !ok {
			t.Fatalf("catalogue lacks %s", code)
		}
		for _, action := range groupAdministrationActions {
			if slices.Contains(p.Actions, action) {
				t.Errorf("%s carries %s", code, action)
			}
		}
	}
}
