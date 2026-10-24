package mwanachamaagency_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestListPowers_OlderPowersListTheActionsDeclaredToday(t *testing.T) {
	mgr := newTestManager(t)
	powers, err := mgr.ListPowers(context.Background())
	if err != nil {
		t.Fatalf("ListPowers: %v", err)
	}
	byCode := map[string]models.Power{}
	for _, p := range powers {
		byCode[p.Code] = p
	}
	want := map[string][]string{
		"member.invite":          {"actor.invitation.create", "actor.invitation.check_batch", "actor.invitation.send_batch"},
		"join_request.decide":    {"actor.join_request.approve", "actor.join_request.decline"},
		"form.approve":           {"forms.approval.approve", "forms.approval.approve_slot", "forms.approval.refuse"},
		"report.publish":         {"forms.results.publish"},
		"poll.run":               {"voting.poll.create", "voting.choice.create"},
		"notice.post":            {"engage.post.create"},
		"event.post":             {"engage.post.create"},
		"volunteer_call.post":    {"engage.post.create"},
		"board_item.hide":        {"catalog.board_item.set_hidden", "catalog.raised_item.remove"},
		"service.publish":        {"catalog.entry.set_visibility"},
		"promise.record_outcome": {},
	}
	for code, actions := range want {
		p, ok := byCode[code]
		if !ok {
			t.Fatalf("power %s missing from the catalog", code)
		}
		if !reflect.DeepEqual(p.Actions, actions) {
			t.Errorf("%s actions = %v, want %v", code, p.Actions, actions)
		}
	}
	if p := byCode["service.publish"]; p.Name != "Publish services" {
		t.Errorf("service.publish name = %q, want Publish services", p.Name)
	}
	if _, ok := byCode["catalog_entry.publish"]; ok {
		t.Errorf("catalog_entry.publish is still in the catalog")
	}
}
