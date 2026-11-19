package models_test

import (
	"slices"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

type catalogRow struct {
	code    string
	module  string
	actions []string
}

func requireCatalogRows(t *testing.T, rows []catalogRow) {
	t.Helper()
	for _, row := range rows {
		p, ok := models.LookupPermission(row.code)
		if !ok {
			t.Errorf("permission %s is missing from the catalog", row.code)
			continue
		}
		if row.module != "" && p.Module != row.module {
			t.Errorf("%s module = %q, want %q", row.code, p.Module, row.module)
		}
		for _, action := range row.actions {
			if !slices.Contains(p.Actions, action) {
				t.Errorf("%s does not carry %s: %v", row.code, action, p.Actions)
			}
		}
	}
}

func TestCatalogCarriesRiskGroupLimitInsightAndBoardActions(t *testing.T) {
	requireCatalogRows(t, []catalogRow{
		{"risk.read", "", []string{"riskmanager.match.list"}},
		{"group.manage", "", []string{"actor.group.set_boundary"}},
		{"group.view", "", []string{"actor.group.list_at_point"}},
		{"role.assign", "", []string{"actor.role_assignment.set_approval_limit"}},
		{"insight.count", "insights", []string{"insights.insight.count"}},
		{"board.manage", "", []string{"catalog.raised_item.reassign"}},
		{"board.read", "forms", []string{"catalog.board_item.list", "catalog.board_item.tally_by_area"}},
		{"resident.read", "", []string{"actor.actor.list", "actor.actor.read", "actor.actor.read_many", "accounting.bill.list", "accounting.bill.read", "accounting.payment.list"}},
	})
}

func TestBoardManageKeepsItsActionsBesideBoardRead(t *testing.T) {
	want := []string{"catalog.raised_item.group", "catalog.raised_item.ungroup", "catalog.board_item.set_outcome", "catalog.raised_item.reassign", "catalog.board_item.assign", "catalog.board_item.read", "catalog.raised_item.file"}
	manage, ok := models.LookupPermission("board.manage")
	if !ok {
		t.Fatal("board.manage is gone from the catalogue")
	}
	if !slices.Equal(manage.Actions, want) {
		t.Errorf("board.manage actions = %v, want %v", manage.Actions, want)
	}
}

func TestCatalogCarriesTheTaskModulePermissions(t *testing.T) {
	requireCatalogRows(t, []catalogRow{
		{"task.read", "taskmanager", []string{
			"taskmanager.task.list", "taskmanager.task.list_overdue", "taskmanager.task.pending",
			"taskmanager.task.read", "taskmanager.task.read_by_name", "taskmanager.task.list_assignments",
			"taskmanager.task_state.list", "taskmanager.backlog.list", "taskmanager.backlog_access.list",
			"taskmanager.relationship.traverse",
		}},
		{"task.edit", "taskmanager", []string{
			"taskmanager.task.create", "taskmanager.task.update", "taskmanager.task.place",
			"taskmanager.task.move", "taskmanager.task.move_to_step",
			"taskmanager.relationship.create", "taskmanager.relationship.delete",
		}},
		{"backlog.manage", "taskmanager", []string{
			"taskmanager.backlog.create", "taskmanager.backlog.rename", "taskmanager.backlog.delete",
			"taskmanager.backlog.manage", "taskmanager.backlog_step.add", "taskmanager.backlog_step.rename",
			"taskmanager.backlog_step.remove", "taskmanager.backlog_step.reorder",
			"taskmanager.backlog_access.give", "taskmanager.backlog_access.take",
			"taskmanager.backlog.set_owner_group",
		}},
	})
}

func TestCatalogCarriesBacklogRulesAndTheAssistantPermissions(t *testing.T) {
	requireCatalogRows(t, []catalogRow{
		{"task.read", "", []string{"taskmanager.backlog.rules_count"}},
		{"backlog.manage", "", []string{"taskmanager.backlog.rules_set"}},
		{"ai.chat", "ai", []string{
			"ai.account.chat", "ai.chat_model.list", "ai.chat_model.pick",
			"ai.assistant_limits.get", "ai.assistant_usage.mine",
		}},
		{"ai.account.read", "", []string{"ai.assistant_limits.get", "ai.assistant_usage.mine"}},
		{"ai.account.manage", "", []string{"ai.assistant_limits.set"}},
	})
}

func TestCatalogCarriesTheDesignReadsAReviewAndAPollNeed(t *testing.T) {
	requireCatalogRows(t, []catalogRow{
		{"controls.edit", "", []string{"agency.control.list", "agency.control.read", "agency.control.list_attachments"}},
		{"risk.read", "", []string{
			"agency.risk.list", "agency.risk.read", "agency.risk_pathway.list", "agency.risk_pathway.read",
			"agency.hazard.list", "agency.hazard.read", "agency.threat.list", "agency.threat.read",
			"agency.consequence.list", "agency.consequence.read", "agency.risk_threat.list",
			"agency.risk_consequence.list", "agency.unguarded_parts.list",
		}},
		{"review.run", "agency", []string{"agency.review.start"}},
		{"poll.run", "", []string{"voting.poll.read", "voting.choice.list"}},
	})
}
