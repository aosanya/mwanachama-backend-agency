package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

var goneRecordRoleFields = map[string][]string{
	"role_id":        {"record.tender.add.*", "record.tender.add.<field>"},
	"edit_role_id":   {"record.tender.edit.*", "record.tender.edit.<field>"},
	"viewer_role_id": {"record.tender.view.*", "record.tender.view.<field>"},
	"delete_role_id": {"record.tender.delete"},
}

func recordDocument(t *testing.T, recordExtra, roleExtra string) models.AgencyDocument {
	t.Helper()
	raw := fmt.Sprintf(`{
		"roles": [{"id": "r1", "code": "R-1", "name": "Clerk"%s}],
		"workflows": [{"id": "w1", "code": "W-1", "name": "Tenders"}],
		"records": [{"id": "c1", "code": "tender", "key": "RC-1", "workflow_id": "w1", "name": "Tender", "form_name": "Tender",
			"questions": [{"prompt": "Closing date", "code": "closing_date", "answer_type": "date", "required": true}]%s}]
	}`, roleExtra, recordExtra)
	var doc models.AgencyDocument
	must(t, json.Unmarshal([]byte(raw), &doc))
	doc.Agency.Name = "Imp"
	return doc
}

func TestRecordRoleFields_NoModelCarriesThem(t *testing.T) {
	for _, shape := range []reflect.Type{reflect.TypeOf(models.Record{}), reflect.TypeOf(models.DocumentRecord{})} {
		for _, name := range []string{"RoleID", "EditRoleID", "DeleteRoleID", "ViewerRoleID"} {
			if _, found := shape.FieldByName(name); found {
				t.Errorf("%s still has the field %s", shape.Name(), name)
			}
		}
	}
	if _, found := reflect.TypeOf(models.Record{}).MethodByName("RoleOfRowAction"); found {
		t.Errorf("Record still has RoleOfRowAction")
	}
}

func TestRecordRoleFields_AnImportThatCarriesOneIsRefusedNamingTheFieldAndTheCodes(t *testing.T) {
	for field, codes := range goneRecordRoleFields {
		t.Run(field, func(t *testing.T) {
			mgr := newTestManager(t)
			before, listErr := mgr.ListDrafts(context.Background(), mwanachamaagency.DraftFilter{})
			must(t, listErr)
			doc := recordDocument(t, fmt.Sprintf(`, %q: "r1"`, field), "")
			_, err := mgr.ImportDocument(context.Background(), "imp", doc)
			if !errors.Is(err, mwanachamaagency.ErrInvalidRecord) {
				t.Fatalf("a record that carries %s: want ErrInvalidRecord, got %v", field, err)
			}
			for _, want := range append([]string{field}, codes...) {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("the refusal of %s does not name %q: %v", field, want, err)
				}
			}
			drafts, listErr := mgr.ListDrafts(context.Background(), mwanachamaagency.DraftFilter{})
			if listErr != nil || len(drafts) != len(before) {
				t.Errorf("a refused import kept %d drafts beside %d, %v", len(drafts), len(before), listErr)
			}
		})
	}
}

func TestRecordRoleFields_AnEmptyOneIsNotRefused(t *testing.T) {
	mgr := newTestManager(t)
	doc := recordDocument(t, `, "role_id": "", "viewer_role_id": "", "edit_role_id": "", "delete_role_id": ""`, "")
	if _, err := mgr.ImportDocument(context.Background(), "imp", doc); err != nil {
		t.Errorf("a record that carries the four fields empty: %v", err)
	}
}

func TestRecordRoleFields_ADocumentWithoutThemImports(t *testing.T) {
	mgr := newTestManager(t)
	out, err := mgr.ImportDocument(context.Background(), "imp", recordDocument(t, "", ""))
	if err != nil {
		t.Fatalf("a document with no role field on its record: %v", err)
	}
	records, err := mgr.ListRecords(context.Background(), out.Draft.ID, mwanachamaagency.RecordFilter{})
	if err != nil || len(records) != 1 || records[0].Code != "tender" {
		t.Errorf("records = %+v, %v", records, err)
	}
}

func TestRecordRoleFields_ADocumentWhoseRolesCarryFieldCodesImports(t *testing.T) {
	mgr := newTestManager(t)
	carried := `, "permissions": ["record.tender.add.*", "record.tender.edit.closing_date", "record.tender.view.*", "record.tender.delete"]`
	out, err := mgr.ImportDocument(context.Background(), "imp", recordDocument(t, "", carried))
	if err != nil {
		t.Fatalf("a document whose role carries record codes: %v", err)
	}
	roles, err := mgr.ListRoles(context.Background(), out.Draft.ID, mwanachamaagency.RoleFilter{})
	if err != nil || len(roles) != 1 || len(roles[0].Permissions) != 4 {
		t.Errorf("roles = %+v, %v", roles, err)
	}
}

