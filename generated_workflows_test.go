package mwanachamaagency_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type genWorld struct {
	rowStepWorld
	db *gorm.DB
	sp *spec.Spec
}

func newGenWorld(t *testing.T) genWorld {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	sp, err := mwanachamaagency.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaagency.Provision(db, sp); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	mgr, err := mwanachamaagency.NewAgencyManager(db, sp)
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	w := stepWorld{mgr: mgr, draftID: o.DraftID, objective: o, roles: map[string]models.Role{}}
	for _, name := range []string{coordinator, farmer, supplier} {
		r, err := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: name})
		if err != nil {
			t.Fatalf("CreateRole %s: %v", name, err)
		}
		w.roles[name] = r
	}
	home := w.workflow(t, "Tender pipeline")
	if _, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: w.draftID, WorkflowID: home.ID, Title: "Find the tender", Ordinality: 1}); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	record, err := mgr.CreateRecord(ctx, models.Record{
		DraftID: w.draftID, WorkflowID: home.ID, Name: "Tenders", RoleID: w.roles[coordinator].ID, FormName: "Tender",
		Questions: []models.StepQuestion{ask("Title", models.AnswerFreeText), ask("Value", models.AnswerCurrency)},
	})
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	mayAdd(t, mgr, record)
	return genWorld{rowStepWorld: rowStepWorld{world: w, record: record}, db: db, sp: sp}
}

func (g genWorld) generatedIn(t *testing.T, draftID string) map[string]models.Workflow {
	t.Helper()
	all, err := g.world.mgr.ListWorkflows(context.Background(), draftID, mwanachamaagency.WorkflowFilter{})
	if err != nil {
		t.Fatalf("ListWorkflows: %v", err)
	}
	out := map[string]models.Workflow{}
	for _, w := range all {
		if w.Generated {
			if _, twice := out[w.Code]; twice {
				t.Errorf("the generated workflow %s is held twice", w.Code)
			}
			out[w.Code] = w
		}
	}
	return out
}

func (g genWorld) stepOf(t *testing.T, draftID string, workflow models.Workflow) models.WorkItem {
	t.Helper()
	steps, err := g.world.mgr.ListWorkItems(context.Background(), draftID, mwanachamaagency.WorkItemFilter{})
	if err != nil {
		t.Fatalf("ListWorkItems: %v", err)
	}
	var found []models.WorkItem
	for _, s := range steps {
		if s.WorkflowID == workflow.ID {
			found = append(found, s)
		}
	}
	if len(found) != 1 {
		t.Fatalf("workflow %s holds %d steps, want 1", workflow.Code, len(found))
	}
	return found[0]
}

func (g genWorld) stripGenerated(t *testing.T) {
	t.Helper()
	wf, steps := tableOf(t, g.sp, "workflow"), tableOf(t, g.sp, "work_item")
	if err := g.db.Exec("delete from "+steps+" where workflow_id in (select id from "+wf+" where generated = ?)", true).Error; err != nil {
		t.Fatalf("strip steps: %v", err)
	}
	if err := g.db.Exec("delete from "+wf+" where generated = ?", true).Error; err != nil {
		t.Fatalf("strip workflows: %v", err)
	}
}

func (g genWorld) setRecordColumn(t *testing.T, column string, value any) {
	t.Helper()
	if err := g.db.Exec("update "+tableOf(t, g.sp, "record")+" set "+column+" = ? where id = ?", value, g.record.ID).Error; err != nil {
		t.Fatalf("set %s: %v", column, err)
	}
}

