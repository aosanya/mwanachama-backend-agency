package mwanachamaagency_test

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func heldByAnUnlockedPermission() map[string]string {
	out := map[string]string{}
	for _, p := range models.PermissionCatalog() {
		if p.Locked {
			continue
		}
		for _, a := range p.Actions {
			out[a] = p.Code
		}
	}
	return out
}

func TestEveryActionOfTheCataloguePermissionGapBelongsToAnUnlockedPermission(t *testing.T) {
	want := map[string]string{
		"ai.account.chat":                      "ai.use",
		"ai.model.list":                        "ai.use",
		"ai.model.refresh":                     "ai.account.manage",
		"ai.chat_model.list":                   "ai.use",
		"ai.chat_model.pick":                   "ai.use",
		"ai.role.set_limits":                   "ai.role.manage",
		"ai.account.default_set":               "ai.account.manage",
		"ai.account.default_read":              "ai.account.read",
		"instance.mount.list":                  "mount.list",
		"actor.role_kind.create":               "role_kind.manage",
		"actor.role_kind.list":                 "role_kind.manage",
		"actor.role_kind.read":                 "role_kind.manage",
		"actor.role_kind.retire":               "role_kind.manage",
		"actor.role_kind.unretire":             "role_kind.manage",
		"actor.role_kind.set_leads_group":      "role_kind.manage",
		"actor.role_kind.list_holders":         "role_kind.manage",
		"actor.group.create":                   "group.manage",
		"actor.group.update":                   "group.manage",
		"actor.group.move":                     "group.manage",
		"actor.group.set_area_node":            "group.manage",
		"actor.group.read":                     "group.view",
		"actor.group.list_children":            "group.view",
		"actor.group.list_leaders":             "group.view",
		"actor.group.list_led":                 "group.view",
		"actor.group.dashboard":                "group.view",
		"actor.code_format.read":               "code_format.manage",
		"actor.code_format.set":                "code_format.manage",
		"forms.form.list":                      "form.manage",
		"forms.form.read":                      "form.manage",
		"forms.form.update":                    "form.manage",
		"forms.form.delete":                    "form.manage",
		"forms.form.submit":                    "form.submit",
		"forms.form.withdraw":                  "form.submit",
		"forms.form.close":                     "form.close",
		"forms.question.update":                "form.manage",
		"forms.question.delete":                "form.manage",
		"forms.question.move":                  "form.manage",
		"forms.question.version":               "form.manage",
		"forms.question_option.list":           "form.manage",
		"forms.question_option.create":         "form.manage",
		"forms.target.create":                  "form.manage",
		"forms.target.list":                    "form.manage",
		"forms.target.delete":                  "form.manage",
		"forms.translation.put":                "form.manage",
		"forms.translation.list":               "form.manage",
		"forms.translation.languages":          "form.manage",
		"forms.results.read":                   "form.results",
		"forms.answer.list":                    "form.results",
		"forms.submission.list":                "form.results",
		"forms.submission.list_missed":         "form.results",
		"forms.approval.list":                  "form.results",
		"forms.approval.list_awaiting":         "form.results",
		"forms.propagation.list":               "form.results",
		"forms.propagation.rollup":             "form.results",
		"forms.submission.mark_paid":           "form.payment",
		"comm.notification.list":               "notification.read",
		"comm.push_delivery.list":              "notification.read",
		"market.poster.apply":                  "market.poster",
		"market.group_order.trail":             "market.browse",
		"market.order.trail":                   "market.browse",
		"market.kind.list":                     "market.browse",
		"market.product.categorise":            "market.categorise",
		"market.group_order.call_off":          "market.group_order",
		"catalog.entry.list":                   "service.read",
		"assetmanager.location.create":         "asset.manage",
		"actor.join_request.list_for_group":    "join_request.operate",
		"actor.join_request.read":              "join_request.operate",
		"actor.join_request.operator_approve":  "join_request.operate",
		"actor.join_request.operator_decline":  "join_request.operate",
		"actor.actor.read":                     "member.view",
		"actor.actor.read_many":                "member.view",
		"actor.assignment.list_for_actor":      "member.view",
		"actor.assignment.list_for_group":      "member.view",
		"actor.role_assignment.read":           "member.view",
		"actor.role_assignment.list_for_actor": "member.view",
		"actor.role_assignment.list_for_group": "member.view",
		"actor.role_assignment.create":         "role.assign",
		"actor.role_assignment.revoke":         "role.assign",
	}
	held := heldByAnUnlockedPermission()
	for action, permission := range want {
		if held[action] == "" {
			t.Errorf("%s belongs to no unlocked permission, want %s", action, permission)
			continue
		}
		p, _ := models.LookupPermission(permission)
		if !carries(p, action) {
			t.Errorf("%s is not carried by %s", action, permission)
		}
	}
}

func TestEveryActionTheCoordinatorAndMemberFlowsNeedBelongsToAnUnlockedPermission(t *testing.T) {
	want := map[string]string{
		"market.group_order.poster":             "market.group_order",
		"market.group_order.line_paid":          "market.group_order",
		"market.group_order.decide_silent_line": "market.group_order",
		"actor.group.list_joinable":             "group.join",
		"actor.group.list_discoverable":         "group.join",
		"actor.group.find_by_code":              "group.join",
		"actor.group.join":                      "group.join",
		"actor.group.join_by_code":              "group.join",
		"actor.invitation.accept":               "group.join",
		"actor.invitation.decline":              "group.join",
		"actor.invitation.list_for_group":       "invitation.read",
		"actor.invitation.read":                 "invitation.read",
		"actor.actor.read_screen_name_by_code":  "member.view",
		"actor.actor.standing":                  "member.view",
		"actor.assignment.home_counts":          "group.view",
		"comm.chat_thread.archive":              "chat.archive",
		"comm.chat_thread.unarchive":            "chat.archive",
		"comm.chat_thread.list_archived":        "chat.archive",
	}
	held := heldByAnUnlockedPermission()
	for action, permission := range want {
		if held[action] == "" {
			t.Errorf("%s belongs to no unlocked permission, want %s", action, permission)
			continue
		}
		p, _ := models.LookupPermission(permission)
		if !carries(p, action) {
			t.Errorf("%s is not carried by %s", action, permission)
		}
	}
}

func TestNoPermissionCarriesAPickersOwnAnswerToARevisedGroupOrder(t *testing.T) {
	if held := heldByAnUnlockedPermission(); held["market.group_order.answer_revision"] != "" {
		t.Errorf("market.group_order.answer_revision is carried by %s, but it is the picker's own answer", held["market.group_order.answer_revision"])
	}
}
