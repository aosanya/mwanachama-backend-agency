package mwanachamaagency_test

import (
	"context"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestRoleBrief_ControlsReadAsOneList(t *testing.T) {
	ctx := context.Background()
	f := newBriefFixture(t)
	pub, err := f.mgr.GetPublicationForChannel(ctx, "")
	must(t, err)
	d, err := f.mgr.CreateDraft(ctx, "controls", pub.Tag, "publication")
	must(t, err)

	seat, other := roleByCode(t, f, d.ID, f.seat.Code), roleByCode(t, f, d.ID, f.other.Code)
	forSeat, err := f.mgr.CreateControl(ctx, models.Control{DraftID: d.ID, Text: "CONTROL-SEAT"})
	must(t, err)
	_, err = f.mgr.CreateControlAttachment(ctx, forSeat.ID, models.TargetTypeRole, seat.ID)
	must(t, err)
	forOther, err := f.mgr.CreateControl(ctx, models.Control{DraftID: d.ID, Text: "CONTROL-OTHER-SEAT"})
	must(t, err)
	_, err = f.mgr.CreateControlAttachment(ctx, forOther.ID, models.TargetTypeRole, other.ID)
	must(t, err)
	_, err = f.mgr.CreateControl(ctx, models.Control{DraftID: d.ID, Text: "CONTROL-ORG"})
	must(t, err)
	_, err = f.mgr.PublishAgency(ctx, d.ID, "", "")
	must(t, err)

	b, err := f.mgr.GetRoleBrief(ctx, f.seat.Code)
	must(t, err)
	for _, want := range []string{"Controls:", "CONTROL-SEAT", "CONTROL-ORG", "RULE-SEAT"} {
		if !strings.Contains(b.Text, want) {
			t.Errorf("brief text lacks %q:\n%s", want, b.Text)
		}
	}
	for _, unwanted := range []string{"CONTROL-OTHER-SEAT", "Values:", "Rules:"} {
		if strings.Contains(b.Text, unwanted) {
			t.Errorf("brief text carries %q:\n%s", unwanted, b.Text)
		}
	}
}

func roleByCode(t *testing.T, f briefFixture, draftID, code string) models.Role {
	t.Helper()
	roles, err := f.mgr.ListRoles(context.Background(), draftID, mwanachamaagency.RoleFilter{})
	must(t, err)
	for _, r := range roles {
		if r.Code == code {
			return r
		}
	}
	t.Fatalf("role %s missing in draft", code)
	return models.Role{}
}
