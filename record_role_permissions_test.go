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
	want := []string{w.code("add"), "report.read", w.code("view")}
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
	role.Permissions = []string{w.code("edit"), "report.read"}
	if _, err := w.mgr.UpdateRole(ctx, role); err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	role.Permissions = []string{"report.read", "record.nothing.edit"}
	_, err := w.mgr.UpdateRole(ctx, role)
	w.refusalNames(t, err, farmer, "record.nothing.edit")
	kept, err := w.mgr.GetRole(ctx, role.ID)
	if err != nil || !reflect.DeepEqual(kept.Permissions, []string{w.code("edit"), "report.read"}) {
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
	_, err = w.mgr.CreateRole(ctx, models.Role{DraftID: other.ID, Name: "Clerk", Permissions: []string{w.code("add")}})
	w.refusalNames(t, err, "Clerk", w.code("add"))
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
	want := []string{w.code("add"), w.code("view"), "report.read"}
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
	pub := publishedWithClerk(t, w, []string{w.code("add"), "report.read"})

	imported, err := w.mgr.ImportDocument(ctx, "same document", pub.Document)
	if err != nil {
		t.Fatalf("ImportDocument of a document whose codes its records give: %v", err)
	}
	roles, err := w.mgr.ListRoles(ctx, imported.Draft.ID, mwanachamaagency.RoleFilter{})
	if err != nil || !reflect.DeepEqual(clerkOf(t, roles).Permissions, []string{w.code("add"), "report.read"}) {
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
	publishedWithClerk(t, w, []string{w.code("add"), w.code("edit"), w.code("delete"), w.code("view")})
}
