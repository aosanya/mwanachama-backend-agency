package models_test

import (
	"slices"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCatalogCarriesTheTagManagementPermission(t *testing.T) {
	requireCatalogRows(t, []catalogRow{
		{"tag.manage", "", []string{"instance.tag.create", "instance.tag.rename", "instance.tag.delete"}},
	})
}

func TestTagManagementPermissionIsGivableAndDoesNotCarryGroupTagging(t *testing.T) {
	p, ok := models.LookupPermission("tag.manage")
	if !ok {
		t.Fatal("catalogue lacks tag.manage")
	}
	if p.Locked {
		t.Error("tag.manage is locked, want it givable to a role")
	}
	for _, action := range []string{"instance.group_tag.give", "instance.group_tag.take", "instance.tag.list"} {
		if slices.Contains(p.Actions, action) {
			t.Errorf("tag.manage carries %s, which stays with group administration", action)
		}
	}
}
