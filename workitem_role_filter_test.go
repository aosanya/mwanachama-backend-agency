package mwanachamaagency_test

import (
	"context"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

var noticeQuestions = []models.StepQuestion{{Prompt: "How many", AnswerType: models.AnswerNumber}}

func TestListWorkItemsNarrowsToTheRolesAskedFor(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "Role filter"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draft, err := mgr.CreateDraft(ctx, "role filter", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	workflow, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draft.ID, Name: "Intake"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	clerk, _ := mgr.CreateRole(ctx, models.Role{DraftID: draft.ID, Name: "Clerk"})
	mayor, _ := mgr.CreateRole(ctx, models.Role{DraftID: draft.ID, Name: "Mayor"})
	stranger, _ := mgr.CreateRole(ctx, models.Role{DraftID: draft.ID, Name: "Stranger"})
	titles := map[string]models.WorkItem{
		"Clerk step":    {DraftID: draft.ID, WorkflowID: workflow.ID, Title: "Clerk step", Kind: models.StepKindNotify, FormName: "Notice form", Questions: noticeQuestions, RoleID: clerk.ID},
		"Mayor step":    {DraftID: draft.ID, WorkflowID: workflow.ID, Title: "Mayor step", Kind: models.StepKindNotify, FormName: "Notice form", Questions: noticeQuestions, RoleID: mayor.ID},
		"Handover step": {DraftID: draft.ID, WorkflowID: workflow.ID, Title: "Handover step", Kind: models.StepKindNotify, FormName: "Notice form", Questions: noticeQuestions, RoleID: stranger.ID, RecipientRoleID: clerk.ID},
		"Other step":    {DraftID: draft.ID, WorkflowID: workflow.ID, Title: "Other step", Kind: models.StepKindNotify, FormName: "Notice form", Questions: noticeQuestions, RoleID: stranger.ID},
	}
	for _, wi := range titles {
		if _, err := mgr.CreateWorkItem(ctx, wi); err != nil {
			t.Fatalf("CreateWorkItem %s: %v", wi.Title, err)
		}
	}

	got, err := mgr.ListWorkItems(ctx, draft.ID, mwanachamaagency.WorkItemFilter{RoleIDs: []string{clerk.ID}})
	if err != nil {
		t.Fatalf("ListWorkItems: %v", err)
	}
	var names []string
	for _, wi := range got {
		names = append(names, wi.Title)
	}
	if len(names) != 2 || !(names[0] == "Clerk step" && names[1] == "Handover step" || names[0] == "Handover step" && names[1] == "Clerk step") {
		t.Errorf("the clerk's steps = %v, want the step the clerk does and the one handed to the clerk", names)
	}

	none, err := mgr.ListWorkItems(ctx, draft.ID, mwanachamaagency.WorkItemFilter{RoleIDs: []string{}})
	if err != nil {
		t.Fatalf("ListWorkItems: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("an empty role list matched %d steps, want none", len(none))
	}

	all, err := mgr.ListWorkItems(ctx, draft.ID, mwanachamaagency.WorkItemFilter{})
	if err != nil {
		t.Fatalf("ListWorkItems: %v", err)
	}
	if len(all) != 4 {
		t.Errorf("no role list matched %d steps, want all four", len(all))
	}
}
