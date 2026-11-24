package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func accountableStep(t *testing.T, w stepWorld, wf models.Workflow, title string, own, accountable string) models.WorkItem {
	t.Helper()
	item := models.WorkItem{DraftID: w.draftID, WorkflowID: wf.ID, Title: title, Prompt: "PROMPT-" + title}
	if own != "" {
		item.Kind = models.StepKindFillForm
		item.RoleID = w.roles[own].ID
		item.FormName = "Form " + title
		item.Questions = []models.StepQuestion{ask("Anything to report", models.AnswerYesNo)}
	}
	if accountable != "" {
		item.AccountableRoleID = w.roles[accountable].ID
	}
	got, err := w.mgr.CreateWorkItem(context.Background(), item)
	must(t, err)
	return got
}

func publishedBrief(t *testing.T, w stepWorld, role models.Role) models.RoleBrief {
	t.Helper()
	ctx := context.Background()
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	brief, err := w.mgr.GetRoleBrief(ctx, role.ID)
	must(t, err)
	return brief
}

func briefStepCodes(brief models.RoleBrief) []string {
	var codes []string
	for _, item := range brief.WorkItems {
		codes = append(codes, item.Code)
	}
	return codes
}

func TestAccountableRole_AStepWithNoRoleOfItsOwnTakesOne(t *testing.T) {
	w := newStepWorld(t)
	wf := w.workflow(t, "Tender")
	step := accountableStep(t, w, wf, "Open the tender", "", farmer)
	got, err := w.mgr.GetWorkItem(context.Background(), step.ID)
	must(t, err)
	if got.AccountableRoleID != w.roles[farmer].ID || got.RoleID != "" {
		t.Errorf("accountable %q role %q, want accountable %q and no role", got.AccountableRoleID, got.RoleID, w.roles[farmer].ID)
	}
}

func TestAccountableRole_AStepWithItsOwnRoleKeepsBoth(t *testing.T) {
	w := newStepWorld(t)
	wf := w.workflow(t, "Tender")
	step := accountableStep(t, w, wf, "Fill the bid", coordinator, farmer)
	got, err := w.mgr.GetWorkItem(context.Background(), step.ID)
	must(t, err)
	if got.AccountableRoleID != w.roles[farmer].ID || got.RoleID != w.roles[coordinator].ID {
		t.Errorf("accountable %q role %q", got.AccountableRoleID, got.RoleID)
	}
}

func TestAccountableRole_RefusesARoleOfNoDraftAndMoreThanOne(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Tender")
	other := newStepWorld(t)
	base := models.WorkItem{DraftID: w.draftID, WorkflowID: wf.ID, Title: "Bad"}

	unknown := base
	unknown.AccountableRoleID = "no-such-role"
	if _, err := w.mgr.CreateWorkItem(ctx, unknown); !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Errorf("unknown accountable role: %v, want ErrRoleNotFound", err)
	}

	foreign := base
	foreign.AccountableRoleID = other.roles[farmer].ID
	if _, err := w.mgr.CreateWorkItem(ctx, foreign); !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Errorf("accountable role of another draft: %v, want ErrRoleNotFound", err)
	}

	two := base
	two.AccountableRoleID = w.roles[farmer].ID + "," + w.roles[supplier].ID
	if _, err := w.mgr.CreateWorkItem(ctx, two); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Errorf("two accountable roles: %v, want ErrInvalidWorkItem", err)
	}
}

func TestAccountableRole_UpdateSetsChangesAndClearsIt(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Tender")
	step := accountableStep(t, w, wf, "Open the tender", "", "")

	step.AccountableRoleID = w.roles[farmer].ID
	updated, err := w.mgr.UpdateWorkItem(ctx, step)
	must(t, err)
	if updated.AccountableRoleID != w.roles[farmer].ID {
		t.Fatalf("set: got %q", updated.AccountableRoleID)
	}

	updated.AccountableRoleID = w.roles[supplier].ID
	updated, err = w.mgr.UpdateWorkItem(ctx, updated)
	must(t, err)
	if updated.AccountableRoleID != w.roles[supplier].ID {
		t.Fatalf("change: got %q", updated.AccountableRoleID)
	}

	updated.AccountableRoleID = ""
	updated, err = w.mgr.UpdateWorkItem(ctx, updated)
	must(t, err)
	got, err := w.mgr.GetWorkItem(ctx, updated.ID)
	must(t, err)
	if got.AccountableRoleID != "" {
		t.Errorf("clear: got %q", got.AccountableRoleID)
	}
}

