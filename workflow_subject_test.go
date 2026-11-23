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

type subjectWorld struct {
	world  stepWorld
	record models.Record
}

func buildSubjectWorld(t *testing.T) subjectWorld {
	t.Helper()
	r := buildRowStepWorld(t)
	return subjectWorld{world: r.world, record: r.record}
}

func (s subjectWorld) workflowNaming(t *testing.T, name, subject string) (models.Workflow, error) {
	t.Helper()
	ctx := context.Background()
	made, err := s.world.mgr.CreateWorkflow(ctx, models.Workflow{DraftID: s.world.draftID, Name: name, SubjectRecord: subject})
	if err != nil {
		return made, err
	}
	linkWorkflowToObjective(t, s.world.mgr, s.world.draftID, made.ID, s.world.objective.ID)
	if _, err := s.world.mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: s.world.draftID, WorkflowID: made.ID, Title: "Decide", Ordinality: 1}); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	return made, nil
}

func TestWorkflowSubject_CreateKeepsTheRecordCodeAndReadsItBack(t *testing.T) {
	s := buildSubjectWorld(t)
	made, err := s.workflowNaming(t, "Pipeline", s.record.Code)
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	got, err := s.world.mgr.GetWorkflow(context.Background(), made.ID)
	if err != nil || got.SubjectRecord != s.record.Code {
		t.Fatalf("read back subject_record %q, %v, want %q", got.SubjectRecord, err, s.record.Code)
	}
}

func TestWorkflowSubject_CreateRefusesARecordTheDraftDoesNotKeepNamingIt(t *testing.T) {
	s := buildSubjectWorld(t)
	_, err := s.workflowNaming(t, "Pipeline", "RC-99")
	if !errors.Is(err, mwanachamaagency.ErrInvalidWorkflow) {
		t.Fatalf("CreateWorkflow error = %v, want ErrInvalidWorkflow", err)
	}
	if !strings.Contains(err.Error(), "RC-99") {
		t.Errorf("the refusal does not name the record: %v", err)
	}
}

