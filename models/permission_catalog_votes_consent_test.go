package models_test

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCatalogCarriesVoteReportConsentAndFindActions(t *testing.T) {
	requireCatalogRows(t, []catalogRow{
		{"case.vote", "taskmanager", []string{"taskmanager.case.vote", "taskmanager.case.take_back_vote", "taskmanager.case.list_votes"}},
		{"report.publish", "", []string{"forms.results.publish", "voting.result.publish"}},
		{"consent.read", "custody", []string{"custody.consent_text.read", "custody.consent.mine"}},
		{"consent.give", "custody", []string{"custody.consent.give", "custody.consent.withdraw"}},
		{"consent.publish", "custody", []string{"custody.consent_version.list", "custody.consent_text.publish"}},
		{"task.read", "taskmanager", []string{"taskmanager.task.find"}},
	})
}

func TestSixPermissionsDeclareTheirModuleAndTheActionsAnswerForIt(t *testing.T) {
	requireCatalogRows(t, []catalogRow{
		{"service.read", "forms", nil},
		{"service.publish", "forms", nil},
		{"board_item.reply", "forms", nil},
		{"board_item.hide", "forms", nil},
		{"board.manage", "forms", nil},
		{"mount.list", "agency", nil},
	})
	declared := map[string]string{
		"catalog.entry.list":           "forms",
		"catalog.reply.send":           "forms",
		"catalog.raised_item.reassign": "forms",
		"instance.mount.list":          "agency",
		"taskmanager.case.vote":        "taskmanager",
	}
	for action, module := range declared {
		if got := models.DeclaredModuleOfAction(action); got != module {
			t.Errorf("DeclaredModuleOfAction(%q) = %q, want %q", action, got, module)
		}
	}
	if got := models.DeclaredModuleOfAction("no.such.action"); got != "" {
		t.Errorf("DeclaredModuleOfAction of an unknown action = %q, want empty", got)
	}
}