func TestAccountableRole_DeletingTheRoleEmptiesTheFieldOnItsSteps(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Tender")
	kept := accountableStep(t, w, wf, "Kept", coordinator, supplier)
	emptied := accountableStep(t, w, wf, "Emptied", "", farmer)
	must(t, w.mgr.DeleteRole(ctx, w.roles[farmer].ID))

	got, err := w.mgr.GetWorkItem(ctx, emptied.ID)
	must(t, err)
	if got.AccountableRoleID != "" {
		t.Errorf("a deleted role stayed accountable: %q", got.AccountableRoleID)
	}
	got, err = w.mgr.GetWorkItem(ctx, kept.ID)
	must(t, err)
	if got.AccountableRoleID != w.roles[supplier].ID || got.RoleID != w.roles[coordinator].ID {
		t.Errorf("an unrelated step changed: accountable %q role %q", got.AccountableRoleID, got.RoleID)
	}
}

func TestAccountableRole_ADraftForkAndThePublicationCarryIt(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Tender")
	step := accountableStep(t, w, wf, "Open the tender", "", farmer)
	pub, err := w.mgr.PublishAgency(ctx, w.draftID, "", "")
	must(t, err)

	var inDocument models.DocumentWorkItem
	for _, wi := range pub.Document.WorkItems {
		if wi.ID == step.ID {
			inDocument = wi
		}
	}
	if inDocument.AccountableRoleID != w.roles[farmer].ID {
		t.Errorf("the publication document carries %q", inDocument.AccountableRoleID)
	}
	raw, err := json.Marshal(pub.Document)
	must(t, err)
	if strings.Contains(string(raw), "role_assignments") {
		t.Errorf("the publication document still has a role_assignments key: %s", raw)
	}

	fork, err := w.mgr.CreateDraft(ctx, "fork", "", "")
	must(t, err)
	forkSteps, err := w.mgr.ListWorkItems(ctx, fork.ID, mwanachamaagency.WorkItemFilter{})
	must(t, err)
	forkRoles, err := w.mgr.ListRoles(ctx, fork.ID, mwanachamaagency.RoleFilter{})
	must(t, err)
	var forkFarmer string
	for _, r := range forkRoles {
		if r.Name == farmer {
			forkFarmer = r.ID
		}
	}
	found := false
	for _, s := range forkSteps {
		if s.Title == "Open the tender" {
			found = true
			if forkFarmer == "" || s.AccountableRoleID != forkFarmer {
				t.Errorf("fork accountable %q, want the fork's own role %q", s.AccountableRoleID, forkFarmer)
			}
		}
	}
	if !found {
		t.Fatal("the step is missing from the fork")
	}
}

func TestAccountableRole_ChangingItIsAChangeToPublish(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Tender")
	step := accountableStep(t, w, wf, "Open the tender", "", farmer)
	_, err := w.mgr.PublishAgency(ctx, w.draftID, "", "")
	must(t, err)
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "", ""); !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("republishing unchanged content: %v", err)
	}
	step.AccountableRoleID = w.roles[supplier].ID
	_, err = w.mgr.UpdateWorkItem(ctx, step)
	must(t, err)
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "", ""); err != nil {
		t.Errorf("a changed accountable role was not a change: %v", err)
	}
}

func TestListIssues_AnAccountableRoleThatIsGoneBlocks(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	_, _, o, wf, _ := seedLiveAgency(t, mgr, ctx)
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "owner"})
	must(t, err)
	step, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: o.DraftID, WorkflowID: wf.ID, Title: "Open the tender", AccountableRoleID: role.ID})
	must(t, err)
	table := tableOf(t, s, "work_item")
	if err := db.Exec("update "+table+" set accountable_role_id = ? where id = ?", "ghost-role", step.ID).Error; err != nil {
		t.Fatalf("plant a dangling accountable role: %v", err)
	}
	found := false
	for _, i := range issuesFor(t, mgr, ctx, o.DraftID, "work_item") {
		if i.Severity == models.IssueSeverityError && i.EntityID == step.ID && strings.Contains(i.Message, "accountable") {
			found = true
		}
	}
	if !found {
		t.Error("no error issue names the step's missing accountable role")
	}
}

