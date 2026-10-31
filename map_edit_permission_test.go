package mwanachamaagency_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestListPermissions_EditTheMapListsTheTwinsWriteActions(t *testing.T) {
	mgr := newTestManager(t)
	permissions, err := mgr.ListPermissions(context.Background())
	if err != nil {
		t.Fatalf("ListPermissions: %v", err)
	}
	var got models.Permission
	for _, p := range permissions {
		if p.Code == "map.edit" {
			got = p
		}
	}
	want := models.Permission{
		Code:   "map.edit",
		Name:   "Edit the map",
		Module: "digitaltwin",
		Actions: []string{
			"digitaltwin.node.create",
			"digitaltwin.node.update",
			"digitaltwin.node.set_status",
			"digitaltwin.link.create",
			"digitaltwin.link.delete",
			"digitaltwin.metric.upsert",
			"digitaltwin.observation.record",
			"digitaltwin.observation.record_batch",
			"digitaltwin.retention.set",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("map.edit = %+v, want %+v", got, want)
	}
}
