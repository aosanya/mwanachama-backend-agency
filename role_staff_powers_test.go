package mwanachamaagency_test

import (
	"context"
	"reflect"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestRole_DuplicatePowerTickDroppedAndStaffPublished(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, role := seedPublishableAgency(t, mgr, ctx)
	role.Powers = []string{"form.send", "member.invite", "form.send"}
	role.Staff = true
	if _, err := mgr.UpdateRole(ctx, role); err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	got, err := mgr.GetRole(ctx, role.ID)
	if err != nil {
		t.Fatalf("GetRole: %v", err)
	}
	if want := []string{"form.send", "member.invite"}; !reflect.DeepEqual(got.Powers, want) || !got.Staff {
		t.Fatalf("role = %+v, want powers %v and staff", got, want)
	}
	pub, err := mgr.PublishAgency(ctx, o.DraftID, "staff-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if len(pub.Document.Roles) != 1 || !pub.Document.Roles[0].Staff {
		t.Fatalf("document roles = %+v, want staff", pub.Document.Roles)
	}
	forked, err := mgr.CreateDraft(ctx, "revisit", "staff-1", "publication")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	roles, err := mgr.ListRoles(ctx, forked.ID, mwanachamaagency.RoleFilter{})
	if err != nil || len(roles) != 1 || !roles[0].Staff {
		t.Fatalf("forked roles = %+v, %v; want staff", roles, err)
	}
}

func TestListPowers_ServesTheCatalogWithActions(t *testing.T) {
	mgr := newTestManager(t)
	powers, err := mgr.ListPowers(context.Background())
	if err != nil {
		t.Fatalf("ListPowers: %v", err)
	}
	byCode := map[string]models.Power{}
	for _, p := range powers {
		if p.Actions == nil {
			t.Fatalf("power %s has nil actions", p.Code)
		}
		byCode[p.Code] = p
	}
	if got := byCode["form.send"].Actions; !reflect.DeepEqual(got, []string{"forms.form.publish"}) {
		t.Fatalf("form.send actions = %v", got)
	}
	if byCode["poll.run"].Module != "voting" {
		t.Fatalf("poll.run module = %q, want voting", byCode["poll.run"].Module)
	}
}
