package mwanachamaagency_test

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func heldByAnUnlockedPower() map[string]string {
	out := map[string]string{}
	for _, p := range models.PowerCatalog() {
		if p.Locked {
			continue
		}
		for _, a := range p.Actions {
			out[a] = p.Code
		}
	}
	return out
}

func TestEveryActionOfTheCataloguePowerGapBelongsToAnUnlockedPower(t *testing.T) {
	want := map[string]string{
		"ai.account.chat":                 "ai.use",
		"ai.model.list":                   "ai.use",
		"ai.model.refresh":                "ai.account.manage",
		"actor.role_kind.create":          "role_kind.manage",
		"actor.role_kind.list":            "role_kind.manage",
		"actor.role_kind.read":            "role_kind.manage",
		"actor.role_kind.retire":          "role_kind.manage",
		"actor.role_kind.unretire":        "role_kind.manage",
		"actor.role_kind.set_leads_group": "role_kind.manage",
		"actor.role_kind.list_holders":    "role_kind.manage",
		"actor.group.create":              "group.manage",
		"actor.group.update":              "group.manage",
		"actor.group.move":                "group.manage",
		"actor.group.set_area_node":       "group.manage",
		"actor.group.read":                "group.view",
		"actor.group.list_children":       "group.view",
		"actor.group.list_leaders":        "group.view",
		"actor.group.list_led":            "group.view",
		"actor.group.dashboard":           "group.view",
		"actor.code_format.read":          "code_format.manage",
		"actor.code_format.set":           "code_format.manage",
		"forms.form.list":                 "form.manage",
		"forms.form.read":                 "form.manage",
		"forms.form.update":               "form.manage",
		"forms.form.delete":               "form.manage",
		"forms.form.submit":               "form.submit",
		"forms.form.withdraw":             "form.submit",
		"forms.form.close":                "form.close",
		"forms.question.update":           "form.manage",
		"forms.question.delete":           "form.manage",
		"forms.question.move":             "form.manage",
		"forms.question.version":          "form.manage",
		"forms.question_option.list":      "form.manage",
		"forms.question_option.create":    "form.manage",
		"forms.target.create":             "form.manage",
		"forms.target.list":               "form.manage",
		"forms.target.delete":             "form.manage",
		"forms.translation.put":           "form.manage",
		"forms.translation.list":          "form.manage",
		"forms.translation.languages":     "form.manage",
		"forms.results.read":              "form.results",
		"forms.answer.list":               "form.results",
		"forms.submission.list":           "form.results",
		"forms.submission.list_missed":    "form.results",
		"forms.approval.list":             "form.results",
		"forms.approval.list_awaiting":    "form.results",
		"forms.propagation.list":          "form.results",
		"forms.propagation.rollup":        "form.results",
		"forms.submission.mark_paid":      "form.payment",
		"comm.notification.list":          "notification.read",
		"comm.push_delivery.list":         "notification.read",
		"market.poster.apply":             "market.poster",
		"market.group_order.trail":        "market.browse",
		"market.order.trail":              "market.browse",
		"catalog.entry.list":              "service.read",
		"assetmanager.location.create":    "asset.manage",
	}
	held := heldByAnUnlockedPower()
	for action, power := range want {
		if held[action] == "" {
			t.Errorf("%s belongs to no unlocked power, want %s", action, power)
			continue
		}
		p, _ := models.LookupPower(power)
		if !carries(p, action) {
			t.Errorf("%s is not carried by %s", action, power)
		}
	}
}
