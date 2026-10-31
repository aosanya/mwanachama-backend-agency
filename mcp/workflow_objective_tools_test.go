package mcp_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type stepDesign struct {
	mgr        mwanachamaagency.AgencyManager
	cs         *gosdkmcp.ClientSession
	draft      models.Draft
	objectiveA models.Objective
	objectiveB models.Objective
	flow       models.Workflow
	otherFlow  models.Workflow
	step       models.WorkItem
	otherStep  models.WorkItem
	owner      models.Role
	helper     models.Role
}

func newStepDesign(t *testing.T) stepDesign {
	t.Helper()
	mgr := newTestManager(t)
	ctx := ctxT(t)
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "SWELab"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draft, err := mgr.CreateDraft(ctx, "design", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	goal, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draft.ID, Title: "Grow"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	d := stepDesign{mgr: mgr, draft: draft}
	if d.objectiveA, err = mgr.CreateObjective(ctx, models.Objective{DraftID: draft.ID, GoalID: goal.ID, Title: "Win members"}); err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	if d.objectiveB, err = mgr.CreateObjective(ctx, models.Objective{DraftID: draft.ID, GoalID: goal.ID, Title: "Keep members"}); err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	if d.flow, err = mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draft.ID, Name: "Onboard"}); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	if d.otherFlow, err = mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draft.ID, Name: "Renew"}); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	if d.step, err = mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: draft.ID, WorkflowID: d.flow.ID, Title: "Welcome"}); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	if d.otherStep, err = mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: draft.ID, WorkflowID: d.otherFlow.ID, Title: "Remind"}); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	if d.owner, err = mgr.CreateRole(ctx, models.Role{DraftID: draft.ID, Name: "Owner"}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if d.helper, err = mgr.CreateRole(ctx, models.Role{DraftID: draft.ID, Name: "Helper"}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	d.cs = mcpConnect(t, srv.URL)
	return d
}

func (d stepDesign) call(t *testing.T, tool string, args map[string]any) *gosdkmcp.CallToolResult {
	t.Helper()
	args["instance_id"] = "any"
	result, err := d.cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool %s: %v", tool, err)
	}
	return result
}

func (d stepDesign) mustCall(t *testing.T, tool string, args map[string]any) map[string]any {
	t.Helper()
	result := d.call(t, tool, args)
	if result.IsError {
		t.Fatalf("%s: IsError = true (%s)", tool, textOf(result))
	}
	sc, _ := result.StructuredContent.(map[string]any)
	return sc
}

