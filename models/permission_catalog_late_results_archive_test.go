package models_test

import "testing"

func TestCatalogCarriesLateMattersResultFilesAndArchivedBacklogs(t *testing.T) {
	requireCatalogRows(t, []catalogRow{
		{"late_matters.read", "forms", []string{"catalog.raised_item.list_overdue"}},
		{"report.publish", "", []string{"forms.results.files"}},
		{"backlog.manage", "taskmanager", []string{"taskmanager.backlog.archive", "taskmanager.backlog.unarchive"}},
	})
}