func findingsOf(t *testing.T, mgr mwanachamaagency.AgencyManager, draftID string, severity models.IssueSeverity, words ...string) []models.Issue {
	t.Helper()
	issues, err := mgr.ListIssues(context.Background(), draftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	var out []models.Issue
	for _, i := range issues {
		match := i.Severity == severity
		for _, w := range words {
			match = match && strings.Contains(i.Message, w)
		}
		if match {
			out = append(out, i)
		}
	}
	return out
}

func TestGenerated_ARecordHasAnAddAnEditAndADeleteWorkflowWithOneFormStepEach(t *testing.T) {
	g := newGenWorld(t)
	got := g.generatedIn(t, g.world.draftID)
	if len(got) != 3 {
		t.Fatalf("generated workflows = %d, want 3: %v", len(got), got)
	}
	want := map[string]struct{ name, action, form string }{
		g.record.Key + ".add":    {"Add a tender", "add", "Tender"},
		g.record.Key + ".edit":   {"Edit a tender", "edit", "Tender"},
		g.record.Key + ".delete": {"Delete a tender", "delete", "Why delete this tender"},
	}
	for code, w := range want {
		wf, ok := got[code]
		if !ok || wf.Name != w.name || wf.GeneratedFor != g.record.Key {
			t.Errorf("%s = %+v, want name %q generated for the key", code, wf, w.name)
			continue
		}
		step := g.stepOf(t, g.world.draftID, wf)
		if step.Kind != models.StepKindFillForm || step.RowRecord != g.record.Key || step.RowAction != w.action ||
			step.RoleID != g.record.RoleID || step.FormName != w.form || step.Ordinality != 1 {
			t.Errorf("%s step = %+v", code, step)
		}
		wantQuestions := g.record.Questions
		if w.action == "delete" {
			wantQuestions = []models.StepQuestion{{Prompt: "Why is it removed", AnswerType: models.AnswerFreeText, Required: true}}
		}
		if !reflect.DeepEqual(step.Questions, wantQuestions) {
			t.Errorf("%s questions = %+v, want %+v", code, step.Questions, wantQuestions)
		}
	}
}

func TestGenerated_TheNameComesFromTheItemNameThenTheFormName(t *testing.T) {
	g := newGenWorld(t)
	ctx := context.Background()
	record := g.record
	record.ItemName = "Office holder"
	if _, err := g.world.mgr.UpdateRecord(ctx, record); err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}
	got := g.generatedIn(t, g.world.draftID)
	if got[g.record.Key+".add"].Name != "Add an office holder" || got[g.record.Key+".edit"].Name != "Edit an office holder" || got[g.record.Key+".delete"].Name != "Delete an office holder" {
		t.Errorf("names = %q, %q, %q", got[g.record.Key+".add"].Name, got[g.record.Key+".edit"].Name, got[g.record.Key+".delete"].Name)
	}
}

func TestGenerated_RegeneratingKeepsTheIdsAndTheCodes(t *testing.T) {
	g := newGenWorld(t)
	ctx := context.Background()
	before := g.generatedIn(t, g.world.draftID)
	stepBefore := g.stepOf(t, g.world.draftID, before[g.record.Key+".add"])
	for i := 0; i < 2; i++ {
		if err := g.world.mgr.EnsureGeneratedWorkflows(ctx, g.world.draftID); err != nil {
			t.Fatalf("EnsureGeneratedWorkflows: %v", err)
		}
	}
	after := g.generatedIn(t, g.world.draftID)
	if len(after) != 3 {
		t.Fatalf("generated workflows = %d, want 3", len(after))
	}
	for code, w := range before {
		if after[code].ID != w.ID {
			t.Errorf("%s changed id from %s to %s", code, w.ID, after[code].ID)
		}
	}
	if stepAfter := g.stepOf(t, g.world.draftID, after[g.record.Key+".add"]); stepAfter.ID != stepBefore.ID || stepAfter.Code != stepBefore.Code {
		t.Errorf("the step changed from %+v to %+v", stepBefore, stepAfter)
	}
}

