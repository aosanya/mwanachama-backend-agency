package mwanachamaagency_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func permissionByCode(t *testing.T, code string) models.Permission {
	t.Helper()
	mgr := newTestManager(t)
	permissions, err := mgr.ListPermissions(context.Background())
	if err != nil {
		t.Fatalf("ListPermissions: %v", err)
	}
	for _, p := range permissions {
		if p.Code == code {
			return p
		}
	}
	t.Fatalf("permission %q is not in the catalogue", code)
	return models.Permission{}
}

func TestListPermissions_EditControlsListsTheControlWriteActions(t *testing.T) {
	want := models.Permission{
		Code:   "controls.edit",
		Name:   "Edit controls",
		Module: "agency",
		Actions: []string{
			"agency.control.create",
			"agency.control.update",
			"agency.control.delete",
			"agency.control.attach",
			"agency.control.detach",
			"agency.hazard.create",
			"agency.hazard.update",
			"agency.hazard.delete",
			"agency.threat.create",
			"agency.threat.update",
			"agency.threat.delete",
			"agency.consequence.create",
			"agency.consequence.update",
			"agency.consequence.delete",
			"agency.risk_threat.create",
			"agency.risk_threat.delete",
			"agency.risk_consequence.create",
			"agency.risk_consequence.delete",
			"agency.control.list",
			"agency.control.read",
			"agency.control.list_attachments",
			"agency.hazard.list",
			"agency.hazard.read",
			"agency.threat.list",
			"agency.threat.read",
			"agency.consequence.list",
			"agency.consequence.read",
			"agency.risk_threat.list",
			"agency.risk_consequence.list",
		},
	}
	if got := permissionByCode(t, "controls.edit"); !reflect.DeepEqual(got, want) {
		t.Fatalf("controls.edit = %+v, want %+v", got, want)
	}
}

func TestListPermissions_ApproveControlChangesIsThePublishAction(t *testing.T) {
	want := models.Permission{
		Code:    "controls.approve",
		Name:    "Approve control changes",
		Module:  "agency",
		Actions: []string{"agency.publication.create"},
	}
	if got := permissionByCode(t, "controls.approve"); !reflect.DeepEqual(got, want) {
		t.Fatalf("controls.approve = %+v, want %+v", got, want)
	}
}

func TestControlPermissionsAreNotLockedToOwners(t *testing.T) {
	for _, code := range []string{"controls.edit", "controls.approve"} {
		if permissionByCode(t, code).Locked {
			t.Fatalf("%s is locked, so no role could hold it", code)
		}
	}
}
