package mwanachamaagency_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

var engagePostingPermissions = []string{"notice.post", "event.post", "volunteer_call.post"}

func TestEngageDefinition_CatalogDeclaresTheModuleSwitchedOff(t *testing.T) {
	for _, entry := range models.ModuleCatalog {
		if entry.Code == "engage" {
			if entry.Enabled || entry.Locked {
				t.Fatalf("engage catalog entry = %+v, want switched off and unlocked", entry)
			}
			return
		}
	}
	t.Fatal("engage is not in the module catalog")
}

func TestEngageDefinition_OnePostingPermissionPerKind(t *testing.T) {
	for _, code := range engagePostingPermissions {
		p, ok := models.LookupPermission(code)
		if !ok {
			t.Fatalf("permission %q is not in the catalog", code)
		}
		if p.Locked || p.Module != "engage" {
			t.Fatalf("permission %q = %+v, want unlocked and owned by engage", code, p)
		}
	}
}

func TestEngageDefinition_PostingPermissionsPublishWithTheRole(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, role := seedPublishableAgency(t, mgr, ctx)
	role.Permissions = append([]string(nil), engagePostingPermissions...)
	if _, err := mgr.UpdateRole(ctx, role); err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	pub, err := mgr.PublishAgency(ctx, o.DraftID, "engage-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if len(pub.Document.Roles) != 1 || !reflect.DeepEqual(pub.Document.Roles[0].Permissions, engagePostingPermissions) {
		t.Fatalf("document roles = %+v, want permissions %v", pub.Document.Roles, engagePostingPermissions)
	}
}