func TestGenerated_AChangedRecordUpdatesTheStepAndFormInPlace(t *testing.T) {
	g := newGenWorld(t)
	ctx := context.Background()
	before := g.generatedIn(t, g.world.draftID)
	record := g.record
	record.RoleID = g.world.roles[farmer].ID
	record.Questions = append(append([]models.StepQuestion{}, record.Questions...), ask("Closing date", models.AnswerDate))
	if _, err := g.world.mgr.UpdateRecord(ctx, record); err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}
	after := g.generatedIn(t, g.world.draftID)
	for _, code := range []string{g.record.Key + ".add", g.record.Key + ".edit"} {
		if after[code].ID != before[code].ID {
			t.Errorf("%s was replaced instead of updated", code)
		}
		step := g.stepOf(t, g.world.draftID, after[code])
		if step.RoleID != record.RoleID || len(step.Questions) != 3 {
			t.Errorf("%s step = role %s with %d questions", code, step.RoleID, len(step.Questions))
		}
	}
	del := g.stepOf(t, g.world.draftID, after[g.record.Key+".delete"])
	if del.RoleID != record.RoleID || len(del.Questions) != 1 {
		t.Errorf("the delete step follows the role and keeps its one question: %+v", del)
	}
}

func TestGenerated_EditAndDeleteAreHeldByTheirOwnRolesWhenTheRecordNamesThem(t *testing.T) {
	g := newGenWorld(t)
	ctx := context.Background()
	record := g.record
	record.EditRoleID = g.world.roles[supplier].ID
	record.DeleteRoleID = g.world.roles[farmer].ID
	if _, err := g.world.mgr.UpdateRecord(ctx, record); err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}
	got := g.generatedIn(t, g.world.draftID)
	roles := map[string]string{g.record.Key + ".add": record.RoleID, g.record.Key + ".edit": record.EditRoleID, g.record.Key + ".delete": record.DeleteRoleID}
	for code, role := range roles {
		if step := g.stepOf(t, g.world.draftID, got[code]); step.RoleID != role {
			t.Errorf("%s is held by %s, want %s", code, step.RoleID, role)
		}
	}
	record.EditRoleID = "no-such-role"
	if _, err := g.world.mgr.UpdateRecord(ctx, record); err == nil {
		t.Errorf("an edit role that is not a role of the draft must be refused")
	}
}

func TestGenerated_AGeneratedWorkflowCannotBeCreatedEditedRenamedOrDeletedThroughTheOperations(t *testing.T) {
	g := newGenWorld(t)
	ctx := context.Background()
	mgr := g.world.mgr
	add := g.generatedIn(t, g.world.draftID)[g.record.Key+".add"]
	step := g.stepOf(t, g.world.draftID, add)

	renamed := add
	renamed.Name = "Something else"
	if _, err := mgr.UpdateWorkflow(ctx, renamed); !errors.Is(err, mwanachamaagency.ErrGeneratedWorkflow) {
		t.Errorf("UpdateWorkflow: want ErrGeneratedWorkflow, got %v", err)
	}
	if err := mgr.DeleteWorkflow(ctx, add.ID); !errors.Is(err, mwanachamaagency.ErrGeneratedWorkflow) {
		t.Errorf("DeleteWorkflow: want ErrGeneratedWorkflow, got %v", err)
	}
	if _, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: g.world.draftID, WorkflowID: add.ID, Title: "Another", Ordinality: 2}); !errors.Is(err, mwanachamaagency.ErrGeneratedWorkflow) {
		t.Errorf("CreateWorkItem into it: want ErrGeneratedWorkflow, got %v", err)
	}
	step.Title = "Renamed"
	if _, err := mgr.UpdateWorkItem(ctx, step); !errors.Is(err, mwanachamaagency.ErrGeneratedWorkflow) {
		t.Errorf("UpdateWorkItem: want ErrGeneratedWorkflow, got %v", err)
	}
	if err := mgr.DeleteWorkItem(ctx, step.ID); !errors.Is(err, mwanachamaagency.ErrGeneratedWorkflow) {
		t.Errorf("DeleteWorkItem: want ErrGeneratedWorkflow, got %v", err)
	}
	plain := g.world.workflow(t, "Plain")
	mover, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: g.world.draftID, WorkflowID: plain.ID, Title: "Mover", Ordinality: 1})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	mover.WorkflowID = add.ID
	if _, err := mgr.UpdateWorkItem(ctx, mover); !errors.Is(err, mwanachamaagency.ErrGeneratedWorkflow) {
		t.Errorf("moving a step into it: want ErrGeneratedWorkflow, got %v", err)
	}

	forged, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: g.world.draftID, Name: "Forged", Generated: true, GeneratedFor: g.record.Key, ReplacedBy: "W-1"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	if forged.Generated || forged.GeneratedFor != "" || forged.ReplacedBy != "" {
		t.Errorf("a created workflow is never generated: %+v", forged)
	}
	forged.Generated = true
	updated, err := mgr.UpdateWorkflow(ctx, forged)
	if err != nil || updated.Generated {
		t.Errorf("an updated workflow stays not generated: %+v, %v", updated, err)
	}
}