func TestRoleBrief_ListsTheStepsOfTheAccountableRole(t *testing.T) {
	w := newStepWorld(t)
	wf := w.workflow(t, "Tender")
	mine := accountableStep(t, w, wf, "Open the tender", "", farmer)
	accountableStep(t, w, wf, "Someone else's", "", supplier)

	brief := publishedBrief(t, w, w.roles[farmer])
	if len(brief.WorkItems) != 1 || brief.WorkItems[0].Code != mine.Code {
		t.Fatalf("brief steps %v, want only %s", briefStepCodes(brief), mine.Code)
	}
	if !brief.WorkItems[0].Accountable {
		t.Errorf("the step is not marked accountable: %+v", brief.WorkItems[0])
	}
	if !strings.Contains(brief.Text, "PROMPT-Open the tender") || strings.Contains(brief.Text, "PROMPT-Someone else's") {
		t.Errorf("brief text:\n%s", brief.Text)
	}
	if len(brief.Objectives) != 1 {
		t.Errorf("the objective of the step's workflow is missing: %+v", brief.Objectives)
	}
}

func TestRoleBrief_AStepWithItsOwnRoleAndNoAccountableListsUnderItsOwnRole(t *testing.T) {
	w := newStepWorld(t)
	wf := w.workflow(t, "Tender")
	mine := accountableStep(t, w, wf, "Fill the bid", coordinator, "")

	brief := publishedBrief(t, w, w.roles[coordinator])
	if len(brief.WorkItems) != 1 || brief.WorkItems[0].Code != mine.Code {
		t.Fatalf("brief steps %v, want %s", briefStepCodes(brief), mine.Code)
	}
	if brief.WorkItems[0].Accountable {
		t.Errorf("a step listed by its own role is not marked accountable: %+v", brief.WorkItems[0])
	}
}

func TestRoleBrief_AStepWithBothListsUnderTheAccountableRoleOnly(t *testing.T) {
	w := newStepWorld(t)
	wf := w.workflow(t, "Tender")
	step := accountableStep(t, w, wf, "Fill the bid", coordinator, farmer)

	if got := briefStepCodes(publishedBrief(t, w, w.roles[farmer])); len(got) != 1 || got[0] != step.Code {
		t.Errorf("accountable role's steps %v, want %s", got, step.Code)
	}
	if got := briefStepCodes(publishedBrief(t, w, w.roles[coordinator])); len(got) != 0 {
		t.Errorf("a step with an accountable role also listed under its own role: %v", got)
	}
}

func TestRoleBrief_AnOldPublicationWithRoleAssignmentsStillReads(t *testing.T) {
	doc := legacyDocument(t, []map[string]string{
		{"work_item_id": "s1", "role_id": "r1", "label": "accountable"},
	})
	pub := models.AgencyPublication{Version: 1, Tag: "v1", Document: doc}
	brief, err := mwanachamaagency.BriefFromDocument(pub, "r1")
	must(t, err)
	if len(brief.WorkItems) != 1 || brief.WorkItems[0].Code != "WI-1" || !brief.WorkItems[0].Accountable {
		t.Errorf("an old publication's accountable row is not read: %+v", brief.WorkItems)
	}
}

func legacyDocument(t *testing.T, assignments []map[string]string) models.AgencyDocument {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"roles":               []map[string]string{{"id": "r1", "code": "R-1", "name": "A"}, {"id": "r2", "code": "R-2", "name": "B"}},
		"goals":               []map[string]string{{"id": "g1", "code": "G-1", "title": "G"}},
		"objectives":          []map[string]string{{"id": "o1", "code": "O-1", "goal_id": "g1", "title": "O"}},
		"workflows":           []map[string]string{{"id": "w1", "code": "W-1", "name": "W"}},
		"workflow_objectives": []map[string]string{{"id": "x", "workflow_id": "w1", "objective_id": "o1"}},
		"work_items": []map[string]any{
			{"id": "s1", "code": "WI-1", "workflow_id": "w1", "title": "No role of its own", "prompt": "P1"},
			{"id": "s2", "code": "WI-2", "workflow_id": "w1", "title": "Own role", "prompt": "P2", "kind": "fill_form", "role_id": "r1",
				"form_name": "F", "questions": []map[string]any{{"prompt": "Q", "answer_type": "yes_no", "required": true}}},
			{"id": "s3", "code": "WI-3", "workflow_id": "w1", "title": "Already accountable", "prompt": "P3", "accountable_role_id": "r2"},
		},
		"role_assignments": assignments,
	})
	must(t, err)
	var doc models.AgencyDocument
	must(t, json.Unmarshal(raw, &doc))
	doc.Agency.Name = "Imp"
	return doc
}