func itemsOf(t *testing.T, sc map[string]any) []map[string]any {
	t.Helper()
	raw, _ := sc["items"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, r := range raw {
		item, _ := r.(map[string]any)
		out = append(out, item)
	}
	return out
}

func TestWorkflowObjectiveTools_LinkListGetAndUnlink(t *testing.T) {
	d := newStepDesign(t)

	linked := d.mustCall(t, "agency_link_workflow_objective", map[string]any{
		"draft_id": d.draft.ID, "workflow_id": d.flow.ID, "objective_id": d.objectiveA.ID,
	})
	linkID, _ := linked["id"].(string)
	if linkID == "" || linked["workflow_id"] != d.flow.ID || linked["objective_id"] != d.objectiveA.ID {
		t.Fatalf("agency_link_workflow_objective returned %+v, want the stored link with its id and both ends", linked)
	}

	d.mustCall(t, "agency_link_workflow_objective", map[string]any{
		"draft_id": d.draft.ID, "workflow_id": d.flow.ID, "objective_id": d.objectiveB.ID,
	})
	d.mustCall(t, "agency_link_workflow_objective", map[string]any{
		"draft_id": d.draft.ID, "workflow_id": d.otherFlow.ID, "objective_id": d.objectiveA.ID,
	})

	all := itemsOf(t, d.mustCall(t, "agency_list_workflow_objectives", map[string]any{"draft_id": d.draft.ID}))
	if len(all) != 3 {
		t.Fatalf("agency_list_workflow_objectives = %+v, want 3 links", all)
	}
	byWorkflow := itemsOf(t, d.mustCall(t, "agency_list_workflow_objectives", map[string]any{"draft_id": d.draft.ID, "workflow_id": d.flow.ID}))
	if len(byWorkflow) != 2 {
		t.Errorf("a workflow serving two objectives listed %d links, want 2: %+v", len(byWorkflow), byWorkflow)
	}
	byObjective := itemsOf(t, d.mustCall(t, "agency_list_workflow_objectives", map[string]any{"draft_id": d.draft.ID, "objective_id": d.objectiveA.ID}))
	if len(byObjective) != 2 {
		t.Errorf("an objective served by two workflows listed %d links, want 2: %+v", len(byObjective), byObjective)
	}
	pair := itemsOf(t, d.mustCall(t, "agency_list_workflow_objectives", map[string]any{
		"draft_id": d.draft.ID, "workflow_id": d.otherFlow.ID, "objective_id": d.objectiveB.ID,
	}))
	if len(pair) != 0 {
		t.Errorf("a pair that was never linked listed %+v", pair)
	}

	got := d.mustCall(t, "agency_get_workflow_objective", map[string]any{"workflow_objective_id": linkID})
	if got["id"] != linkID || got["workflow_id"] != d.flow.ID || got["objective_id"] != d.objectiveA.ID {
		t.Errorf("agency_get_workflow_objective = %+v, want the link made first", got)
	}

	d.mustCall(t, "agency_delete_workflow_objective", map[string]any{"workflow_objective_id": linkID})

	if gone := d.call(t, "agency_get_workflow_objective", map[string]any{"workflow_objective_id": linkID}); !gone.IsError {
		t.Errorf("a link that was unlinked can still be read: %s", textOf(gone))
	}
	if left, err := d.mgr.ListWorkflowObjectives(ctxT(t), d.draft.ID, mwanachamaagency.WorkflowObjectiveFilter{}); err != nil || len(left) != 2 {
		t.Errorf("after one unlink ListWorkflowObjectives = %+v err=%v, want the other 2", left, err)
	}
	if _, err := d.mgr.GetObjective(ctxT(t), d.objectiveA.ID); err != nil {
		t.Errorf("unlinking removed the objective: %v", err)
	}
	if _, err := d.mgr.GetWorkflow(ctxT(t), d.flow.ID); err != nil {
		t.Errorf("unlinking removed the workflow: %v", err)
	}
}

func TestWorkflowObjectiveTools_RefuseTheSamePairTwice(t *testing.T) {
	d := newStepDesign(t)
	args := func() map[string]any {
		return map[string]any{"draft_id": d.draft.ID, "workflow_id": d.flow.ID, "objective_id": d.objectiveA.ID}
	}
	d.mustCall(t, "agency_link_workflow_objective", args())

	again := d.call(t, "agency_link_workflow_objective", args())
	if !again.IsError {
		t.Fatalf("the same workflow and objective linked twice was accepted: %+v", again.StructuredContent)
	}
	if links, _ := d.mgr.ListWorkflowObjectives(ctxT(t), d.draft.ID, mwanachamaagency.WorkflowObjectiveFilter{}); len(links) != 1 {
		t.Errorf("the refused link was stored anyway: %+v", links)
	}
}

func TestWorkflowObjectiveTools_RefuseAnEndInAnotherDraft(t *testing.T) {
	d := newStepDesign(t)
	ctx := ctxT(t)
	other, err := d.mgr.CreateDraft(ctx, "other", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	goal, _ := d.mgr.CreateGoal(ctx, models.Goal{DraftID: other.ID, Title: "Elsewhere"})
	foreign, err := d.mgr.CreateObjective(ctx, models.Objective{DraftID: other.ID, GoalID: goal.ID, Title: "Foreign"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}

	crossed := d.call(t, "agency_link_workflow_objective", map[string]any{
		"draft_id": d.draft.ID, "workflow_id": d.flow.ID, "objective_id": foreign.ID,
	})
	if !crossed.IsError {
		t.Fatalf("a workflow linked to an objective of another draft was accepted: %+v", crossed.StructuredContent)
	}
	missing := d.call(t, "agency_link_workflow_objective", map[string]any{
		"draft_id": d.draft.ID, "workflow_id": d.flow.ID, "objective_id": "no-such-objective",
	})
	if !missing.IsError {
		t.Fatalf("a workflow linked to an objective that does not exist was accepted: %+v", missing.StructuredContent)
	}
	for _, draftID := range []string{d.draft.ID, other.ID} {
		if links, _ := d.mgr.ListWorkflowObjectives(ctx, draftID, mwanachamaagency.WorkflowObjectiveFilter{}); len(links) != 0 {
			t.Errorf("draft %s holds links after the refused calls: %+v", draftID, links)
		}
	}
}

func TestWorkflowObjectiveTools_ALinkedWorkflowStopsBeingAnIssue(t *testing.T) {
	d := newStepDesign(t)
	if _, err := d.mgr.CreateWorkItem(ctxT(t), models.WorkItem{DraftID: d.draft.ID, WorkflowID: d.otherFlow.ID, Title: "Second"}); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	unserved := func() int {
		n := 0
		for _, issue := range itemsOf(t, d.mustCall(t, "agency_list_issues", map[string]any{"draft_id": d.draft.ID})) {
			if issue["entity_type"] == "workflow" && strings.Contains(issue["message"].(string), "has no Objective") {
				n++
			}
		}
		return n
	}
	if got := unserved(); got != 2 {
		t.Fatalf("two workflows with no objective raised %d issues, want 2", got)
	}
	d.mustCall(t, "agency_link_workflow_objective", map[string]any{"draft_id": d.draft.ID, "workflow_id": d.flow.ID, "objective_id": d.objectiveA.ID})
	if got := unserved(); got != 1 {
		t.Errorf("after linking one workflow %d issues remain, want 1", got)
	}
}

func TestRaciTools_SeatsAreLabelledOnAStep(t *testing.T) {
	d := newStepDesign(t)

	set := func(stepID, roleID, label string) *gosdkmcp.CallToolResult {
		return d.call(t, "agency_set_raci", map[string]any{"draft_id": d.draft.ID, "work_item_id": stepID, "role_id": roleID, "label": label})
	}
	for _, c := range []struct{ step, role, label string }{
		{d.step.ID, d.owner.ID, "accountable"},
		{d.step.ID, d.helper.ID, "accountable"},
		{d.step.ID, d.owner.ID, "responsible"},
		{d.step.ID, d.helper.ID, "informed"},
		{d.otherStep.ID, d.helper.ID, "consulted"},
	} {
		if result := set(c.step, c.role, c.label); result.IsError {
			t.Fatalf("agency_set_raci %+v refused: %s", c, textOf(result))
		}
	}

	set1 := d.mustCall(t, "agency_list_role_assignments_for_work_item", map[string]any{"draft_id": d.draft.ID, "work_item_id": d.step.ID})
	seats := itemsOf(t, set1)
	if len(seats) != 4 {
		t.Fatalf("agency_list_role_assignments_for_work_item = %+v, want the 4 seats of the step and none of the other step's", seats)
	}
	for _, seat := range seats {
		if seat["work_item_id"] != d.step.ID {
			t.Errorf("a seat of another step was listed: %+v", seat)
		}
		if _, hasObjective := seat["objective_id"]; hasObjective {
			t.Errorf("a seat still carries an objective_id: %+v", seat)
		}
	}
	accountable := 0
	for _, seat := range seats {
		if seat["label"] == "accountable" {
			accountable++
		}
	}
	if accountable != 2 {
		t.Errorf("a step holds %d accountable seats, want the 2 that were set", accountable)
	}

	onlyAccountable := itemsOf(t, d.mustCall(t, "agency_list_role_assignments_for_work_item", map[string]any{
		"draft_id": d.draft.ID, "work_item_id": d.step.ID, "label": "accountable",
	}))
	if len(onlyAccountable) != 2 {
		t.Errorf("label filter returned %+v, want the 2 accountable seats", onlyAccountable)
	}
	onlyHelper := itemsOf(t, d.mustCall(t, "agency_list_role_assignments_for_work_item", map[string]any{
		"draft_id": d.draft.ID, "work_item_id": d.step.ID, "role_id": d.helper.ID,
	}))
	if len(onlyHelper) != 2 {
		t.Errorf("role filter returned %+v, want the helper's 2 seats", onlyHelper)
	}

	other := itemsOf(t, d.mustCall(t, "agency_list_role_assignments_for_work_item", map[string]any{"draft_id": d.draft.ID, "work_item_id": d.otherStep.ID}))
	if len(other) != 1 || other[0]["label"] != "consulted" {
		t.Errorf("the other step lists %+v, want its one consulted seat", other)
	}
}

func TestRaciTools_TheSameSeatAndLabelOnAStepIsTheOnlyRefusal(t *testing.T) {
	d := newStepDesign(t)
	args := map[string]any{"draft_id": d.draft.ID, "work_item_id": d.step.ID, "role_id": d.owner.ID, "label": "accountable"}
	d.mustCall(t, "agency_set_raci", args)

	if again := d.call(t, "agency_set_raci", args); !again.IsError {
		t.Fatalf("the same seat with the same label on the same step twice was accepted: %+v", again.StructuredContent)
	}
	elsewhere := d.call(t, "agency_set_raci", map[string]any{"draft_id": d.draft.ID, "work_item_id": d.otherStep.ID, "role_id": d.owner.ID, "label": "accountable"})
	if elsewhere.IsError {
		t.Errorf("the same seat and label on another step was refused: %s", textOf(elsewhere))
	}
	seats, err := d.mgr.ListRoleAssignmentsForWorkItem(ctxT(t), d.step.ID, d.draft.ID, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil || len(seats) != 1 {
		t.Errorf("the step holds %+v err=%v, want the one seat", seats, err)
	}
}

func TestRaciTools_RefuseAStepOrSeatThatIsNotThere(t *testing.T) {
	d := newStepDesign(t)
	for name, args := range map[string]map[string]any{
		"unknown step": {"work_item_id": "no-such-step", "role_id": d.owner.ID, "label": "accountable"},
		"unknown role": {"work_item_id": d.step.ID, "role_id": "no-such-role", "label": "accountable"},
		"bad label":    {"work_item_id": d.step.ID, "role_id": d.owner.ID, "label": "boss"},
	} {
		args["draft_id"] = d.draft.ID
		if result := d.call(t, "agency_set_raci", args); !result.IsError {
			t.Errorf("%s: agency_set_raci accepted %+v", name, result.StructuredContent)
		}
	}
	if seats, _ := d.mgr.ListRoleAssignments(ctxT(t), d.draft.ID, mwanachamaagency.RoleAssignmentFilter{}); len(seats) != 0 {
		t.Errorf("refused calls stored seats: %+v", seats)
	}
}

func TestDeletingAStepOrAWorkflowOrAnObjectiveLeavesNoDanglingJoinRow(t *testing.T) {
	d := newStepDesign(t)
	ctx := ctxT(t)
	d.mustCall(t, "agency_link_workflow_objective", map[string]any{"draft_id": d.draft.ID, "workflow_id": d.flow.ID, "objective_id": d.objectiveA.ID})
	d.mustCall(t, "agency_link_workflow_objective", map[string]any{"draft_id": d.draft.ID, "workflow_id": d.otherFlow.ID, "objective_id": d.objectiveB.ID})
	d.mustCall(t, "agency_set_raci", map[string]any{"draft_id": d.draft.ID, "work_item_id": d.step.ID, "role_id": d.owner.ID, "label": "accountable"})
	d.mustCall(t, "agency_set_raci", map[string]any{"draft_id": d.draft.ID, "work_item_id": d.otherStep.ID, "role_id": d.owner.ID, "label": "informed"})

	d.mustCall(t, "agency_delete_objective", map[string]any{"objective_id": d.objectiveB.ID})
	links, _ := d.mgr.ListWorkflowObjectives(ctx, d.draft.ID, mwanachamaagency.WorkflowObjectiveFilter{})
	if len(links) != 1 || links[0].ObjectiveID != d.objectiveA.ID {
		t.Errorf("deleting an objective left %+v, want only the other objective's link", links)
	}

	d.mustCall(t, "agency_delete_workflow", map[string]any{"workflow_id": d.flow.ID})
	links, _ = d.mgr.ListWorkflowObjectives(ctx, d.draft.ID, mwanachamaagency.WorkflowObjectiveFilter{})
	seats, _ := d.mgr.ListRoleAssignments(ctx, d.draft.ID, mwanachamaagency.RoleAssignmentFilter{})
	if len(links) != 0 {
		t.Errorf("deleting a workflow left its objective links: %+v", links)
	}
	if len(seats) != 1 || seats[0].WorkItemID != d.otherStep.ID {
		t.Errorf("deleting a workflow left %+v, want only the seat of the other workflow's step", seats)
	}
}
