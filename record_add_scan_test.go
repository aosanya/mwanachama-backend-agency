package mwanachamaagency_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

const farmerField = "select_the_farmer"

func (w recordPermissionWorld) addScanErrors(t *testing.T) []models.Issue {
	t.Helper()
	issues, err := w.mgr.ListIssues(context.Background(), w.draftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	var found []models.Issue
	for _, issue := range issues {
		if issue.Severity == models.IssueSeverityError && issue.EntityType == "record" && issue.EntityID == w.record.ID {
			found = append(found, issue)
		}
	}
	return found
}

func (w recordPermissionWorld) roleCarrying(t *testing.T, name string, permissions ...string) {
	t.Helper()
	if _, err := w.mgr.CreateRole(context.Background(), models.Role{DraftID: w.draftID, Name: name, Permissions: permissions}); err != nil {
		t.Fatalf("CreateRole %s: %v", name, err)
	}
}

func TestIssueScan_ARecordNoRoleMayAddToIsAnErrorNamingTheRecordAndRefusesThePublish(t *testing.T) {
	w := newRecordPermissionWorld(t)
	w.roleCarrying(t, "Reader", w.fieldCode("view", "*"), w.fieldCode("edit", "*"), w.code("delete"))

	found := w.addScanErrors(t)
	if len(found) != 1 {
		t.Fatalf("errors of the record = %+v, want exactly one", found)
	}
	for _, part := range []string{w.record.Code, "Payments", "no role may add"} {
		if !strings.Contains(found[0].Message, part) {
			t.Errorf("finding = %q, want it to say %q", found[0].Message, part)
		}
	}
	_, err := w.mgr.PublishAgency(context.Background(), w.draftID, "", "")
	if !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) || !strings.Contains(err.Error(), w.record.Code) {
		t.Fatalf("PublishAgency = %v, want ErrPublishValidationFailed naming %s", err, w.record.Code)
	}
}

func TestIssueScan_ARecordWithNoRoleAtAllIsFlaggedOnce(t *testing.T) {
	w := newRecordPermissionWorld(t)
	if found := w.addScanErrors(t); len(found) != 1 || !strings.Contains(found[0].Message, "no role may add") {
		t.Fatalf("errors of the record = %+v, want one finding that no role may add", found)
	}
}

func TestIssueScan_ARoleWithAddStarClearsBothFindings(t *testing.T) {
	w := newRecordPermissionWorld(t)
	w.roleCarrying(t, "Clerk", w.fieldCode("add", "*"))
	if found := w.addScanErrors(t); len(found) != 0 {
		t.Fatalf("errors of the record = %+v, want none", found)
	}
	if _, err := w.mgr.PublishAgency(context.Background(), w.draftID, "", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
}

func TestIssueScan_OneAddFieldCodeLetsARoleAddButLeavesTheOtherRequiredFieldFlagged(t *testing.T) {
	w := newRecordPermissionWorld(t)
	w.roleCarrying(t, "Clerk", w.fieldCode("add", amountField))

	found := w.addScanErrors(t)
	if len(found) != 1 {
		t.Fatalf("errors of the record = %+v, want exactly one", found)
	}
	for _, part := range []string{w.record.Code, "Payments", farmerField, "required"} {
		if !strings.Contains(found[0].Message, part) {
			t.Errorf("finding = %q, want it to say %q", found[0].Message, part)
		}
	}
	if strings.Contains(found[0].Message, amountField) {
		t.Errorf("finding = %q, must not name the field a role can add", found[0].Message)
	}
	_, err := w.mgr.PublishAgency(context.Background(), w.draftID, "", "")
	if !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) || !strings.Contains(err.Error(), farmerField) {
		t.Fatalf("PublishAgency = %v, want ErrPublishValidationFailed naming %s", err, farmerField)
	}
}

func TestIssueScan_RequiredFieldsAddedByDifferentRolesScanClean(t *testing.T) {
	w := newRecordPermissionWorld(t)
	w.roleCarrying(t, "Farmer picker", w.fieldCode("add", farmerField))
	w.roleCarrying(t, "Amount clerk", w.fieldCode("add", amountField))
	if found := w.addScanErrors(t); len(found) != 0 {
		t.Fatalf("errors of the record = %+v, want none", found)
	}
}

func TestIssueScan_AnOptionalFieldNoRoleMayAddIsNotFlagged(t *testing.T) {
	w := newRecordPermissionWorld(t)
	ctx := context.Background()
	workflow := w.workflow(t, "Tenders")
	optional := ask("Closing note", models.AnswerFreeText)
	optional.Required = false
	record, err := w.mgr.CreateRecord(ctx, models.Record{
		DraftID: w.draftID, WorkflowID: workflow.ID, Name: "Tender", RoleID: w.roles[coordinator].ID,
		FormName: "Tender", Questions: []models.StepQuestion{ask("Enter the title", models.AnswerFreeText), optional},
	})
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	w.roleCarrying(t, "Payments clerk", w.fieldCode("add", "*"))
	w.roleCarrying(t, "Title clerk", "record."+record.Code+".add.enter_the_title")

	issues, err := w.mgr.ListIssues(ctx, w.draftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	for _, issue := range issues {
		if issue.EntityID == record.ID && issue.Severity == models.IssueSeverityError {
			t.Errorf("finding = %q, want none for a record whose only required field a role may add", issue.Message)
		}
	}
}

func TestIssueScan_AddWithoutViewEditWithoutViewAndDeleteWithoutViewAreNotFindings(t *testing.T) {
	w := newRecordPermissionWorld(t)
	w.roleCarrying(t, "Adder", w.fieldCode("add", "*"))
	w.roleCarrying(t, "Editor", w.fieldCode("edit", "*"))
	w.roleCarrying(t, "Remover", w.code("delete"))
	if found := w.addScanErrors(t); len(found) != 0 {
		t.Fatalf("errors of the record = %+v, want none", found)
	}
}

func TestIssueScan_AnAgencyWithoutRecordsScansAsBefore(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	if _, err := w.mgr.CreateRole(ctx, models.Role{DraftID: w.draftID, Name: "Reader", Permissions: []string{"report.read"}}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	issues, err := w.mgr.ListIssues(ctx, w.draftID, mwanachamaagency.IssueFilter{Severity: string(models.IssueSeverityError)})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("issues = %+v, want none", issues)
	}
}