func importedSteps(t *testing.T, doc models.AgencyDocument) map[string]models.WorkItem {
	t.Helper()
	mgr := newTestManager(t)
	ctx := context.Background()
	res, err := mgr.ImportDocument(ctx, "imp", doc)
	must(t, err)
	steps, err := mgr.ListWorkItems(ctx, res.Draft.ID, mwanachamaagency.WorkItemFilter{})
	must(t, err)
	roles, err := mgr.ListRoles(ctx, res.Draft.ID, mwanachamaagency.RoleFilter{})
	must(t, err)
	codeOfRole := map[string]string{}
	for _, r := range roles {
		codeOfRole[r.ID] = r.Code
	}
	byCode := map[string]models.WorkItem{}
	for _, s := range steps {
		s.AccountableRoleID = codeOfRole[s.AccountableRoleID]
		s.RoleID = codeOfRole[s.RoleID]
		byCode[s.Title] = s
	}
	return byCode
}

func TestImportDocument_AnOldAccountableRowBecomesTheStepsAccountableRole(t *testing.T) {
	steps := importedSteps(t, legacyDocument(t, []map[string]string{
		{"work_item_id": "s1", "role_id": "r2", "label": "accountable"},
	}))
	if got := steps["No role of its own"].AccountableRoleID; got != "R-2" {
		t.Errorf("accountable role %q, want R-2", got)
	}
}

func TestImportDocument_AnOldRowThatRepeatsTheStepsOwnRoleIsDropped(t *testing.T) {
	steps := importedSteps(t, legacyDocument(t, []map[string]string{
		{"work_item_id": "s2", "role_id": "r1", "label": "accountable"},
	}))
	if got := steps["Own role"].AccountableRoleID; got != "" {
		t.Errorf("a duplicate of the step's own role became the accountable role: %q", got)
	}
	if got := steps["Own role"].RoleID; got != "R-1" {
		t.Errorf("the step lost its own role: %q", got)
	}
}

func TestImportDocument_OldRowsOfTheOtherThreeLabelsAreDropped(t *testing.T) {
	steps := importedSteps(t, legacyDocument(t, []map[string]string{
		{"work_item_id": "s1", "role_id": "r1", "label": "responsible"},
		{"work_item_id": "s1", "role_id": "r1", "label": "consulted"},
		{"work_item_id": "s1", "role_id": "r2", "label": "informed"},
	}))
	if got := steps["No role of its own"].AccountableRoleID; got != "" {
		t.Errorf("a row that is not accountable became the accountable role: %q", got)
	}
}

func TestImportDocument_TheFirstOldAccountableRowOfAStepWinsAndAFieldAlreadySetStays(t *testing.T) {
	steps := importedSteps(t, legacyDocument(t, []map[string]string{
		{"work_item_id": "s1", "role_id": "r2", "label": "accountable"},
		{"work_item_id": "s1", "role_id": "r1", "label": "accountable"},
		{"work_item_id": "s3", "role_id": "r1", "label": "accountable"},
	}))
	if got := steps["No role of its own"].AccountableRoleID; got != "R-2" {
		t.Errorf("first row did not win: %q", got)
	}
	if got := steps["Already accountable"].AccountableRoleID; got != "R-2" {
		t.Errorf("an accountable role already on the step was replaced: %q", got)
	}
}

func TestImportDocument_ANewFileCarriesTheFieldAndReadsNoRoleAssignments(t *testing.T) {
	steps := importedSteps(t, legacyDocument(t, nil))
	if got := steps["Already accountable"].AccountableRoleID; got != "R-2" {
		t.Errorf("accountable_role_id of a new file: %q", got)
	}
}