func TestGenerated_ADesignedWorkflowMayRunAGeneratedOne(t *testing.T) {
	g := newGenWorld(t)
	ctx := context.Background()
	add := g.generatedIn(t, g.world.draftID)[g.record.Key+".add"]
	caller := g.world.workflow(t, "Win a client")
	made, err := g.world.mgr.CreateWorkItem(ctx, models.WorkItem{
		DraftID: g.world.draftID, WorkflowID: caller.ID, Title: "Add the client", Ordinality: 1, Kind: models.StepKindRunWorkflow, RunWorkflowID: add.ID,
	})
	if err != nil || made.RunWorkflowID != add.ID {
		t.Fatalf("run_workflow naming a generated workflow: %+v, %v", made, err)
	}
	fork, err := g.world.mgr.CreateDraft(ctx, "fork", g.world.draftID, "draft")
	if err != nil {
		t.Fatalf("fork: %v", err)
	}
	forkedAdd := g.generatedIn(t, fork.ID)[g.record.Key+".add"]
	if got := copiedSteps(t, g.world.mgr, fork.ID)["Add the client"]; got.RunWorkflowID != forkedAdd.ID || forkedAdd.ID == add.ID {
		t.Errorf("the fork's step runs %s, want the fork's generated workflow %s", got.RunWorkflowID, forkedAdd.ID)
	}
}

func TestGenerated_AForkAnImportAndAPublicationGiveEveryRecordItsThree(t *testing.T) {
	g := newGenWorld(t)
	ctx := context.Background()
	mgr := g.world.mgr
	pub, err := mgr.PublishAgency(ctx, g.world.draftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	inDocument := 0
	for _, w := range pub.Document.Workflows {
		if w.Generated && w.GeneratedFor == g.record.Key {
			inDocument++
		}
	}
	if inDocument != 3 {
		t.Errorf("the publication document carries the three generated workflows, found %d", inDocument)
	}

	g.stripGenerated(t)
	fork, err := mgr.CreateDraft(ctx, "fork of an old draft", g.world.draftID, "draft")
	if err != nil {
		t.Fatalf("fork: %v", err)
	}
	if got := g.generatedIn(t, fork.ID); len(got) != 3 {
		t.Errorf("a fork of a draft with none regenerates them: %d", len(got))
	}

	var older models.AgencyDocument = pub.Document
	older.Workflows = nil
	for _, w := range pub.Document.Workflows {
		if !w.Generated {
			older.Workflows = append(older.Workflows, w)
		}
	}
	generatedIDs := map[string]bool{}
	for _, w := range pub.Document.Workflows {
		if w.Generated {
			generatedIDs[w.ID] = true
		}
	}
	older.WorkItems = nil
	for _, s := range pub.Document.WorkItems {
		if !generatedIDs[s.WorkflowID] {
			older.WorkItems = append(older.WorkItems, s)
		}
	}
	imported, err := mgr.ImportDocument(ctx, "older agency", older)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	if got := g.generatedIn(t, imported.Draft.ID); len(got) != 3 {
		t.Errorf("an import of an agency with records and no generated workflows regenerates them: %d", len(got))
	}

	g.stripGenerated(t)
	if _, err := mgr.PublishAgency(ctx, g.world.draftID, "", ""); !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Errorf("regenerating alone does not change the content hash, got %v", err)
	}
	g.stripGenerated(t)
	changed := g.world.workflow(t, "A change to publish")
	if _, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: g.world.draftID, WorkflowID: changed.ID, Title: "Do it", Ordinality: 1}); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, g.world.draftID, "", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if got := g.generatedIn(t, g.world.draftID); len(got) != 3 {
		t.Errorf("publishing regenerates the missing workflows: %d", len(got))
	}

	restored, err := mgr.CreateDraft(ctx, "restored", pub.Tag, "publication")
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if got := g.generatedIn(t, restored.ID); len(got) != 3 {
		t.Errorf("a draft restored from a publication holds the three: %d", len(got))
	}
}

