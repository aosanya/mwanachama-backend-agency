package mcp

import (
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAWorkItemLineNamesItsAccountableRole(t *testing.T) {
	got := describe([]models.WorkItem{{
		ID:                "item-3",
		DraftID:           "draft-1",
		Title:             "Open the tender",
		AccountableRoleID: "role-7",
		CreatedAt:         "2026-09-25T10:00:00Z",
	}})

	for _, want := range []string{"accountable role id: role-7", "id: item-3"} {
		if !strings.Contains(got, want) {
			t.Errorf("a step line without %q cannot be correlated to the role accountable for it:\n%s", want, got)
		}
	}
	if strings.Contains(got, "draft-1") {
		t.Errorf("every row in one listing shares the draft:\n%s", got)
	}
	if strings.Contains(got, "2026-09-25") {
		t.Errorf("a timestamp says nothing about which row this is:\n%s", got)
	}
}

func TestAWorkItemLineNamesItsWorkflowAndNoObjective(t *testing.T) {
	got := describe([]models.WorkItem{{
		ID:         "item-1",
		Code:       "WI-4",
		Title:      "Draft the release notes",
		WorkflowID: "workflow-2",
	}})

	for _, want := range []string{`WI-4 "Draft the release notes"`, "workflow id: workflow-2"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "objective") {
		t.Errorf("a step serves no objective of its own:\n%s", got)
	}
}

func TestAWorkflowObjectiveLineNamesBothEndsOfTheLink(t *testing.T) {
	got := describe([]models.WorkflowObjective{{
		ID:          "link-1",
		DraftID:     "draft-1",
		WorkflowID:  "workflow-2",
		ObjectiveID: "objective-3",
		CreatedAt:   "2026-09-25T10:00:00Z",
	}})

	for _, want := range []string{"workflow id: workflow-2", "objective id: objective-3", "id: link-1"} {
		if !strings.Contains(got, want) {
			t.Errorf("a workflow objective line without %q cannot be correlated to its workflow or objective:\n%s", want, got)
		}
	}
	if strings.Contains(got, "draft-1") || strings.Contains(got, "2026-09-25") {
		t.Errorf("a draft or a timestamp says nothing about which link this is:\n%s", got)
	}
}

func TestAnAttachmentLineNamesWhatItAttachesToWhat(t *testing.T) {
	got := describe([]models.ValueAttachment{{
		ID:         "attachment-1",
		ValueCode:  "V-2",
		TargetType: models.TargetTypeRole,
		TargetCode: "R-5",
	}})

	for _, want := range []string{"value code: V-2", "target code: R-5", "target type: role"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q:\n%s", want, got)
		}
	}
}

func TestAnEmptyListingSaysSo(t *testing.T) {
	if got := describe([]models.Role{}); got != "There are none yet." {
		t.Errorf("got %q", got)
	}
}
