package mwanachamaagency_test

import (
	"context"
	"reflect"
	"testing"
)

func TestARoleMayDeclareTheRulePermissionsAndStillPublish(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, role := seedPublishableAgency(t, mgr, ctx)
	role.Permissions = []string{"rule.read", "rule.edit"}
	if _, err := mgr.UpdateRole(ctx, role); err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	pub, err := mgr.PublishAgency(ctx, o.DraftID, "rules-1", "")
	if err != nil {
		t.Fatalf("PublishAgency refused a role holding the rule permissions: %v", err)
	}
	if len(pub.Document.Roles) != 1 || !reflect.DeepEqual(pub.Document.Roles[0].Permissions, role.Permissions) {
		t.Fatalf("document roles = %+v, want permissions %v", pub.Document.Roles, role.Permissions)
	}
}

func TestThePermissionListServesTheRulePermissions(t *testing.T) {
	mgr := newTestManager(t)
	permissions, err := mgr.ListPermissions(context.Background())
	if err != nil {
		t.Fatalf("ListPermissions: %v", err)
	}
	have := map[string]bool{}
	for _, p := range permissions {
		have[p.Code] = true
	}
	for _, code := range []string{"rule.read", "rule.edit"} {
		if !have[code] {
			t.Errorf("GET /permissions does not list %s", code)
		}
	}
}