func TestWorkflowSubject_ARecordOfAnotherDraftIsNotKnown(t *testing.T) {
	s := buildSubjectWorld(t)
	other, err := s.world.mgr.CreateDraft(context.Background(), "other", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	_, err = s.world.mgr.CreateWorkflow(context.Background(), models.Workflow{DraftID: other.ID, Name: "Pipeline", SubjectRecord: s.record.Code})
	if !errors.Is(err, mwanachamaagency.ErrInvalidWorkflow) {
		t.Fatalf("CreateWorkflow error = %v, want ErrInvalidWorkflow", err)
	}
}

func TestWorkflowSubject_NoSubjectBehavesAsToday(t *testing.T) {
	w := newStepWorld(t)
	made, err := w.mgr.CreateWorkflow(context.Background(), models.Workflow{DraftID: w.draftID, Name: "Plain"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	got, err := w.mgr.GetWorkflow(context.Background(), made.ID)
	if err != nil || got.SubjectRecord != "" {
		t.Fatalf("a workflow with no subject reads back %q, %v", got.SubjectRecord, err)
	}
}

func TestWorkflowSubject_UpdateSetsItRefusesAnUnknownOneAndClearsIt(t *testing.T) {
	s := buildSubjectWorld(t)
	ctx := context.Background()
	made, err := s.workflowNaming(t, "Pipeline", "")
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	var update models.Workflow
	consoleBody(t, &update, map[string]any{"name": "Pipeline", "subject_record": s.record.Code})
	update.ID = made.ID
	if _, err := s.world.mgr.UpdateWorkflow(ctx, update); err != nil {
		t.Fatalf("UpdateWorkflow set: %v", err)
	}
	if got, _ := s.world.mgr.GetWorkflow(ctx, made.ID); got.SubjectRecord != s.record.Code {
		t.Fatalf("subject_record after the update = %q, want %q", got.SubjectRecord, s.record.Code)
	}

	var unknown models.Workflow
	consoleBody(t, &unknown, map[string]any{"name": "Pipeline", "subject_record": "RC-99"})
	unknown.ID = made.ID
	_, err = s.world.mgr.UpdateWorkflow(ctx, unknown)
	if !errors.Is(err, mwanachamaagency.ErrInvalidWorkflow) || !strings.Contains(err.Error(), "RC-99") {
		t.Fatalf("UpdateWorkflow unknown error = %v, want ErrInvalidWorkflow naming RC-99", err)
	}

	var clear models.Workflow
	consoleBody(t, &clear, map[string]any{"name": "Pipeline", "subject_record": ""})
	clear.ID = made.ID
	if _, err := s.world.mgr.UpdateWorkflow(ctx, clear); err != nil {
		t.Fatalf("UpdateWorkflow clear: %v", err)
	}
	if got, _ := s.world.mgr.GetWorkflow(ctx, made.ID); got.SubjectRecord != "" {
		t.Errorf("subject_record after clearing = %q, want empty", got.SubjectRecord)
	}
}

func TestWorkflowSubject_AnUpdateThatDoesNotSendItKeepsIt(t *testing.T) {
	s := buildSubjectWorld(t)
	ctx := context.Background()
	made, err := s.workflowNaming(t, "Pipeline", s.record.Code)
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	var update models.Workflow
	consoleBody(t, &update, map[string]any{"name": "Pipeline renamed"})
	update.ID = made.ID
	if _, err := s.world.mgr.UpdateWorkflow(ctx, update); err != nil {
		t.Fatalf("UpdateWorkflow: %v", err)
	}
	if got, _ := s.world.mgr.GetWorkflow(ctx, made.ID); got.SubjectRecord != s.record.Code {
		t.Errorf("subject_record after a rename = %q, want %q", got.SubjectRecord, s.record.Code)
	}
}

func TestWorkflowSubject_TheIssueScanReportsAWorkflowWhoseRecordIsGone(t *testing.T) {
	s := buildSubjectWorld(t)
	ctx := context.Background()
	if _, err := s.workflowNaming(t, "Pipeline", s.record.Code); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	if err := s.world.mgr.DeleteRecord(ctx, s.record.ID); err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}
	issues, err := s.world.mgr.ListIssues(ctx, s.world.draftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	for _, issue := range issues {
		if issue.Severity == models.IssueSeverityError && strings.Contains(issue.Message, s.record.Code) && strings.Contains(issue.Message, "Pipeline") {
			return
		}
	}
	t.Errorf("no error names the workflow and the missing record %s: %+v", s.record.Code, issues)
}

func subjectDocument(subject string) models.AgencyDocument {
	return models.AgencyDocument{
		Agency: models.DocumentAgency{Name: "Subject agency"},
		Roles:  []models.DocumentRole{{ID: "role-scout", Code: "R-1", Name: "Scout"}},
		Workflows: []models.DocumentWorkflow{
			{ID: "wf-keeps", Code: "WF-1", Name: "Keeps tenders"},
			{ID: "wf-pipeline", Code: "WF-2", Name: "Pipeline", SubjectRecord: subject},
		},
		Records: []models.DocumentRecord{{
			ID: "rec-1", Code: "RC-1", WorkflowID: "wf-keeps", Name: "Tenders", RoleID: "role-scout", FormName: "Tender",
			Questions: []models.StepQuestion{{Prompt: "Tender title", AnswerType: models.AnswerFreeText, Required: true}},
		}},
	}
}

func workflowInDraft(t *testing.T, mgr mwanachamaagency.AgencyManager, draftID, name string) models.Workflow {
	t.Helper()
	all, err := mgr.ListWorkflows(context.Background(), draftID, mwanachamaagency.WorkflowFilter{})
	if err != nil {
		t.Fatalf("ListWorkflows: %v", err)
	}
	for _, w := range all {
		if w.Name == name {
			return w
		}
	}
	t.Fatalf("no workflow named %q", name)
	return models.Workflow{}
}

func TestImportDocument_KeepsTheSubjectRecordOfAWorkflow(t *testing.T) {
	mgr := newTestManager(t)
	out, err := mgr.ImportDocument(context.Background(), "subject", subjectDocument("RC-1"))
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	if got := workflowInDraft(t, mgr, out.Draft.ID, "Pipeline"); got.SubjectRecord != "RC-1" {
		t.Errorf("the imported workflow carries subject_record %q, want RC-1", got.SubjectRecord)
	}
	if got := workflowInDraft(t, mgr, out.Draft.ID, "Keeps tenders"); got.SubjectRecord != "" {
		t.Errorf("a workflow with no subject came back with %q", got.SubjectRecord)
	}
}

func TestImportDocument_RefusesAWorkflowWhoseSubjectRecordIsNotInTheFile(t *testing.T) {
	mgr := newTestManager(t)
	before, err := mgr.ListDrafts(context.Background(), mwanachamaagency.DraftFilter{})
	if err != nil {
		t.Fatalf("ListDrafts: %v", err)
	}
	_, err = mgr.ImportDocument(context.Background(), "subject", subjectDocument("RC-9"))
	if !errors.Is(err, mwanachamaagency.ErrInvalidWorkflow) {
		t.Fatalf("ImportDocument error = %v, want ErrInvalidWorkflow", err)
	}
	if !strings.Contains(err.Error(), "RC-9") {
		t.Errorf("the refusal does not name the missing record: %v", err)
	}
	after, listErr := mgr.ListDrafts(context.Background(), mwanachamaagency.DraftFilter{})
	if listErr != nil || len(after) != len(before) {
		t.Errorf("a refused import changed the drafts from %d to %d, %v", len(before), len(after), listErr)
	}
}

func TestImportDocument_AnOlderFileWithNoSubjectImportsAsBefore(t *testing.T) {
	mgr := newTestManager(t)
	out, err := mgr.ImportDocument(context.Background(), "plain", subjectDocument(""))
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	if got := workflowInDraft(t, mgr, out.Draft.ID, "Pipeline"); got.SubjectRecord != "" {
		t.Errorf("subject_record = %q, want empty", got.SubjectRecord)
	}
}

func TestWorkflowSubject_ForkKeepsItAndThePublishedDocumentCarriesIt(t *testing.T) {
	s := buildSubjectWorld(t)
	ctx := context.Background()
	if _, err := s.workflowNaming(t, "Pipeline", s.record.Code); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	fork, err := s.world.mgr.CreateDraft(ctx, "fork", s.world.draftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if got := workflowInDraft(t, s.world.mgr, fork.ID, "Pipeline"); got.SubjectRecord != s.record.Code {
		t.Errorf("the forked workflow carries %q, want %q", got.SubjectRecord, s.record.Code)
	}

	pub, err := s.world.mgr.PublishAgency(ctx, s.world.draftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	carried := ""
	for _, w := range pub.Document.Workflows {
		if w.Name == "Pipeline" {
			carried = w.SubjectRecord
		}
	}
	if carried != s.record.Code {
		t.Errorf("the published document carries %q, want %q", carried, s.record.Code)
	}
	raw, err := json.Marshal(pub.Document)
	if err != nil || !strings.Contains(string(raw), `"subject_record":"`+s.record.Code+`"`) {
		t.Errorf("the published document does not spell subject_record: %v", err)
	}
}

func TestWorkflowSubject_ChangingItIsAChangeToPublish(t *testing.T) {
	s := buildSubjectWorld(t)
	ctx := context.Background()
	made, err := s.workflowNaming(t, "Pipeline", "")
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	if _, err := s.world.mgr.PublishAgency(ctx, s.world.draftID, "", ""); err != nil {
		t.Fatalf("first PublishAgency: %v", err)
	}
	var update models.Workflow
	consoleBody(t, &update, map[string]any{"name": "Pipeline", "subject_record": s.record.Code})
	update.ID = made.ID
	if _, err := s.world.mgr.UpdateWorkflow(ctx, update); err != nil {
		t.Fatalf("UpdateWorkflow: %v", err)
	}
	if _, err := s.world.mgr.PublishAgency(ctx, s.world.draftID, "", ""); errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("naming a subject was not seen as a change: %v", err)
	}
}
