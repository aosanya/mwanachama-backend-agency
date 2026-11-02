package mwanachamaagency_test

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCataloguePermissionChatReachCarriesOnlyTheReachOfOnePerson(t *testing.T) {
	p := permissionNamed(t, "chat.reach")
	if len(p.Actions) != 1 || !carries(p, "comm.person_reach.list") {
		t.Fatalf("chat.reach actions = %v, want only comm.person_reach.list", p.Actions)
	}
	if p.Module != "comm" {
		t.Errorf("chat.reach module = %q, want comm", p.Module)
	}
}

func TestNoOtherPermissionCarriesTheReachOfOnePerson(t *testing.T) {
	for _, p := range models.PermissionCatalog() {
		if p.Code != "chat.reach" && carries(p, "comm.person_reach.list") {
			t.Errorf("%s carries comm.person_reach.list, which only chat.reach may", p.Code)
		}
	}
	if held := heldByAnUnlockedPermission()["comm.person_reach.list"]; held != "chat.reach" {
		t.Errorf("comm.person_reach.list is held by %q, want chat.reach", held)
	}
}
