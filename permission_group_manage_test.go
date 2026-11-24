package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

var groupAdministrationCodes = []string{
	"instance.group_tag.give",
	"instance.group_tag.take",
	"instance.group.disable",
	"instance.group.enable",
}

func requireHeldCodesAllowGroupAdministration(t *testing.T, held []string) {
	t.Helper()
	if !slices.Contains(held, "group.manage") {
		t.Fatalf("role holds %v, want group.manage kept", held)
	}
	var allowed []string
	for _, code := range held {
		p, ok := models.LookupPermission(code)
		if !ok {
			t.Fatalf("held code %s is not in the catalogue", code)
		}
		allowed = append(allowed, p.Actions...)
	}
	for _, action := range groupAdministrationCodes {
		if !slices.Contains(allowed, action) {
			t.Errorf("a role holding %v does not allow %s", held, action)
		}
	}
}

func TestAGroupAdministratorRoleIsAcceptedOnCreate(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "group administrator", Permissions: []string{"group.manage"}})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	reread, err := mgr.GetRole(ctx, role.ID)
	if err != nil {
		t.Fatalf("GetRole: %v", err)
	}
	requireHeldCodesAllowGroupAdministration(t, reread.Permissions)
}

func TestAGroupAdministratorRoleIsAcceptedOnUpdateAndSet(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, role := seedPublishableAgency(t, mgr, ctx)
	role.Permissions = []string{"group.manage", "tag.manage"}
	updated, err := mgr.UpdateRole(ctx, role)
	if err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	requireHeldCodesAllowGroupAdministration(t, updated.Permissions)

	set, err := mgr.SetRolePermissions(ctx, role.ID, []string{"group.manage"})
	if err != nil {
		t.Fatalf("SetRolePermissions: %v", err)
	}
	requireHeldCodesAllowGroupAdministration(t, set.Permissions)
}

func TestAGroupAdministratorRoleIsAcceptedOnImport(t *testing.T) {
	var doc models.AgencyDocument
	raw := `{"agency": {"name": "Ward office"}, "roles": [{"id": "r1", "code": "R-1", "name": "Group administrator", "permissions": ["group.manage"]}]}`
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	mgr := newTestManager(t)
	ctx := context.Background()
	out, err := mgr.ImportDocument(ctx, "ward", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	roles := listRoles(t, mgr, ctx, out.Draft.ID)
	if len(roles) != 1 {
		t.Fatalf("imported roles = %+v, want one", roles)
	}
	requireHeldCodesAllowGroupAdministration(t, roles[0].Permissions)
}

func TestAGroupAdministratorRoleIsAcceptedOnPublish(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, role := seedPublishableAgency(t, mgr, ctx)
	role.Permissions = []string{"group.manage"}
	if _, err := mgr.UpdateRole(ctx, role); err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	pub, err := mgr.PublishAgency(ctx, o.DraftID, "group-administration-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if len(pub.Document.Roles) != 1 {
		t.Fatalf("document roles = %+v, want one", pub.Document.Roles)
	}
	requireHeldCodesAllowGroupAdministration(t, pub.Document.Roles[0].Permissions)
}

func TestAnUnknownGroupAdministrationCodeIsStillRefused(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, role := seedPublishableAgency(t, mgr, ctx)
	for _, code := range []string{"group.administer", "instance.group.disable"} {
		_, err := mgr.SetRolePermissions(ctx, role.ID, []string{"group.manage", code})
		if !errors.Is(err, mwanachamaagency.ErrUnknownPermission) {
			t.Errorf("SetRolePermissions with %s = %v, want ErrUnknownPermission", code, err)
		}
	}
}
