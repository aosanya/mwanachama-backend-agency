package mwanachamaagency

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

type collectedReport struct{ messages []string }

func (r *collectedReport) Add(severity models.IssueSeverity, entityType, entityID, format string, args ...any) {
	r.messages = append(r.messages, fmt.Sprintf("%s: %s", severity, fmt.Sprintf(format, args...)))
}

func TestHoldScan_WhatAnImportCanCarryIn(t *testing.T) {
	calc := models.WorkItem{ID: "s1", Code: "WI-1", Title: "Work out the interest", WorkflowID: "w1", Ordinality: 1, Kind: models.StepKindCompute}
	approve := models.WorkItem{ID: "s2", Code: "WI-2", Title: "Approve the interest", WorkflowID: "w1", Ordinality: 2, Kind: models.StepKindYesNo, RoleID: "r1", HeldStepID: "s1"}

	cases := []struct {
		name  string
		step  models.WorkItem
		steps []models.WorkItem
		says  string
	}{
		{"a hold on a step of another kind",
			models.WorkItem{ID: "s3", Code: "WI-3", Title: "Tell the farmer", WorkflowID: "w1", Ordinality: 3, Kind: models.StepKindNotify, Hold: &models.StepHold{ExpiresAfterDays: 14}},
			nil, "is a notify step with a hold, and only a calculating step can hold its result"},
		{"a held step on a step of another kind",
			models.WorkItem{ID: "s3", Code: "WI-3", Title: "Tell the farmer", WorkflowID: "w1", Ordinality: 3, Kind: models.StepKindNotify, HeldStepID: "s1"},
			[]models.WorkItem{calc}, "only a yes_no or post step can release one"},
		{"a hold of 91 days",
			models.WorkItem{ID: "s1", Code: "WI-1", Title: "Work out the interest", WorkflowID: "w1", Ordinality: 1, Kind: models.StepKindCompute, Hold: &models.StepHold{ExpiresAfterDays: 91}},
			[]models.WorkItem{approve}, "holds its result for 91 days, and a hold lasts 1 to 90 days"},
		{"a hold of 0 days",
			models.WorkItem{ID: "s1", Code: "WI-1", Title: "Work out the interest", WorkflowID: "w1", Ordinality: 1, Kind: models.StepKindCompute, Hold: &models.StepHold{}},
			[]models.WorkItem{approve}, "holds its result for 0 days"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report := &collectedReport{}
			holdScan(report, tc.step, append([]models.WorkItem{tc.step}, tc.steps...))
			if len(report.messages) != 1 || !strings.HasPrefix(report.messages[0], "error: ") || !strings.Contains(report.messages[0], tc.says) {
				t.Fatalf("want one error saying %q, got %q", tc.says, report.messages)
			}
		})
	}
}