func TestGenerated_ARecordChangeChangesTheHashAndTheGeneratedWorkflowsDoNotCountTwice(t *testing.T) {
	g := newGenWorld(t)
	ctx := context.Background()
	if _, err := g.world.mgr.PublishAgency(ctx, g.world.draftID, "", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if _, err := g.world.mgr.PublishAgency(ctx, g.world.draftID, "", ""); !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("nothing changed: want ErrNoChangesDetected, got %v", err)
	}
	record := g.record
	record.Questions = append(append([]models.StepQuestion{}, record.Questions...), ask("Closing date", models.AnswerDate))
	if _, err := g.world.mgr.UpdateRecord(ctx, record); err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}
	if _, err := g.world.mgr.PublishAgency(ctx, g.world.draftID, "", ""); err != nil {
		t.Errorf("a changed record is a change to publish, got %v", err)
	}
}

func TestGenerated_DeletingARecordOrItsWorkflowRemovesItsGeneratedWorkflows(t *testing.T) {
	g := newGenWorld(t)
	ctx := context.Background()
	if err := g.world.mgr.DeleteRecord(ctx, g.record.ID); err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}
	if got := g.generatedIn(t, g.world.draftID); len(got) != 0 {
		t.Errorf("a deleted record leaves no generated workflow: %v", got)
	}
	steps, _ := g.world.mgr.ListWorkItems(ctx, g.world.draftID, mwanachamaagency.WorkItemFilter{})
	for _, s := range steps {
		if s.RowRecord != "" {
			t.Errorf("a row step outlived its record: %+v", s)
		}
	}

	second := newGenWorld(t)
	if err := second.world.mgr.DeleteWorkflow(ctx, second.record.WorkflowID); err != nil {
		t.Fatalf("DeleteWorkflow: %v", err)
	}
	if got := second.generatedIn(t, second.world.draftID); len(got) != 0 {
		t.Errorf("deleting the workflow that keeps the record removes its generated workflows: %v", got)
	}
}

