package mwanachamaagency_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestRolePermissions_PublishAndForkCarryThem(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, role := seedPublishableAgency(t, mgr, ctx)
	role.Permissions = []string{"report.publish", "join_request.decide"}
	if _, err := mgr.UpdateRole(ctx, role); err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	reread, err := mgr.GetRole(ctx, role.ID)
	if err != nil {
		t.Fatalf("GetRole: %v", err)
	}
	if !reflect.DeepEqual(reread.Permissions, role.Permissions) {
		t.Fatalf("stored permissions = %v, want %v", reread.Permissions, role.Permissions)
	}

	pub, err := mgr.PublishAgency(ctx, o.DraftID, "permissions-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if len(pub.Document.Roles) != 1 || !reflect.DeepEqual(pub.Document.Roles[0].Permissions, role.Permissions) {
		t.Fatalf("document roles = %+v, want permissions %v", pub.Document.Roles, role.Permissions)
	}

	forked, err := mgr.CreateDraft(ctx, "revisit permissions-1", "permissions-1", "publication")
	if err != nil {
		t.Fatalf("CreateDraft from publication: %v", err)
	}
	roles, err := mgr.ListRoles(ctx, forked.ID, mwanachamaagency.RoleFilter{})
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	if len(roles) != 1 || !reflect.DeepEqual(roles[0].Permissions, role.Permissions) {
		t.Fatalf("forked roles = %+v, want permissions %v", roles, role.Permissions)
	}
}

func TestRolePermissions_PublishRefusesUnknownAndLockedPermissions(t *testing.T) {
	for _, tc := range []struct {
		permission string
		want       string
	}{
		{"teleport.anyone", "unknown permission"},
		{"agency.delete", "locked to owners"},
	} {
		t.Run(tc.permission, func(t *testing.T) {
			mgr := newTestManager(t)
			ctx := context.Background()
			_, o, role := seedPublishableAgency(t, mgr, ctx)
			role.Permissions = []string{"member.invite", tc.permission}
			if _, err := mgr.UpdateRole(ctx, role); err != nil {
				t.Fatalf("UpdateRole: %v", err)
			}
			_, err := mgr.PublishAgency(ctx, o.DraftID, "", "")
			if !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("PublishAgency = %v, want ErrPublishValidationFailed naming %q", err, tc.want)
			}
		})
	}
}

func TestPermissionCatalog_LocksTheOwnerPermissions(t *testing.T) {
	locked := map[string]bool{}
	for _, p := range models.PermissionCatalog() {
		if p.Locked {
			locked[p.Code] = true
		}
	}
	for _, code := range []string{"agent.manage", "api_key.manage", "agency.go_live", "agency.delete", "permission.override"} {
		if !locked[code] {
			t.Errorf("permission %q is not locked to owners", code)
		}
	}
}
