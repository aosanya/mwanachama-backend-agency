package mwanachamaagency_test

import (
	"context"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

func TestTheLabsResidentRolesHoldMountListAndTheAgentDoesNot(t *testing.T) {
	doc := readDocument(t, "testdata/hub-demo/siaya-lab/agency.json")
	mgr := newTestManager(t)
	ctx := context.Background()
	out, err := mgr.ImportDocument(ctx, "lab", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	roles, err := mgr.ListRoles(ctx, out.Draft.ID, mwanachamaagency.RoleFilter{})
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	holds := map[string]bool{}
	for _, r := range roles {
		for _, p := range r.Permissions {
			if p == "mount.list" {
				holds[r.Name] = true
			}
		}
	}
	for _, name := range []string{"Lab manager", "Member"} {
		if !holds[name] {
			t.Errorf("%s imported without mount.list", name)
		}
	}
	if holds["Hub Steward"] {
		t.Error("Hub Steward must not list the agency's modules")
	}
	if _, err := mgr.PublishAgency(ctx, out.Draft.ID, "lab-1", "production"); err != nil {
		t.Fatalf("PublishAgency refused the lab template: %v", err)
	}
}

func TestMountListIsAnUnlockedPermissionCarryingOnlyTheMountList(t *testing.T) {
	have := permissionActions(t, "mount.list")
	if len(have) != 1 || !have["instance.mount.list"] {
		t.Fatalf("mount.list carries %v, want only instance.mount.list", have)
	}
}

func TestTheRoleLimitsAndChatModelActionsSitInTheirPermissions(t *testing.T) {
	for permission, actions := range map[string][]string{
		"ai.role.manage":    {"ai.role.set_limits"},
		"ai.use":            {"ai.chat_model.list", "ai.chat_model.pick"},
		"ai.account.manage": {"ai.account.default_set"},
		"ai.account.read":   {"ai.account.default_read"},
	} {
		have := permissionActions(t, permission)
		for _, a := range actions {
			if !have[a] {
				t.Errorf("%s missing %s", permission, a)
			}
		}
	}
	if permissionActions(t, "ai.use")["ai.role.set_limits"] {
		t.Error("ai.use must not set role limits")
	}
}