func TestGenerated_ThePublishGuardNamesAMissingAlteredOrStaleGeneratedWorkflow(t *testing.T) {
	wf := func(g genWorld) string { return tableOf(t, g.sp, "workflow") }
	steps := func(g genWorld) string { return tableOf(t, g.sp, "work_item") }

	missing := newGenWorld(t)
	if err := missing.db.Exec("delete from "+wf(missing)+" where code = ?", missing.record.Key+".edit").Error; err != nil {
		t.Fatal(err)
	}
	if got := findingsOf(t, missing.world.mgr, missing.world.draftID, models.IssueSeverityError, missing.record.Key, missing.record.Key+".edit"); len(got) == 0 {
		t.Errorf("a missing generated workflow is an error naming the record and the workflow")
	}

	altered := newGenWorld(t)
	if err := altered.db.Exec("update "+steps(altered)+" set row_action = ? where workflow_id in (select id from "+wf(altered)+" where code = ?)", "delete", altered.record.Key+".add").Error; err != nil {
		t.Fatal(err)
	}
	if got := findingsOf(t, altered.world.mgr, altered.world.draftID, models.IssueSeverityError, altered.record.Key+".add"); len(got) == 0 {
		t.Errorf("an altered generated step is an error naming the workflow")
	}

	stale := newGenWorld(t)
	stale.setRecordColumn(t, "questions", `[{"prompt":"Title","answer_type":"free_text","required":true}]`)
	if got := findingsOf(t, stale.world.mgr, stale.world.draftID, models.IssueSeverityError, stale.record.Key); len(got) == 0 {
		t.Errorf("a stale generated workflow, made from questions the record no longer has, is an error")
	}

	reroled := newGenWorld(t)
	reroled.setRecordColumn(t, "edit_role_id", reroled.world.roles[farmer].ID)
	if got := findingsOf(t, reroled.world.mgr, reroled.world.draftID, models.IssueSeverityError, reroled.record.Key+".edit"); len(got) == 0 {
		t.Errorf("a generated edit workflow held by another role than the record's edit role is an error")
	}
}

func TestGenerated_ARecordWithNoAddingRoleBlocksThePublishWithAFinding(t *testing.T) {
	g := newGenWorld(t)
	ctx := context.Background()
	g.setRecordColumn(t, "role_id", "")
	if got := findingsOf(t, g.world.mgr, g.world.draftID, models.IssueSeverityError, g.record.Code, "adding role"); len(got) == 0 {
		t.Errorf("a record with no adding role is an error")
	}
	if _, err := g.world.mgr.PublishAgency(ctx, g.world.draftID, "", ""); !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) {
		t.Errorf("publishing is refused, got %v", err)
	}
}

func TestGenerated_ADuplicatedGeneratedWorkflowIsAFindingAndRegeneratingKeepsOne(t *testing.T) {
	g := newGenWorld(t)
	ctx := context.Background()
	wfTable := tableOf(t, g.sp, "workflow")
	original := g.generatedIn(t, g.world.draftID)[g.record.Key+".add"]
	var copyRow map[string]any
	if err := g.db.Table(wfTable).Where("id = ?", original.ID).Take(&copyRow).Error; err != nil {
		t.Fatal(err)
	}
	copyRow["id"] = "forged-duplicate"
	if err := g.db.Table(wfTable).Create(copyRow).Error; err != nil {
		t.Fatal(err)
	}
	if got := findingsOf(t, g.world.mgr, g.world.draftID, models.IssueSeverityError, g.record.Key+".add"); len(got) == 0 {
		t.Errorf("a duplicated generated workflow is an error")
	}
	if err := g.world.mgr.EnsureGeneratedWorkflows(ctx, g.world.draftID); err != nil {
		t.Fatalf("EnsureGeneratedWorkflows: %v", err)
	}
	if got := g.generatedIn(t, g.world.draftID); len(got) != 3 || got[g.record.Key+".add"].ID != original.ID {
		t.Errorf("regenerating keeps the first and removes the copy: %v", got)
	}
	if got := findingsOf(t, g.world.mgr, g.world.draftID, models.IssueSeverityError, g.record.Key); len(got) != 0 {
		t.Errorf("a regenerated draft passes the guard: %v", got)
	}
}

func TestGenerated_ANormalDraftPassesTheGuardAndGeneratedWorkflowsNeedNoObjective(t *testing.T) {
	g := newGenWorld(t)
	if got := findingsOf(t, g.world.mgr, g.world.draftID, models.IssueSeverityError); len(got) != 0 {
		t.Errorf("a normal draft has no errors: %v", got)
	}
	if _, err := g.world.mgr.PublishAgency(context.Background(), g.world.draftID, "", ""); err != nil {
		t.Errorf("PublishAgency: %v", err)
	}
}

