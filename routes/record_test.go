package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestRecordRoutes(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	ctx := t.Context()
	wf, _ := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftID, Name: "Group order"})
	coordinator, _ := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "Group coordinator"})
	farmer, _ := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "Farmer"})

	send := func(method, path, payload string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		serve(mgr, rec, httptest.NewRequest(method, path, strings.NewReader(payload)))
		return rec
	}

	created := send(http.MethodPost, "/records", `{"draft_id":"`+draftID+`","workflow_id":"`+wf.ID+`","name":"Payments","role_id":"`+coordinator.ID+`",`+
		`"form_name":"Payment","questions":[{"prompt":"Enter the amount paid","answer_type":"currency","required":true}]}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create record: status = %d, body = %s", created.Code, created.Body.String())
	}
	var record models.Record
	if err := json.Unmarshal(created.Body.Bytes(), &record); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if record.Code == "" || record.RoleID != coordinator.ID {
		t.Fatalf("created record = %+v", record)
	}

	if rec := send(http.MethodPost, "/records", `{"draft_id":"`+draftID+`","workflow_id":"`+wf.ID+`","name":"","role_id":"`+coordinator.ID+`"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("a record with no name: status = %d, want 400, body = %s", rec.Code, rec.Body.String())
	}
	if rec := send(http.MethodGet, "/records/no-such-record", ""); rec.Code != http.StatusNotFound {
		t.Errorf("a record that does not exist: status = %d, want 404", rec.Code)
	}

	listed := send(http.MethodGet, "/records?draft_id="+draftID, "")
	var records []models.Record
	if err := json.Unmarshal(listed.Body.Bytes(), &records); err != nil || len(records) != 1 || records[0].ID != record.ID {
		t.Fatalf("list records: status = %d, body = %s", listed.Code, listed.Body.String())
	}

	row := func(role string) string {
		return `{"draft_id":"` + draftID + `","record_id":"` + record.ID + `","role_id":"` + role + `","answers":{"Enter the amount paid":"1200"}}`
	}
	if rec := send(http.MethodPost, "/record-rows", row(farmer.ID)); rec.Code != http.StatusForbidden {
		t.Errorf("a row from a role that does not add rows: status = %d, want 403, body = %s", rec.Code, rec.Body.String())
	}
	if rec := send(http.MethodPost, "/record-rows", row(coordinator.ID)); rec.Code != http.StatusCreated {
		t.Fatalf("a row from the role that adds rows: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec := send(http.MethodPost, "/record-rows", strings.Replace(row(coordinator.ID), "1200", "plenty", 1)); rec.Code != http.StatusBadRequest {
		t.Errorf("a row whose answer does not fit: status = %d, want 400, body = %s", rec.Code, rec.Body.String())
	}

	rows := send(http.MethodGet, "/record-rows?draft_id="+draftID+"&record_id="+record.ID, "")
	var got []models.RecordRow
	if err := json.Unmarshal(rows.Body.Bytes(), &got); err != nil || len(got) != 1 || got[0].RecordCode != record.Code {
		t.Fatalf("list rows: status = %d, body = %s", rows.Code, rows.Body.String())
	}

	if rec := send(http.MethodDelete, "/records/"+record.ID, ""); rec.Code != http.StatusNoContent {
		t.Errorf("delete record: status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
