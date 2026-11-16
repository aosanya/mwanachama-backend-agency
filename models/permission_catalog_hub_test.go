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
	})
}
