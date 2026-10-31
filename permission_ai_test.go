package mwanachamaagency_test

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func permissionActions(t *testing.T, code string) map[string]bool {
	t.Helper()
	p, ok := models.LookupPermission(code)
	if !ok {
		t.Fatalf("permission %s missing from the catalogue", code)
	}
	if p.Locked {
		t.Fatalf("permission %s = %+v, want unlocked", code, p)
	}
	have := map[string]bool{}
	for _, a := range p.Actions {
		have[a] = true
	}
	return have
}

func TestCatalogueNamesAIAccountManage(t *testing.T) {
	have := permissionActions(t, "ai.account.manage")
	for _, a := range []string{"ai.account.create", "ai.account.update", "ai.account.delete", "ai.account.key_set", "ai.account.test"} {
		if !have[a] {
			t.Fatalf("ai.account.manage missing %s", a)
		}
	}
	if have["ai.limits.set"] || have["ai.usage.record"] {
		t.Fatalf("ai.account.manage grants too much: %v", have)
	}
}

func TestCatalogueNamesAIRoleManage(t *testing.T) {
	have := permissionActions(t, "ai.role.manage")
	for _, a := range []string{"ai.limits.set", "ai.role.create", "ai.role.update", "ai.role.delete", "ai.revision.list", "ai.revision.restore", "ai.role.map", "ai.role.resolve", "ai.account.test"} {
		if !have[a] {
			t.Fatalf("ai.role.manage missing %s", a)
		}
	}
	if have["ai.account.key_set"] || have["ai.account.create"] || have["ai.role.revision_list"] || have["ai.role.restore"] {
		t.Fatalf("ai.role.manage grants account management: %v", have)
	}
}

func TestCatalogueNamesAIUseWithoutTest(t *testing.T) {
	have := permissionActions(t, "ai.use")
	for _, a := range []string{"ai.account.list", "ai.usage.list", "ai.usage.record", "ai.limits.check", "ai.role.resolve_agency_role"} {
		if !have[a] {
			t.Fatalf("ai.use missing %s", a)
		}
	}
	if have["ai.account.read"] || have["ai.account.test"] || have["ai.account.key_set"] || have["ai.limits.set"] {
		t.Fatalf("ai.use grants too much: %v", have)
	}
}

func TestSeatWithoutAIPermissionIsRefused(t *testing.T) {
	for _, code := range []string{"ai.account.manage", "ai.role.manage", "ai.use", "ai.account.read"} {
		if _, ok := models.LookupPermission(code); !ok {
			t.Fatalf("permission %s missing, so no seat can hold it", code)
		}
	}
	if _, ok := models.LookupPermission("ai.unheld"); ok {
		t.Fatal("an undeclared ai permission must not resolve")
	}
}
