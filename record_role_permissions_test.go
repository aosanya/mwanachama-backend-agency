package mwanachamaagency_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type recordPermissionWorld struct {
	stepWorld
	record models.Record
}

func newRecordPermissionWorld(t *testing.T) recordPermissionWorld {
	t.Helper()
	w := newStepWorld(t)
	workflow := w.workflow(t, "Group order")
	step := models.WorkItem{DraftID: w.draftID, WorkflowID: workflow.ID, Title: "Greet the member", Ordinality: 1}
	if _, err := w.mgr.CreateWorkItem(context.Background(), step); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	record, err := w.mgr.CreateRecord(context.Background(), paymentsRecord(w, workflow))
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	return recordPermissionWorld{stepWorld: w, record: record}
}

func (w recordPermissionWorld) code(action string) string {
	return mwanachamaagency.RecordPermissionCode(w.record.Code, action)
}

func (w recordPermissionWorld) refusalNames(t *testing.T, err error, role, code string) {
	t.Helper()
	if !errors.Is(err, mwanachamaagency.ErrUnknownPermission) {
		t.Fatalf("err = %v, want ErrUnknownPermission", err)
	}
	if !strings.Contains(err.Error(), role) || !strings.Contains(err.Error(), code) {
		t.Fatalf("err = %q, want it to name the role %q and the code %q", err, role, code)
	}
}

func TestRolePermissions_CreateTakesTheCodesOfARecordOfTheDraftBesideCatalogueCodes(t *testing.T) {
	w := newRecordPermissionWorld(t)
	want := []string{w.fieldCode("add", amountField), "report.read", w.fieldCode("view", amountField)}
	created, err := w.mgr.CreateRole(context.Background(), models.Role{DraftID: w.draftID, Name: "Clerk", Permissions: want})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	got, err := w.mgr.GetRole(context.Background(), created.ID)
	if err != nil || !reflect.DeepEqual(got.Permissions, want) {
		t.Fatalf("GetRole = %+v, %v; want permissions %v", got, err, want)
	}
}

func TestRolePermissions_CreateRefusesARecordCodeNoRecordGives(t *testing.T) {
	w := newRecordPermissionWorld(t)
	for _, code := range []string{"record.nothing.add", w.code("sell"), "record." + w.record.Code} {
		_, err := w.mgr.CreateRole(context.Background(), models.Role{DraftID: w.draftID, Name: "Clerk", Permissions: []string{"report.read", code}})
		w.refusalNames(t, err, "Clerk", code)
	}
	roles, err := w.mgr.ListRoles(context.Background(), w.draftID, mwanachamaagency.RoleFilter{Name: "Clerk"})
	if err != nil || len(roles) != 0 {
		t.Fatalf("a refused role was kept: %+v, %v", roles, err)
	}
}

func TestRolePermissions_UpdateTakesAndRefusesTheSameCodes(t *testing.T) {
	w := newRecordPermissionWorld(t)
	ctx := context.Background()
	role := w.roles[farmer]
	role.Permissions = []string{w.fieldCode("edit", amountField), "report.read"}
	if _, err := w.mgr.UpdateRole(ctx, role); err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	role.Permissions = []string{"report.read", "record.nothing.edit"}
	_, err := w.mgr.UpdateRole(ctx, role)
	w.refusalNames(t, err, farmer, "record.nothing.edit")
	kept, err := w.mgr.GetRole(ctx, role.ID)
	if err != nil || !reflect.DeepEqual(kept.Permissions, []string{w.fieldCode("edit", amountField), "report.read"}) {
		t.Fatalf("GetRole = %+v, %v; want the accepted list kept", kept, err)
	}
}