func TestRecordRoleFields_ARecordBodyThatCarriesOneIsRefusedByCreateAndUpdate(t *testing.T) {
	for field, codes := range goneRecordRoleFields {
		t.Run(field, func(t *testing.T) {
			w := newStepWorld(t)
			ctx := context.Background()
			wf := w.workflow(t, "Tenders")
			body := fmt.Sprintf(`{"draft_id":%q,"workflow_id":%q,"name":"Tender","form_name":"Tender",`+
				`"questions":[{"prompt":"Closing date","answer_type":"date","required":true}],%q:%q}`,
				w.draftID, wf.ID, field, w.roles[coordinator].ID)
			var in models.Record
			must(t, json.Unmarshal([]byte(body), &in))
			_, err := w.mgr.CreateRecord(ctx, in)
			if !errors.Is(err, mwanachamaagency.ErrInvalidRecord) {
				t.Fatalf("create with %s: want ErrInvalidRecord, got %v", field, err)
			}
			for _, want := range append([]string{field}, codes...) {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("the refusal of %s does not name %q: %v", field, want, err)
				}
			}

			clean := strings.Replace(body, fmt.Sprintf(`,%q:%q`, field, w.roles[coordinator].ID), "", 1)
			var plain models.Record
			must(t, json.Unmarshal([]byte(clean), &plain))
			made, err := w.mgr.CreateRecord(ctx, plain)
			if err != nil {
				t.Fatalf("create without %s: %v", field, err)
			}
			var edited models.Record
			must(t, json.Unmarshal([]byte(body), &edited))
			edited.ID = made.ID
			if _, err := w.mgr.UpdateRecord(ctx, edited); !errors.Is(err, mwanachamaagency.ErrInvalidRecord) {
				t.Errorf("update with %s: want ErrInvalidRecord, got %v", field, err)
			}
		})
	}
}

func TestRecordRoleFields_AFieldAddCodeLetsARoleAddRowsAndOtherRecordCodesDoNot(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	w := o.world
	row := func(role models.Role) models.RecordRow {
		return models.RecordRow{DraftID: w.draftID, RecordID: o.record.ID, RoleID: role.ID, AddedBy: "member-1",
			Answers: map[string]string{"Select the farmer": "Wanjiru", "Enter the amount paid": "1200"}}
	}
	for name, code := range map[string]string{
		"view star": mwanachamaagency.RecordPermissionCode(o.record.Code, mwanachamaagency.RecordViewAction, mwanachamaagency.RecordFieldStar),
		"edit star": mwanachamaagency.RecordPermissionCode(o.record.Code, models.RowActionEdit, mwanachamaagency.RecordFieldStar),
		"delete":    mwanachamaagency.RecordPermissionCode(o.record.Code, models.RowActionDelete),
		"catalogue": "report.read",
	} {
		holder := w.roles[supplier]
		if _, err := w.mgr.SetRolePermissions(ctx, holder.ID, []string{code}); err != nil {
			t.Fatalf("%s: SetRolePermissions: %v", name, err)
		}
		if _, err := w.mgr.CreateRecordRow(ctx, row(holder)); !errors.Is(err, mwanachamaagency.ErrRecordRowRefused) {
			t.Errorf("a role that carries only %s adds a row: want ErrRecordRowRefused, got %v", name, err)
		}
	}
	field := fieldOf(t, o.record, "Enter the amount paid")
	named := mwanachamaagency.RecordPermissionCode(o.record.Code, models.RowActionAdd, field.Code)
	if _, err := w.mgr.SetRolePermissions(ctx, w.roles[supplier].ID, []string{named}); err != nil {
		t.Fatalf("SetRolePermissions: %v", err)
	}
	if _, err := w.mgr.CreateRecordRow(ctx, row(w.roles[supplier])); err != nil {
		t.Errorf("a role that carries %s adds a row: %v", named, err)
	}
}

func TestRecordRoleFields_AStoredPublicationThatCarriesOneStillReadsAndIsRefusedWhenForked(t *testing.T) {
	g := newGenWorld(t)
	ctx := context.Background()
	mgr := g.world.mgr
	pub, err := mgr.PublishAgency(ctx, g.world.draftID, "old-shape", "")
	must(t, err)
	var stored string
	must(t, g.db.Raw("select document from "+tableOf(t, g.sp, "agency_publication")+" where id = ?", pub.ID).Scan(&stored).Error)
	var document map[string]any
	must(t, json.Unmarshal([]byte(stored), &document))
	records := document["records"].([]any)
	records[0].(map[string]any)["role_id"] = g.world.roles[coordinator].ID
	rewritten, err := json.Marshal(document)
	must(t, err)
	must(t, g.db.Exec("update "+tableOf(t, g.sp, "agency_publication")+" set document = ? where id = ?", string(rewritten), pub.ID).Error)

	listed, err := mgr.ListPublications(ctx, mwanachamaagency.PublicationFilter{})
	if err != nil || len(listed) == 0 {
		t.Fatalf("a stored publication that carries role_id does not list: %v", err)
	}
	_, err = mgr.CreateDraft(ctx, "revisit", "old-shape", "publication")
	if !errors.Is(err, mwanachamaagency.ErrInvalidRecord) || !strings.Contains(err.Error(), "role_id") {
		t.Errorf("forking it: want ErrInvalidRecord naming role_id, got %v", err)
	}
}
