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
			if listErr != nil || len(drafts) != 0 {
				t.Errorf("a refused import kept %d drafts, %v", len(drafts), listErr)
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