func TestRolePermissions_SetRolePermissionsTakesAndRefusesTheSameCodes(t *testing.T) {
	w := newRecordPermissionWorld(t)
	ctx := context.Background()
	role := w.roles[supplier]
	got, err := w.mgr.SetRolePermissions(ctx, role.ID, []string{w.code("delete"), "report.read"})
	if err != nil || !reflect.DeepEqual(got.Permissions, []string{w.code("delete"), "report.read"}) {
		t.Fatalf("SetRolePermissions = %+v, %v", got, err)
	}
	_, err = w.mgr.SetRolePermissions(ctx, role.ID, []string{"record.nothing.delete"})
	w.refusalNames(t, err, supplier, "record.nothing.delete")
}

func TestRolePermissions_ARecordOfAnotherDraftGivesNoCode(t *testing.T) {
	w := newRecordPermissionWorld(t)
	ctx := context.Background()
	other, err := w.mgr.CreateDraft(ctx, "another draft", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	_, err = w.mgr.CreateRole(ctx, models.Role{DraftID: other.ID, Name: "Clerk", Permissions: []string{w.fieldCode("add", amountField)}})
	w.refusalNames(t, err, "Clerk", w.fieldCode("add", amountField))
}

func TestRolePermissions_CatalogueOnlyRolesBehaveAsBefore(t *testing.T) {
	w := newRecordPermissionWorld(t)
	ctx := context.Background()
	created, err := w.mgr.CreateRole(ctx, models.Role{DraftID: w.draftID, Name: "Reader", Permissions: []string{"report.read", "teleport.anyone"}})
	if err != nil || !reflect.DeepEqual(created.Permissions, []string{"report.read", "teleport.anyone"}) {
		t.Fatalf("CreateRole = %+v, %v; the catalogue check stays at the publish scan", created, err)
	}
}

func publishedWithClerk(t *testing.T, w recordPermissionWorld, permissions []string) models.AgencyPublication {
	t.Helper()
	ctx := context.Background()
	if _, err := w.mgr.CreateRole(ctx, models.Role{DraftID: w.draftID, Name: "Clerk", Permissions: permissions}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	mayAdd(t, w.mgr, w.record)
	pub, err := w.mgr.PublishAgency(ctx, w.draftID, "records-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	return pub
}

func clerkOf(t *testing.T, roles []models.Role) models.Role {
	t.Helper()
	for _, r := range roles {
		if r.Name == "Clerk" {
			return r
		}
	}
	t.Fatalf("no role Clerk in %+v", roles)
	return models.Role{}
}

func TestRolePermissions_PublishCarriesTheRecordCodesAndAForkKeepsThem(t *testing.T) {
	w := newRecordPermissionWorld(t)
	ctx := context.Background()
	want := []string{w.fieldCode("add", amountField), w.fieldCode("view", amountField), "report.read"}
	pub := publishedWithClerk(t, w, want)

	var inDocument []string
	for _, r := range pub.Document.Roles {
		if r.Name == "Clerk" {
			inDocument = r.Permissions
		}
	}
	if !reflect.DeepEqual(inDocument, want) {
		t.Fatalf("document Clerk permissions = %v, want %v", inDocument, want)
	}

	for _, fork := range []struct{ from, kind string }{{"records-1", "publication"}, {w.draftID, "draft"}} {
		draft, err := w.mgr.CreateDraft(ctx, "revisit", fork.from, fork.kind)
		if err != nil {
			t.Fatalf("CreateDraft from a %s: %v", fork.kind, err)
		}
		roles, err := w.mgr.ListRoles(ctx, draft.ID, mwanachamaagency.RoleFilter{})
		if err != nil || !reflect.DeepEqual(clerkOf(t, roles).Permissions, want) {
			t.Fatalf("fork from a %s: roles = %+v, %v; want Clerk with %v", fork.kind, roles, err, want)
		}
	}
}

func TestRolePermissions_ImportTakesTheCodesOfTheDocumentsRecordsAndRefusesTheRest(t *testing.T) {
	w := newRecordPermissionWorld(t)
	ctx := context.Background()
	pub := publishedWithClerk(t, w, []string{w.fieldCode("add", amountField), "report.read"})

	imported, err := w.mgr.ImportDocument(ctx, "same document", pub.Document)
	if err != nil {
		t.Fatalf("ImportDocument of a document whose codes its records give: %v", err)
	}
	roles, err := w.mgr.ListRoles(ctx, imported.Draft.ID, mwanachamaagency.RoleFilter{})
	if err != nil || !reflect.DeepEqual(clerkOf(t, roles).Permissions, []string{w.fieldCode("add", amountField), "report.read"}) {
		t.Fatalf("imported roles = %+v, %v", roles, err)
	}

	for _, code := range []string{"record.nothing.add", w.code("sell")} {
		bad := pub.Document
		bad.Roles = append([]models.DocumentRole(nil), pub.Document.Roles...)
		for i := range bad.Roles {
			if bad.Roles[i].Name == "Clerk" {
				bad.Roles[i].Permissions = []string{"report.read", code}
			}
		}
		_, err := w.mgr.ImportDocument(ctx, "bad document", bad)
		w.refusalNames(t, err, "Clerk", code)
	}
}

func TestRolePermissions_PublishAcceptsTheCodesOfARecordOfTheDraft(t *testing.T) {
	w := newRecordPermissionWorld(t)
	publishedWithClerk(t, w, []string{w.fieldCode("add", amountField), w.fieldCode("edit", amountField), w.code("delete"), w.fieldCode("view", amountField)})
}

func TestRolePermissions_DeletingARecordLeavesTheCodesAsAnErrorIssueAndRefusesThePublish(t *testing.T) {
	w := newRecordPermissionWorld(t)
	ctx := context.Background()
	role, err := w.mgr.CreateRole(ctx, models.Role{DraftID: w.draftID, Name: "Clerk", Permissions: []string{w.fieldCode("add", amountField), "report.read"}})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	dangling := w.fieldCode("add", amountField)
	if err := w.mgr.DeleteRecord(ctx, w.record.ID); err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}

	kept, err := w.mgr.GetRole(ctx, role.ID)
	if err != nil || !reflect.DeepEqual(kept.Permissions, []string{dangling, "report.read"}) {
		t.Fatalf("GetRole = %+v, %v; the role keeps what it carried", kept, err)
	}

	issues, err := w.mgr.ListIssues(ctx, w.draftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	var found []models.Issue
	for _, issue := range issues {
		if issue.EntityID == role.ID {
			found = append(found, issue)
		}
	}
	if len(found) != 1 || found[0].Severity != models.IssueSeverityError {
		t.Fatalf("issues of the role = %+v, want one error", found)
	}
	for _, part := range []string{role.Code, "Clerk", dangling, "no record gives"} {
		if !strings.Contains(found[0].Message, part) {
			t.Errorf("issue = %q, want it to say %q", found[0].Message, part)
		}
	}

	_, err = w.mgr.PublishAgency(ctx, w.draftID, "", "")
	if !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) || !strings.Contains(err.Error(), "Clerk") || !strings.Contains(err.Error(), dangling) {
		t.Fatalf("PublishAgency = %v, want ErrPublishValidationFailed naming the role and %s", err, dangling)
	}
}

func TestRolePermissions_ACodeNoRecordGivesIsNotCalledAnUnknownCatalogueCode(t *testing.T) {
	w := newRecordPermissionWorld(t)
	ctx := context.Background()
	if _, err := w.mgr.CreateRole(ctx, models.Role{DraftID: w.draftID, Name: "Reader", Permissions: []string{"teleport.anyone"}}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	_, err := w.mgr.PublishAgency(ctx, w.draftID, "", "")
	if err == nil || !strings.Contains(err.Error(), "unknown permission") || strings.Contains(err.Error(), "no record gives") {
		t.Fatalf("PublishAgency = %v, want the catalogue wording kept for a catalogue code", err)
	}
}

func (w recordPermissionWorld) fieldCode(action, field string) string {
	return "record." + w.record.Code + "." + action + "." + field
}

const amountField = "enter_the_amount_paid"

func TestRolePermissions_FieldLevelCodesStarsAndDeleteAreTakenWithNoViewNeeded(t *testing.T) {
	w := newRecordPermissionWorld(t)
	ctx := context.Background()
	want := []string{
		w.fieldCode("add", amountField), w.fieldCode("edit", amountField), w.fieldCode("edit", "*"),
		w.fieldCode("view", "select_the_farmer"), w.fieldCode("add", "*"), w.fieldCode("view", "*"), "record." + w.record.Code + ".delete",
	}
	created, err := w.mgr.CreateRole(ctx, models.Role{DraftID: w.draftID, Name: "Clerk", Permissions: want})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	got, err := w.mgr.GetRole(ctx, created.ID)
	if err != nil || !reflect.DeepEqual(got.Permissions, want) {
		t.Fatalf("GetRole = %+v, %v; want permissions %v", got, err, want)
	}
	for _, only := range [][]string{{w.fieldCode("edit", amountField)}, {w.fieldCode("add", amountField)}} {
		if _, err := w.mgr.CreateRole(ctx, models.Role{DraftID: w.draftID, Name: "Blind " + only[0], Permissions: only}); err != nil {
			t.Fatalf("CreateRole with %v and no view: %v", only, err)
		}
	}
}

func TestRolePermissions_ThePlainAddEditAndViewCodesAreRefused(t *testing.T) {
	w := newRecordPermissionWorld(t)
	for _, action := range []string{"add", "edit", "view"} {
		code := "record." + w.record.Code + "." + action
		_, err := w.mgr.CreateRole(context.Background(), models.Role{DraftID: w.draftID, Name: "Clerk", Permissions: []string{code}})
		w.refusalNames(t, err, "Clerk", code)
	}
}

func TestRolePermissions_AFieldNoRecordHasAndADeleteWithAFieldPartAreRefused(t *testing.T) {
	w := newRecordPermissionWorld(t)
	for _, code := range []string{
		w.fieldCode("edit", "nothing"), w.fieldCode("add", "Enter the amount paid"), w.fieldCode("view", ""),
		w.fieldCode("delete", amountField), w.fieldCode("delete", "*"), w.fieldCode("edit", "*") + ".more",
	} {
		_, err := w.mgr.CreateRole(context.Background(), models.Role{DraftID: w.draftID, Name: "Clerk", Permissions: []string{code}})
		w.refusalNames(t, err, "Clerk", code)
	}
}

func TestRolePermissions_UpdateAndSetFollowTheFieldLevelCodes(t *testing.T) {
	w := newRecordPermissionWorld(t)
	ctx := context.Background()
	role := w.roles[farmer]
	role.Permissions = []string{w.fieldCode("edit", amountField)}
	if _, err := w.mgr.UpdateRole(ctx, role); err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	role.Permissions = []string{w.fieldCode("edit", "nothing")}
	_, err := w.mgr.UpdateRole(ctx, role)
	w.refusalNames(t, err, farmer, w.fieldCode("edit", "nothing"))
	got, err := w.mgr.SetRolePermissions(ctx, w.roles[supplier].ID, []string{w.fieldCode("view", "*"), "record." + w.record.Code + ".delete"})
	if err != nil || len(got.Permissions) != 2 {
		t.Fatalf("SetRolePermissions = %+v, %v", got, err)
	}
}

func TestRolePermissions_ImportReadsTheFieldCodesOfAnOldDocumentWithNoStoredCodes(t *testing.T) {
	w := newRecordPermissionWorld(t)
	ctx := context.Background()
	pub := publishedWithClerk(t, w, []string{w.fieldCode("edit", amountField), w.fieldCode("view", "*")})
	old := pub.Document
	old.Records = append([]models.DocumentRecord(nil), pub.Document.Records...)
	for i := range old.Records {
		old.Records[i].Questions = withoutIdentity(old.Records[i].Questions)
	}
	if _, err := w.mgr.ImportDocument(ctx, "old document", old); err != nil {
		t.Fatalf("ImportDocument of a document with no field codes: %v", err)
	}
	bad := old
	bad.Roles = append([]models.DocumentRole(nil), old.Roles...)
	for i := range bad.Roles {
		if bad.Roles[i].Name == "Clerk" {
			bad.Roles[i].Permissions = []string{w.fieldCode("edit", "nothing")}
		}
	}
	_, err := w.mgr.ImportDocument(ctx, "bad document", bad)
	w.refusalNames(t, err, "Clerk", w.fieldCode("edit", "nothing"))
}

func TestRolePermissions_ARemovedFieldLeavesItsCodeAsAnErrorWhileTheStarKeepsHolding(t *testing.T) {
	w := newRecordPermissionWorld(t)
	ctx := context.Background()
	named := w.fieldCode("view", amountField)
	star := w.fieldCode("view", "*")
	role, err := w.mgr.CreateRole(ctx, models.Role{DraftID: w.draftID, Name: "Clerk", Permissions: []string{named, star}})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	record := w.record
	record.Questions = record.Questions[:1]
	if _, err := w.mgr.UpdateRecord(ctx, record); err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}
	issues, err := w.mgr.ListIssues(ctx, w.draftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	var found []models.Issue
	for _, issue := range issues {
		if issue.EntityID == role.ID {
			found = append(found, issue)
		}
	}
	if len(found) != 1 || found[0].Severity != models.IssueSeverityError || !strings.Contains(found[0].Message, named) || strings.Contains(found[0].Message, star) {
		t.Fatalf("issues of the role = %+v, want one error naming %s and not the star", found, named)
	}
}

func (w recordPermissionWorld) refusalSays(t *testing.T, err error, parts ...string) {
	t.Helper()
	if !errors.Is(err, mwanachamaagency.ErrUnknownPermission) {
		t.Fatalf("err = %v, want ErrUnknownPermission", err)
	}
	for _, part := range parts {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("err = %q, want it to say %q", err, part)
		}
	}
}

func (w recordPermissionWorld) refusedOnEveryPath(t *testing.T, permissions []string) map[string]error {
	t.Helper()
	ctx := context.Background()
	refusals := map[string]error{}
	_, refusals["create"] = w.mgr.CreateRole(ctx, models.Role{DraftID: w.draftID, Name: "Clerk", Permissions: permissions})
	role := w.roles[farmer]
	role.Permissions = permissions
	_, refusals["update"] = w.mgr.UpdateRole(ctx, role)
	_, refusals["set"] = w.mgr.SetRolePermissions(ctx, w.roles[supplier].ID, permissions)
	pub := publishedWithClerk(t, w, []string{"report.read"})
	bad := pub.Document
	bad.Roles = append([]models.DocumentRole(nil), pub.Document.Roles...)
	for i := range bad.Roles {
		if bad.Roles[i].Name == "Clerk" {
			bad.Roles[i].Permissions = permissions
		}
	}
	_, refusals["import"] = w.mgr.ImportDocument(ctx, "bad document", bad)
	return refusals
}

func TestRolePermissions_APlainCodeIsRefusedSayingWhatReplacedItOnEveryPath(t *testing.T) {
	for _, action := range []string{"add", "edit", "view"} {
		w := newRecordPermissionWorld(t)
		plain := "record." + w.record.Code + "." + action
		for path, err := range w.refusedOnEveryPath(t, []string{plain}) {
			t.Run(action+" on "+path, func(t *testing.T) {
				w.refusalSays(t, err, plain, plain+".*", plain+".<field>", w.record.Code)
			})
		}
	}
}

func TestRolePermissions_AFieldTheRecordLacksIsRefusedNamingTheRoleTheCodeAndTheRecordOnEveryPath(t *testing.T) {
	w := newRecordPermissionWorld(t)
	missing := w.fieldCode("edit", "nothing")
	for path, err := range w.refusedOnEveryPath(t, []string{missing}) {
		t.Run(path, func(t *testing.T) {
			w.refusalSays(t, err, missing, `record "`+w.record.Code+`"`, `no field "nothing"`)
		})
	}
	_, err := w.mgr.CreateRole(context.Background(), models.Role{DraftID: w.draftID, Name: "Clerk", Permissions: []string{missing}})
	w.refusalNames(t, err, "Clerk", missing)
}

func TestRolePermissions_FieldCodesStarsAndDeleteAreTakenOnEveryPath(t *testing.T) {
	w := newRecordPermissionWorld(t)
	ctx := context.Background()
	want := []string{w.fieldCode("edit", amountField), w.fieldCode("view", "*"), w.code("delete")}
	if _, err := w.mgr.CreateRole(ctx, models.Role{DraftID: w.draftID, Name: "Clerk", Permissions: want}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	role := w.roles[farmer]
	role.Permissions = want
	if _, err := w.mgr.UpdateRole(ctx, role); err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	if _, err := w.mgr.SetRolePermissions(ctx, w.roles[supplier].ID, want); err != nil {
		t.Fatalf("SetRolePermissions: %v", err)
	}
	pub, err := w.mgr.PublishAgency(ctx, w.draftID, "fields-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	imported, err := w.mgr.ImportDocument(ctx, "same document", pub.Document)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	roles, err := w.mgr.ListRoles(ctx, imported.Draft.ID, mwanachamaagency.RoleFilter{Name: "Clerk"})
	if err != nil || len(roles) != 1 || !reflect.DeepEqual(roles[0].Permissions, want) {
		t.Fatalf("imported Clerk = %+v, %v; want permissions %v", roles, err, want)
	}
}

func TestRolePermissions_TheStarIsGivenAndTakenForARecordWithNoFields(t *testing.T) {
	bare := models.Record{Code: "notes", Name: "Notes"}
	star := mwanachamaagency.RecordPermissionCode("notes", "add", "*")
	gave := false
	for _, p := range mwanachamaagency.RecordPermissionsOf(bare) {
		gave = gave || p.Code == star
	}
	if !gave {
		t.Fatalf("RecordPermissionsOf(no fields) = %+v, want %s", mwanachamaagency.RecordPermissionsOf(bare), star)
	}
	w := newRecordPermissionWorld(t)
	pub := publishedWithClerk(t, w, []string{"report.read"})
	doc := pub.Document
	doc.Records = append(append([]models.DocumentRecord(nil), doc.Records...), models.DocumentRecord{Code: "notes", Name: "Notes"})
	doc.Roles = append([]models.DocumentRole(nil), doc.Roles...)
	for i := range doc.Roles {
		if doc.Roles[i].Name == "Clerk" {
			doc.Roles[i].Permissions = []string{star}
		}
	}
	if _, err := w.mgr.ImportDocument(context.Background(), "bare record", doc); err != nil {
		t.Fatalf("ImportDocument with the star on a record with no fields: %v", err)
	}
}

func TestRolePermissions_TheFindingForARemovedFieldNamesTheRoleTheCodeAndTheField(t *testing.T) {
	w := newRecordPermissionWorld(t)
	ctx := context.Background()
	named := w.fieldCode("edit", amountField)
	role, err := w.mgr.CreateRole(ctx, models.Role{DraftID: w.draftID, Name: "Clerk", Permissions: []string{named}})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	record := w.record
	record.Questions = record.Questions[:1]
	if _, err := w.mgr.UpdateRecord(ctx, record); err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}
	issues, err := w.mgr.ListIssues(ctx, w.draftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	var found []models.Issue
	for _, issue := range issues {
		if issue.EntityID == role.ID {
			found = append(found, issue)
		}
	}
	if len(found) != 1 || found[0].Severity != models.IssueSeverityError {
		t.Fatalf("issues of the role = %+v, want one error", found)
	}
	for _, part := range []string{role.Code, "Clerk", named, `field "` + amountField + `"`, `record "` + w.record.Code + `"`} {
		if !strings.Contains(found[0].Message, part) {
			t.Errorf("finding = %q, want it to say %q", found[0].Message, part)
		}
	}
}