func TestGenerated_AWorkflowCanStandInForOneOfTheThreeAndTheGeneratedOneSaysSo(t *testing.T) {
	g := newGenWorld(t)
	ctx := context.Background()
	mgr := g.world.mgr
	designed := g.world.workflow(t, "Win a client")
	add := g.generatedIn(t, g.world.draftID)[g.record.Key+".add"]

	record := g.record
	record.AddReplacedBy = "W-99"
	if _, err := mgr.UpdateRecord(ctx, record); !errors.Is(err, mwanachamaagency.ErrInvalidRecord) {
		t.Errorf("a workflow that does not exist: want ErrInvalidRecord, got %v", err)
	}
	record.AddReplacedBy = add.Code
	if _, err := mgr.UpdateRecord(ctx, record); !errors.Is(err, mwanachamaagency.ErrInvalidRecord) {
		t.Errorf("a generated workflow cannot stand in for itself: want ErrInvalidRecord, got %v", err)
	}
	record.AddReplacedBy = designed.Code
	if _, err := mgr.UpdateRecord(ctx, record); err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}
	got := g.generatedIn(t, g.world.draftID)
	if got[g.record.Key+".add"].ReplacedBy != designed.Code || got[g.record.Key+".edit"].ReplacedBy != "" {
		t.Errorf("replaced_by = %q and %q", got[g.record.Key+".add"].ReplacedBy, got[g.record.Key+".edit"].ReplacedBy)
	}
	if warn := findingsOf(t, mgr, g.world.draftID, models.IssueSeverityWarning, designed.Code, g.record.Key+".add"); len(warn) == 0 {
		t.Errorf("a stand-in with no run_workflow step naming the generated workflow is a warning")
	}

	if _, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: g.world.draftID, WorkflowID: designed.ID, Title: "Add the client", Ordinality: 1, Kind: models.StepKindRunWorkflow, RunWorkflowID: got[g.record.Key+".add"].ID}); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	if warn := findingsOf(t, mgr, g.world.draftID, models.IssueSeverityWarning, designed.Code, g.record.Key+".add"); len(warn) != 0 {
		t.Errorf("a stand-in that runs the generated workflow has no warning: %v", warn)
	}
	if errs := findingsOf(t, mgr, g.world.draftID, models.IssueSeverityError); len(errs) != 0 {
		t.Errorf("the generated workflow stays and the guard still passes: %v", errs)
	}
}

func TestGenerated_ADatabaseFromBeforeTheColumnsExistedReadsAndRegeneratesOnTheNextFork(t *testing.T) {
	g := newGenWorld(t)
	ctx := context.Background()
	wf := tableOf(t, g.sp, "workflow")
	g.stripGenerated(t)
	for _, column := range []string{"generated", "generated_for", "replaced_by"} {
		if err := g.db.Exec("alter table " + wf + " drop column " + column).Error; err != nil {
			t.Fatalf("drop %s: %v", column, err)
		}
	}
	if err := mwanachamaagency.Provision(g.db, g.sp); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	old, err := g.world.mgr.ListWorkflows(ctx, g.world.draftID, mwanachamaagency.WorkflowFilter{})
	if err != nil || len(old) == 0 {
		t.Fatalf("workflows of an old database read back: %v, %v", old, err)
	}
	for _, w := range old {
		if w.Generated {
			t.Errorf("an old workflow is not generated: %+v", w)
		}
	}
	fork, err := g.world.mgr.CreateDraft(ctx, "fork", g.world.draftID, "draft")
	if err != nil {
		t.Fatalf("fork: %v", err)
	}
	if got := g.generatedIn(t, fork.ID); len(got) != 3 {
		t.Errorf("the next fork of an old draft regenerates the three: %d", len(got))
	}
}
