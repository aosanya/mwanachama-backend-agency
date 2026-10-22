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

func TestRolePowers_PublishAndForkCarryThem(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, role := seedPublishableAgency(t, mgr, ctx)
	role.Powers = []string{"report.publish", "join_request.decide"}
	if _, err := mgr.UpdateRole(ctx, role); err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	reread, err := mgr.GetRole(ctx, role.ID)
	if err != nil {
		t.Fatalf("GetRole: %v", err)
	}
	if !reflect.DeepEqual(reread.Powers, role.Powers) {
		t.Fatalf("stored powers = %v, want %v", reread.Powers, role.Powers)
	}

	pub, err := mgr.PublishAgency(ctx, o.DraftID, "powers-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if len(pub.Document.Roles) != 1 || !reflect.DeepEqual(pub.Document.Roles[0].Powers, role.Powers) {
		t.Fatalf("document roles = %+v, want powers %v", pub.Document.Roles, role.Powers)
	}

	forked, err := mgr.CreateDraft(ctx, "revisit powers-1", "powers-1", "publication")
	if err != nil {
		t.Fatalf("CreateDraft from publication: %v", err)
	}
	roles, err := mgr.ListRoles(ctx, forked.ID, mwanachamaagency.RoleFilter{})
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	if len(roles) != 1 || !reflect.DeepEqual(roles[0].Powers, role.Powers) {
		t.Fatalf("forked roles = %+v, want powers %v", roles, role.Powers)
	}
}

func TestRolePowers_PublishRefusesUnknownAndLockedPowers(t *testing.T) {
	for _, tc := range []struct {
		power string
		want  string
	}{
		{"teleport.anyone", "unknown power"},
		{"agency.delete", "locked to owners"},
	} {
		t.Run(tc.power, func(t *testing.T) {
			mgr := newTestManager(t)
			ctx := context.Background()
			_, o, role := seedPublishableAgency(t, mgr, ctx)
			role.Powers = []string{"member.invite", tc.power}
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

func TestPowerCatalog_LocksTheOwnerPowers(t *testing.T) {
	locked := map[string]bool{}
	for _, p := range models.PowerCatalog() {
		if p.Locked {
			locked[p.Code] = true
		}
	}
	for _, code := range []string{"agent.manage", "api_key.manage", "agency.go_live", "agency.delete", "power.override"} {
		if !locked[code] {
			t.Errorf("power %q is not locked to owners", code)
		}
	}
}
